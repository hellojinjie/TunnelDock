import Combine
import Foundation

public protocol TunnelClock: Sendable {
    func sleep(for duration: Duration) async throws
}

public struct ContinuousTunnelClock: TunnelClock {
    public init() {}

    public func sleep(for duration: Duration) async throws {
        try await Task.sleep(for: duration)
    }
}

public enum TunnelManagerError: Error, Sendable, Equatable, CustomStringConvertible {
    case hostNotFound
    case configurationError(String)
    case localPortInUse(UInt16, owner: PortOwner)
    case portCheckFailed(String)
    case editingActiveTunnel
    case deletingActiveTunnel
    case runtimeNotFound
    case alreadyActive
    case applicationShuttingDown
    case connectionFailed(SSHUserError)

    public var description: String {
        switch self {
        case .hostNotFound: SSHUserError.hostNotFound.message
        case let .configurationError(message): message
        case let .localPortInUse(port, owner):
            owner == .tunnelDock
                ? "Local port is already in use."
                : "Local port \(port) is already in use. Choose another local port."
        case let .portCheckFailed(message): message
        case .editingActiveTunnel: "Disconnect the tunnel before editing forwarding settings."
        case .deletingActiveTunnel: "Disconnect the tunnel before deleting it."
        case .runtimeNotFound: "Tunnel runtime was not found."
        case .alreadyActive: "Tunnel is already active."
        case .applicationShuttingDown: "TunnelDock is quitting and cannot start a new tunnel."
        case let .connectionFailed(error): error.message
        }
    }
}

@MainActor
public final class TunnelManager: ObservableObject {
    @Published public private(set) var runtimes: [TunnelRuntimeSnapshot] = []

    private let repository: TunnelRepository
    private let portChecker: any PortChecking
    private let socketManager: any ControlSocketManaging
    private let processController: any SSHProcessControlling
    private let listenerProbe: any LocalListenerProbing
    private let tunnelClock: any TunnelClock
    private let now: @Sendable () -> Date
    private var hosts: [String: SSHHost]
    private var storage: [TunnelRuntimeID: ManagedRuntime] = [:]
    private var isShuttingDown = false

    public init(
        repository: TunnelRepository,
        portChecker: any PortChecking = SystemPortAvailabilityChecker(),
        socketManager: any ControlSocketManaging = ControlSocketManager(),
        processController: any SSHProcessControlling = SSHProcessController(),
        listenerProbe: any LocalListenerProbing = SystemLocalListenerProbe(),
        hosts: [SSHHost] = [],
        tunnelClock: any TunnelClock = ContinuousTunnelClock(),
        now: @escaping @Sendable () -> Date = Date.init
    ) {
        self.repository = repository
        self.portChecker = portChecker
        self.socketManager = socketManager
        self.processController = processController
        self.listenerProbe = listenerProbe
        self.tunnelClock = tunnelClock
        self.hosts = Dictionary(uniqueKeysWithValues: hosts.map { ($0.alias, $0) })
        self.now = now
    }

    public func updateHosts(_ hosts: [SSHHost]) {
        self.hosts = Dictionary(uniqueKeysWithValues: hosts.map { ($0.alias, $0) })
        for runtime in storage.values where runtime.state == .reconnecting {
            guard !hostIsAvailable(runtime.definition.hostAlias) else { continue }
            runtime.retryTask?.cancel()
            runtime.desiredConnection = false
            runtime.state = .failed
            runtime.lastError = SSHUserError.hostNotFound.message
            if let endpoint = runtime.endpoint {
                Task { await portChecker.release(endpoint) }
                runtime.endpoint = nil
            }
        }
        publish()
    }

    public func snapshot(id: TunnelRuntimeID) -> TunnelRuntimeSnapshot? {
        storage[id]?.snapshot
    }

    public func loadSavedDefinitions() async throws {
        let definitions = try await repository.load()
        for definition in definitions {
            let id = TunnelRuntimeID.saved(definition.id)
            if storage[id] == nil {
                storage[id] = ManagedRuntime(id: id, definition: definition, isTemporary: false)
            }
        }
        publish()
    }

    public func connectTemporary(
        hostAlias: String,
        input: TunnelInput
    ) async throws -> TunnelRuntimeID {
        guard !isShuttingDown else { throw TunnelManagerError.applicationShuttingDown }
        try validateHost(alias: hostAlias)
        let definition = try makeDefinition(hostAlias: hostAlias, input: input)
        let id = TunnelRuntimeID.temporary(definition.id)
        let runtime = ManagedRuntime(id: id, definition: definition, isTemporary: true)
        storage[id] = runtime
        do {
            try await runConnectionOperation(runtime)
            return id
        } catch {
            storage.removeValue(forKey: id)
            publish()
            throw error
        }
    }

    public func connectRecent(
        hostAlias: String,
        input: TunnelInput
    ) async throws -> TunnelRuntimeID {
        let temporaryID = try await connectTemporary(hostAlias: hostAlias, input: input)
        do {
            let savedID = try await saveAsRecent(id: temporaryID, input: input)
            return .saved(savedID)
        } catch {
            try? await disconnect(id: temporaryID)
            throw error
        }
    }

    private func saveAsRecent(id: TunnelRuntimeID, input: TunnelInput) async throws -> UUID {
        guard case .temporary = id,
              let runtime = storage[id], runtime.isTemporary
        else {
            throw TunnelManagerError.runtimeNotFound
        }
        if let existing = try await repository.findMatchingForward(
            hostAlias: runtime.definition.hostAlias,
            localAddress: runtime.definition.localAddress,
            localPort: runtime.definition.localPort,
            remoteHost: runtime.definition.remoteHost,
            remotePort: runtime.definition.remotePort
        ) {
            var updated = existing
            updated.webProtocol = runtime.definition.webProtocol
            if let name = try InputValidator.optionalName(input.name) {
                updated.name = name
            }
            updated.lastConnectedAt = now()
            updated.updatedAt = now()
            updated = try await repository.update(updated)
            storage.removeValue(forKey: id)
            runtime.definition = updated
            runtime.id = .saved(existing.id)
            runtime.isTemporary = false
            storage[runtime.id] = runtime
            publish()
            return existing.id
        }
        runtime.definition.lastConnectedAt = now()
        return try await saveTemporary(id: id, name: input.name)
    }

    public func connectSaved(id: UUID) async throws {
        guard !isShuttingDown else { throw TunnelManagerError.applicationShuttingDown }
        let runtimeID = TunnelRuntimeID.saved(id)
        guard let runtime = storage[runtimeID] else {
            throw TunnelManagerError.runtimeNotFound
        }
        try validateHost(alias: runtime.definition.hostAlias)
        try await runConnectionOperation(runtime)
    }

    public func saveTemporary(id: TunnelRuntimeID, name: String?) async throws -> UUID {
        guard case let .temporary(uuid) = id,
              let runtime = storage[id], runtime.isTemporary
        else {
            throw TunnelManagerError.runtimeNotFound
        }
        runtime.definition.name = try InputValidator.optionalName(name)
        runtime.definition.updatedAt = now()
        try await repository.save(runtime.definition)
        storage.removeValue(forKey: id)
        runtime.id = .saved(uuid)
        runtime.isTemporary = false
        storage[runtime.id] = runtime
        publish()
        return uuid
    }

    public func disconnect(id: TunnelRuntimeID) async throws {
        guard let runtime = storage[id] else { throw TunnelManagerError.runtimeNotFound }
        if let task = runtime.disconnectTask {
            await task.value
            return
        }
        let task = Task { @MainActor [weak self, weak runtime] in
            guard let self, let runtime else { return }
            await self.performDisconnect(runtime, id: id)
        }
        runtime.disconnectTask = task
        await task.value
        runtime.disconnectTask = nil
    }

    private func performDisconnect(_ runtime: ManagedRuntime, id: TunnelRuntimeID) async {
        runtime.desiredConnection = false
        runtime.generation &+= 1
        let operationTask = runtime.operationTask
        operationTask?.cancel()
        runtime.monitorTask?.cancel()
        runtime.monitorTask = nil
        let retryTask = runtime.retryTask
        runtime.retryTask?.cancel()
        runtime.retryTask = nil
        _ = try? await operationTask?.value
        runtime.operationTask = nil
        await retryTask?.value
        if let socket = runtime.socket {
            await processController.requestExit(alias: runtime.definition.hostAlias, socket: socket)
        }
        if let handle = runtime.handle {
            try? await tunnelClock.sleep(for: .seconds(2))
            if processController.isRunning(handle) {
                processController.terminate(handle)
            }
            try? await tunnelClock.sleep(for: .seconds(1))
            if processController.isRunning(handle) {
                processController.kill(handle)
            }
        }
        removeProcessResources(runtime)
        if let endpoint = runtime.endpoint {
            await portChecker.release(endpoint)
            runtime.endpoint = nil
        }
        runtime.state = .disconnected
        runtime.lastError = nil
        runtime.log.append("Disconnected.", at: now())
        if runtime.isTemporary {
            storage.removeValue(forKey: id)
        }
        publish()
    }

    public func rename(id: TunnelRuntimeID, name: String?) async throws {
        guard let runtime = storage[id] else { throw TunnelManagerError.runtimeNotFound }
        let normalized = try InputValidator.optionalName(name)
        if case let .saved(uuid) = id {
            runtime.definition = try await repository.rename(id: uuid, name: normalized)
        } else {
            runtime.definition.name = normalized
            runtime.definition.updatedAt = now()
        }
        publish()
    }

    public func edit(id: TunnelRuntimeID, input: TunnelInput) async throws {
        guard let runtime = storage[id] else { throw TunnelManagerError.runtimeNotFound }
        guard runtime.state == .disconnected || runtime.state == .failed else {
            throw TunnelManagerError.editingActiveTunnel
        }
        let specification = try ForwardSpecification(
            localAddress: input.localAddress,
            localPort: input.localPort,
            remoteHost: input.remoteHost,
            remotePort: input.remotePort
        )
        runtime.definition.name = try InputValidator.optionalName(input.name)
        runtime.definition.localAddress = specification.localAddress
        runtime.definition.localPort = specification.localPort
        runtime.definition.remoteHost = specification.remoteHost
        runtime.definition.remotePort = specification.remotePort
        runtime.definition.webProtocol = input.webProtocol
        runtime.definition.updatedAt = now()
        if case .saved = id {
            runtime.definition = try await repository.update(runtime.definition)
        }
        publish()
    }

    public func delete(id: TunnelRuntimeID) async throws {
        guard let runtime = storage[id] else { throw TunnelManagerError.runtimeNotFound }
        guard runtime.state == .disconnected || runtime.state == .failed else {
            throw TunnelManagerError.deletingActiveTunnel
        }
        if case let .saved(uuid) = id {
            try await repository.delete(id: uuid)
        }
        storage.removeValue(forKey: id)
        publish()
    }

    public func shutdownAll() async {
        guard !isShuttingDown else { return }
        isShuttingDown = true
        let disconnects = storage.values.compactMap(\.disconnectTask)
        for task in disconnects { await task.value }
        for runtime in storage.values { runtime.disconnectTask = nil }
        var operations: [Task<Void, Error>] = []
        var retries: [Task<Void, Never>] = []
        for runtime in storage.values {
            runtime.desiredConnection = false
            runtime.generation &+= 1
            if let task = runtime.operationTask {
                task.cancel()
                operations.append(task)
            }
            runtime.retryTask?.cancel()
            if let task = runtime.retryTask { retries.append(task) }
            runtime.retryTask = nil
            runtime.monitorTask?.cancel()
            runtime.monitorTask = nil
        }
        for task in operations { _ = try? await task.value }
        for task in retries { await task.value }
        for runtime in storage.values { runtime.operationTask = nil }

        let active = storage.values.compactMap { runtime -> (ManagedRuntime, SSHMasterHandle, URL)? in
            guard let handle = runtime.handle, let socket = runtime.socket else { return nil }
            return (runtime, handle, socket)
        }
        for (runtime, _, socket) in active {
            await processController.requestExit(alias: runtime.definition.hostAlias, socket: socket)
        }
        try? await tunnelClock.sleep(for: .seconds(2))
        for (_, handle, _) in active where processController.isRunning(handle) {
            processController.terminate(handle)
        }
        try? await tunnelClock.sleep(for: .seconds(1))
        for (_, handle, _) in active where processController.isRunning(handle) {
            processController.kill(handle)
        }
        for runtime in storage.values {
            removeProcessResources(runtime)
            if let endpoint = runtime.endpoint {
                await portChecker.release(endpoint)
                runtime.endpoint = nil
            }
            runtime.state = .disconnected
            runtime.lastError = nil
        }
        publish()
    }

    private func connect(_ runtime: ManagedRuntime) async throws {
        guard runtime.state == .disconnected || runtime.state == .failed else {
            throw TunnelManagerError.alreadyActive
        }
        let specification = try ForwardSpecification(
            localAddress: runtime.definition.localAddress,
            localPort: String(runtime.definition.localPort),
            remoteHost: runtime.definition.remoteHost,
            remotePort: String(runtime.definition.remotePort)
        )
        let endpoint = LocalEndpoint(
            address: specification.localAddress,
            port: specification.localPort
        )
        switch await portChecker.check(endpoint) {
        case .available:
            break
        case let .occupied(owner):
            throw TunnelManagerError.localPortInUse(endpoint.port, owner: owner)
        case let .unavailable(message):
            throw TunnelManagerError.portCheckFailed(message)
        }
        guard await portChecker.reserve(endpoint) else {
            throw TunnelManagerError.localPortInUse(endpoint.port, owner: .tunnelDock)
        }

        runtime.generation &+= 1
        let generation = runtime.generation
        runtime.desiredConnection = true
        runtime.state = .connecting
        runtime.endpoint = endpoint
        runtime.lastError = nil
        runtime.log.removeAll()
        runtime.log.append("Connecting...", at: now())
        publish()

        var operationSocket: URL?
        var operationHandle: SSHMasterHandle?
        do {
            let socket = try socketManager.allocateSocketURL()
            operationSocket = socket
            runtime.socket = socket
            let handle = try await processController.startMaster(
                alias: runtime.definition.hostAlias,
                socket: socket
            )
            operationHandle = handle
            try requireCurrent(runtime, generation: generation)
            runtime.handle = handle
            startMonitoring(runtime, generation: generation, handle: handle)

            guard await processController.waitUntilReady(
                alias: runtime.definition.hostAlias,
                socket: socket
            ) else {
                throw processController.readinessError(for: handle)
            }
            try requireCurrent(runtime, generation: generation)
            runtime.log.append("SSH transport established.", at: now())
            try await processController.addForward(
                alias: runtime.definition.hostAlias,
                socket: socket,
                specification: specification
            )
            try requireCurrent(runtime, generation: generation)
            runtime.log.append("Forward established.", at: now())
            guard await listenerProbe.waitUntilListening(endpoint, timeout: .seconds(2)) else {
                throw TunnelManagerError.connectionFailed(.localPortInUse)
            }
            try requireCurrent(runtime, generation: generation)
            runtime.state = .connected
            runtime.hasEverConnected = true
            runtime.retryAttempt = 0
            runtime.log.append("Connected.", at: now())
            await markConnected(runtime)
            publish()
        } catch {
            await cleanupOperationResources(
                alias: runtime.definition.hostAlias,
                socket: operationSocket,
                handle: operationHandle
            )
            guard storage[runtime.id] === runtime,
                  runtime.generation == generation,
                  runtime.desiredConnection
            else {
                throw mappedConnectionError(error)
            }
            runtime.desiredConnection = false
            runtime.state = .failed
            appendControllerStderr(from: error, to: runtime)
            runtime.lastError = userMessage(for: error)
            runtime.log.append(runtime.lastError ?? "Connection failed.", at: now())
            runtime.socket = nil
            runtime.handle = nil
            let endpoint = runtime.endpoint
            runtime.endpoint = nil
            publish()
            if let endpoint { await portChecker.release(endpoint) }
            throw mappedConnectionError(error)
        }
    }

    private func runConnectionOperation(_ runtime: ManagedRuntime) async throws {
        guard runtime.operationTask == nil, runtime.disconnectTask == nil else {
            throw TunnelManagerError.alreadyActive
        }
        let task = Task { @MainActor [weak self, weak runtime] in
            guard let self, let runtime else { return }
            try await self.connect(runtime)
        }
        runtime.operationTask = task
        defer {
            if runtime.operationTask != nil { runtime.operationTask = nil }
        }
        try await task.value
    }

    private func validateHost(alias: String) throws {
        guard let host = hosts[alias] else { throw TunnelManagerError.hostNotFound }
        if case let .configurationError(message) = host.availability {
            throw TunnelManagerError.configurationError(message)
        }
    }

    private func makeDefinition(hostAlias: String, input: TunnelInput) throws -> TunnelDefinition {
        let alias = try InputValidator.text(hostAlias, field: .hostAlias)
        let specification = try ForwardSpecification(
            localAddress: input.localAddress,
            localPort: input.localPort,
            remoteHost: input.remoteHost,
            remotePort: input.remotePort
        )
        let date = now()
        return TunnelDefinition(
            id: UUID(),
            hostAlias: alias,
            name: try InputValidator.optionalName(input.name),
            remoteHost: specification.remoteHost,
            remotePort: specification.remotePort,
            localAddress: specification.localAddress,
            localPort: specification.localPort,
            createdAt: date,
            updatedAt: date,
            webProtocol: input.webProtocol
        )
    }

    private func requireCurrent(_ runtime: ManagedRuntime, generation: UInt64) throws {
        guard runtime.generation == generation, runtime.desiredConnection else {
            throw CancellationError()
        }
    }

    private func startMonitoring(
        _ runtime: ManagedRuntime,
        generation: UInt64,
        handle: SSHMasterHandle
    ) {
        runtime.monitorTask?.cancel()
        runtime.monitorTask = Task { [weak self] in
            guard let self else { return }
            for await event in processController.events(for: handle) {
                if Task.isCancelled { return }
                await self.receive(event, runtime: runtime, generation: generation)
            }
        }
    }

    private func receive(
        _ event: ProcessEvent,
        runtime: ManagedRuntime,
        generation: UInt64
    ) async {
        guard storage[runtime.id] === runtime, runtime.generation == generation else { return }
        switch event {
        case let .stdout(data):
            append(data: data, prefix: "ssh stdout", to: runtime)
        case let .stderr(data):
            append(data: data, prefix: "ssh stderr", to: runtime)
        case let .terminated(status):
            runtime.log.append("SSH process exited: \(status)", at: now())
            if runtime.state == .connecting || runtime.state == .reconnecting {
                break
            }
            removeProcessResources(runtime)
            if runtime.state == .connected,
               runtime.desiredConnection,
               hostIsAvailable(runtime.definition.hostAlias) {
                runtime.state = .reconnecting
                runtime.lastError = nil
                scheduleRetry(runtime)
            } else {
                runtime.generation &+= 1
                runtime.desiredConnection = false
                runtime.state = .failed
                runtime.lastError = hosts[runtime.definition.hostAlias] == nil
                    ? SSHUserError.hostNotFound.message
                    : SSHUserError.processExitedUnexpectedly.message
                if let endpoint = runtime.endpoint {
                    await portChecker.release(endpoint)
                    runtime.endpoint = nil
                }
            }
        }
        publish()
    }

    private func scheduleRetry(_ runtime: ManagedRuntime) {
        guard runtime.desiredConnection, hostIsAvailable(runtime.definition.hostAlias) else {
            runtime.state = .failed
            runtime.desiredConnection = false
            runtime.lastError = SSHUserError.hostNotFound.message
            return
        }
        let delays: [Duration] = [.seconds(1), .seconds(2), .seconds(5), .seconds(10), .seconds(30)]
        let delay = delays[min(runtime.retryAttempt, delays.count - 1)]
        runtime.retryAttempt += 1
        runtime.log.append("Reconnecting in \(durationSeconds(delay)) seconds...", at: now())
        let generation = runtime.generation
        runtime.retryTask?.cancel()
        runtime.retryTask = Task { [weak self] in
            guard let self else { return }
            do {
                try await tunnelClock.sleep(for: delay)
            } catch {
                return
            }
            if Task.isCancelled { return }
            await self.performReconnect(runtime, expectedGeneration: generation)
        }
    }

    private func performReconnect(
        _ runtime: ManagedRuntime,
        expectedGeneration: UInt64
    ) async {
        guard storage[runtime.id] === runtime,
              runtime.generation == expectedGeneration,
              runtime.desiredConnection,
              hostIsAvailable(runtime.definition.hostAlias),
              let endpoint = runtime.endpoint
        else { return }

        runtime.generation &+= 1
        let generation = runtime.generation
        runtime.state = .reconnecting
        var operationSocket: URL?
        var operationHandle: SSHMasterHandle?
        do {
            let specification = try ForwardSpecification(
                localAddress: runtime.definition.localAddress,
                localPort: String(runtime.definition.localPort),
                remoteHost: runtime.definition.remoteHost,
                remotePort: String(runtime.definition.remotePort)
            )
            let socket = try socketManager.allocateSocketURL()
            operationSocket = socket
            runtime.socket = socket
            let handle = try await processController.startMaster(
                alias: runtime.definition.hostAlias,
                socket: socket
            )
            operationHandle = handle
            try requireCurrent(runtime, generation: generation)
            runtime.handle = handle
            startMonitoring(runtime, generation: generation, handle: handle)
            guard await processController.waitUntilReady(
                alias: runtime.definition.hostAlias,
                socket: socket
            ) else {
                throw processController.readinessError(for: handle)
            }
            try requireCurrent(runtime, generation: generation)
            try await processController.addForward(
                alias: runtime.definition.hostAlias,
                socket: socket,
                specification: specification
            )
            try requireCurrent(runtime, generation: generation)
            guard await listenerProbe.waitUntilListening(endpoint, timeout: .seconds(2)) else {
                throw TunnelManagerError.connectionFailed(.localPortInUse)
            }
            try requireCurrent(runtime, generation: generation)
            runtime.state = .connected
            runtime.retryAttempt = 0
            runtime.lastError = nil
            runtime.log.append("Connected.", at: now())
            await markConnected(runtime)
            publish()
        } catch {
            await cleanupOperationResources(
                alias: runtime.definition.hostAlias,
                socket: operationSocket,
                handle: operationHandle
            )
            guard storage[runtime.id] === runtime,
                  runtime.generation == generation
            else { return }
            runtime.socket = nil
            runtime.handle = nil
            guard runtime.desiredConnection, hostIsAvailable(runtime.definition.hostAlias) else {
                runtime.state = .failed
                runtime.desiredConnection = false
                runtime.lastError = SSHUserError.hostNotFound.message
                if let endpoint = runtime.endpoint {
                    await portChecker.release(endpoint)
                    runtime.endpoint = nil
                }
                publish()
                return
            }
            runtime.state = .reconnecting
            appendControllerStderr(from: error, to: runtime)
            runtime.lastError = userMessage(for: error)
            runtime.log.append(runtime.lastError ?? "Reconnect failed.", at: now())
            scheduleRetry(runtime)
            publish()
        }
    }

    private func markConnected(_ runtime: ManagedRuntime) async {
        let date = now()
        runtime.definition.lastConnectedAt = date
        guard case .saved = runtime.id else { return }
        runtime.definition.updatedAt = date
        if let updated = try? await repository.update(runtime.definition) {
            runtime.definition = updated
        }
    }

    private func hostIsAvailable(_ alias: String) -> Bool {
        guard let host = hosts[alias] else { return false }
        return host.availability == .available
    }

    private func durationSeconds(_ duration: Duration) -> Int {
        let components = duration.components
        return Int(components.seconds)
    }

    private func append(data: Data, prefix: String, to runtime: ManagedRuntime) {
        guard let value = String(data: data, encoding: .utf8) else { return }
        for line in value.split(whereSeparator: \Character.isNewline) {
            runtime.log.append("\(prefix): \(line)", at: now())
        }
    }

    private func removeProcessResources(_ runtime: ManagedRuntime) {
        if let socket = runtime.socket { try? socketManager.removeSocket(at: socket) }
        runtime.socket = nil
        runtime.handle = nil
    }

    private func cleanupOperationResources(
        alias: String,
        socket: URL?,
        handle: SSHMasterHandle?
    ) async {
        if let socket {
            await processController.requestExit(alias: alias, socket: socket)
        }
        if let handle {
            if processController.isRunning(handle) {
                processController.terminate(handle)
            }
            if processController.isRunning(handle) {
                processController.kill(handle)
            }
        }
        if let socket { try? socketManager.removeSocket(at: socket) }
    }

    private func mappedConnectionError(_ error: Error) -> Error {
        if let controllerError = error as? SSHProcessControllerError {
            return TunnelManagerError.connectionFailed(controllerError.userError)
        }
        return error
    }

    private func appendControllerStderr(from error: Error, to runtime: ManagedRuntime) {
        guard case let .commandFailed(_, stderr, _) = error as? SSHProcessControllerError,
              !stderr.isEmpty
        else { return }
        for line in stderr.split(whereSeparator: \Character.isNewline) {
            runtime.log.append("ssh stderr: \(line)", at: now())
        }
    }

    private func userMessage(for error: Error) -> String {
        if let error = error as? TunnelManagerError { return error.description }
        if let error = error as? SSHProcessControllerError { return error.userError.message }
        if let error = error as? InputValidationError { return error.description }
        return SSHUserError.connectionFailed.message
    }

    private func publish() {
        runtimes = storage.values
            .filter { !$0.isTemporary || $0.hasEverConnected }
            .map(\.snapshot)
            .sorted { lhs, rhs in
                if lhs.hostAlias != rhs.hostAlias { return lhs.hostAlias < rhs.hostAlias }
                return lhs.localPort < rhs.localPort
            }
    }
}

@MainActor
private final class ManagedRuntime {
    var id: TunnelRuntimeID
    var definition: TunnelDefinition
    var isTemporary: Bool
    var state: TunnelState = .disconnected
    var desiredConnection = false
    var hasEverConnected = false
    var retryAttempt = 0
    var generation: UInt64 = 0
    var handle: SSHMasterHandle?
    var socket: URL?
    var endpoint: LocalEndpoint?
    var lastError: String?
    var log = TunnelLogBuffer()
    var monitorTask: Task<Void, Never>?
    var retryTask: Task<Void, Never>?
    var operationTask: Task<Void, Error>?
    var disconnectTask: Task<Void, Never>?

    init(id: TunnelRuntimeID, definition: TunnelDefinition, isTemporary: Bool) {
        self.id = id
        self.definition = definition
        self.isTemporary = isTemporary
    }

    var snapshot: TunnelRuntimeSnapshot {
        TunnelRuntimeSnapshot(
            id: id,
            hostAlias: definition.hostAlias,
            name: definition.name,
            remoteHost: definition.remoteHost,
            remotePort: definition.remotePort,
            localAddress: definition.localAddress,
            localPort: definition.localPort,
            state: state,
            desiredConnection: desiredConnection,
            lastError: lastError,
            logLines: log.entries.map(\.formattedMessage),
            lastConnectedAt: definition.lastConnectedAt,
            webProtocol: definition.webProtocol
        )
    }
}

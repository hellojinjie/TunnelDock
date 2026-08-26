import Foundation
import TestSupport
import TunnelDockCore

enum TunnelManagerConnectionTests {
    static let all: [TestCase] = [
        TestCase("TunnelManagerConnectionTests.connectedTunnelIsAutomaticallySavedAsRecentAndRetainedAfterDisconnect") {
            try await withFixture { fixture in
                let recentID = try await fixture.manager.connectRecent(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "8888", localPort: "8888")
                )
                guard case let .saved(savedID) = recentID else {
                    throw TestFailure("Connected tunnel should become a saved recent tunnel")
                }

                try expectEqual(await fixture.manager.snapshot(id: .saved(savedID))?.state, .connected)
                try expectEqual(fixture.controller.launchCount, 1)
                try expectEqual(try await fixture.repository.load().count, 1)

                try await fixture.manager.disconnect(id: .saved(savedID))
                try expectEqual(await fixture.manager.snapshot(id: .saved(savedID))?.state, .disconnected)
                let reloadedManager = await TunnelManager(repository: fixture.repository)
                try await reloadedManager.loadSavedDefinitions()
                try expectEqual(await reloadedManager.snapshot(id: .saved(savedID))?.state, .disconnected)
                try expectEqual(fixture.controller.requestExitCount, 1)
                try expectEqual(fixture.controller.terminateCount, 1)
                try expectEqual(fixture.controller.killCount, 1)
            }
        },
        TestCase("TunnelManagerConnectionTests.invalidInputAndPortCollisionLaunchNoProcess") {
            try await withFixture(portAvailability: .occupied(.system)) { fixture in
                do {
                    _ = try await fixture.manager.connectTemporary(
                        hostAlias: "gpu",
                        input: TunnelInput(remotePort: "8888", localPort: "8888")
                    )
                    throw TestFailure("Expected local port collision")
                } catch let error as TunnelManagerError {
                    try expectEqual(error, .localPortInUse(8_888, owner: .system))
                }
                try expectEqual(fixture.controller.launchCount, 0)

                do {
                    _ = try await fixture.manager.connectTemporary(
                        hostAlias: "gpu",
                        input: TunnelInput(remotePort: "zero", localPort: "zero")
                    )
                    throw TestFailure("Expected invalid input")
                } catch is InputValidationError { }
                try expectEqual(fixture.controller.launchCount, 0)
            }
        },
        TestCase("TunnelManagerConnectionTests.missingAndConfigurationErrorHostsCannotConnect") {
            try await withFixture { fixture in
                await fixture.manager.updateHosts([])
                do {
                    _ = try await fixture.manager.connectTemporary(
                        hostAlias: "gpu",
                        input: TunnelInput(remotePort: "8888", localPort: "8888")
                    )
                    throw TestFailure("Expected missing host")
                } catch let error as TunnelManagerError {
                    try expectEqual(error, .hostNotFound)
                }

                await fixture.manager.updateHosts([host(availability: .configurationError("bad"))])
                do {
                    _ = try await fixture.manager.connectTemporary(
                        hostAlias: "gpu",
                        input: TunnelInput(remotePort: "8888", localPort: "8888")
                    )
                    throw TestFailure("Expected configuration error")
                } catch let error as TunnelManagerError {
                    try expectEqual(error, .configurationError("bad"))
                }
                try expectEqual(fixture.controller.launchCount, 0)
            }
        },
        TestCase("TunnelManagerConnectionTests.runningTunnelCanRenameButCannotEditOrDelete") {
            try await withFixture { fixture in
                let id = try await fixture.manager.connectRecent(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "8888", localPort: "8888")
                )

                try await fixture.manager.rename(id: id, name: "Jupyter")
                try expectEqual(await fixture.manager.snapshot(id: id)?.name, "Jupyter")
                try expectEqual(try await fixture.repository.load().first?.name, "Jupyter")
                try await expectManagerError(.editingActiveTunnel) {
                    try await fixture.manager.edit(
                        id: id,
                        input: TunnelInput(remotePort: "6006", localPort: "6006")
                    )
                }
                try await expectManagerError(.deletingActiveTunnel) {
                    try await fixture.manager.delete(id: id)
                }
            }
        },
        TestCase("TunnelManagerConnectionTests.disconnectRemovesTemporaryRuntime") {
            try await withFixture { fixture in
                let id = try await fixture.manager.connectTemporary(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "8888", localPort: "8888")
                )

                try await fixture.manager.disconnect(id: id)

                try expectEqual(await fixture.manager.snapshot(id: id), nil)
            }
        },
        TestCase("TunnelManagerConnectionTests.disconnectDuringStartLeavesDisconnectedAndCleansLateProcess") {
            let directory = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockStaleConnectTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            defer { try? FileManager.default.removeItem(at: directory) }
            let definition = TunnelDefinition(
                id: UUID(), hostAlias: "gpu", name: nil,
                remoteHost: "127.0.0.1", remotePort: 8_888,
                localAddress: "127.0.0.1", localPort: 8_888,
                createdAt: .distantPast, updatedAt: .distantPast
            )
            let repository = TunnelRepository(fileURL: directory.appending(path: "saved-tunnels.json"))
            try await repository.replaceAll([definition])
            let controller = BlockingStartProcessController()
            let manager = await TunnelManager(
                repository: repository,
                portChecker: StubPortChecker(availability: .available),
                socketManager: StubControlSocketManager(directory: directory),
                processController: controller,
                listenerProbe: AlwaysListeningProbe(),
                hosts: [host()],
                tunnelClock: NoWaitTunnelClock()
            )
            try await manager.loadSavedDefinitions()

            let connectTask = Task { try await manager.connectSaved(id: definition.id) }
            for _ in 0..<2_000 where !controller.hasStarted {
                try await Task.sleep(for: .milliseconds(1))
            }
            guard controller.hasStarted else { throw TestFailure("Start did not suspend") }
            let disconnectTask = Task { try await manager.disconnect(id: .saved(definition.id)) }
            await Task.yield()
            controller.resumeStart()
            _ = try? await connectTask.value
            try await disconnectTask.value

            try expectEqual(await manager.snapshot(id: .saved(definition.id))?.state, .disconnected)
            try expectEqual(controller.terminateCount, 1)
            try expectEqual(controller.killCount, 1)
        },
        TestCase("TunnelManagerConnectionTests.recentConnectionsReuseExistingDefinitionInsteadOfDuplicating") {
            try await withFixture { fixture in
                let firstID = try await fixture.manager.connectRecent(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "8888", localPort: "8888")
                )
                try await fixture.manager.disconnect(id: firstID)

                let secondID = try await fixture.manager.connectRecent(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "8888", localPort: "8888", webProtocol: .https)
                )

                guard case let .saved(firstSaved) = firstID,
                      case let .saved(secondSaved) = secondID
                else {
                    throw TestFailure("Connected tunnels should be saved recent tunnels")
                }
                try expectEqual(secondSaved, firstSaved)
                try expectEqual(try await fixture.repository.load().count, 1)
                try expectEqual(try await fixture.repository.load().first?.webProtocol, .https)
                try expectNotEqual(try await fixture.repository.load().first?.lastConnectedAt, nil)

                let differentPort = try await fixture.manager.connectRecent(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "9090", localPort: "9090")
                )
                guard case let .saved(differentSaved) = differentPort else {
                    throw TestFailure("Connected tunnel should be a saved recent tunnel")
                }
                try expectNotEqual(differentSaved, firstSaved)
                try expectEqual(try await fixture.repository.load().count, 2)
            }
        },
        TestCase("TunnelManagerConnectionTests.concurrentDisconnectsShareOneCleanupOperation") {
            let clock = BlockingDisconnectClock()
            try await withFixture(clock: clock) { fixture in
                let runtimeID = try await fixture.manager.connectRecent(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "8888", localPort: "8888")
                )

                let first = Task { try await fixture.manager.disconnect(id: runtimeID) }
                let second = Task { try await fixture.manager.disconnect(id: runtimeID) }
                for _ in 0..<2_000 where await clock.sleepCount < 1 {
                    try await Task.sleep(for: .milliseconds(1))
                }
                let exitCountBeforeResume = fixture.controller.requestExitCount
                await clock.resumeAll()
                for _ in 0..<2_000 where await clock.sleepCount < 2 {
                    try await Task.sleep(for: .milliseconds(1))
                }
                await clock.resumeAll()
                _ = try? await first.value
                _ = try? await second.value

                try expectEqual(exitCountBeforeResume, 1)
                try expectEqual(fixture.controller.requestExitCount, 1)
            }
        },
    ]

    private static func host(availability: SSHHostAvailability = .available) -> SSHHost {
        SSHHost(
            alias: "gpu", hostname: "gpu.internal", user: "tester", port: 22,
            configOrder: 0, availability: availability
        )
    }

    private static func withFixture(
        portAvailability: PortAvailability = .available,
        clock: any TunnelClock = NoWaitTunnelClock(),
        _ body: (ManagerFixture) async throws -> Void
    ) async throws {
        let directory = FileManager.default.temporaryDirectory
            .appending(path: "TunnelDockManagerTests-\(UUID().uuidString)", directoryHint: .isDirectory)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let repository = TunnelRepository(fileURL: directory.appending(path: "saved-tunnels.json"))
        let controller = ImmediateSSHProcessController()
        let manager = await TunnelManager(
            repository: repository,
            portChecker: StubPortChecker(availability: portAvailability),
            socketManager: StubControlSocketManager(directory: directory),
            processController: controller,
            listenerProbe: AlwaysListeningProbe(),
            hosts: [host()],
            tunnelClock: clock
        )
        try await body(ManagerFixture(manager: manager, repository: repository, controller: controller))
    }

    private static func expectManagerError(
        _ expected: TunnelManagerError,
        operation: () async throws -> Void
    ) async throws {
        do {
            try await operation()
            throw TestFailure("Expected \(expected)")
        } catch let error as TunnelManagerError {
            try expectEqual(error, expected)
        }
    }
}

private struct ManagerFixture: Sendable {
    let manager: TunnelManager
    let repository: TunnelRepository
    let controller: ImmediateSSHProcessController
}

private actor StubPortChecker: PortChecking {
    private let availability: PortAvailability
    private var reservations: Set<LocalEndpoint> = []

    init(availability: PortAvailability) { self.availability = availability }
    func check(_ endpoint: LocalEndpoint) -> PortAvailability {
        reservations.contains(endpoint) ? .occupied(.tunnelDock) : availability
    }
    func reserve(_ endpoint: LocalEndpoint) -> Bool { reservations.insert(endpoint).inserted }
    func release(_ endpoint: LocalEndpoint) { reservations.remove(endpoint) }
    func isReserved(_ endpoint: LocalEndpoint) -> Bool { reservations.contains(endpoint) }
}

private struct StubControlSocketManager: ControlSocketManaging {
    let directory: URL
    func prepareDirectory() throws -> URL { directory }
    func allocateSocketURL() throws -> URL { directory.appending(path: "manager.sock") }
    func removeSocket(at url: URL) throws {}
    func removeStaleSockets() throws {}
}

private struct AlwaysListeningProbe: LocalListenerProbing {
    func waitUntilListening(_ endpoint: LocalEndpoint, timeout: Duration) async -> Bool { true }
}

private struct NoWaitTunnelClock: TunnelClock {
    func sleep(for duration: Duration) async throws {}
}

private actor BlockingDisconnectClock: TunnelClock {
    private(set) var sleepCount = 0
    private var continuations: [CheckedContinuation<Void, Never>] = []

    func sleep(for duration: Duration) async throws {
        sleepCount += 1
        await withCheckedContinuation { continuations.append($0) }
    }

    func resumeAll() {
        let pending = continuations
        continuations.removeAll()
        pending.forEach { $0.resume() }
    }
}

private final class ManagerRunningProcess: RunningProcess, @unchecked Sendable {
    let processIdentifier: Int32 = 7_777
    let isRunning = true
    func events() -> AsyncStream<ProcessEvent> { AsyncStream { _ in } }
    func terminate() {}
    func interruptWithKill() {}
}

private final class ImmediateSSHProcessController: SSHProcessControlling, @unchecked Sendable {
    private let lock = NSLock()
    private var launches = 0
    private var exits = 0
    private var terminations = 0
    private var kills = 0

    var launchCount: Int { lock.withLock { launches } }
    var requestExitCount: Int { lock.withLock { exits } }
    var terminateCount: Int { lock.withLock { terminations } }
    var killCount: Int { lock.withLock { kills } }

    func startMaster(alias: String, socket: URL) async throws -> SSHMasterHandle {
        lock.withLock { launches += 1 }
        return SSHMasterHandle(alias: alias, socket: socket, process: ManagerRunningProcess())
    }
    func waitUntilReady(alias: String, socket: URL) async -> Bool { true }
    func addForward(alias: String, socket: URL, specification: ForwardSpecification) async throws {}
    func requestExit(alias: String, socket: URL) async { lock.withLock { exits += 1 } }
    func events(for handle: SSHMasterHandle) -> AsyncStream<ProcessEvent> { AsyncStream { _ in } }
    func terminate(_ handle: SSHMasterHandle) { lock.withLock { terminations += 1 } }
    func kill(_ handle: SSHMasterHandle) { lock.withLock { kills += 1 } }
    func isRunning(_ handle: SSHMasterHandle) -> Bool { true }
}

private final class BlockingStartProcessController: SSHProcessControlling, @unchecked Sendable {
    private let lock = NSLock()
    private var startContinuation: CheckedContinuation<Void, Never>?
    private var started = false
    private var terminations = 0
    private var kills = 0

    var hasStarted: Bool { lock.withLock { started } }
    var terminateCount: Int { lock.withLock { terminations } }
    var killCount: Int { lock.withLock { kills } }

    func startMaster(alias: String, socket: URL) async throws -> SSHMasterHandle {
        await withCheckedContinuation { continuation in
            lock.withLock {
                started = true
                startContinuation = continuation
            }
        }
        return SSHMasterHandle(alias: alias, socket: socket, process: ManagerRunningProcess())
    }

    func resumeStart() {
        let continuation = lock.withLock { () -> CheckedContinuation<Void, Never>? in
            let value = startContinuation
            startContinuation = nil
            return value
        }
        continuation?.resume()
    }

    func waitUntilReady(alias: String, socket: URL) async -> Bool { true }
    func addForward(alias: String, socket: URL, specification: ForwardSpecification) async throws {}
    func requestExit(alias: String, socket: URL) async {}
    func events(for handle: SSHMasterHandle) -> AsyncStream<ProcessEvent> { AsyncStream { _ in } }
    func terminate(_ handle: SSHMasterHandle) { lock.withLock { terminations += 1 } }
    func kill(_ handle: SSHMasterHandle) { lock.withLock { kills += 1 } }
    func isRunning(_ handle: SSHMasterHandle) -> Bool { true }
}

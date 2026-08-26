import Foundation
import TestSupport
import TunnelDockCore

enum TunnelManagerRecoveryTests {
    static let all: [TestCase] = [
        TestCase("TunnelManagerRecoveryTests.firstConnectionFailureDoesNotRetry") {
            try await withRecoveryFixture(readiness: [false]) { fixture in
                do {
                    try await fixture.manager.connectSaved(id: fixture.definition.id)
                    throw TestFailure("Expected first connection failure")
                } catch is TunnelManagerError { }

                try expectEqual(await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.state, .failed)
                try expectEqual(await fixture.clock.sleeps, [])
            }
        },
        TestCase("TunnelManagerRecoveryTests.establishedFailureUsesBackoffAndResetsAfterSuccess") {
            try await withRecoveryFixture(readiness: [true, false, false, false, false, false, true]) { fixture in
                try await fixture.manager.connectSaved(id: fixture.definition.id)
                fixture.controller.emitToLatest(.terminated(255))
                let expected: [Duration] = [.seconds(1), .seconds(2), .seconds(5), .seconds(10), .seconds(30), .seconds(30)]

                for (index, delay) in expected.enumerated() {
                    try await eventually {
                        await fixture.clock.sleeps.count == index + 1
                    }
                    try expectEqual(await fixture.clock.sleeps[index], delay)
                    await fixture.clock.resumeNext()
                }
                try await eventually {
                    await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.state == .connected
                }

                fixture.controller.emitToLatest(.terminated(255))
                try await eventually { await fixture.clock.sleeps.count == expected.count + 1 }
                try expectEqual(await fixture.clock.sleeps.last, .seconds(1))
                await fixture.clock.resumeNext()
            }
        },
        TestCase("TunnelManagerRecoveryTests.temporaryTunnelRemainsVisibleWhileReconnecting") {
            try await withRecoveryFixture(readiness: [true, true]) { fixture in
                let id = try await fixture.manager.connectTemporary(
                    hostAlias: "gpu",
                    input: TunnelInput(remotePort: "9999", localPort: "9999")
                )
                fixture.controller.emitToLatest(.terminated(255))

                try await eventually { await fixture.clock.sleeps.count == 1 }
                let visibleState = await fixture.manager.runtimes.first { $0.id == id }?.state
                try expectEqual(visibleState, .reconnecting)
                await fixture.clock.resumeNext()
                try await eventually { await fixture.manager.snapshot(id: id)?.state == .connected }
            }
        },
        TestCase("TunnelManagerRecoveryTests.processExitDuringReconnectContinuesBackoff") {
            try await withRecoveryFixture(
                readiness: [true, false, true],
                terminateDuringReadinessCalls: [2]
            ) { fixture in
                try await fixture.manager.connectSaved(id: fixture.definition.id)
                fixture.controller.emitToLatest(.terminated(255))
                try await eventually { await fixture.clock.sleeps.count == 1 }
                await fixture.clock.resumeNext()

                try await eventually { await fixture.clock.sleeps.count == 2 }
                let retrying = await fixture.manager.snapshot(id: .saved(fixture.definition.id))
                try expectEqual(retrying?.state, .reconnecting)
                try expectEqual(retrying?.desiredConnection, true)

                await fixture.clock.resumeNext()
                try await eventually {
                    await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.state == .connected
                }
            }
        },
        TestCase("TunnelManagerRecoveryTests.hostRemovalKeepsLiveTunnelButPreventsReconnectAfterExit") {
            try await withRecoveryFixture(readiness: [true]) { fixture in
                try await fixture.manager.connectSaved(id: fixture.definition.id)

                await fixture.manager.updateHosts([])
                try expectEqual(await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.state, .connected)
                fixture.controller.emitToLatest(.terminated(255))

                try await eventually {
                    await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.state == .failed
                }
                try expectEqual(await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.desiredConnection, false)
                try expectEqual(await fixture.clock.sleeps, [])
            }
        },
        TestCase("TunnelManagerRecoveryTests.remoteServiceRefusalOnlyAppendsLog") {
            try await withRecoveryFixture(readiness: [true]) { fixture in
                try await fixture.manager.connectSaved(id: fixture.definition.id)

                fixture.controller.emitToLatest(.stderr(Data("connect failed: Connection refused".utf8)))

                try await eventually {
                    await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.logLines
                        .contains(where: { $0.contains("Connection refused") }) == true
                }
                try expectEqual(await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.state, .connected)
            }
        },
        TestCase("TunnelManagerRecoveryTests.forwardFailurePreservesRawSSHStderr") {
            let raw = "bind [127.0.0.1]:8888: Address already in use"
            try await withRecoveryFixture(
                readiness: [true],
                forwardError: .commandFailed(
                    userError: .localPortInUse,
                    stderr: raw,
                    exitStatus: 255
                )
            ) { fixture in
                _ = try? await fixture.manager.connectSaved(id: fixture.definition.id)
                let lines = await fixture.manager.snapshot(id: .saved(fixture.definition.id))?.logLines ?? []
                try expectEqual(lines.contains(where: { $0.contains(raw) }), true)
            }
        },
        TestCase("TunnelManagerRecoveryTests.shutdownDisablesReconnectAndCleansRuntime") {
            try await withRecoveryFixture(readiness: [true], clock: ImmediateTunnelClock()) { fixture in
                try await fixture.manager.connectSaved(id: fixture.definition.id)

                await fixture.manager.shutdownAll()

                let snapshot = await fixture.manager.snapshot(id: .saved(fixture.definition.id))
                try expectEqual(snapshot?.state, .disconnected)
                try expectEqual(snapshot?.desiredConnection, false)
                try expectEqual(fixture.controller.requestExitCount, 1)
                try expectEqual(fixture.controller.terminateCount, 1)
                try expectEqual(fixture.controller.killCount, 1)
            }
        },
        TestCase("TunnelManagerRecoveryTests.idleShutdownRejectsNewConnectionsWithoutWaiting") {
            try await withRecoveryFixture(readiness: [true]) { fixture in
                await fixture.manager.shutdownAll()

                var observedError: TunnelManagerError?
                do {
                    try await fixture.manager.connectSaved(id: fixture.definition.id)
                } catch let error as TunnelManagerError {
                    observedError = error
                }
                try expectEqual(observedError, .applicationShuttingDown)
                try expectEqual(await fixture.clock.sleeps, [])
                try expectEqual(fixture.controller.requestExitCount, 0)
            }
        },
    ]

    private static func withRecoveryFixture(
        readiness: [Bool],
        terminateDuringReadinessCalls: Set<Int> = [],
        forwardError: SSHProcessControllerError? = nil,
        clock suppliedClock: (any TunnelClock)? = nil,
        _ body: (RecoveryFixture) async throws -> Void
    ) async throws {
        let directory = FileManager.default.temporaryDirectory
            .appending(path: "TunnelDockRecoveryTests-\(UUID().uuidString)", directoryHint: .isDirectory)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let definition = TunnelDefinition(
            id: UUID(), hostAlias: "gpu", name: "Jupyter",
            remoteHost: "127.0.0.1", remotePort: 8_888,
            localAddress: "127.0.0.1", localPort: 8_888,
            createdAt: .distantPast, updatedAt: .distantPast
        )
        let repository = TunnelRepository(fileURL: directory.appending(path: "saved-tunnels.json"))
        try await repository.replaceAll([definition])
        let controller = RecoveryProcessController(
            readiness: readiness,
            terminateDuringReadinessCalls: terminateDuringReadinessCalls,
            forwardError: forwardError
        )
        let manualClock = suppliedClock as? ManualTunnelClock ?? ManualTunnelClock()
        let clock: any TunnelClock = suppliedClock ?? manualClock
        let manager = await TunnelManager(
            repository: repository,
            portChecker: RecoveryPortChecker(),
            socketManager: RecoverySocketManager(directory: directory),
            processController: controller,
            listenerProbe: RecoveryListenerProbe(),
            hosts: [SSHHost(alias: "gpu", hostname: "gpu", user: "tester", port: 22, configOrder: 0, availability: .available)],
            tunnelClock: clock
        )
        try await manager.loadSavedDefinitions()
        try await body(RecoveryFixture(
            manager: manager,
            definition: definition,
            controller: controller,
            clock: manualClock
        ))
    }

    private static func eventually(
        _ predicate: @escaping @Sendable () async -> Bool
    ) async throws {
        for _ in 0..<2_000 {
            if await predicate() { return }
            try await Task.sleep(for: .milliseconds(1))
        }
        throw TestFailure("Condition was not reached")
    }
}

private struct RecoveryFixture: Sendable {
    let manager: TunnelManager
    let definition: TunnelDefinition
    let controller: RecoveryProcessController
    let clock: ManualTunnelClock
}

private actor ManualTunnelClock: TunnelClock {
    private(set) var sleeps: [Duration] = []
    private var continuations: [CheckedContinuation<Void, Never>] = []

    func sleep(for duration: Duration) async throws {
        sleeps.append(duration)
        await withCheckedContinuation { continuations.append($0) }
    }

    func resumeNext() {
        guard !continuations.isEmpty else { return }
        continuations.removeFirst().resume()
    }
}

private struct ImmediateTunnelClock: TunnelClock {
    func sleep(for duration: Duration) async throws {}
}

private actor RecoveryPortChecker: PortChecking {
    private var reservations: Set<LocalEndpoint> = []
    func check(_ endpoint: LocalEndpoint) -> PortAvailability {
        reservations.contains(endpoint) ? .occupied(.tunnelDock) : .available
    }
    func reserve(_ endpoint: LocalEndpoint) -> Bool { reservations.insert(endpoint).inserted }
    func release(_ endpoint: LocalEndpoint) { reservations.remove(endpoint) }
    func isReserved(_ endpoint: LocalEndpoint) -> Bool { reservations.contains(endpoint) }
}

private struct RecoverySocketManager: ControlSocketManaging {
    let directory: URL
    func prepareDirectory() throws -> URL { directory }
    func allocateSocketURL() throws -> URL { directory.appending(path: "\(UUID().uuidString).sock") }
    func removeSocket(at url: URL) throws {}
    func removeStaleSockets() throws {}
}

private struct RecoveryListenerProbe: LocalListenerProbing {
    func waitUntilListening(_ endpoint: LocalEndpoint, timeout: Duration) async -> Bool { true }
}

private final class RecoveryRunningProcess: RunningProcess, @unchecked Sendable {
    let processIdentifier: Int32
    let isRunning = true
    init(processIdentifier: Int32) { self.processIdentifier = processIdentifier }
    func events() -> AsyncStream<ProcessEvent> { AsyncStream { _ in } }
    func terminate() {}
    func interruptWithKill() {}
}

private final class RecoveryProcessController: SSHProcessControlling, @unchecked Sendable {
    private let lock = NSLock()
    private var readiness: [Bool]
    private var nextPID: Int32 = 9_000
    private var streams: [Int32: AsyncStream<ProcessEvent>] = [:]
    private var continuations: [Int32: AsyncStream<ProcessEvent>.Continuation] = [:]
    private var latestPID: Int32?
    private var exitRequests = 0
    private var terminations = 0
    private var kills = 0
    private var readinessCallCount = 0
    private let terminateDuringReadinessCalls: Set<Int>
    private let forwardError: SSHProcessControllerError?

    init(
        readiness: [Bool],
        terminateDuringReadinessCalls: Set<Int> = [],
        forwardError: SSHProcessControllerError? = nil
    ) {
        self.readiness = readiness
        self.terminateDuringReadinessCalls = terminateDuringReadinessCalls
        self.forwardError = forwardError
    }
    var requestExitCount: Int { lock.withLock { exitRequests } }
    var terminateCount: Int { lock.withLock { terminations } }
    var killCount: Int { lock.withLock { kills } }

    func startMaster(alias: String, socket: URL) async throws -> SSHMasterHandle {
        let pid = lock.withLock { () -> Int32 in
            nextPID += 1
            let pid = nextPID
            let pair = AsyncStream.makeStream(of: ProcessEvent.self)
            streams[pid] = pair.stream
            continuations[pid] = pair.continuation
            latestPID = pid
            return pid
        }
        return SSHMasterHandle(alias: alias, socket: socket, process: RecoveryRunningProcess(processIdentifier: pid))
    }
    func waitUntilReady(alias: String, socket: URL) async -> Bool {
        let result = lock.withLock { () -> (Bool, AsyncStream<ProcessEvent>.Continuation?) in
            readinessCallCount += 1
            let shouldTerminate = terminateDuringReadinessCalls.contains(readinessCallCount)
            let continuation = shouldTerminate ? latestPID.flatMap { continuations[$0] } : nil
            return (readiness.isEmpty ? true : readiness.removeFirst(), continuation)
        }
        result.1?.yield(.terminated(255))
        await Task.yield()
        return result.0
    }
    func addForward(alias: String, socket: URL, specification: ForwardSpecification) async throws {
        if let forwardError { throw forwardError }
    }
    func requestExit(alias: String, socket: URL) async { lock.withLock { exitRequests += 1 } }
    func events(for handle: SSHMasterHandle) -> AsyncStream<ProcessEvent> {
        lock.withLock { streams[handle.processIdentifier] ?? AsyncStream { $0.finish() } }
    }
    func terminate(_ handle: SSHMasterHandle) { lock.withLock { terminations += 1 } }
    func kill(_ handle: SSHMasterHandle) { lock.withLock { kills += 1 } }
    func isRunning(_ handle: SSHMasterHandle) -> Bool { true }

    func emitToLatest(_ event: ProcessEvent) {
        lock.withLock {
            guard let latestPID else { return }
            continuations[latestPID]?.yield(event)
        }
    }
}

import Foundation
import TestSupport
import TunnelDockCore

enum SSHProcessControllerTests {
    static let all: [TestCase] = [
        TestCase("SSHProcessControllerTests.buildsExactMasterAndForwardArguments") {
            let builder = SSHCommandBuilder()
            let socket = URL(fileURLWithPath: "/tmp/tunneldock-501/abc.sock")
            let specification = try ForwardSpecification(
                localAddress: "127.0.0.1",
                localPort: "18888",
                remoteHost: "127.0.0.1",
                remotePort: "8888"
            )

            try expectEqual(builder.masterArguments(alias: "gpu", socket: socket), [
                "-M", "-S", socket.path, "-N", "-T", "-n",
                "-o", "ControlPersist=no",
                "-o", "ClearAllForwardings=yes",
                "-o", "ExitOnForwardFailure=yes",
                "-o", "ServerAliveInterval=15",
                "-o", "ServerAliveCountMax=3",
                "-o", "BatchMode=yes",
                "-o", "StrictHostKeyChecking=yes",
                "gpu",
            ])
            try expectEqual(
                builder.forwardArguments(alias: "gpu", socket: socket, specification: specification),
                ["-S", socket.path, "-O", "forward", "-L", "127.0.0.1:18888:127.0.0.1:8888", "gpu"]
            )
            try expectEqual(builder.checkArguments(alias: "gpu", socket: socket), [
                "-S", socket.path, "-O", "check", "gpu",
            ])
            try expectEqual(builder.exitArguments(alias: "gpu", socket: socket), [
                "-S", socket.path, "-O", "exit", "gpu",
            ])
        },
        TestCase("SSHProcessControllerTests.startsMasterOnlyThroughUsrBinSSH") {
            let launcher = RecordingProcessLauncher()
            let controller = SSHProcessController(
                executor: QueueProcessExecutor(results: []),
                launcher: launcher,
                clock: ImmediateControlClock()
            )
            let socket = URL(fileURLWithPath: "/tmp/tunneldock-501/abc.sock")

            let handle = try await controller.startMaster(alias: "gpu", socket: socket)

            try expectEqual(handle.processIdentifier, 4_242)
            let calls = await launcher.calls
            try expectEqual(calls.count, 1)
            try expectEqual(calls.first?.executableURL, URL(fileURLWithPath: "/usr/bin/ssh"))
            try expectEqual(calls.first?.arguments, SSHCommandBuilder().masterArguments(alias: "gpu", socket: socket))
        },
        TestCase("SSHProcessControllerTests.pollsUntilMasterCheckSucceeds") {
            let executor = QueueProcessExecutor(results: [
                ProcessResult(exitStatus: 255, stdout: Data(), stderr: Data()),
                ProcessResult(exitStatus: 0, stdout: Data("Master running".utf8), stderr: Data()),
            ])
            let clock = ImmediateControlClock()
            let controller = SSHProcessController(
                executor: executor,
                launcher: RecordingProcessLauncher(),
                clock: clock,
                maximumReadinessAttempts: 3
            )
            let socket = URL(fileURLWithPath: "/tmp/tunneldock-501/abc.sock")

            try expectEqual(await controller.waitUntilReady(alias: "gpu", socket: socket), true)
            try expectEqual(await clock.sleeps, [.milliseconds(100)])
            try expectEqual(await executor.calls.count, 2)
        },
        TestCase("SSHProcessControllerTests.forwardFailureThrowsCommandError") {
            let executor = QueueProcessExecutor(results: [
                ProcessResult(exitStatus: 255, stdout: Data(), stderr: Data("Address already in use".utf8)),
            ])
            let controller = SSHProcessController(
                executor: executor,
                launcher: RecordingProcessLauncher(),
                clock: ImmediateControlClock()
            )
            let specification = try ForwardSpecification(
                localAddress: "127.0.0.1", localPort: "8888",
                remoteHost: "127.0.0.1", remotePort: "8888"
            )

            do {
                try await controller.addForward(
                    alias: "gpu",
                    socket: URL(fileURLWithPath: "/tmp/tunneldock-501/abc.sock"),
                    specification: specification
                )
                throw TestFailure("Expected SSHProcessControllerError.commandFailed")
            } catch let error as SSHProcessControllerError {
                try expectEqual(error.userError, .localPortInUse)
            }
        },
        TestCase("SSHProcessControllerTests.readinessFailureClassifiesMasterStderr") {
            let executor = QueueProcessExecutor(results: [
                ProcessResult(exitStatus: 255, stdout: Data(), stderr: Data()),
            ])
            let controller = SSHProcessController(
                executor: executor,
                launcher: CapturedStderrProcessLauncher(
                    stderr: Data("Permission denied (publickey).".utf8)
                ),
                clock: ImmediateControlClock(),
                maximumReadinessAttempts: 1
            )
            let socket = URL(fileURLWithPath: "/tmp/tunneldock-501/auth.sock")
            let handle = try await controller.startMaster(alias: "gpu", socket: socket)

            try expectEqual(await controller.waitUntilReady(alias: "gpu", socket: socket), false)
            try expectEqual(controller.readinessError(for: handle).userError, .authenticationFailed)
        },
    ]
}

private struct LaunchCall: Sendable, Equatable {
    let executableURL: URL
    let arguments: [String]
}

private final class StubRunningProcess: RunningProcess, @unchecked Sendable {
    let processIdentifier: Int32 = 4_242
    let isRunning = true
    private let stream: AsyncStream<ProcessEvent>

    init() {
        stream = AsyncStream { _ in }
    }

    func events() -> AsyncStream<ProcessEvent> { stream }
    func terminate() {}
    func interruptWithKill() {}
}

private actor RecordingProcessLauncher: ProcessLaunching {
    private(set) var calls: [LaunchCall] = []

    func launch(executableURL: URL, arguments: [String]) async throws -> any RunningProcess {
        calls.append(LaunchCall(executableURL: executableURL, arguments: arguments))
        return StubRunningProcess()
    }
}

private final class CapturedStderrRunningProcess: RunningProcess, @unchecked Sendable {
    let processIdentifier: Int32 = 4_243
    let isRunning = false
    let capturedStderr: Data

    init(stderr: Data) { capturedStderr = stderr }
    func events() -> AsyncStream<ProcessEvent> { AsyncStream { $0.finish() } }
    func terminate() {}
    func interruptWithKill() {}
}

private struct CapturedStderrProcessLauncher: ProcessLaunching {
    let stderr: Data

    func launch(executableURL: URL, arguments: [String]) async throws -> any RunningProcess {
        CapturedStderrRunningProcess(stderr: stderr)
    }
}

private actor QueueProcessExecutor: ProcessExecuting {
    private var results: [ProcessResult]
    private(set) var calls: [(URL, [String])] = []

    init(results: [ProcessResult]) {
        self.results = results
    }

    func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult {
        calls.append((executableURL, arguments))
        guard !results.isEmpty else {
            return ProcessResult(exitStatus: 255, stdout: Data(), stderr: Data())
        }
        return results.removeFirst()
    }
}

private actor ImmediateControlClock: SSHControlClock {
    private(set) var sleeps: [Duration] = []

    func sleep(for duration: Duration) async throws {
        sleeps.append(duration)
    }
}

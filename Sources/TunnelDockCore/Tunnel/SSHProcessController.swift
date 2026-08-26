import Darwin
import Foundation

public struct SSHCommandBuilder: Sendable {
    public init() {}

    public func masterArguments(alias: String, socket: URL) -> [String] {
        [
            "-M", "-S", socket.path, "-N", "-T", "-n",
            "-o", "ControlPersist=no",
            "-o", "ClearAllForwardings=yes",
            "-o", "ExitOnForwardFailure=yes",
            "-o", "ServerAliveInterval=15",
            "-o", "ServerAliveCountMax=3",
            "-o", "BatchMode=yes",
            "-o", "StrictHostKeyChecking=yes",
            alias,
        ]
    }

    public func checkArguments(alias: String, socket: URL) -> [String] {
        ["-S", socket.path, "-O", "check", alias]
    }

    public func forwardArguments(
        alias: String,
        socket: URL,
        specification: ForwardSpecification
    ) -> [String] {
        ["-S", socket.path, "-O", "forward", "-L", specification.openSSHArgument, alias]
    }

    public func exitArguments(alias: String, socket: URL) -> [String] {
        ["-S", socket.path, "-O", "exit", alias]
    }
}

public enum ProcessEvent: Sendable, Equatable {
    case stdout(Data)
    case stderr(Data)
    case terminated(Int32)
}

public protocol RunningProcess: Sendable {
    var processIdentifier: Int32 { get }
    var isRunning: Bool { get }
    var capturedStderr: Data { get }
    func events() -> AsyncStream<ProcessEvent>
    func terminate()
    func interruptWithKill()
}

public extension RunningProcess {
    var capturedStderr: Data { Data() }
}

public protocol ProcessLaunching: Sendable {
    func launch(executableURL: URL, arguments: [String]) async throws -> any RunningProcess
}

public struct FoundationProcessLauncher: ProcessLaunching {
    public init() {}

    public func launch(executableURL: URL, arguments: [String]) async throws -> any RunningProcess {
        try FoundationRunningProcess(executableURL: executableURL, arguments: arguments)
    }
}

private final class FoundationRunningProcess: RunningProcess, @unchecked Sendable {
    private let process: Process
    private let eventStream: AsyncStream<ProcessEvent>
    private let stderrLock = NSLock()
    private var stderrData = Data()

    var processIdentifier: Int32 { process.processIdentifier }
    var isRunning: Bool { process.isRunning }
    var capturedStderr: Data { stderrLock.withLock { stderrData } }

    init(executableURL: URL, arguments: [String]) throws {
        let process = Process()
        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        let pair = AsyncStream.makeStream(of: ProcessEvent.self)

        self.process = process
        self.eventStream = pair.stream

        process.executableURL = executableURL
        process.arguments = arguments
        process.standardInput = FileHandle.nullDevice
        process.standardOutput = stdoutPipe
        process.standardError = stderrPipe

        stdoutPipe.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            if !data.isEmpty { pair.continuation.yield(.stdout(data)) }
        }
        stderrPipe.fileHandleForReading.readabilityHandler = { handle in
            let data = handle.availableData
            if !data.isEmpty {
                self.stderrLock.withLock { self.stderrData.append(data) }
                pair.continuation.yield(.stderr(data))
            }
        }
        process.terminationHandler = { completed in
            stdoutPipe.fileHandleForReading.readabilityHandler = nil
            stderrPipe.fileHandleForReading.readabilityHandler = nil
            let stdout = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
            let stderr = stderrPipe.fileHandleForReading.readDataToEndOfFile()
            if !stdout.isEmpty { pair.continuation.yield(.stdout(stdout)) }
            if !stderr.isEmpty {
                self.stderrLock.withLock { self.stderrData.append(stderr) }
                pair.continuation.yield(.stderr(stderr))
            }
            pair.continuation.yield(.terminated(completed.terminationStatus))
            pair.continuation.finish()
        }
        do {
            try process.run()
        } catch {
            stdoutPipe.fileHandleForReading.readabilityHandler = nil
            stderrPipe.fileHandleForReading.readabilityHandler = nil
            pair.continuation.finish()
            throw error
        }
    }

    func events() -> AsyncStream<ProcessEvent> { eventStream }

    func terminate() {
        if process.isRunning { process.terminate() }
    }

    func interruptWithKill() {
        if process.isRunning { Darwin.kill(process.processIdentifier, SIGKILL) }
    }
}

public struct SSHMasterHandle: Sendable {
    public let alias: String
    public let socket: URL
    public let processIdentifier: Int32
    fileprivate let process: any RunningProcess

    public init(alias: String, socket: URL, process: any RunningProcess) {
        self.alias = alias
        self.socket = socket
        self.processIdentifier = process.processIdentifier
        self.process = process
    }
}

public protocol SSHControlClock: Sendable {
    func sleep(for duration: Duration) async throws
}

public struct ContinuousSSHControlClock: SSHControlClock {
    public init() {}

    public func sleep(for duration: Duration) async throws {
        try await Task.sleep(for: duration)
    }
}

public enum SSHProcessControllerError: Error, Sendable, Equatable {
    case commandFailed(userError: SSHUserError, stderr: String, exitStatus: Int32)

    public var userError: SSHUserError {
        switch self {
        case let .commandFailed(userError, _, _): userError
        }
    }
}

public protocol SSHProcessControlling: Sendable {
    func startMaster(alias: String, socket: URL) async throws -> SSHMasterHandle
    func waitUntilReady(alias: String, socket: URL) async -> Bool
    func addForward(alias: String, socket: URL, specification: ForwardSpecification) async throws
    func requestExit(alias: String, socket: URL) async
    func events(for handle: SSHMasterHandle) -> AsyncStream<ProcessEvent>
    func terminate(_ handle: SSHMasterHandle)
    func kill(_ handle: SSHMasterHandle)
    func isRunning(_ handle: SSHMasterHandle) -> Bool
    func readinessError(for handle: SSHMasterHandle) -> SSHProcessControllerError
}

public extension SSHProcessControlling {
    func readinessError(for handle: SSHMasterHandle) -> SSHProcessControllerError {
        .commandFailed(
            userError: .processExitedUnexpectedly,
            stderr: "",
            exitStatus: 255
        )
    }
}

public struct SSHProcessController: SSHProcessControlling {
    private let executor: any ProcessExecuting
    private let launcher: any ProcessLaunching
    private let clock: any SSHControlClock
    private let builder: SSHCommandBuilder
    private let classifier: SSHErrorClassifier
    private let maximumReadinessAttempts: Int
    private let executableURL = URL(fileURLWithPath: "/usr/bin/ssh")

    public init(
        executor: any ProcessExecuting = FoundationProcessExecutor(),
        launcher: any ProcessLaunching = FoundationProcessLauncher(),
        clock: any SSHControlClock = ContinuousSSHControlClock(),
        builder: SSHCommandBuilder = SSHCommandBuilder(),
        classifier: SSHErrorClassifier = SSHErrorClassifier(),
        maximumReadinessAttempts: Int = 100
    ) {
        self.executor = executor
        self.launcher = launcher
        self.clock = clock
        self.builder = builder
        self.classifier = classifier
        self.maximumReadinessAttempts = max(1, maximumReadinessAttempts)
    }

    public func startMaster(alias: String, socket: URL) async throws -> SSHMasterHandle {
        let process = try await launcher.launch(
            executableURL: executableURL,
            arguments: builder.masterArguments(alias: alias, socket: socket)
        )
        return SSHMasterHandle(
            alias: alias,
            socket: socket,
            process: process
        )
    }

    public func waitUntilReady(alias: String, socket: URL) async -> Bool {
        for attempt in 0..<maximumReadinessAttempts {
            if Task.isCancelled { return false }
            if let result = try? await executor.run(
                executableURL: executableURL,
                arguments: builder.checkArguments(alias: alias, socket: socket)
            ), result.exitStatus == 0 {
                return true
            }
            if attempt + 1 < maximumReadinessAttempts {
                try? await clock.sleep(for: .milliseconds(100))
            }
        }
        return false
    }

    public func addForward(
        alias: String,
        socket: URL,
        specification: ForwardSpecification
    ) async throws {
        let result = try await executor.run(
            executableURL: executableURL,
            arguments: builder.forwardArguments(
                alias: alias,
                socket: socket,
                specification: specification
            )
        )
        guard result.exitStatus == 0 else { throw commandError(from: result) }
    }

    public func requestExit(alias: String, socket: URL) async {
        _ = try? await executor.run(
            executableURL: executableURL,
            arguments: builder.exitArguments(alias: alias, socket: socket)
        )
    }

    public func events(for handle: SSHMasterHandle) -> AsyncStream<ProcessEvent> {
        handle.process.events()
    }

    public func terminate(_ handle: SSHMasterHandle) {
        handle.process.terminate()
    }

    public func kill(_ handle: SSHMasterHandle) {
        handle.process.interruptWithKill()
    }

    public func isRunning(_ handle: SSHMasterHandle) -> Bool {
        handle.process.isRunning
    }

    public func readinessError(for handle: SSHMasterHandle) -> SSHProcessControllerError {
        let stderrData = handle.process.capturedStderr
        let stderr = String(data: stderrData, encoding: .utf8) ?? ""
        return .commandFailed(
            userError: classifier.classify(stderr: stderr, exitStatus: 255),
            stderr: stderr,
            exitStatus: 255
        )
    }

    private func commandError(from result: ProcessResult) -> SSHProcessControllerError {
        let stderr = String(data: result.stderr, encoding: .utf8) ?? ""
        return .commandFailed(
            userError: classifier.classify(stderr: stderr, exitStatus: result.exitStatus),
            stderr: stderr,
            exitStatus: result.exitStatus
        )
    }
}

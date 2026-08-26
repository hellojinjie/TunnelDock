import Foundation

public struct ProcessResult: Sendable, Equatable {
    public let exitStatus: Int32
    public let stdout: Data
    public let stderr: Data

    public init(exitStatus: Int32, stdout: Data, stderr: Data) {
        self.exitStatus = exitStatus
        self.stdout = stdout
        self.stderr = stderr
    }
}

public protocol ProcessExecuting: Sendable {
    func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult
}

public struct FoundationProcessExecutor: ProcessExecuting {
    public init() {}

    public func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult {
        try await Task.detached {
            let process = Process()
            let stdoutPipe = Pipe()
            let stderrPipe = Pipe()
            let stdout = LockedDataBuffer()
            let stderr = LockedDataBuffer()

            stdoutPipe.fileHandleForReading.readabilityHandler = { handle in
                stdout.append(handle.availableData)
            }
            stderrPipe.fileHandleForReading.readabilityHandler = { handle in
                stderr.append(handle.availableData)
            }

            process.executableURL = executableURL
            process.arguments = arguments
            process.standardInput = FileHandle.nullDevice
            process.standardOutput = stdoutPipe
            process.standardError = stderrPipe
            try process.run()
            process.waitUntilExit()

            stdoutPipe.fileHandleForReading.readabilityHandler = nil
            stderrPipe.fileHandleForReading.readabilityHandler = nil
            stdout.append(stdoutPipe.fileHandleForReading.readDataToEndOfFile())
            stderr.append(stderrPipe.fileHandleForReading.readDataToEndOfFile())

            return ProcessResult(
                exitStatus: process.terminationStatus,
                stdout: stdout.value,
                stderr: stderr.value
            )
        }.value
    }
}

private final class LockedDataBuffer: @unchecked Sendable {
    private let lock = NSLock()
    private var data = Data()

    var value: Data {
        lock.withLock { data }
    }

    func append(_ newData: Data) {
        guard !newData.isEmpty else { return }
        lock.withLock {
            data.append(newData)
        }
    }
}

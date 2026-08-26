import Foundation
import TunnelDockCore

public enum SSHConfigEditorOpenerError: Error, LocalizedError {
    case launchFailed(String)

    public var errorDescription: String? {
        switch self {
        case .launchFailed(let message):
            message.isEmpty
                ? "macOS could not open the SSH configuration in the default text editor."
                : message
        }
    }
}

public struct SSHConfigEditorOpener: Sendable {
    private let executor: any ProcessExecuting
    private let executableURL = URL(fileURLWithPath: "/usr/bin/open")

    public init(executor: any ProcessExecuting = FoundationProcessExecutor()) {
        self.executor = executor
    }

    public func open(configURL: URL) async throws {
        let result = try await executor.run(
            executableURL: executableURL,
            arguments: ["-t", configURL.path]
        )
        guard result.exitStatus == 0 else {
            let message = String(data: result.stderr, encoding: .utf8)?
                .trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            throw SSHConfigEditorOpenerError.launchFailed(message)
        }
    }
}

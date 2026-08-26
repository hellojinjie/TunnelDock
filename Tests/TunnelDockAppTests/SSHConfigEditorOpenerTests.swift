import Foundation
import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum SSHConfigEditorOpenerTests {
    static let all: [TestCase] = [
        TestCase("SSHConfigEditorOpenerTests.opensConfigWithSystemDefaultTextEditor") {
            let executor = RecordingOpenExecutor()
            let opener = SSHConfigEditorOpener(executor: executor)
            let configURL = URL(fileURLWithPath: "/tmp/example-ssh-config")

            try await opener.open(configURL: configURL)

            try expectEqual(
                await executor.calls,
                [OpenCall(executableURL: URL(fileURLWithPath: "/usr/bin/open"), arguments: ["-t", "/tmp/example-ssh-config"])]
            )
        },
    ]
}

private struct OpenCall: Sendable, Equatable {
    let executableURL: URL
    let arguments: [String]
}

private actor RecordingOpenExecutor: ProcessExecuting {
    private(set) var calls: [OpenCall] = []

    func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult {
        calls.append(OpenCall(executableURL: executableURL, arguments: arguments))
        return ProcessResult(exitStatus: 0, stdout: Data(), stderr: Data())
    }
}

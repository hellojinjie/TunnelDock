import Foundation
import TestSupport
import TunnelDockCore

enum SSHHostResolverTests {
    static let all: [TestCase] = [
        TestCase("SSHHostResolverTests.resolvesEffectiveFieldsWithExactSSHArguments") {
            let executor = RecordingProcessExecutor(results: [
                "gpu": ProcessResult(
                    exitStatus: 0,
                    stdout: Data("hostname 10.0.0.21\nuser researcher\nport 2222\n".utf8),
                    stderr: Data()
                ),
            ])

            let host = await SSHHostResolver(executor: executor).resolve(alias: "gpu", order: 3)

            try expectEqual(host.hostname, "10.0.0.21")
            try expectEqual(host.user, "researcher")
            try expectEqual(host.port, 2_222)
            try expectEqual(host.configOrder, 3)
            try expectEqual(host.availability, .available)
            let calls = await executor.calls
            try expectEqual(calls, [
                ProcessCall(executableURL: URL(fileURLWithPath: "/usr/bin/ssh"), arguments: ["-G", "gpu"]),
            ])
        },
        TestCase("SSHHostResolverTests.nonzeroExitMarksOnlyThatHostAsConfigurationError") {
            let executor = RecordingProcessExecutor(results: [
                "bad": ProcessResult(
                    exitStatus: 255,
                    stdout: Data(),
                    stderr: Data("Bad configuration option".utf8)
                ),
            ])

            let host = await SSHHostResolver(executor: executor).resolve(alias: "bad", order: 0)

            try expectConfigurationError(host)
            try expectEqual(host.alias, "bad")
        },
        TestCase("SSHHostResolverTests.missingFieldAndInvalidPortMarkConfigurationError") {
            let executor = RecordingProcessExecutor(results: [
                "missing": ProcessResult(exitStatus: 0, stdout: Data("hostname host\nport 22\n".utf8), stderr: Data()),
                "invalid": ProcessResult(exitStatus: 0, stdout: Data("hostname host\nuser me\nport nope\n".utf8), stderr: Data()),
            ])
            let resolver = SSHHostResolver(executor: executor)

            try expectConfigurationError(await resolver.resolve(alias: "missing", order: 0))
            try expectConfigurationError(await resolver.resolve(alias: "invalid", order: 1))
        },
        TestCase("SSHHostResolverTests.searchIsCaseInsensitiveAcrossEveryEffectiveField") {
            let host = SSHHost(
                alias: "GPU-Lab",
                hostname: "10.0.0.21",
                user: "Researcher",
                port: 2_222,
                configOrder: 0,
                availability: .available
            )

            for query in ["gpu", "10.0.0", "RESEARCHER", "2222"] {
                try expectEqual(host.matches(query: query), true, "query \(query)")
            }
            try expectEqual(host.matches(query: "missing"), false)
        },
    ]

    private static func expectConfigurationError(_ host: SSHHost) throws {
        guard case .configurationError = host.availability else {
            throw TestFailure("Expected configuration error for \(host.alias)")
        }
    }
}

private struct ProcessCall: Sendable, Equatable {
    let executableURL: URL
    let arguments: [String]
}

private actor RecordingProcessExecutor: ProcessExecuting {
    private let results: [String: ProcessResult]
    private(set) var calls: [ProcessCall] = []

    init(results: [String: ProcessResult]) {
        self.results = results
    }

    func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult {
        calls.append(ProcessCall(executableURL: executableURL, arguments: arguments))
        return results[arguments.last ?? ""]
            ?? ProcessResult(exitStatus: 1, stdout: Data(), stderr: Data("missing fixture".utf8))
    }
}

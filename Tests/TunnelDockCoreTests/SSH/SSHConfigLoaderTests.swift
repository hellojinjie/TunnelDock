import Foundation
import TestSupport
import TunnelDockCore

enum SSHConfigLoaderTests {
    static let all: [TestCase] = [
        TestCase("SSHConfigLoaderTests.restoresConfigOrderAfterConcurrentResolution") {
            let root = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockLoaderTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            let sshDirectory = root.appending(path: ".ssh", directoryHint: .isDirectory)
            try FileManager.default.createDirectory(at: sshDirectory, withIntermediateDirectories: true)
            defer { try? FileManager.default.removeItem(at: root) }
            let config = sshDirectory.appending(path: "config")
            try "Host first\nHost second\nHost third\n".write(to: config, atomically: true, encoding: .utf8)
            let resolver = OutOfOrderHostResolver()
            let loader = SSHConfigLoader(
                includeResolver: SSHIncludeResolver(userSSHDirectory: sshDirectory),
                scanner: SSHConfigScanner(),
                hostResolver: resolver,
                maximumConcurrentResolutions: 2
            )

            let snapshot = try await loader.load(rootURL: config)

            try expectEqual(snapshot.hosts.map(\.alias), ["first", "second", "third"])
            try expectEqual(snapshot.hosts.map(\.configOrder), [0, 1, 2])
            try expectEqual(snapshot.expanded.watchedFiles.contains(config.standardizedFileURL), true)
        },
    ]
}

private actor OutOfOrderHostResolver: SSHHostResolving {
    func resolve(alias: String, order: Int) async -> SSHHost {
        if alias == "first" {
            try? await Task.sleep(for: .milliseconds(20))
        }
        return SSHHost(
            alias: alias,
            hostname: "\(alias).internal",
            user: "tester",
            port: 22,
            configOrder: order,
            availability: .available
        )
    }
}

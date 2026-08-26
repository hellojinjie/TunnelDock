import Foundation
import TestSupport
import TunnelDockCore

enum SSHIncludeResolverTests {
    static let all: [TestCase] = [
        TestCase("SSHIncludeResolverTests.expandsGlobAtDirectivePositionInBytewiseOrder") {
            try withSSHDirectory { sshDirectory in
                try write("Host beta\n", to: sshDirectory.appending(path: "config.d/b"))
                try write("Host alpha\n", to: sshDirectory.appending(path: "config.d/a"))
                let root = sshDirectory.appending(path: "config")
                try write("Host before\nInclude config.d/*\nHost after\n", to: root)

                let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory).resolve(rootURL: root)

                try expectEqual(
                    SSHConfigScanner().discoverAliases(in: expanded.lines),
                    ["before", "alpha", "beta", "after"]
                )
                let expectedDirectory = sshDirectory.appending(path: "config.d")
                try expectEqual(
                    expanded.watchedDirectories.contains(expectedDirectory),
                    true,
                    "expected \(expectedDirectory.absoluteString), got \(expanded.watchedDirectories.map(\.absoluteString).sorted())"
                )
            }
        },
        TestCase("SSHIncludeResolverTests.nestedRelativeIncludeRemainsBasedAtUserSSHDirectory") {
            try withSSHDirectory { sshDirectory in
                try write("Host common\n", to: sshDirectory.appending(path: "common"))
                try write("Host nested\nInclude common\n", to: sshDirectory.appending(path: "nested/one"))
                let root = sshDirectory.appending(path: "config")
                try write("Include nested/one\n", to: root)

                let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory).resolve(rootURL: root)

                try expectEqual(
                    SSHConfigScanner().discoverAliases(in: expanded.lines),
                    ["nested", "common"]
                )
            }
        },
        TestCase("SSHIncludeResolverTests.detectsActiveStackCycleWithoutBlockingRepeatedNoncyclicIncludes") {
            try withSSHDirectory { sshDirectory in
                try write("Host a\nInclude b\n", to: sshDirectory.appending(path: "a"))
                try write("Host b\nInclude a\n", to: sshDirectory.appending(path: "b"))
                try write("Host shared\n", to: sshDirectory.appending(path: "shared"))
                let root = sshDirectory.appending(path: "config")
                try write("Include a\nInclude shared\nInclude shared\n", to: root)

                let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory).resolve(rootURL: root)

                try expectEqual(expanded.diagnostics.count, 1)
                try expectEqual(
                    SSHConfigScanner().discoverAliases(in: expanded.lines),
                    ["a", "b", "shared"]
                )
                try expectEqual(expanded.lines.filter { $0 == "Host shared" }.count, 2)
            }
        },
        TestCase("SSHIncludeResolverTests.expandsTildeAndAbsoluteIncludes") {
            try withSSHDirectory { sshDirectory in
                let home = sshDirectory.deletingLastPathComponent()
                let tildeFile = home.appending(path: "tilde-config")
                let absoluteFile = home.appending(path: "absolute-config")
                try write("Host tilde\n", to: tildeFile)
                try write("Host absolute\n", to: absoluteFile)
                let root = sshDirectory.appending(path: "config")
                try write("Include ~/tilde-config\nInclude \(absoluteFile.path)\n", to: root)

                let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory).resolve(rootURL: root)

                try expectEqual(
                    SSHConfigScanner().discoverAliases(in: expanded.lines),
                    ["tilde", "absolute"]
                )
            }
        },
        TestCase("SSHIncludeResolverTests.missingRootProducesEmptySnapshotAndWatchesSSHDirectory") {
            try withSSHDirectory { sshDirectory in
                let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory)
                    .resolve(rootURL: sshDirectory.appending(path: "config"))

                try expectEqual(expanded.lines, [])
                try expectEqual(expanded.watchedDirectories.contains(sshDirectory), true)
            }
        },
        TestCase("SSHIncludeResolverTests.acceptsOpenSSHIncludeEqualsSyntax") {
            try withSSHDirectory { sshDirectory in
                try write("Host included\n", to: sshDirectory.appending(path: "included.conf"))
                try write("Host spaced\n", to: sshDirectory.appending(path: "spaced.conf"))
                let root = sshDirectory.appending(path: "config")
                try write("Include=included.conf\nInclude = spaced.conf\n", to: root)

                let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory).resolve(rootURL: root)
                try expectEqual(SSHConfigScanner().discoverAliases(in: expanded.lines), ["included", "spaced"])
            }
        },
    ]

    private static func withSSHDirectory(
        _ body: (URL) throws -> Void
    ) throws {
        let root = FileManager.default.temporaryDirectory
            .appending(path: "TunnelDockTests-\(UUID().uuidString)", directoryHint: .isDirectory)
        let sshDirectory = root.appending(path: ".ssh", directoryHint: .isDirectory)
        try FileManager.default.createDirectory(at: sshDirectory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }
        try body(sshDirectory)
    }

    private static func write(_ contents: String, to url: URL) throws {
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        try contents.write(to: url, atomically: true, encoding: .utf8)
    }
}

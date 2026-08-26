import Foundation
import TestSupport
import TunnelDockCore

enum SSHConfigAppenderTests {
    static let all: [TestCase] = [
        TestCase("SSHConfigAppenderTests.appendsNewHostWithoutChangingExistingConfiguration") {
            let root = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockAppenderTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            defer { try? FileManager.default.removeItem(at: root) }
            try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
            let config = root.appending(path: "config")
            let existing = "Host existing\n    HostName existing.example.com\n"
            try existing.write(to: config, atomically: true, encoding: .utf8)

            try SSHConfigAppender().append(
                SSHHostConfiguration(
                    alias: "new-host",
                    hostname: "new.example.com",
                    user: "alice",
                    port: 2200
                ),
                to: config
            )

            let content = try String(contentsOf: config, encoding: .utf8)
            try expectEqual(
                content,
                existing + "\nHost new-host\n    HostName new.example.com\n    User alice\n    Port 2200\n"
            )
        },
        TestCase("SSHConfigAppenderTests.createsMissingConfigurationBeforeAppending") {
            let root = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockAppenderTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            defer { try? FileManager.default.removeItem(at: root) }
            let config = root.appending(path: ".ssh/config")

            try SSHConfigAppender().append(
                SSHHostConfiguration(alias: "db", hostname: "db.internal", user: "alice", port: 22),
                to: config
            )

            try expectEqual(FileManager.default.fileExists(atPath: config.path), true)
            try expectEqual(
                try String(contentsOf: config, encoding: .utf8),
                "Host db\n    HostName db.internal\n    User alice\n    Port 22\n"
            )
        },
        TestCase("SSHConfigAppenderTests.rejectsUnsafeHostTokens") {
            let root = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockAppenderTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            defer { try? FileManager.default.removeItem(at: root) }
            let config = root.appending(path: "config")

            do {
                try SSHConfigAppender().append(
                    SSHHostConfiguration(alias: "*", hostname: "db.internal", user: "alice", port: 22),
                    to: config
                )
                throw TestFailure("Expected wildcard aliases to be rejected")
            } catch SSHConfigAppenderError.invalidAlias {
                try expectEqual(FileManager.default.fileExists(atPath: config.path), false)
            }
        },
    ]
}

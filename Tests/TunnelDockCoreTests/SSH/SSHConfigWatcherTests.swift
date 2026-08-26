import Foundation
import TestSupport
import TunnelDockCore

enum SSHConfigWatcherTests {
    static let all: [TestCase] = [
        TestCase("SSHConfigWatcherTests.emitsDebouncedEventForWatchedFileOrDirectoryChange") {
            let directory = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockWatcherTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            defer { try? FileManager.default.removeItem(at: directory) }
            let config = directory.appending(path: "config")
            try "Host before\n".write(to: config, atomically: true, encoding: .utf8)
            let expanded = ExpandedSSHConfig(
                lines: [],
                watchedFiles: [config.standardizedFileURL],
                watchedDirectories: [directory.standardizedFileURL],
                diagnostics: []
            )
            let stream = SSHConfigWatcher(debounceDuration: .milliseconds(30))
                .events(watching: expanded)

            let eventTask = Task { await receivesOneEvent(stream) }
            try "Host one\n".write(to: config, atomically: true, encoding: .utf8)
            try "Host two\n".write(to: config, atomically: true, encoding: .utf8)
            try "Host three\n".write(to: config, atomically: true, encoding: .utf8)

            try expectEqual(await eventTask.value, true)
        },
    ]

    private static func receivesOneEvent(_ stream: AsyncStream<Void>) async -> Bool {
        await withTaskGroup(of: Bool.self) { group in
            group.addTask {
                for await _ in stream { return true }
                return false
            }
            group.addTask {
                try? await Task.sleep(for: .seconds(1))
                return false
            }
            let result = await group.next() ?? false
            group.cancelAll()
            return result
        }
    }
}

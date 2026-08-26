import Foundation
import TestSupport
import TunnelDockCore

enum ControlSocketManagerTests {
    static let all: [TestCase] = [
        TestCase("ControlSocketManagerTests.createsPrivateShortRuntimeDirectory") {
            let base = URL(fileURLWithPath: "/tmp/td-\(UUID().uuidString.prefix(8))", isDirectory: true)
            defer { try? FileManager.default.removeItem(at: base) }
            let manager = ControlSocketManager(uid: 12_345, baseDirectory: base)

            let directory = try manager.prepareDirectory()
            let attributes = try FileManager.default.attributesOfItem(atPath: directory.path)
            let mode = (attributes[.posixPermissions] as? NSNumber)?.intValue ?? -1
            let socket = try manager.allocateSocketURL()

            try expectEqual(mode & 0o777, 0o700)
            try expectEqual(socket.path.utf8.count < 104, true)
            try expectEqual(socket.pathExtension, "sock")
        },
        TestCase("ControlSocketManagerTests.removesOnlySockEntriesInsideOwnedDirectory") {
            let base = URL(fileURLWithPath: "/tmp/td-\(UUID().uuidString.prefix(8))", isDirectory: true)
            defer { try? FileManager.default.removeItem(at: base) }
            let manager = ControlSocketManager(uid: 12_346, baseDirectory: base)
            let directory = try manager.prepareDirectory()
            let stale = directory.appending(path: "stale.sock")
            let keep = directory.appending(path: "keep.txt")
            try Data().write(to: stale)
            try Data().write(to: keep)

            try manager.removeStaleSockets()

            try expectEqual(FileManager.default.fileExists(atPath: stale.path), false)
            try expectEqual(FileManager.default.fileExists(atPath: keep.path), true)
        },
    ]
}

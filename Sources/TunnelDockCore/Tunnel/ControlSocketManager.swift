import Darwin
import Foundation

public enum ControlSocketError: Error, Sendable, Equatable {
    case invalidRuntimeDirectory(URL)
    case invalidSocketURL(URL)
}

public protocol ControlSocketManaging: Sendable {
    func prepareDirectory() throws -> URL
    func allocateSocketURL() throws -> URL
    func removeSocket(at url: URL) throws
    func removeStaleSockets() throws
}

public struct ControlSocketManager: ControlSocketManaging {
    private let uid: UInt32
    private let baseDirectory: URL

    public init(
        uid: UInt32 = getuid(),
        baseDirectory: URL = URL(fileURLWithPath: "/tmp", isDirectory: true)
    ) {
        self.uid = uid
        self.baseDirectory = baseDirectory.standardizedFileURL
    }

    public func prepareDirectory() throws -> URL {
        let directory = runtimeDirectory
        var isDirectory: ObjCBool = false
        if FileManager.default.fileExists(atPath: directory.path, isDirectory: &isDirectory) {
            guard isDirectory.boolValue else {
                throw ControlSocketError.invalidRuntimeDirectory(directory)
            }
        } else {
            try FileManager.default.createDirectory(
                at: directory,
                withIntermediateDirectories: true,
                attributes: [.posixPermissions: 0o700]
            )
        }
        try FileManager.default.setAttributes(
            [.posixPermissions: 0o700],
            ofItemAtPath: directory.path
        )
        return directory
    }

    public func allocateSocketURL() throws -> URL {
        let directory = try prepareDirectory()
        let identifier = UUID().uuidString
            .replacingOccurrences(of: "-", with: "")
            .prefix(12)
            .lowercased()
        return directory.appending(path: "\(identifier).sock")
    }

    public func removeSocket(at url: URL) throws {
        let standardized = url.standardizedFileURL
        guard standardized.deletingLastPathComponent() == runtimeDirectory,
              standardized.pathExtension == "sock"
        else {
            throw ControlSocketError.invalidSocketURL(url)
        }
        if FileManager.default.fileExists(atPath: standardized.path) {
            try FileManager.default.removeItem(at: standardized)
        }
    }

    public func removeStaleSockets() throws {
        let directory = try prepareDirectory()
        let entries = try FileManager.default.contentsOfDirectory(
            at: directory,
            includingPropertiesForKeys: nil
        )
        for entry in entries where entry.pathExtension == "sock" {
            try removeSocket(at: entry)
        }
    }

    private var runtimeDirectory: URL {
        baseDirectory.appending(path: "tunneldock-\(uid)", directoryHint: .isDirectory)
            .standardizedFileURL
    }
}

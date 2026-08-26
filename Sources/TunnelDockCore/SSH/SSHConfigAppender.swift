import Foundation

public struct SSHHostConfiguration: Sendable, Equatable {
    public let alias: String
    public let hostname: String
    public let user: String
    public let port: UInt16

    public init(alias: String, hostname: String, user: String, port: UInt16) {
        self.alias = alias
        self.hostname = hostname
        self.user = user
        self.port = port
    }
}

public enum SSHConfigAppenderError: Error, Equatable, LocalizedError {
    case invalidAlias
    case invalidHostname
    case invalidUser
    case invalidPort
    case configurationPathIsDirectory
    case couldNotCreateConfiguration

    public var errorDescription: String? {
        switch self {
        case .invalidAlias:
            "Host aliases must not contain whitespace, comments, or wildcard characters."
        case .invalidHostname:
            "HostName is required and must not contain whitespace or comments."
        case .invalidUser:
            "User is required and must not contain whitespace or comments."
        case .invalidPort:
            "Port must be between 1 and 65535."
        case .configurationPathIsDirectory:
            "~/.ssh/config is a directory, not a configuration file."
        case .couldNotCreateConfiguration:
            "Could not create ~/.ssh/config."
        }
    }
}

public struct SSHConfigAppender {
    private let fileManager: FileManager

    public init(fileManager: FileManager = .default) {
        self.fileManager = fileManager
    }

    public func ensureConfigurationFile(at configURL: URL) throws {
        let directoryURL = configURL.deletingLastPathComponent()
        var isDirectory: ObjCBool = false
        if fileManager.fileExists(atPath: directoryURL.path, isDirectory: &isDirectory) {
            guard isDirectory.boolValue else { throw SSHConfigAppenderError.couldNotCreateConfiguration }
        } else {
            try fileManager.createDirectory(
                at: directoryURL,
                withIntermediateDirectories: true,
                attributes: [.posixPermissions: 0o700]
            )
        }

        if fileManager.fileExists(atPath: configURL.path, isDirectory: &isDirectory) {
            guard !isDirectory.boolValue else { throw SSHConfigAppenderError.configurationPathIsDirectory }
            return
        }

        guard fileManager.createFile(
            atPath: configURL.path,
            contents: nil,
            attributes: [.posixPermissions: 0o600]
        ) else {
            throw SSHConfigAppenderError.couldNotCreateConfiguration
        }
    }

    public func append(_ host: SSHHostConfiguration, to configURL: URL) throws {
        try validate(host)
        try ensureConfigurationFile(at: configURL)

        let attributes = try fileManager.attributesOfItem(atPath: configURL.path)
        let prefix = (attributes[.size] as? NSNumber)?.uint64Value == 0 ? "" : "\n"
        let data = Data((prefix + hostBlock(for: host)).utf8)
        let fileHandle = try FileHandle(forWritingTo: configURL)
        defer { try? fileHandle.close() }
        try fileHandle.seekToEnd()
        try fileHandle.write(contentsOf: data)
    }

    private func validate(_ host: SSHHostConfiguration) throws {
        guard isValidToken(host.alias), !host.alias.contains(where: { "*!?[]".contains($0) }) else {
            throw SSHConfigAppenderError.invalidAlias
        }
        guard isValidToken(host.hostname) else { throw SSHConfigAppenderError.invalidHostname }
        guard isValidToken(host.user) else { throw SSHConfigAppenderError.invalidUser }
        guard host.port > 0 else { throw SSHConfigAppenderError.invalidPort }
    }

    private func isValidToken(_ value: String) -> Bool {
        !value.isEmpty
            && value.rangeOfCharacter(from: .whitespacesAndNewlines) == nil
            && !value.contains("#")
    }

    private func hostBlock(for host: SSHHostConfiguration) -> String {
        "Host \(host.alias)\n"
            + "    HostName \(host.hostname)\n"
            + "    User \(host.user)\n"
            + "    Port \(host.port)\n"
    }
}

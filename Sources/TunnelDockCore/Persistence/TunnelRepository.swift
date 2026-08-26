import Foundation

public struct TunnelEnvelope: Codable, Sendable, Equatable {
    public let schemaVersion: Int
    public let tunnels: [TunnelDefinition]

    public init(schemaVersion: Int = 1, tunnels: [TunnelDefinition]) {
        self.schemaVersion = schemaVersion
        self.tunnels = tunnels
    }
}

public enum TunnelRepositoryError: Error, Sendable, Equatable, CustomStringConvertible {
    case malformedFile(String)
    case unsupportedSchema(Int)
    case writeLocked
    case definitionNotFound(UUID)
    case duplicateDefinition(UUID)

    public var description: String {
        switch self {
        case let .malformedFile(message):
            return "Saved tunnels could not be read: \(message)"
        case let .unsupportedSchema(version):
            return "Saved tunnels use unsupported schema version \(version)."
        case .writeLocked:
            return "Saved tunnels cannot be changed until the storage file is valid."
        case let .definitionNotFound(id):
            return "Saved tunnel \(id) was not found."
        case let .duplicateDefinition(id):
            return "Saved tunnel \(id) already exists."
        }
    }
}

public actor TunnelRepository {
    private let fileURL: URL
    private let now: @Sendable () -> Date
    private var cachedTunnels: [TunnelDefinition]?
    private var isWriteLocked = false

    public init(
        fileURL: URL,
        now: @escaping @Sendable () -> Date = Date.init
    ) {
        self.fileURL = fileURL
        self.now = now
    }

    public func load() throws -> [TunnelDefinition] {
        guard FileManager.default.fileExists(atPath: fileURL.path) else {
            cachedTunnels = []
            isWriteLocked = false
            return []
        }
        do {
            let data = try Data(contentsOf: fileURL)
            let envelope = try Self.decoder.decode(TunnelEnvelope.self, from: data)
            guard envelope.schemaVersion == 1 else {
                isWriteLocked = true
                throw TunnelRepositoryError.unsupportedSchema(envelope.schemaVersion)
            }
            cachedTunnels = envelope.tunnels
            isWriteLocked = false
            return envelope.tunnels
        } catch let error as TunnelRepositoryError {
            throw error
        } catch {
            isWriteLocked = true
            throw TunnelRepositoryError.malformedFile(error.localizedDescription)
        }
    }

    public func replaceAll(_ tunnels: [TunnelDefinition]) throws {
        try requireWritable()
        try persist(tunnels)
        cachedTunnels = tunnels
    }

    public func save(_ definition: TunnelDefinition) throws {
        var tunnels = try currentTunnels()
        guard !tunnels.contains(where: { $0.id == definition.id }) else {
            throw TunnelRepositoryError.duplicateDefinition(definition.id)
        }
        tunnels.append(definition)
        try persist(tunnels)
        cachedTunnels = tunnels
    }

    public func findMatchingForward(
        hostAlias: String,
        localAddress: String,
        localPort: UInt16,
        remoteHost: String,
        remotePort: UInt16
    ) throws -> TunnelDefinition? {
        try currentTunnels().first {
            $0.hostAlias == hostAlias
                && $0.localAddress == localAddress
                && $0.localPort == localPort
                && $0.remoteHost == remoteHost
                && $0.remotePort == remotePort
        }
    }

    @discardableResult
    public func rename(id: UUID, name: String?) throws -> TunnelDefinition {
        var tunnels = try currentTunnels()
        guard let index = tunnels.firstIndex(where: { $0.id == id }) else {
            throw TunnelRepositoryError.definitionNotFound(id)
        }
        tunnels[index].name = name
        tunnels[index].updatedAt = now()
        try persist(tunnels)
        cachedTunnels = tunnels
        return tunnels[index]
    }

    @discardableResult
    public func update(_ definition: TunnelDefinition) throws -> TunnelDefinition {
        var tunnels = try currentTunnels()
        guard let index = tunnels.firstIndex(where: { $0.id == definition.id }) else {
            throw TunnelRepositoryError.definitionNotFound(definition.id)
        }
        var updated = definition
        updated = TunnelDefinition(
            id: definition.id,
            hostAlias: definition.hostAlias,
            name: definition.name,
            remoteHost: definition.remoteHost,
            remotePort: definition.remotePort,
            localAddress: definition.localAddress,
            localPort: definition.localPort,
            createdAt: tunnels[index].createdAt,
            updatedAt: now(),
            lastConnectedAt: definition.lastConnectedAt,
            webProtocol: definition.webProtocol
        )
        tunnels[index] = updated
        try persist(tunnels)
        cachedTunnels = tunnels
        return updated
    }

    public func delete(id: UUID) throws {
        var tunnels = try currentTunnels()
        guard let index = tunnels.firstIndex(where: { $0.id == id }) else {
            throw TunnelRepositoryError.definitionNotFound(id)
        }
        tunnels.remove(at: index)
        try persist(tunnels)
        cachedTunnels = tunnels
    }

    private func currentTunnels() throws -> [TunnelDefinition] {
        try requireWritable()
        if let cachedTunnels { return cachedTunnels }
        return try load()
    }

    private func requireWritable() throws {
        guard !isWriteLocked else { throw TunnelRepositoryError.writeLocked }
    }

    private func persist(_ tunnels: [TunnelDefinition]) throws {
        let directory = fileURL.deletingLastPathComponent()
        try FileManager.default.createDirectory(
            at: directory,
            withIntermediateDirectories: true,
            attributes: [.posixPermissions: 0o700]
        )
        let data = try Self.encoder.encode(TunnelEnvelope(tunnels: tunnels))
        try data.write(to: fileURL, options: .atomic)
    }

    private static let encoder: JSONEncoder = {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .iso8601
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        return encoder
    }()

    private static let decoder: JSONDecoder = {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        return decoder
    }()
}

import Foundation

public enum TunnelProtocol: String, CaseIterable, Codable, Sendable, Equatable, Identifiable {
    case http
    case https

    public var id: String { rawValue }
    public var displayName: String { rawValue.uppercased() }
}

public struct TunnelDefinition: Identifiable, Codable, Sendable, Equatable {
    public let id: UUID
    public var hostAlias: String
    public var name: String?
    public var remoteHost: String
    public var remotePort: UInt16
    public var localAddress: String
    public var localPort: UInt16
    public var webProtocol: TunnelProtocol
    public let createdAt: Date
    public var updatedAt: Date
    public var lastConnectedAt: Date?

    public init(
        id: UUID,
        hostAlias: String,
        name: String?,
        remoteHost: String,
        remotePort: UInt16,
        localAddress: String,
        localPort: UInt16,
        createdAt: Date,
        updatedAt: Date,
        lastConnectedAt: Date? = nil,
        webProtocol: TunnelProtocol = .http
    ) {
        self.id = id
        self.hostAlias = hostAlias
        self.name = name
        self.remoteHost = remoteHost
        self.remotePort = remotePort
        self.localAddress = localAddress
        self.localPort = localPort
        self.webProtocol = webProtocol
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.lastConnectedAt = lastConnectedAt
    }

    public var displayName: String {
        if let name, !name.isEmpty {
            return name
        }
        if localPort == remotePort {
            return String(remotePort)
        }
        return "\(localPort) → \(remotePort)"
    }

    private enum CodingKeys: String, CodingKey {
        case id, hostAlias, name, remoteHost, remotePort, localAddress, localPort, webProtocol, createdAt, updatedAt, lastConnectedAt
    }

    public init(from decoder: any Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(UUID.self, forKey: .id)
        hostAlias = try container.decode(String.self, forKey: .hostAlias)
        name = try container.decodeIfPresent(String.self, forKey: .name)
        remoteHost = try container.decode(String.self, forKey: .remoteHost)
        remotePort = try container.decode(UInt16.self, forKey: .remotePort)
        localAddress = try container.decode(String.self, forKey: .localAddress)
        localPort = try container.decode(UInt16.self, forKey: .localPort)
        webProtocol = try container.decodeIfPresent(TunnelProtocol.self, forKey: .webProtocol) ?? .http
        createdAt = try container.decode(Date.self, forKey: .createdAt)
        updatedAt = try container.decode(Date.self, forKey: .updatedAt)
        lastConnectedAt = try container.decodeIfPresent(Date.self, forKey: .lastConnectedAt)
    }

    public func encode(to encoder: any Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(id, forKey: .id)
        try container.encode(hostAlias, forKey: .hostAlias)
        try container.encodeIfPresent(name, forKey: .name)
        try container.encode(remoteHost, forKey: .remoteHost)
        try container.encode(remotePort, forKey: .remotePort)
        try container.encode(localAddress, forKey: .localAddress)
        try container.encode(localPort, forKey: .localPort)
        try container.encode(webProtocol, forKey: .webProtocol)
        try container.encode(createdAt, forKey: .createdAt)
        try container.encode(updatedAt, forKey: .updatedAt)
        try container.encodeIfPresent(lastConnectedAt, forKey: .lastConnectedAt)
    }
}

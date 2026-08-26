import Foundation

public enum TunnelState: Sendable, Equatable {
    case disconnected
    case connecting
    case connected
    case reconnecting
    case failed
}

public enum TunnelRuntimeID: Hashable, Sendable {
    case saved(UUID)
    case temporary(UUID)
}

public struct TunnelRuntimeSnapshot: Identifiable, Sendable, Equatable {
    public let id: TunnelRuntimeID
    public let hostAlias: String
    public let name: String?
    public let remoteHost: String
    public let remotePort: UInt16
    public let localAddress: String
    public let localPort: UInt16
    public let webProtocol: TunnelProtocol
    public let state: TunnelState
    public let desiredConnection: Bool
    public let lastError: String?
    public let logLines: [String]
    public let lastConnectedAt: Date?

    public init(
        id: TunnelRuntimeID,
        hostAlias: String,
        name: String?,
        remoteHost: String,
        remotePort: UInt16,
        localAddress: String,
        localPort: UInt16,
        state: TunnelState,
        desiredConnection: Bool,
        lastError: String?,
        logLines: [String],
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
        self.state = state
        self.desiredConnection = desiredConnection
        self.lastError = lastError
        self.logLines = logLines
        self.lastConnectedAt = lastConnectedAt
    }

    public var displayName: String {
        if let name, !name.isEmpty { return name }
        if localPort == remotePort { return String(remotePort) }
        return "\(localPort) → \(remotePort)"
    }

    public var browserURL: URL? {
        var components = URLComponents()
        components.scheme = webProtocol.rawValue
        let host = browserHost
        if host.contains(":") {
            components.percentEncodedHost = "[\(host)]"
        } else {
            components.host = host
        }
        components.port = Int(localPort)
        return components.url
    }

    private var browserHost: String {
        guard localAddress != "0.0.0.0", localAddress != "::" else { return "127.0.0.1" }
        guard localAddress.hasPrefix("["), localAddress.hasSuffix("]") else { return localAddress }
        return String(localAddress.dropFirst().dropLast())
    }
}

import Foundation

public enum SSHHostAvailability: Sendable, Equatable {
    case available
    case configurationError(String)
}

public struct SSHHost: Identifiable, Sendable, Equatable {
    public var id: String { alias }
    public let alias: String
    public let hostname: String
    public let user: String
    public let port: UInt16
    public let configOrder: Int
    public let availability: SSHHostAvailability

    public init(
        alias: String,
        hostname: String,
        user: String,
        port: UInt16,
        configOrder: Int,
        availability: SSHHostAvailability
    ) {
        self.alias = alias
        self.hostname = hostname
        self.user = user
        self.port = port
        self.configOrder = configOrder
        self.availability = availability
    }

    public func matches(query: String) -> Bool {
        guard !query.isEmpty else { return true }
        return alias.localizedCaseInsensitiveContains(query)
            || hostname.localizedCaseInsensitiveContains(query)
            || user.localizedCaseInsensitiveContains(query)
            || String(port).localizedCaseInsensitiveContains(query)
    }
}

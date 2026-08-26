import Darwin
import Foundation

public struct LocalEndpoint: Hashable, Sendable {
    public let address: String
    public let port: UInt16

    public init(address: String, port: UInt16) {
        self.address = address
        self.port = port
    }
}

public enum PortOwner: Sendable, Equatable {
    case tunnelDock
    case system
}

public enum PortAvailability: Sendable, Equatable {
    case available
    case occupied(PortOwner)
    case unavailable(String)
}

public protocol PortChecking: Sendable {
    func check(_ endpoint: LocalEndpoint) async -> PortAvailability
    func reserve(_ endpoint: LocalEndpoint) async -> Bool
    func release(_ endpoint: LocalEndpoint) async
    func isReserved(_ endpoint: LocalEndpoint) async -> Bool
}

public actor SystemPortAvailabilityChecker: PortChecking {
    private var reservations: Set<LocalEndpoint> = []

    public init() {}

    public func check(_ endpoint: LocalEndpoint) -> PortAvailability {
        if reservations.contains(endpoint) {
            return .occupied(.tunnelDock)
        }
        return Self.canBind(endpoint)
    }

    public func reserve(_ endpoint: LocalEndpoint) -> Bool {
        reservations.insert(endpoint).inserted
    }

    public func release(_ endpoint: LocalEndpoint) {
        reservations.remove(endpoint)
    }

    public func isReserved(_ endpoint: LocalEndpoint) -> Bool {
        reservations.contains(endpoint)
    }

    private static func canBind(_ endpoint: LocalEndpoint) -> PortAvailability {
        var hints = addrinfo()
        hints.ai_family = AF_UNSPEC
        hints.ai_socktype = SOCK_STREAM
        hints.ai_protocol = IPPROTO_TCP
        hints.ai_flags = AI_PASSIVE
        var result: UnsafeMutablePointer<addrinfo>?
        let status = getaddrinfo(endpoint.address, String(endpoint.port), &hints, &result)
        guard status == 0, let first = result else {
            return .unavailable(String(cString: gai_strerror(status)))
        }
        defer { freeaddrinfo(first) }

        var candidate: UnsafeMutablePointer<addrinfo>? = first
        while let info = candidate?.pointee {
            let descriptor = socket(info.ai_family, info.ai_socktype, info.ai_protocol)
            if descriptor >= 0 {
                var reuse: Int32 = 0
                setsockopt(
                    descriptor,
                    SOL_SOCKET,
                    SO_REUSEADDR,
                    &reuse,
                    socklen_t(MemoryLayout<Int32>.size)
                )
                let bindStatus = Darwin.bind(descriptor, info.ai_addr, info.ai_addrlen)
                Darwin.close(descriptor)
                if bindStatus == 0 {
                    return .available
                }
            }
            candidate = info.ai_next
        }
        return .occupied(.system)
    }
}

public protocol LocalListenerProbing: Sendable {
    func waitUntilListening(_ endpoint: LocalEndpoint, timeout: Duration) async -> Bool
}

public struct SystemLocalListenerProbe: LocalListenerProbing {
    public init() {}

    public func waitUntilListening(_ endpoint: LocalEndpoint, timeout: Duration) async -> Bool {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: timeout)
        repeat {
            let connected = await Task.detached {
                Self.canConnect(endpoint)
            }.value
            if connected { return true }
            if clock.now >= deadline { return false }
            try? await Task.sleep(for: .milliseconds(50))
        } while !Task.isCancelled
        return false
    }

    private static func canConnect(_ endpoint: LocalEndpoint) -> Bool {
        var hints = addrinfo()
        hints.ai_family = AF_UNSPEC
        hints.ai_socktype = SOCK_STREAM
        hints.ai_protocol = IPPROTO_TCP
        var result: UnsafeMutablePointer<addrinfo>?
        let status = getaddrinfo(endpoint.address, String(endpoint.port), &hints, &result)
        guard status == 0, let first = result else { return false }
        defer { freeaddrinfo(first) }

        var candidate: UnsafeMutablePointer<addrinfo>? = first
        while let info = candidate?.pointee {
            let descriptor = socket(info.ai_family, info.ai_socktype, info.ai_protocol)
            if descriptor >= 0 {
                let connectStatus = Darwin.connect(descriptor, info.ai_addr, info.ai_addrlen)
                Darwin.close(descriptor)
                if connectStatus == 0 { return true }
            }
            candidate = info.ai_next
        }
        return false
    }
}

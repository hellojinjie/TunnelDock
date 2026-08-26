import Combine
import TunnelDockCore

public enum QuickForwardField: Hashable, Sendable {
    case remotePort
    case localPort
    case remoteHost
    case localAddress
}

@MainActor
public final class QuickForwardModel: ObservableObject {
    @Published public var remotePort = ""
    @Published public private(set) var localPort = ""
    @Published public var remoteHost = "127.0.0.1"
    @Published public var localAddress = "127.0.0.1"
    @Published public var webProtocol: TunnelProtocol = .http
    @Published public var isAdvancedExpanded = false
    @Published public var focusedField: QuickForwardField?
    @Published public var errorMessage: String?

    private var localPortWasEdited = false

    public init() {}

    public func setRemotePort(_ value: String) {
        remotePort = value
        if !localPortWasEdited {
            localPort = value
        }
    }

    public func setLocalPort(_ value: String, userInitiated: Bool) {
        localPort = value
        if userInitiated { localPortWasEdited = true }
    }

    public func handle(_ error: TunnelManagerError) {
        errorMessage = error.description
        if case .localPortInUse = error {
            isAdvancedExpanded = true
            focusedField = .localPort
        }
    }

    public func reset() {
        remotePort = ""
        localPort = ""
        remoteHost = "127.0.0.1"
        localAddress = "127.0.0.1"
        webProtocol = .http
        isAdvancedExpanded = false
        focusedField = .remotePort
        errorMessage = nil
        localPortWasEdited = false
    }
}

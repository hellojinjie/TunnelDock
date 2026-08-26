import Foundation
import TunnelDockCore

public enum MenuBarTunnelAction: Sendable, Equatable {
    case connect
    case disconnect

    public init(state: TunnelState) {
        switch state {
        case .disconnected, .failed:
            self = .connect
        case .connecting, .connected, .reconnecting:
            self = .disconnect
        }
    }

    public var systemImage: String {
        switch self {
        case .connect: "play.fill"
        case .disconnect: "stop.fill"
        }
    }

    public var accessibilityLabel: String {
        switch self {
        case .connect: "Connect"
        case .disconnect: "Disconnect"
        }
    }
}

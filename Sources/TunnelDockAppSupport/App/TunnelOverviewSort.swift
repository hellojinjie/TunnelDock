import Foundation
import TunnelDockCore

public enum TunnelOverviewSort {
    public static func sorted(_ runtimes: [TunnelRuntimeSnapshot]) -> [TunnelRuntimeSnapshot] {
        runtimes.sorted { lhs, rhs in
            let (leftRank, rightRank) = (stateRank(lhs.state), stateRank(rhs.state))
            if leftRank != rightRank { return leftRank < rightRank }
            switch (lhs.lastConnectedAt, rhs.lastConnectedAt) {
            case let (left?, right?):
                if left != right { return left > right }
            case (.some, .none):
                return true
            case (.none, .some):
                return false
            case (.none, .none):
                break
            }
            if lhs.hostAlias != rhs.hostAlias { return lhs.hostAlias < rhs.hostAlias }
            return lhs.localPort < rhs.localPort
        }
    }

    static func stateRank(_ state: TunnelState) -> Int {
        switch state {
        case .connected: 0
        case .connecting: 1
        case .reconnecting: 2
        case .failed: 3
        case .disconnected: 4
        }
    }
}

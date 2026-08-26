import Foundation
import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum TunnelOverviewSortTests {
    static let all: [TestCase] = [
        TestCase("TunnelOverviewSortTests.ordersByStateThenLastConnectedAtDescending") {
            let connectedRecent = snapshot(state: .connected, lastConnectedAt: hoursAgo(1), port: 8_001)
            let connectedOlder = snapshot(state: .connected, lastConnectedAt: hoursAgo(4), port: 8_002)
            let connecting = snapshot(state: .connecting, lastConnectedAt: hoursAgo(9), port: 8_003)
            let reconnecting = snapshot(state: .reconnecting, lastConnectedAt: hoursAgo(9), port: 8_004)
            let failed = snapshot(state: .failed, lastConnectedAt: hoursAgo(1), port: 8_005)
            let disconnectedRecent = snapshot(state: .disconnected, lastConnectedAt: hoursAgo(1), port: 8_006)
            let disconnectedOlder = snapshot(state: .disconnected, lastConnectedAt: hoursAgo(5), port: 8_007)
            let disconnectedNever = snapshot(state: .disconnected, lastConnectedAt: nil, port: 8_008)

            let sorted = TunnelOverviewSort.sorted([
                disconnectedNever, disconnectedOlder, disconnectedRecent, failed,
                reconnecting, connecting, connectedOlder, connectedRecent,
            ])

            try expectEqual(
                sorted.map(\.localPort),
                [8_001, 8_002, 8_003, 8_004, 8_005, 8_006, 8_007, 8_008]
            )
        },
        TestCase("TunnelOverviewSortTests.breaksTiesDeterministically") {
            let first = snapshot(state: .connected, lastConnectedAt: hoursAgo(1), host: "alpha", port: 9_001)
            let second = snapshot(state: .connected, lastConnectedAt: hoursAgo(1), host: "alpha", port: 9_002)
            let third = snapshot(state: .connected, lastConnectedAt: nil, host: "beta", port: 9_000)

            for _ in 0..<3 {
                let sorted = TunnelOverviewSort.sorted([third, second, first])
                try expectEqual(sorted.map(\.id), [first.id, second.id, third.id])
            }
        },
    ]

    private static func hoursAgo(_ hours: Int) -> Date {
        Date(timeIntervalSince1970: 1_000_000 - TimeInterval(hours * 3_600))
    }

    private static func snapshot(
        state: TunnelState,
        lastConnectedAt: Date?,
        host: String = "gpu",
        port: UInt16
    ) -> TunnelRuntimeSnapshot {
        TunnelRuntimeSnapshot(
            id: .saved(UUID()),
            hostAlias: host,
            name: nil,
            remoteHost: "127.0.0.1",
            remotePort: port,
            localAddress: "127.0.0.1",
            localPort: port,
            state: state,
            desiredConnection: state != .disconnected && state != .failed,
            lastError: nil,
            logLines: [],
            lastConnectedAt: lastConnectedAt
        )
    }
}

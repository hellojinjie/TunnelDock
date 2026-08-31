import Foundation
import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum TunnelRowPresentationTests {
    static let all: [TestCase] = [
        TestCase("TunnelRowPresentationTests.allTunnelsProvidesHostBadge") {
            try expectEqual(
                TunnelRowPresentation.hostBadgeTitle(
                    for: snapshot(),
                    showsHostAlias: true
                ),
                "gpu"
            )
        },
        TestCase("TunnelRowPresentationTests.hostDetailOmitsHostBadge") {
            try expectEqual(
                TunnelRowPresentation.hostBadgeTitle(
                    for: snapshot(),
                    showsHostAlias: false
                ),
                nil
            )
        },
        TestCase("TunnelRowPresentationTests.subtitleContainsOnlyForwarding") {
            try expectEqual(
                TunnelRowPresentation.subtitle(for: snapshot()),
                "127.0.0.1:8888 → 127.0.0.1:8888"
            )
        },
    ]

    private static func snapshot() -> TunnelRuntimeSnapshot {
        TunnelRuntimeSnapshot(
            id: .saved(UUID()),
            hostAlias: "gpu",
            name: nil,
            remoteHost: "127.0.0.1",
            remotePort: 8_888,
            localAddress: "127.0.0.1",
            localPort: 8_888,
            state: .disconnected,
            desiredConnection: false,
            lastError: nil,
            logLines: [],
            lastConnectedAt: nil
        )
    }
}

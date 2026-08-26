import Foundation
import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum MenuBarModelTests {
    static let all: [TestCase] = [
        TestCase("MenuBarModelTests.excludesTemporaryTunnelsAndGroupsSavedByHost") {
            let jupyter = snapshot(id: .saved(UUID()), host: "gpu", name: "Jupyter", local: 8_888, remote: 8_888)
            let web = snapshot(id: .saved(UUID()), host: "nas", name: "Web UI", local: 9_000, remote: 80)
            let scratch = snapshot(id: .temporary(UUID()), host: "gpu", name: nil, local: 4_444, remote: 4_444)

            let result = MenuBarModel(runtimes: [scratch, web, jupyter])
            try expectEqual(result.rows.map(\.id), [jupyter.id, web.id])
            try expectEqual(result.groups.map(\.hostAlias), ["gpu", "nas"])
        },
        TestCase("MenuBarModelTests.searchesHostNameAndPorts") {
            let jupyter = snapshot(id: .saved(UUID()), host: "gpu", name: "Jupyter", local: 8_888, remote: 8_889)
            let web = snapshot(id: .saved(UUID()), host: "nas", name: "Web UI", local: 9_000, remote: 80)

            try expectEqual(MenuBarModel(runtimes: [jupyter, web], query: "8889").rows.map(\.id), [jupyter.id])
            try expectEqual(MenuBarModel(runtimes: [jupyter, web], query: "NAS").rows.map(\.id), [web.id])
        },
        TestCase("MenuBarTunnelActionTests.usesStopIconForEveryActiveTunnelState") {
            let actions = [
                TunnelState.disconnected,
                .failed,
                .connecting,
                .connected,
                .reconnecting,
            ].map(MenuBarTunnelAction.init(state:))

            try expectEqual(actions, [.connect, .connect, .disconnect, .disconnect, .disconnect])
            try expectEqual(
                actions.map(\.systemImage),
                ["play.fill", "play.fill", "stop.fill", "stop.fill", "stop.fill"]
            )
        },
    ]

    private static func snapshot(
        id: TunnelRuntimeID,
        host: String,
        name: String?,
        local: UInt16,
        remote: UInt16
    ) -> TunnelRuntimeSnapshot {
        TunnelRuntimeSnapshot(
            id: id, hostAlias: host, name: name,
            remoteHost: "127.0.0.1", remotePort: remote,
            localAddress: "127.0.0.1", localPort: local,
            state: .disconnected, desiredConnection: false,
            lastError: nil, logLines: []
        )
    }
}

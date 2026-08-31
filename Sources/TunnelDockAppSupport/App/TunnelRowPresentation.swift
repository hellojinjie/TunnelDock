import TunnelDockCore

public enum TunnelRowPresentation {
    public static func hostBadgeTitle(
        for snapshot: TunnelRuntimeSnapshot,
        showsHostAlias: Bool
    ) -> String? {
        showsHostAlias ? snapshot.hostAlias : nil
    }

    public static func subtitle(for snapshot: TunnelRuntimeSnapshot) -> String {
        "\(snapshot.localAddress):\(snapshot.localPort) → \(snapshot.remoteHost):\(snapshot.remotePort)"
    }
}

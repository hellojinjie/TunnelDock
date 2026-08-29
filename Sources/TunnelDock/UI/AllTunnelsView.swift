import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct AllTunnelsView: View {
    @ObservedObject var appState: AppState
    @ObservedObject var tunnelManager: TunnelManager

    private var tunnels: [TunnelRuntimeSnapshot] {
        TunnelOverviewSort.sorted(tunnelManager.runtimes)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                Text("All Tunnels").font(.largeTitle).bold()
                GroupBox {
                    VStack(spacing: 0) {
                        if tunnels.isEmpty {
                            Text("Connect a host to see its tunnel here.")
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8)
                        } else {
                            ForEach(tunnels) { tunnel in
                                TunnelRow(
                                    snapshot: tunnel,
                                    manager: tunnelManager,
                                    canConnect: canConnect(tunnel),
                                    showsHostAlias: true
                                )
                                if tunnel.id != tunnels.last?.id { Divider() }
                            }
                        }
                    }
                }
            }
            .padding(24)
            .frame(maxWidth: 760, alignment: .leading)
        }
    }

    private func canConnect(_ snapshot: TunnelRuntimeSnapshot) -> Bool {
        appState.allHosts.first { $0.alias == snapshot.hostAlias }?.availability == .available
    }
}

import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct HostDetailView: View {
    let alias: String
    @ObservedObject var appState: AppState
    @ObservedObject var tunnelManager: TunnelManager

    private var host: SSHHost? { appState.allHosts.first { $0.alias == alias } }
    private var tunnels: [TunnelRuntimeSnapshot] {
        tunnelManager.runtimes.filter { $0.hostAlias == alias }
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(alias).font(.largeTitle).bold()
                    if let host {
                        Text("\(host.user) @ \(host.hostname):\(host.port)")
                            .foregroundStyle(.secondary)
                        if case .configurationError = host.availability {
                            Label("SSH configuration could not be resolved.", systemImage: "exclamationmark.triangle")
                                .foregroundStyle(.orange)
                        }
                    } else {
                        Label("Host not found", systemImage: "exclamationmark.triangle")
                            .foregroundStyle(.orange)
                    }
                }

                GroupBox("Recent Tunnels") {
                    VStack(spacing: 0) {
                        if tunnels.isEmpty {
                            Text("No tunnels for this host.")
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(.vertical, 8)
                        } else {
                            ForEach(tunnels) { tunnel in
                                TunnelRow(
                                    snapshot: tunnel,
                                    manager: tunnelManager,
                                    canConnect: host?.availability == .available
                                )
                                if tunnel.id != tunnels.last?.id { Divider() }
                            }
                        }
                    }
                }

                if let host, host.availability == .available {
                    GroupBox("Quick Forward") {
                        QuickForwardView(hostAlias: alias, manager: tunnelManager)
                            .padding(.top, 4)
                    }
                }
            }
            .padding(24)
            .frame(maxWidth: 760, alignment: .leading)
        }
    }
}

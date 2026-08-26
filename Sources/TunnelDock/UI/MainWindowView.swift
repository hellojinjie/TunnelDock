import AppKit
import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct MainWindowView: View {
    @ObservedObject var model: AppModel

    var body: some View {
        NavigationSplitView {
            HostSidebar(
                appState: model.appState,
                selection: $model.selectedPaneID,
                refresh: { Task { await model.refreshSSHConfig() } },
                addHost: { host in try await model.addSSHHost(host) },
                openConfigInEditor: { model.openSSHConfigInDefaultEditor() }
            )
            .navigationSplitViewColumnWidth(min: 210, ideal: 240)
        } detail: {
            switch model.selectedPane {
            case .allTunnels:
                AllTunnelsView(
                    appState: model.appState,
                    tunnelManager: model.tunnelManager
                )
            case .host(let alias):
                HostDetailView(
                    alias: alias,
                    appState: model.appState,
                    tunnelManager: model.tunnelManager
                )
            case .none:
                VStack(spacing: 10) {
                    Image(systemName: "network.slash")
                        .font(.system(size: 36))
                        .foregroundStyle(.secondary)
                    Text("No SSH hosts found").font(.title2)
                    Text("TunnelDock reads hosts from ~/.ssh/config")
                        .foregroundStyle(.secondary)
                }
            }
        }
        .overlay(alignment: .bottom) {
            if let error = model.lifecycleError {
                Text(error)
                    .font(.callout)
                    .padding(10)
                    .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 8))
                    .padding()
            }
        }
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                if #available(macOS 14.0, *) {
                    SettingsLink {
                        Label("Settings", systemImage: "gearshape")
                    }
                    .help("Settings")
                } else {
                    Button {
                        NSApp.sendAction(Selector(("showSettingsWindow:")), to: nil, from: nil)
                        NSApp.activate(ignoringOtherApps: true)
                    } label: {
                        Label("Settings", systemImage: "gearshape")
                    }
                    .help("Settings")
                }
            }
        }
    }
}

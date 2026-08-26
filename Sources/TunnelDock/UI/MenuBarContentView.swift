import AppKit
import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct MenuBarContentView: View {
    @ObservedObject var model: AppModel
    @ObservedObject var tunnelManager: TunnelManager
    @Environment(\.openWindow) private var openWindow
    @State private var query = ""
    @State private var errorMessage: String?

    private var menu: MenuBarModel {
        MenuBarModel(runtimes: tunnelManager.runtimes, query: query)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            TextField("Search Saved Tunnels", text: $query)
                .textFieldStyle(.roundedBorder)

            if menu.groups.isEmpty {
                Text("No saved tunnels")
                    .foregroundStyle(.secondary)
            } else {
                ForEach(menu.groups) { group in
                    VStack(alignment: .leading, spacing: 2) {
                        Text(group.hostAlias)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        ForEach(group.rows) { row in
                            let action = MenuBarTunnelAction(state: row.state)
                            HStack(spacing: 8) {
                                VStack(alignment: .leading, spacing: 0) {
                                    Text(row.displayName)
                                        .lineLimit(1)
                                    Text("\(row.localPort) → \(row.remoteHost):\(row.remotePort)")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                        .lineLimit(1)
                                }
                                Spacer(minLength: 8)
                                Button {
                                    toggle(row)
                                } label: {
                                    Image(systemName: action.systemImage)
                                        .frame(width: 20, height: 20)
                                }
                                .buttonStyle(.plain)
                                .help(action.accessibilityLabel)
                                .accessibilityLabel(action.accessibilityLabel)
                                .disabled(action == .connect && !canConnect(row))
                            }
                            .padding(.vertical, 2)
                        }
                    }
                }
            }

            if let errorMessage {
                Text(errorMessage).font(.caption).foregroundStyle(.red)
            }
            Divider()
            HStack {
                Button("Open TunnelDock") { openMainWindow() }
                Button("Refresh") { Task { await model.refreshSSHConfig() } }
            }
            HStack {
                if #available(macOS 14.0, *) {
                    SettingsLink {
                        Text("Settings…")
                    }
                } else {
                    Button("Settings…") {
                        NSApp.sendAction(Selector(("showSettingsWindow:")), to: nil, from: nil)
                        NSApp.activate(ignoringOtherApps: true)
                    }
                }
                Spacer()
                Button("Quit") { NSApp.terminate(nil) }
            }
        }
        .padding(12)
        .frame(width: 320)
    }

    private func toggle(_ row: TunnelRuntimeSnapshot) {
        errorMessage = nil
        Task {
            do {
                if MenuBarTunnelAction(state: row.state) == .disconnect {
                    try await tunnelManager.disconnect(id: row.id)
                } else if case let .saved(id) = row.id {
                    try await tunnelManager.connectSaved(id: id)
                }
            } catch {
                errorMessage = (error as? TunnelManagerError)?.description ?? error.localizedDescription
            }
        }
    }

    private func canConnect(_ row: TunnelRuntimeSnapshot) -> Bool {
        model.appState.allHosts.first { $0.alias == row.hostAlias }?.availability == .available
    }

    private func openMainWindow() {
        let mainWindow = NSApp.windows.first { $0.title == "TunnelDock" }
        NSApp.activate(ignoringOtherApps: true)
        switch AppReopenCoordinator.mainWindowPresentation(hasExistingMainWindow: mainWindow != nil) {
        case .revealExisting:
            mainWindow?.makeKeyAndOrderFront(nil)
        case .openNew:
            openWindow(id: "main")
        }
    }
}

import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct HostSidebar: View {
    @ObservedObject var appState: AppState
    @Binding var selection: String?
    let refresh: () -> Void
    let addHost: @MainActor (SSHHostConfiguration) async throws -> Void
    let openConfigInEditor: () -> Void
    @State private var isPresentingAddHost = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 4) {
                sidebarRow(isSelected: selection == MainPane.allTunnelsID) {
                    Label("All Tunnels", systemImage: "rectangle.stack")
                }
                .onTapGesture { selection = MainPane.allTunnelsID }
                .padding(.bottom, 12)

                HStack(spacing: 8) {
                    Text("SSH Hosts")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    Spacer()
                    if !appState.allHosts.isEmpty {
                        searchField
                    }
                }
                .padding(.bottom, 4)

                ForEach(appState.normalHosts) { host in
                    sidebarRow(isSelected: selection == host.alias) {
                        HStack {
                            Label(host.alias, systemImage: hostIcon(host))
                            Spacer(minLength: 0)
                            if appState.activeTunnelHostAliases.contains(host.alias) {
                                Image(systemName: "point.3.connected.trianglepath.dotted")
                                    .foregroundStyle(.green)
                                    .accessibilityLabel("Tunnel active")
                            }
                        }
                    }
                    .onTapGesture { selection = host.alias }
                }

                if !appState.filteredMissingHosts.isEmpty {
                    Text("Missing Hosts")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .padding(.top, 12)
                        .padding(.bottom, 4)
                    ForEach(appState.filteredMissingHosts) { host in
                        sidebarRow(isSelected: selection == host.alias) {
                            Label(host.alias, systemImage: "exclamationmark.triangle")
                        }
                        .onTapGesture { selection = host.alias }
                    }
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 8)
        }
        .toolbar {
            Button {
                isPresentingAddHost = true
            } label: {
                Label("Quick Add SSH Host", systemImage: "plus")
            }
            .help("Quick Add SSH Host")

            Button(action: openConfigInEditor) {
                Label("Open SSH Config in Default Editor", systemImage: "square.and.pencil")
            }
            .help("Open ~/.ssh/config in Default Editor")

            Button(action: refresh) {
                Label("Refresh SSH Config", systemImage: "arrow.clockwise")
            }
            .disabled(appState.isRefreshing)
        }
        .sheet(isPresented: $isPresentingAddHost) {
            SSHHostQuickAddView(save: addHost)
        }
    }

    private var searchField: some View {
        HStack(spacing: 4) {
            Image(systemName: "magnifyingglass")
                .font(.caption)
                .foregroundStyle(.secondary)
            SidebarSearchField(text: $appState.searchQuery)
                .frame(height: 18)
            if !appState.searchQuery.isEmpty {
                Button {
                    appState.searchQuery = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(.horizontal, 7)
        .padding(.vertical, 4)
        .background(Capsule().fill(Color(nsColor: .controlBackgroundColor)))
        .overlay(Capsule().strokeBorder(Color.primary.opacity(0.08)))
        .frame(maxWidth: 150)
    }

    @ViewBuilder
    private func sidebarRow<Content: View>(
        isSelected: Bool,
        @ViewBuilder content: () -> Content
    ) -> some View {
        content()
            .frame(maxWidth: .infinity, minHeight: 24, alignment: .leading)
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background(
                RoundedRectangle(cornerRadius: 6)
                    .fill(isSelected ? Color.accentColor.opacity(0.2) : Color.clear)
            )
            .contentShape(Rectangle())
    }

    private func hostIcon(_ host: SSHHost) -> String {
        host.availability == .available ? "server.rack" : "exclamationmark.triangle"
    }
}

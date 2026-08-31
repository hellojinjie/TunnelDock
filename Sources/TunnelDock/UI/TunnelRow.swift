import AppKit
import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct TunnelRow: View {
    let snapshot: TunnelRuntimeSnapshot
    @ObservedObject var manager: TunnelManager
    let canConnect: Bool
    var showsHostAlias = false
    @State private var isWorking = false
    @State private var presentedSheet: PresentedSheet?
    @State private var errorMessage: String?
    @State private var isConfirmingDelete = false

    private enum PresentedSheet: Identifiable {
        case edit
        case log
        case rename

        var id: Int {
            switch self {
            case .edit: 0
            case .log: 1
            case .rename: 2
            }
        }
    }

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: stateIcon)
                .foregroundStyle(stateColor)
                .frame(width: 18)
            VStack(alignment: .leading, spacing: 3) {
                Text(snapshot.displayName)
                    .fontWeight(.medium)
                HStack(spacing: 6) {
                    if let hostAlias = TunnelRowPresentation.hostBadgeTitle(
                        for: snapshot,
                        showsHostAlias: showsHostAlias
                    ) {
                        Text(hostAlias)
                            .font(.caption2)
                            .fontWeight(.semibold)
                            .foregroundStyle(.blue)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.blue.opacity(0.12), in: Capsule())
                    }
                    Text(TunnelRowPresentation.subtitle(for: snapshot))
                    .font(.caption)
                    .foregroundStyle(.secondary)
                }
                if let error = snapshot.lastError {
                    Text(error).font(.caption).foregroundStyle(.red)
                } else if let errorMessage {
                    Text(errorMessage).font(.caption).foregroundStyle(.red)
                }
            }
            Spacer()
            Text(stateTitle)
                .font(.caption)
                .foregroundStyle(.secondary)
            if snapshot.state == .connected {
                Button(action: openInBrowser) {
                    Label("Open in Browser", systemImage: "globe")
                }
                .labelStyle(.iconOnly)
                .help("Open in Browser")
            }
            Button(snapshot.state == .connected || snapshot.state == .connecting || snapshot.state == .reconnecting ? "Disconnect" : "Connect") {
                toggleConnection()
            }
            .disabled(isWorking || (canModify && !canConnect))
            Menu {
                Button("View Log") { presentedSheet = .log }
                if !isTemporary {
                    Button("Rename…") { presentedSheet = .rename }
                    Button("Edit…") { presentedSheet = .edit }
                        .disabled(!canModify)
                    Button("Delete", role: .destructive) { isConfirmingDelete = true }
                        .disabled(!canModify)
                }
            } label: {
                Image(systemName: "ellipsis.circle")
            }
            .menuStyle(.borderlessButton)
            .frame(width: 28)
        }
        .padding(.vertical, 9)
        .sheet(item: $presentedSheet) { sheet in
            switch sheet {
            case .edit:
                TunnelEditorView(snapshot: snapshot, manager: manager)
            case .log:
                TunnelLogView(snapshot: snapshot)
            case .rename:
                TunnelNameView(title: "Rename Tunnel", initialName: snapshot.name) { name in
                    try await manager.rename(id: snapshot.id, name: name)
                }
            }
        }
        .confirmationDialog(
            "Delete this recent tunnel?",
            isPresented: $isConfirmingDelete,
            titleVisibility: .visible
        ) {
            Button("Delete", role: .destructive) { delete() }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("The recent tunnel will be removed from the list and from saved tunnels.")
        }
    }

    private var canModify: Bool {
        snapshot.state == .disconnected || snapshot.state == .failed
    }

    private var isTemporary: Bool {
        if case .temporary = snapshot.id { return true }
        return false
    }

    private var stateTitle: String {
        switch snapshot.state {
        case .disconnected: "Disconnected"
        case .connecting: "Connecting"
        case .connected: "Connected"
        case .reconnecting: "Reconnecting"
        case .failed: "Failed"
        }
    }

    private var stateIcon: String {
        switch snapshot.state {
        case .connected: "checkmark.circle.fill"
        case .connecting, .reconnecting: "arrow.triangle.2.circlepath.circle"
        case .failed: "exclamationmark.circle.fill"
        case .disconnected: "circle"
        }
    }

    private var stateColor: Color {
        switch snapshot.state {
        case .connected: .green
        case .connecting, .reconnecting: .blue
        case .failed: .red
        case .disconnected: .secondary
        }
    }

    private func toggleConnection() {
        isWorking = true
        errorMessage = nil
        Task {
            defer { isWorking = false }
            do {
                if canModify {
                    guard case let .saved(id) = snapshot.id else { return }
                    try await manager.connectSaved(id: id)
                } else {
                    try await manager.disconnect(id: snapshot.id)
                }
            } catch {
                errorMessage = (error as? TunnelManagerError)?.description ?? error.localizedDescription
            }
        }
    }

    private func delete() {
        Task {
            do { try await manager.delete(id: snapshot.id) }
            catch { errorMessage = (error as? TunnelManagerError)?.description ?? error.localizedDescription }
        }
    }

    private func openInBrowser() {
        guard let url = snapshot.browserURL else {
            errorMessage = "Cannot build a browser URL for \(snapshot.localAddress)."
            return
        }
        NSWorkspace.shared.open(url)
    }
}

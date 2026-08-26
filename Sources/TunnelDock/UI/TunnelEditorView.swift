import SwiftUI
import TunnelDockCore

struct TunnelEditorView: View {
    let snapshot: TunnelRuntimeSnapshot
    @ObservedObject var manager: TunnelManager
    @Environment(\.dismiss) private var dismiss
    @State private var name: String
    @State private var remoteHost: String
    @State private var remotePort: String
    @State private var localAddress: String
    @State private var localPort: String
    @State private var webProtocol: TunnelProtocol
    @State private var errorMessage: String?
    @State private var isSaving = false

    init(snapshot: TunnelRuntimeSnapshot, manager: TunnelManager) {
        self.snapshot = snapshot
        self.manager = manager
        _name = State(initialValue: snapshot.name ?? "")
        _remoteHost = State(initialValue: snapshot.remoteHost)
        _remotePort = State(initialValue: String(snapshot.remotePort))
        _localAddress = State(initialValue: snapshot.localAddress)
        _localPort = State(initialValue: String(snapshot.localPort))
        _webProtocol = State(initialValue: snapshot.webProtocol)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Edit Tunnel").font(.title2).bold()
            Form {
                TextField("Name", text: $name)
                TextField("Remote Host", text: $remoteHost)
                TextField("Remote Port", text: $remotePort)
                TextField("Local Address", text: $localAddress)
                TextField("Local Port", text: $localPort)
                Picker("Browser URL Scheme", selection: $webProtocol) {
                    ForEach(TunnelProtocol.allCases) { webProtocol in
                        Text(webProtocol.displayName).tag(webProtocol)
                    }
                }
                Text("Only used by Open in Browser; SSH forwarding is unchanged.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if let errorMessage {
                Text(errorMessage).font(.callout).foregroundStyle(.red)
            }
            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button("Save") { save() }
                    .keyboardShortcut(.defaultAction)
                    .disabled(isSaving)
            }
        }
        .padding(24)
        .frame(width: 440)
    }

    private func save() {
        isSaving = true
        errorMessage = nil
        Task {
            defer { isSaving = false }
            do {
                try await manager.edit(
                    id: snapshot.id,
                    input: TunnelInput(
                        name: name,
                        remoteHost: remoteHost,
                        remotePort: remotePort,
                        localAddress: localAddress,
                        localPort: localPort,
                        webProtocol: webProtocol
                    )
                )
                dismiss()
            } catch {
                errorMessage = (error as? TunnelManagerError)?.description ?? error.localizedDescription
            }
        }
    }
}

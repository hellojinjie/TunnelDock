import AppKit
import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct SSHHostQuickAddView: View {
    let save: @MainActor (SSHHostConfiguration) async throws -> Void

    @Environment(\.dismiss) private var dismiss
    @StateObject private var draft = SSHHostDraft()
    @State private var errorMessage: String?
    @State private var isSaving = false

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Text("Quick Add SSH Host")
                .font(.title2.weight(.semibold))

            Grid(alignment: .leading, horizontalSpacing: 12, verticalSpacing: 12) {
                GridRow {
                    Text("HostName")
                    TextField("Required", text: $draft.hostname)
                }
                GridRow {
                    Text("Host")
                    TextField("Defaults to HostName", text: Binding(
                        get: { draft.alias },
                        set: { draft.setAlias($0) }
                    ))
                }
                GridRow {
                    Text("User")
                    TextField("Current user", text: $draft.user)
                }
                GridRow {
                    Text("Port")
                    TextField("22", text: $draft.port)
                }
            }
            .textFieldStyle(.roundedBorder)

            Text("A new Host block will be appended to ~/.ssh/config. Existing configuration is never changed or deleted.")
                .font(.caption)
                .foregroundStyle(.secondary)

            if let errorMessage {
                Text(errorMessage)
                    .font(.callout)
                    .foregroundStyle(.red)
            }

            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                    .disabled(isSaving)
                Button(isSaving ? "Adding…" : "Add Host") {
                    submit()
                }
                .disabled(isSaving || draft.hostname.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }
        }
        .padding(24)
        .frame(width: 450)
        .onAppear {
            DispatchQueue.main.async {
                NSApp.keyWindow?.makeFirstResponder(nil)
            }
        }
    }

    private func submit() {
        Task {
            isSaving = true
            defer { isSaving = false }
            do {
                try await save(try draft.configuration())
                dismiss()
            } catch {
                errorMessage = error.localizedDescription
            }
        }
    }
}

import SwiftUI
import TunnelDockCore

struct TunnelNameView: View {
    let title: String
    let action: @MainActor (String?) async throws -> Void
    @Environment(\.dismiss) private var dismiss
    @State private var name: String
    @State private var errorMessage: String?
    @State private var isWorking = false

    init(
        title: String,
        initialName: String?,
        action: @escaping @MainActor (String?) async throws -> Void
    ) {
        self.title = title
        self.action = action
        _name = State(initialValue: initialName ?? "")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text(title).font(.title2).bold()
            TextField("Name (optional)", text: $name)
                .textFieldStyle(.roundedBorder)
            if let errorMessage {
                Text(errorMessage).font(.callout).foregroundStyle(.red)
            }
            HStack {
                Spacer()
                Button("Cancel") { dismiss() }
                Button(title.replacingOccurrences(of: " Tunnel", with: "")) { submit() }
                    .keyboardShortcut(.defaultAction)
                    .disabled(isWorking)
            }
        }
        .padding(24)
        .frame(width: 400)
    }

    private func submit() {
        isWorking = true
        errorMessage = nil
        Task {
            defer { isWorking = false }
            do {
                try await action(name)
                dismiss()
            } catch {
                errorMessage = (error as? TunnelManagerError)?.description ?? error.localizedDescription
            }
        }
    }
}

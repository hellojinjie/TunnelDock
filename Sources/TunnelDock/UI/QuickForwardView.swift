import SwiftUI
import TunnelDockAppSupport
import TunnelDockCore

struct QuickForwardView: View {
    let hostAlias: String
    @ObservedObject var manager: TunnelManager
    @StateObject private var model = QuickForwardModel()
    @FocusState private var focus: QuickForwardField?
    @State private var isConnecting = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                TextField("Remote Port", text: Binding(
                    get: { model.remotePort },
                    set: { model.setRemotePort($0) }
                ))
                .textFieldStyle(.roundedBorder)
                .frame(maxWidth: 180)
                .focused($focus, equals: .remotePort)
                Button(isConnecting ? "Connecting…" : "Connect") {
                    connect()
                }
                .disabled(isConnecting || model.remotePort.isEmpty)
            }

            DisclosureGroup("Advanced", isExpanded: $model.isAdvancedExpanded) {
                Grid(alignment: .leading, horizontalSpacing: 12, verticalSpacing: 10) {
                    GridRow {
                        Text("Local Port")
                        TextField("Local Port", text: Binding(
                            get: { model.localPort },
                            set: { model.setLocalPort($0, userInitiated: true) }
                        ))
                        .focused($focus, equals: .localPort)
                    }
                    GridRow {
                        Text("Remote Host")
                        TextField("Remote Host", text: $model.remoteHost)
                            .focused($focus, equals: .remoteHost)
                    }
                    GridRow {
                        Text("Local Address")
                        TextField("Local Address", text: $model.localAddress)
                            .focused($focus, equals: .localAddress)
                    }
                    GridRow {
                        Text("Browser URL Scheme")
                        Picker("Browser URL Scheme", selection: $model.webProtocol) {
                            ForEach(TunnelProtocol.allCases) { webProtocol in
                                Text(webProtocol.displayName).tag(webProtocol)
                            }
                        }
                        .labelsHidden()
                    }
                }
                .textFieldStyle(.roundedBorder)
                .padding(.top, 8)
                Text("Only used by Open in Browser; SSH forwarding is unchanged.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .padding(.top, 4)
            }

            if let error = model.errorMessage {
                Text(error).font(.callout).foregroundStyle(.red)
            }
        }
        .onReceive(model.$focusedField) { focus = $0 }
    }

    private func connect() {
        isConnecting = true
        model.errorMessage = nil
        Task {
            defer { isConnecting = false }
            do {
                _ = try await manager.connectRecent(
                    hostAlias: hostAlias,
                    input: TunnelInput(
                        remoteHost: model.remoteHost,
                        remotePort: model.remotePort,
                        localAddress: model.localAddress,
                        localPort: model.localPort,
                        webProtocol: model.webProtocol
                    )
                )
                model.reset()
            } catch let error as TunnelManagerError {
                model.handle(error)
            } catch {
                model.errorMessage = error.localizedDescription
            }
        }
    }
}

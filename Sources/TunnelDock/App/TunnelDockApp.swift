import SwiftUI

@main
struct TunnelDockApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var appDelegate
    @StateObject private var model = AppModel()
    @AppStorage("showMenuBar") private var showMenuBar = true

    var body: some Scene {
        WindowGroup("TunnelDock", id: "main") {
            MainWindowView(model: model)
                .frame(minWidth: 820, minHeight: 560)
                .task {
                    appDelegate.model = model
                    await model.start()
                }
        }
        .windowStyle(.titleBar)
        .commands {
            CommandGroup(after: .appInfo) {
                Button("Refresh SSH Config") {
                    Task { await model.refreshSSHConfig() }
                }
                .keyboardShortcut("r", modifiers: .command)
            }
        }

        MenuBarExtra(
            "TunnelDock",
            systemImage: "point.3.connected.trianglepath.dotted",
            isInserted: $showMenuBar
        ) {
            MenuBarContentView(model: model, tunnelManager: model.tunnelManager)
        }
        .menuBarExtraStyle(.window)

        Settings {
            SettingsView(showMenuBar: $showMenuBar)
        }
    }
}

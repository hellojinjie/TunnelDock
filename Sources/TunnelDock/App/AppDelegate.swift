import AppKit
import TunnelDockAppSupport

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    weak var model: AppModel?
    private var terminationCoordinator: AppTerminationCoordinator?
    private let reopenCoordinator = AppReopenCoordinator()

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        false
    }

    func applicationShouldHandleReopen(
        _ sender: NSApplication,
        hasVisibleWindows _: Bool
    ) -> Bool {
        reopenCoordinator.applicationShouldHandleReopen(
            activate: { sender.activate(ignoringOtherApps: true) },
            revealMainWindow: {
                sender.windows.first { $0.title == "TunnelDock" }?.makeKeyAndOrderFront(nil)
            }
        )
    }

    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        guard let model else { return .terminateNow }
        let coordinator: AppTerminationCoordinator
        if let terminationCoordinator {
            coordinator = terminationCoordinator
        } else {
            coordinator = AppTerminationCoordinator(
                shutdown: { [weak model] in await model?.tunnelManager.shutdownAll() },
                reply: { sender.reply(toApplicationShouldTerminate: $0) }
            )
            terminationCoordinator = coordinator
        }
        _ = coordinator.applicationShouldTerminate()
        return .terminateLater
    }
}

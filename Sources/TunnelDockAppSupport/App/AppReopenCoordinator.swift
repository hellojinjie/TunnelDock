import Foundation

public enum MainWindowPresentation: Sendable, Equatable {
    case revealExisting
    case openNew
}

@MainActor
public final class AppReopenCoordinator {
    public typealias Activate = @MainActor () -> Void
    public typealias RevealMainWindow = @MainActor () -> Void

    public init() {}

    nonisolated public static func mainWindowPresentation(
        hasExistingMainWindow: Bool
    ) -> MainWindowPresentation {
        hasExistingMainWindow ? .revealExisting : .openNew
    }

    public func applicationShouldHandleReopen(
        activate: Activate,
        revealMainWindow: RevealMainWindow
    ) -> Bool {
        activate()
        revealMainWindow()
        return true
    }
}

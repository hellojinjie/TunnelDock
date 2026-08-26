import Foundation

public enum ApplicationTerminationDecision: Sendable, Equatable {
    case terminateLater
}

@MainActor
public final class AppTerminationCoordinator {
    public typealias Shutdown = @MainActor @Sendable () async -> Void
    public typealias Reply = @MainActor @Sendable (Bool) -> Void

    private let shutdown: Shutdown
    private let reply: Reply
    private var terminationTask: Task<Void, Never>?

    public init(
        shutdown: @escaping Shutdown,
        reply: @escaping Reply
    ) {
        self.shutdown = shutdown
        self.reply = reply
    }

    public func applicationShouldTerminate() -> ApplicationTerminationDecision {
        if terminationTask == nil {
            terminationTask = Task { [shutdown, reply] in
                await shutdown()
                reply(true)
            }
        }
        return .terminateLater
    }

    public func waitUntilFinished() async {
        await terminationTask?.value
    }
}

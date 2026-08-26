import TestSupport
import TunnelDockAppSupport

@MainActor
private final class TerminationHarness {
    var shutdownCount = 0
    var replies: [Bool] = []

    func shutdown() async { shutdownCount += 1 }
}

enum AppTerminationCoordinatorTests {
    static let all: [TestCase] = [
        TestCase("AppReopenCoordinator.reusesExistingMainWindowBeforeOpeningAnother") {
            try expectEqual(
                AppReopenCoordinator.mainWindowPresentation(hasExistingMainWindow: true),
                .revealExisting
            )
            try expectEqual(
                AppReopenCoordinator.mainWindowPresentation(hasExistingMainWindow: false),
                .openNew
            )
        },
        TestCase("AppReopenCoordinator.acceptsDockReopenWhenMainWindowWasClosed") {
            var activationCount = 0
            var revealCount = 0
            let coordinator = await AppReopenCoordinator()

            let shouldHandle = await MainActor.run {
                coordinator.applicationShouldHandleReopen(
                    activate: { activationCount += 1 },
                    revealMainWindow: { revealCount += 1 }
                )
            }

            try expectEqual(shouldHandle, true)
            try expectEqual(activationCount, 1)
            try expectEqual(revealCount, 1)
        },
        TestCase("AppTerminationCoordinatorTests.delaysAndRepliesAfterOneShutdown") {
            let harness = TerminationHarness()
            let coordinator = await AppTerminationCoordinator(
                shutdown: { await harness.shutdown() },
                reply: { harness.replies.append($0) }
            )

            let decisions = await MainActor.run {
                (coordinator.applicationShouldTerminate(), coordinator.applicationShouldTerminate())
            }
            try expectEqual(decisions.0, .terminateLater)
            try expectEqual(decisions.1, .terminateLater)
            await coordinator.waitUntilFinished()

            let result = await MainActor.run { (harness.shutdownCount, harness.replies) }
            try expectEqual(result.0, 1)
            try expectEqual(result.1, [true])
        },
    ]
}

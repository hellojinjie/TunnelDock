import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum QuickForwardModelTests {
    static let all: [TestCase] = [
        TestCase("QuickForwardModelTests.localPortFollowsUntilExplicitlyEdited") {
            let values = await MainActor.run { () -> [String] in
                let model = QuickForwardModel()
                model.setRemotePort("8888")
                let followed = model.localPort
                model.setLocalPort("18888", userInitiated: true)
                model.setRemotePort("6006")
                return [followed, model.localPort]
            }

            try expectEqual(values, ["8888", "18888"])
        },
        TestCase("QuickForwardModelTests.collisionExpandsAdvancedAndFocusesLocalPort") {
            let values = await MainActor.run { () -> (Bool, QuickForwardField?) in
                let model = QuickForwardModel()
                model.handle(.localPortInUse(8_888, owner: .system))
                return (model.isAdvancedExpanded, model.focusedField)
            }

            try expectEqual(values.0, true)
            try expectEqual(values.1, .localPort)
        },
        TestCase("QuickForwardModelTests.resetRestoresSafeLoopbackDefaults") {
            let values = await MainActor.run { () -> [String] in
                let model = QuickForwardModel()
                model.remoteHost = "db.internal"
                model.localAddress = "0.0.0.0"
                model.setLocalPort("18888", userInitiated: true)
                model.reset()
                return [model.remoteHost, model.localAddress, model.localPort, model.remotePort]
            }

            try expectEqual(values, ["127.0.0.1", "127.0.0.1", "", ""])
        },
        TestCase("QuickForwardModelTests.protocolDefaultsToHTTPAndResetsToHTTP") {
            let values = await MainActor.run { () -> [TunnelProtocol] in
                let model = QuickForwardModel()
                let initial = model.webProtocol
                model.webProtocol = .https
                model.reset()
                return [initial, model.webProtocol]
            }

            try expectEqual(values, [.http, .http])
        },
    ]
}

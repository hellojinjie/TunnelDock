import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum SSHHostDraftTests {
    static let all: [TestCase] = [
        TestCase("SSHHostDraftTests.defaultsUserAndPortAndFollowsHostnameForAlias") {
            let values = try await MainActor.run { () throws -> (String, String, String, SSHHostConfiguration) in
                let draft = SSHHostDraft(currentUser: "alice")
                draft.hostname = "db.internal"
                let followingAlias = draft.alias
                draft.setAlias("production-db")
                draft.hostname = "replacement.internal"
                return (followingAlias, draft.user, draft.port, try draft.configuration())
            }

            try expectEqual(values.0, "db.internal")
            try expectEqual(values.1, "alice")
            try expectEqual(values.2, "22")
            try expectEqual(
                values.3,
                SSHHostConfiguration(
                    alias: "production-db",
                    hostname: "replacement.internal",
                    user: "alice",
                    port: 22
                )
            )
        },
        TestCase("SSHHostDraftTests.emptyAliasFallsBackToHostname") {
            let configuration = try await MainActor.run { () throws -> SSHHostConfiguration in
                let draft = SSHHostDraft(currentUser: "alice")
                draft.hostname = "db.internal"
                draft.setAlias("")
                return try draft.configuration()
            }

            try expectEqual(configuration.alias, "db.internal")
        },
    ]
}

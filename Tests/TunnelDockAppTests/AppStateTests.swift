import Foundation
import TestSupport
import TunnelDockAppSupport
import TunnelDockCore

enum AppStateTests {
    static let all: [TestCase] = [
        TestCase("AppStateTests.joinsMissingAliasesAndRestoresConfigOrder") {
            let nasTunnel = definition(hostAlias: "nas")
            let gpu = host(alias: "gpu", order: 0)
            let state = await AppState()

            let missingResult = await MainActor.run { () -> ([String], [String]) in
                state.apply(hosts: [gpu], definitions: [nasTunnel])
                return (state.normalHosts.map(\.alias), state.missingHosts.map(\.alias))
            }
            try expectEqual(missingResult.0, ["gpu"])
            try expectEqual(missingResult.1, ["nas"])

            let restoredResult = await MainActor.run { () -> ([String], [String]) in
                state.apply(hosts: [host(alias: "nas", order: 0), host(alias: "gpu", order: 1)], definitions: [nasTunnel])
                return (state.normalHosts.map(\.alias), state.missingHosts.map(\.alias))
            }
            try expectEqual(restoredResult.0, ["nas", "gpu"])
            try expectEqual(restoredResult.1, [])
        },
        TestCase("AppStateTests.searchFiltersAllEffectiveFieldsWithoutReordering") {
            let state = await AppState()
            let values = await MainActor.run { () -> [String] in
                state.apply(
                    hosts: [
                        SSHHost(alias: "first", hostname: "10.0.0.1", user: "root", port: 22, configOrder: 0, availability: .available),
                        SSHHost(alias: "second", hostname: "10.0.0.2", user: "ROOT", port: 2_222, configOrder: 1, availability: .available),
                    ],
                    definitions: []
                )
                state.searchQuery = "root"
                return state.normalHosts.map(\.alias)
            }

            try expectEqual(values, ["first", "second"])
        },
        TestCase("AppStateTests.searchAlsoFiltersMissingHosts") {
            let state = await AppState()
            let aliases = await MainActor.run { () -> [String] in
                state.apply(
                    hosts: [],
                    definitions: [definition(hostAlias: "missing-gpu"), definition(hostAlias: "archive")]
                )
                state.searchQuery = "GPU"
                return state.filteredMissingHosts.map(\.alias)
            }
            try expectEqual(aliases, ["missing-gpu"])
        },
        TestCase("AppStateTests.deletingLastSavedTunnelRemovesMissingHostImmediately") {
            let directory = FileManager.default.temporaryDirectory
                .appending(path: "TunnelDockAppStateTests-\(UUID().uuidString)", directoryHint: .isDirectory)
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            defer { try? FileManager.default.removeItem(at: directory) }
            let saved = definition(hostAlias: "retired-host")
            let repository = TunnelRepository(fileURL: directory.appending(path: "saved.json"))
            try await repository.replaceAll([saved])
            let manager = await TunnelManager(repository: repository)
            try await manager.loadSavedDefinitions()
            let state = await AppState(tunnelManager: manager)
            await MainActor.run { state.apply(hosts: [], definitions: [saved]) }
            try expectEqual(await MainActor.run { state.missingHosts.map(\.alias) }, ["retired-host"])

            try await manager.delete(id: .saved(saved.id))
            try expectEqual(await MainActor.run { state.missingHosts.map(\.alias) }, [])
        },
        TestCase("AppStateTests.activeTunnelHostsIncludeConnectingConnectedAndReconnecting") {
            let aliases = AppState.activeTunnelHostAliases(from: [
                snapshot(hostAlias: "gpu", state: .connecting),
                snapshot(hostAlias: "nas", state: .connected),
                snapshot(hostAlias: "gpu", state: .reconnecting),
                snapshot(hostAlias: "archive", state: .disconnected),
                snapshot(hostAlias: "failed", state: .failed),
            ])

            try expectEqual(aliases, Set(["gpu", "nas"]))
        },
    ]

    private static func host(alias: String, order: Int) -> SSHHost {
        SSHHost(alias: alias, hostname: "\(alias).internal", user: "tester", port: 22, configOrder: order, availability: .available)
    }

    private static func definition(hostAlias: String) -> TunnelDefinition {
        TunnelDefinition(
            id: UUID(), hostAlias: hostAlias, name: nil,
            remoteHost: "127.0.0.1", remotePort: 8_888,
            localAddress: "127.0.0.1", localPort: 8_888,
            createdAt: .distantPast, updatedAt: .distantPast
        )
    }

    private static func snapshot(hostAlias: String, state: TunnelState) -> TunnelRuntimeSnapshot {
        TunnelRuntimeSnapshot(
            id: .saved(UUID()), hostAlias: hostAlias, name: nil,
            remoteHost: "127.0.0.1", remotePort: 8_888,
            localAddress: "127.0.0.1", localPort: 8_888,
            state: state, desiredConnection: state != .disconnected,
            lastError: nil, logLines: []
        )
    }
}

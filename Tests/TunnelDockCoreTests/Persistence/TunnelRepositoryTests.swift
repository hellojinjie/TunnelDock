import Foundation
import TestSupport
import TunnelDockCore

enum TunnelRepositoryTests {
    static let all: [TestCase] = [
        TestCase("TunnelRepositoryTests.roundTripsSchemaOneWithoutRuntimeFields") {
            try await withRepository { repository, fileURL in
                let definition = fixture()

                try await repository.replaceAll([definition])

                try expectEqual(try await repository.load(), [definition])
                let json = try String(contentsOf: fileURL, encoding: .utf8)
                try expectEqual(json.contains("\"schemaVersion\""), true)
                try expectEqual(json.contains("connected"), false)
                try expectEqual(json.contains("pid"), false)
            }
        },
        TestCase("TunnelRepositoryTests.roundTripsSelectedWebProtocol") {
            try await withRepository { repository, _ in
                var definition = fixture()
                definition.webProtocol = .https

                try await repository.replaceAll([definition])

                try expectEqual(try await repository.load(), [definition])
            }
        },
        TestCase("TunnelRepositoryTests.roundTripsLastConnectedAtAndUpdatePreservesCreatedAt") {
            let createdAt = Date(timeIntervalSince1970: 1_000)
            try await withRepository { repository, _ in
                let definition = TunnelDefinition(
                    id: fixture().id,
                    hostAlias: "gpu",
                    name: nil,
                    remoteHost: "127.0.0.1",
                    remotePort: 8_888,
                    localAddress: "127.0.0.1",
                    localPort: 8_888,
                    createdAt: createdAt,
                    updatedAt: createdAt
                )

                try await repository.replaceAll([definition])

                var connected = definition
                connected.lastConnectedAt = Date(timeIntervalSince1970: 5_000)
                let updated = try await repository.update(connected)

                try expectEqual(updated.lastConnectedAt, connected.lastConnectedAt)
                try expectEqual(updated.createdAt, createdAt)
                try expectEqual(try await repository.load(), [updated])
            }
        },
        TestCase("TunnelRepositoryTests.malformedFileIsPreservedAndLocksMutations") {
            try await withRepository(seed: Data("not-json".utf8)) { repository, fileURL in
                try await expectRepositoryFailure { _ = try await repository.load() }
                try await expectRepositoryFailure { try await repository.replaceAll([]) }
                try expectEqual(try Data(contentsOf: fileURL), Data("not-json".utf8))
            }
        },
        TestCase("TunnelRepositoryTests.unsupportedSchemaIsPreserved") {
            let data = Data("{\"schemaVersion\":2,\"tunnels\":[]}".utf8)
            try await withRepository(seed: data) { repository, fileURL in
                try await expectRepositoryFailure { _ = try await repository.load() }
                try expectEqual(try Data(contentsOf: fileURL), data)
            }
        },
        TestCase("TunnelRepositoryTests.renameUpdatesOnlyNameAndUpdatedAt") {
            let now = Date(timeIntervalSince1970: 2_000)
            try await withRepository(now: now) { repository, _ in
                let definition = fixture()
                try await repository.replaceAll([definition])

                let renamed = try await repository.rename(id: definition.id, name: "Jupyter")

                try expectEqual(renamed.name, "Jupyter")
                try expectEqual(renamed.createdAt, definition.createdAt)
                try expectEqual(renamed.updatedAt, now)
            }
        },
        TestCase("TunnelRepositoryTests.deleteRemovesOnlyRequestedDefinition") {
            try await withRepository { repository, _ in
                let first = fixture()
                var second = fixture()
                second = TunnelDefinition(
                    id: UUID(uuidString: "00000000-0000-0000-0000-000000000002")!,
                    hostAlias: second.hostAlias,
                    name: second.name,
                    remoteHost: second.remoteHost,
                    remotePort: second.remotePort,
                    localAddress: second.localAddress,
                    localPort: second.localPort,
                    createdAt: second.createdAt,
                    updatedAt: second.updatedAt
                )
                try await repository.replaceAll([first, second])

                try await repository.delete(id: first.id)

                try expectEqual(try await repository.load(), [second])
            }
        },
    ]

    private static func fixture() -> TunnelDefinition {
        TunnelDefinition(
            id: UUID(uuidString: "00000000-0000-0000-0000-000000000001")!,
            hostAlias: "gpu",
            name: nil,
            remoteHost: "127.0.0.1",
            remotePort: 8_888,
            localAddress: "127.0.0.1",
            localPort: 8_888,
            createdAt: Date(timeIntervalSince1970: 1_000),
            updatedAt: Date(timeIntervalSince1970: 1_000)
        )
    }

    private static func withRepository(
        seed: Data? = nil,
        now: Date = Date(timeIntervalSince1970: 3_000),
        _ body: (TunnelRepository, URL) async throws -> Void
    ) async throws {
        let directory = FileManager.default.temporaryDirectory
            .appending(path: "TunnelDockRepositoryTests-\(UUID().uuidString)", directoryHint: .isDirectory)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let fileURL = directory.appending(path: "saved-tunnels.json")
        if let seed { try seed.write(to: fileURL) }
        let repository = TunnelRepository(fileURL: fileURL, now: { now })
        try await body(repository, fileURL)
    }

    private static func expectRepositoryFailure(
        _ operation: () async throws -> Void
    ) async throws {
        do {
            try await operation()
            throw TestFailure("Expected TunnelRepositoryError")
        } catch is TunnelRepositoryError {
            return
        }
    }
}

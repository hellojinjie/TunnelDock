import Foundation
import TestSupport
import TunnelDockCore

enum TunnelDefinitionTests {
    static let all: [TestCase] = [
        TestCase("TunnelDefinitionTests.unnamedDisplayNameUsesOnePortWhenPortsMatch") {
            let tunnel = TunnelDefinition.fixture(name: nil, localPort: 8_888, remotePort: 8_888)
            try expectEqual(tunnel.displayName, "8888")
        },
        TestCase("TunnelDefinitionTests.unnamedDisplayNameShowsMappingWhenPortsDiffer") {
            let tunnel = TunnelDefinition.fixture(name: nil, localPort: 18_888, remotePort: 8_888)
            try expectEqual(tunnel.displayName, "18888 → 8888")
        },
        TestCase("TunnelDefinitionTests.explicitNameTakesPrecedenceOverPortDisplay") {
            let tunnel = TunnelDefinition.fixture(name: "Jupyter", localPort: 18_888, remotePort: 8_888)
            try expectEqual(tunnel.displayName, "Jupyter")
        },
        TestCase("TunnelDefinitionTests.runtimeSnapshotUsesTheSameDisplayNameRules") {
            let snapshot = TunnelRuntimeSnapshot(
                id: .temporary(UUID()), hostAlias: "gpu", name: nil,
                remoteHost: "127.0.0.1", remotePort: 8_888,
                localAddress: "127.0.0.1", localPort: 18_888,
                state: .connected, desiredConnection: true,
                lastError: nil, logLines: []
            )
            try expectEqual(snapshot.displayName, "18888 → 8888")
        },
        TestCase("TunnelDefinitionTests.decodesMissingWebProtocolAsHTTP") {
            let json = Data("""
            {
              "id": "00000000-0000-0000-0000-000000000001",
              "hostAlias": "gpu",
              "name": null,
              "remoteHost": "127.0.0.1",
              "remotePort": 8888,
              "localAddress": "127.0.0.1",
              "localPort": 8888,
              "createdAt": 0,
              "updatedAt": 0
            }
            """.utf8)

            let tunnel = try JSONDecoder().decode(TunnelDefinition.self, from: json)
            try expectEqual(tunnel.webProtocol, .http)
            try expectEqual(tunnel.lastConnectedAt, nil)
        },
        TestCase("TunnelDefinitionTests.browserURLUsesSelectedProtocolAndLoopbackForWildcardAddress") {
            let snapshot = TunnelRuntimeSnapshot(
                id: .saved(UUID()), hostAlias: "gpu", name: nil,
                remoteHost: "127.0.0.1", remotePort: 8_443,
                localAddress: "0.0.0.0", localPort: 8_443,
                state: .connected, desiredConnection: true,
                lastError: nil, logLines: [], webProtocol: .https
            )

            try expectEqual(snapshot.browserURL?.absoluteString, "https://127.0.0.1:8443")
        },
        TestCase("TunnelDefinitionTests.browserURLMapsIPv6WildcardToLoopback") {
            let snapshot = TunnelRuntimeSnapshot(
                id: .saved(UUID()), hostAlias: "gpu", name: nil,
                remoteHost: "127.0.0.1", remotePort: 8_443,
                localAddress: "::", localPort: 8_443,
                state: .connected, desiredConnection: true,
                lastError: nil, logLines: [], webProtocol: .http
            )

            try expectEqual(snapshot.browserURL?.absoluteString, "http://127.0.0.1:8443")
        },
        TestCase("TunnelDefinitionTests.browserURLBracketsBareIPv6Address") {
            let snapshot = TunnelRuntimeSnapshot(
                id: .saved(UUID()), hostAlias: "gpu", name: nil,
                remoteHost: "127.0.0.1", remotePort: 8_443,
                localAddress: "fe80::1", localPort: 8_443,
                state: .connected, desiredConnection: true,
                lastError: nil, logLines: [], webProtocol: .http
            )

            try expectEqual(snapshot.browserURL?.absoluteString, "http://[fe80::1]:8443")
        },
    ]
}

private extension TunnelDefinition {
    static func fixture(name: String?, localPort: UInt16, remotePort: UInt16) -> Self {
        .init(
            id: UUID(uuidString: "00000000-0000-0000-0000-000000000001")!,
            hostAlias: "gpu",
            name: name,
            remoteHost: "127.0.0.1",
            remotePort: remotePort,
            localAddress: "127.0.0.1",
            localPort: localPort,
            createdAt: .distantPast,
            updatedAt: .distantPast
        )
    }
}

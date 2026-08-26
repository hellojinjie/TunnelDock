import Foundation
import TunnelDockCore

public struct MenuBarGroup: Identifiable, Sendable, Equatable {
    public var id: String { hostAlias }
    public let hostAlias: String
    public let rows: [TunnelRuntimeSnapshot]
}

public struct MenuBarModel: Sendable, Equatable {
    public let rows: [TunnelRuntimeSnapshot]
    public let groups: [MenuBarGroup]

    public init(runtimes: [TunnelRuntimeSnapshot], query: String = "") {
        let needle = query.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        rows = runtimes.filter { snapshot in
            guard case .saved = snapshot.id else { return false }
            guard !needle.isEmpty else { return true }
            return [
                snapshot.hostAlias,
                snapshot.name ?? "",
                snapshot.remoteHost,
                String(snapshot.remotePort),
                snapshot.localAddress,
                String(snapshot.localPort),
            ].contains { $0.lowercased().contains(needle) }
        }.sorted { lhs, rhs in
            if lhs.hostAlias != rhs.hostAlias { return lhs.hostAlias < rhs.hostAlias }
            if lhs.localPort != rhs.localPort { return lhs.localPort < rhs.localPort }
            return String(describing: lhs.id) < String(describing: rhs.id)
        }

        groups = Dictionary(grouping: rows, by: \.hostAlias)
            .map { MenuBarGroup(hostAlias: $0.key, rows: $0.value) }
            .sorted { $0.hostAlias < $1.hostAlias }
    }
}

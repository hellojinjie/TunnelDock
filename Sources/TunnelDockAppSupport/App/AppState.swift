import Combine
import Foundation
import TunnelDockCore

public struct MissingSSHHost: Identifiable, Sendable, Equatable {
    public var id: String { alias }
    public let alias: String

    public init(alias: String) {
        self.alias = alias
    }
}

@MainActor
public final class AppState: ObservableObject {
    @Published public var searchQuery = ""
    @Published public private(set) var allHosts: [SSHHost] = []
    @Published public private(set) var definitions: [TunnelDefinition] = []
    @Published public private(set) var missingHosts: [MissingSSHHost] = []
    @Published public private(set) var activeTunnelHostAliases: Set<String> = []
    @Published public private(set) var isRefreshing = false
    @Published public private(set) var refreshError: String?

    private let tunnelManager: TunnelManager?
    private var cancellables: Set<AnyCancellable> = []

    public init(tunnelManager: TunnelManager? = nil) {
        self.tunnelManager = tunnelManager
        tunnelManager?.$runtimes
            .sink { @MainActor [weak self] runtimes in
                self?.activeTunnelHostAliases = Self.activeTunnelHostAliases(from: runtimes)
                self?.recomputeMissingHosts(using: runtimes)
            }
            .store(in: &cancellables)
    }

    public var normalHosts: [SSHHost] {
        guard !searchQuery.isEmpty else { return allHosts }
        return allHosts.filter { $0.matches(query: searchQuery) }
    }

    public var filteredMissingHosts: [MissingSSHHost] {
        guard !searchQuery.isEmpty else { return missingHosts }
        return missingHosts.filter { $0.alias.localizedCaseInsensitiveContains(searchQuery) }
    }

    public func apply(hosts: [SSHHost], definitions: [TunnelDefinition]) {
        self.allHosts = hosts.sorted { $0.configOrder < $1.configOrder }
        self.definitions = definitions
        recomputeMissingHosts()
        tunnelManager?.updateHosts(self.allHosts)
    }

    nonisolated public static func activeTunnelHostAliases(from runtimes: [TunnelRuntimeSnapshot]) -> Set<String> {
        Set(runtimes.compactMap { runtime in
            switch runtime.state {
            case .connecting, .connected, .reconnecting:
                return runtime.hostAlias
            case .disconnected, .failed:
                return nil
            }
        })
    }

    private func recomputeMissingHosts(using publishedRuntimes: [TunnelRuntimeSnapshot]? = nil) {
        let availableAliases = Set(allHosts.map(\.alias))
        let hostAliases: [String]
        if let tunnelManager {
            hostAliases = (publishedRuntimes ?? tunnelManager.runtimes).compactMap { runtime in
                guard case .saved = runtime.id else { return nil }
                return runtime.hostAlias
            }
        } else {
            hostAliases = definitions.map(\.hostAlias)
        }
        var seen: Set<String> = []
        missingHosts = hostAliases.compactMap { alias in
            guard !availableAliases.contains(alias),
                  seen.insert(alias).inserted
            else { return nil }
            return MissingSSHHost(alias: alias)
        }
    }

    public func setRefreshing(_ value: Bool) {
        isRefreshing = value
    }

    public func refresh(
        using loader: any SSHConfigLoading,
        rootURL: URL
    ) async {
        isRefreshing = true
        defer { isRefreshing = false }
        do {
            let snapshot = try await loader.load(rootURL: rootURL)
            refreshError = nil
            apply(hosts: snapshot.hosts, definitions: definitions)
        } catch {
            refreshError = error.localizedDescription
        }
    }
}

import Foundation

public struct SSHConfigSnapshot: Sendable, Equatable {
    public let hosts: [SSHHost]
    public let expanded: ExpandedSSHConfig

    public init(hosts: [SSHHost], expanded: ExpandedSSHConfig) {
        self.hosts = hosts
        self.expanded = expanded
    }
}

public protocol SSHConfigLoading: Sendable {
    func load(rootURL: URL) async throws -> SSHConfigSnapshot
}

public struct SSHConfigLoader: SSHConfigLoading {
    private let includeResolver: SSHIncludeResolver
    private let scanner: SSHConfigScanner
    private let hostResolver: any SSHHostResolving
    private let maximumConcurrentResolutions: Int

    public init(
        includeResolver: SSHIncludeResolver,
        scanner: SSHConfigScanner = SSHConfigScanner(),
        hostResolver: any SSHHostResolving = SSHHostResolver(),
        maximumConcurrentResolutions: Int = 8
    ) {
        self.includeResolver = includeResolver
        self.scanner = scanner
        self.hostResolver = hostResolver
        self.maximumConcurrentResolutions = max(1, maximumConcurrentResolutions)
    }

    public func load(rootURL: URL) async throws -> SSHConfigSnapshot {
        let expanded = try includeResolver.resolve(rootURL: rootURL)
        let aliases = scanner.discoverAliases(in: expanded.lines)
        var hosts: [SSHHost] = []
        hosts.reserveCapacity(aliases.count)

        for start in stride(from: 0, to: aliases.count, by: maximumConcurrentResolutions) {
            let end = min(start + maximumConcurrentResolutions, aliases.count)
            let batch = Array(aliases[start..<end].enumerated()).map { offset, alias in
                (start + offset, alias)
            }
            let resolved = await withTaskGroup(of: SSHHost.self, returning: [SSHHost].self) { group in
                for (order, alias) in batch {
                    group.addTask {
                        await hostResolver.resolve(alias: alias, order: order)
                    }
                }
                var values: [SSHHost] = []
                for await host in group {
                    values.append(host)
                }
                return values
            }
            hosts.append(contentsOf: resolved)
        }
        hosts.sort { $0.configOrder < $1.configOrder }
        return SSHConfigSnapshot(hosts: hosts, expanded: expanded)
    }
}

import Combine
import Foundation
import TunnelDockAppSupport
import TunnelDockCore

enum MainPane: Hashable {
    case allTunnels
    case host(String)

    static let allTunnelsID = "__tunneldock.allTunnels"

    init(id: String) {
        if id == Self.allTunnelsID {
            self = .allTunnels
        } else {
            self = .host(id)
        }
    }

    var id: String {
        switch self {
        case .allTunnels: Self.allTunnelsID
        case .host(let alias): alias
        }
    }
}

@MainActor
final class AppModel: ObservableObject {
    let repository: TunnelRepository
    let tunnelManager: TunnelManager
    let appState: AppState

    @Published var selectedPaneID: String? = MainPane.allTunnelsID
    @Published var lifecycleError: String?

    var selectedPane: MainPane? {
        selectedPaneID.map(MainPane.init(id:))
    }

    private let configURL: URL
    private let loader: SSHConfigLoader
    private let watcher = SSHConfigWatcher()
    private var watchTask: Task<Void, Never>?
    private var hasStarted = false

    init() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let sshDirectory = home.appending(path: ".ssh", directoryHint: .isDirectory)
        configURL = sshDirectory.appending(path: "config")
        let supportDirectory = FileManager.default.urls(
            for: .applicationSupportDirectory,
            in: .userDomainMask
        ).first!.appending(path: "TunnelDock", directoryHint: .isDirectory)
        repository = TunnelRepository(fileURL: supportDirectory.appending(path: "saved-tunnels.json"))
        let manager = TunnelManager(repository: repository)
        tunnelManager = manager
        appState = AppState(tunnelManager: manager)
        loader = SSHConfigLoader(includeResolver: SSHIncludeResolver(userSSHDirectory: sshDirectory))
    }

    func start() async {
        guard !hasStarted else { return }
        hasStarted = true
        do {
            try ControlSocketManager().removeStaleSockets()
            try await tunnelManager.loadSavedDefinitions()
        } catch {
            lifecycleError = error.localizedDescription
        }
        await refreshSSHConfig()
    }

    func refreshSSHConfig() async {
        appState.setRefreshing(true)
        defer { appState.setRefreshing(false) }
        do {
            async let snapshot = loader.load(rootURL: configURL)
            async let definitions = repository.load()
            let (loadedSnapshot, loadedDefinitions) = try await (snapshot, definitions)
            appState.apply(hosts: loadedSnapshot.hosts, definitions: loadedDefinitions)
            lifecycleError = nil
            replaceWatcher(with: loadedSnapshot.expanded)
        } catch {
            lifecycleError = error.localizedDescription
        }
    }

    private func replaceWatcher(with expanded: ExpandedSSHConfig) {
        watchTask?.cancel()
        let events = watcher.events(watching: expanded)
        watchTask = Task { [weak self] in
            for await _ in events {
                if Task.isCancelled { return }
                await self?.refreshSSHConfig()
            }
        }
    }
}

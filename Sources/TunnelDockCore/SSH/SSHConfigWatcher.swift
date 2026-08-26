import Darwin
import Dispatch
import Foundation

public protocol SSHConfigWatching: Sendable {
    func events(watching config: ExpandedSSHConfig) -> AsyncStream<Void>
}

public struct SSHConfigWatcher: SSHConfigWatching {
    private let debounceDuration: Duration

    public init(debounceDuration: Duration = .milliseconds(300)) {
        self.debounceDuration = debounceDuration
    }

    public func events(watching config: ExpandedSSHConfig) -> AsyncStream<Void> {
        AsyncStream { continuation in
            let lifetime = WatchLifetime(
                continuation: continuation,
                debounceDuration: debounceDuration
            )
            let paths = config.watchedFiles.union(config.watchedDirectories)
            for url in paths {
                lifetime.watch(url)
            }
            continuation.onTermination = { _ in
                lifetime.cancel()
            }
        }
    }
}

private final class WatchLifetime: @unchecked Sendable {
    private let lock = NSLock()
    private let queue = DispatchQueue(label: "com.tunneldock.ssh-config-watcher")
    private let continuation: AsyncStream<Void>.Continuation
    private let delay: DispatchTimeInterval
    private var generation: UInt64 = 0
    private var sources: [DispatchSourceFileSystemObject] = []
    private var cancelled = false

    init(
        continuation: AsyncStream<Void>.Continuation,
        debounceDuration: Duration
    ) {
        self.continuation = continuation
        let components = debounceDuration.components
        let milliseconds = max(
            0,
            Int(components.seconds * 1_000)
                + Int(components.attoseconds / 1_000_000_000_000_000)
        )
        self.delay = .milliseconds(milliseconds)
    }

    func watch(_ url: URL) {
        let descriptor = open(url.path, O_EVTONLY)
        guard descriptor >= 0 else { return }
        let source = DispatchSource.makeFileSystemObjectSource(
            fileDescriptor: descriptor,
            eventMask: [.write, .delete, .rename, .extend, .attrib, .link],
            queue: queue
        )
        source.setEventHandler { [weak self] in self?.receivedEvent() }
        source.setCancelHandler { Darwin.close(descriptor) }
        lock.withLock { sources.append(source) }
        source.resume()
    }

    func cancel() {
        let current: [DispatchSourceFileSystemObject] = lock.withLock {
            guard !cancelled else { return [] }
            cancelled = true
            generation &+= 1
            let current = sources
            sources.removeAll()
            return current
        }
        current.forEach { $0.cancel() }
        continuation.finish()
    }

    private func receivedEvent() {
        let eventGeneration: UInt64 = lock.withLock {
            generation &+= 1
            return generation
        }
        queue.asyncAfter(deadline: .now() + delay) { [weak self] in
            guard let self else { return }
            let shouldYield = lock.withLock {
                !cancelled && generation == eventGeneration
            }
            if shouldYield { continuation.yield(()) }
        }
    }
}

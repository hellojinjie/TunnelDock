import Foundation

public struct TunnelLogEntry: Identifiable, Sendable, Equatable {
    public let id: UUID
    public let timestamp: Date
    public let message: String

    public init(id: UUID = UUID(), timestamp: Date, message: String) {
        self.id = id
        self.timestamp = timestamp
        self.message = message
    }

    public var formattedMessage: String {
        let components = Calendar.current.dateComponents([.hour, .minute, .second], from: timestamp)
        return String(
            format: "[%02d:%02d:%02d] %@",
            components.hour ?? 0,
            components.minute ?? 0,
            components.second ?? 0,
            message
        )
    }
}

public struct TunnelLogBuffer: Sendable, Equatable {
    public private(set) var entries: [TunnelLogEntry] = []
    public let capacity: Int

    public init(capacity: Int = 500) {
        self.capacity = max(0, capacity)
    }

    public mutating func append(_ message: String, at timestamp: Date = Date()) {
        guard capacity > 0 else { return }
        entries.append(TunnelLogEntry(timestamp: timestamp, message: message))
        let overflow = entries.count - capacity
        if overflow > 0 {
            entries.removeFirst(overflow)
        }
    }

    public mutating func removeAll() {
        entries.removeAll(keepingCapacity: true)
    }
}

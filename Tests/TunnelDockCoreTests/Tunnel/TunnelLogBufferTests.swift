import Foundation
import TestSupport
import TunnelDockCore

enum TunnelLogBufferTests {
    static let all: [TestCase] = [
        TestCase("TunnelLogBufferTests.keepsNewestFiveHundredLines") {
            let date = Date(timeIntervalSince1970: 1_000)
            var buffer = TunnelLogBuffer(capacity: 500)
            for value in 0..<505 {
                buffer.append("line \(value)", at: date)
            }

            try expectEqual(buffer.entries.count, 500)
            try expectEqual(buffer.entries.first?.message, "line 5")
            try expectEqual(buffer.entries.last?.message, "line 504")
        },
        TestCase("TunnelLogBufferTests.zeroCapacityRetainsNoLines") {
            var buffer = TunnelLogBuffer(capacity: 0)
            buffer.append("ignored", at: .distantPast)

            try expectEqual(buffer.entries, [])
        },
        TestCase("TunnelLogBufferTests.formatsTimestampForPresentation") {
            let entry = TunnelLogEntry(timestamp: Date(timeIntervalSince1970: 0), message: "Connected.")
            try expectEqual(entry.formattedMessage.hasPrefix("["), true)
            try expectEqual(entry.formattedMessage.contains("] Connected."), true)
        },
    ]
}

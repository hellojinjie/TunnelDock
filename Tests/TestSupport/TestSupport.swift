import Foundation

public struct TestCase: Sendable {
    public let name: String
    public let body: @Sendable () async throws -> Void

    public init(_ name: String, body: @escaping @Sendable () async throws -> Void) {
        self.name = name
        self.body = body
    }
}

public struct TestFailure: Error, CustomStringConvertible, Sendable {
    public let description: String

    public init(_ description: String) {
        self.description = description
    }
}

public func expectEqual<T: Equatable & Sendable>(
    _ actual: T,
    _ expected: T,
    _ message: String = ""
) throws {
    guard actual == expected else {
        let suffix = message.isEmpty ? "" : ": \(message)"
        throw TestFailure("Expected \(expected), got \(actual)\(suffix)")
    }
}

public func expectNotEqual<T: Equatable & Sendable>(
    _ actual: T,
    _ unexpected: T,
    _ message: String = ""
) throws {
    guard actual != unexpected else {
        let suffix = message.isEmpty ? "" : ": \(message)"
        throw TestFailure("Expected value different from \(actual)\(suffix)")
    }
}

public enum TestRunner {
    public static func run(_ tests: [TestCase], filter: String?) async -> Int32 {
        let selected = tests.filter { test in
            guard let filter, !filter.isEmpty else { return true }
            return test.name.localizedCaseInsensitiveContains(filter)
        }
        var failures = 0
        for test in selected {
            do {
                try await test.body()
                print("PASS \(test.name)")
            } catch {
                failures += 1
                print("FAIL \(test.name): \(error)")
            }
        }
        print("Executed \(selected.count) tests, \(failures) failures")
        return failures == 0 ? 0 : 1
    }
}

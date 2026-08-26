import TestSupport
import TunnelDockCore

enum ForwardSpecificationTests {
    static let all: [TestCase] = [
        TestCase("ForwardSpecificationTests.formatsIPv6EndpointsWithBrackets") {
            let value = try ForwardSpecification(
                localAddress: "::1",
                localPort: "8888",
                remoteHost: "2001:db8::10",
                remotePort: "8888"
            )

            try expectEqual(value.openSSHArgument, "[::1]:8888:[2001:db8::10]:8888")
        },
        TestCase("ForwardSpecificationTests.formatsHostnamesAndDifferentPorts") {
            let value = try ForwardSpecification(
                localAddress: "127.0.0.1",
                localPort: "18888",
                remoteHost: "db.internal",
                remotePort: "8888"
            )

            try expectEqual(value.openSSHArgument, "127.0.0.1:18888:db.internal:8888")
        },
        TestCase("ForwardSpecificationTests.acceptsPortBoundaries") {
            try expectEqual(try InputValidator.port("1"), 1)
            try expectEqual(try InputValidator.port("65535"), 65_535)
        },
        TestCase("ForwardSpecificationTests.rejectsNonDecimalAndOutOfRangePorts") {
            for value in ["0", "65536", "8_888", " 8888", "", "+22", "２２"] {
                try expectValidationFailure { try InputValidator.port(value) }
            }
        },
        TestCase("ForwardSpecificationTests.rejectsControlCharactersInAddresses") {
            try expectValidationFailure {
                try InputValidator.text("db.internal\n-o ProxyCommand=x", field: .remoteHost)
            }
            try expectValidationFailure {
                try InputValidator.text("127.0.0.1\0", field: .localAddress)
            }
        },
        TestCase("ForwardSpecificationTests.tunnelInputDefaultsToLoopback") {
            let input = TunnelInput(remotePort: "8888", localPort: "8888")

            try expectEqual(input.remoteHost, "127.0.0.1")
            try expectEqual(input.localAddress, "127.0.0.1")
        },
    ]

    private static func expectValidationFailure<T>(
        _ operation: () throws -> T
    ) throws {
        do {
            _ = try operation()
            throw TestFailure("Expected InputValidationError")
        } catch is InputValidationError {
            return
        }
    }
}

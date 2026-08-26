import TestSupport
import TunnelDockCore

enum SSHErrorClassifierTests {
    static let all: [TestCase] = [
        TestCase("SSHErrorClassifierTests.classifiesRequiredOpenSSHErrors") {
            let classifier = SSHErrorClassifier()
            let fixtures: [(String, SSHUserError)] = [
                ("bind [127.0.0.1]:8888: Address already in use", .localPortInUse),
                ("Permission denied (publickey,password).", .authenticationFailed),
                ("Host key verification failed.", .hostVerificationRequired),
                ("WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!", .hostVerificationRequired),
                ("Host not found", .hostNotFound),
                ("Bad configuration option: Broken", .configurationError),
                ("Connection timed out", .connectionTimedOut),
                ("Could not resolve hostname gpu: nodename nor servname provided", .couldNotResolveHost),
                ("connect to host gpu port 22: Connection refused", .connectionRefused),
            ]

            for (stderr, expected) in fixtures {
                try expectEqual(
                    classifier.classify(stderr: stderr, exitStatus: 255),
                    expected,
                    stderr
                )
            }
        },
        TestCase("SSHErrorClassifierTests.usesUnexpectedExitForEmptyStderr") {
            try expectEqual(
                SSHErrorClassifier().classify(stderr: "", exitStatus: 255),
                .processExitedUnexpectedly
            )
        },
        TestCase("SSHErrorClassifierTests.usesGenericFallbackForUnknownStderr") {
            try expectEqual(
                SSHErrorClassifier().classify(stderr: "unrecognized failure", exitStatus: 255),
                .connectionFailed
            )
        },
        TestCase("SSHErrorClassifierTests.hostVerificationMessageRequiresTerminalVerification") {
            let error = SSHUserError.hostVerificationRequired

            try expectEqual(error.message.contains("Terminal"), true)
            try expectEqual(error.message.contains("ssh <host>"), true)
        },
    ]
}

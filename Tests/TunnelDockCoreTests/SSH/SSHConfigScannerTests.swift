import TestSupport
import TunnelDockCore

enum SSHConfigScannerTests {
    static let all: [TestCase] = [
        TestCase("SSHConfigScannerTests.discoversOnlyExplicitAliasesOnceInTextOrder") {
            let lines = [
                "Host *",
                "Host gpu gpu-server lab-gpu",
                "Host gpu-* lab-? [abc] !blocked",
                "Host gpu",
                "  Host quoted # trailing comment",
            ]

            try expectEqual(
                SSHConfigScanner().discoverAliases(in: lines),
                ["gpu", "gpu-server", "lab-gpu", "quoted"]
            )
        },
        TestCase("SSHConfigScannerTests.lexerPreservesHashesAndSpacesInsideQuotes") {
            let lines = ["Host \"hash#alias\" 'space alias' plain # comment"]

            try expectEqual(
                SSHConfigScanner().discoverAliases(in: lines),
                ["hash#alias", "space alias", "plain"]
            )
        },
        TestCase("SSHConfigScannerTests.matchesHostKeywordCaseInsensitively") {
            try expectEqual(
                SSHConfigScanner().discoverAliases(in: ["hOsT GPU"]),
                ["GPU"]
            )
        },
        TestCase("SSHConfigScannerTests.acceptsOpenSSHKeywordEqualsSyntax") {
            try expectEqual(
                SSHConfigScanner().discoverAliases(in: [
                    "Host=eqalias",
                    "hOsT=\"space alias\"",
                    "Host = spaced",
                    "Host =joined",
                ]),
                ["eqalias", "space alias", "spaced", "joined"]
            )
        },
    ]
}

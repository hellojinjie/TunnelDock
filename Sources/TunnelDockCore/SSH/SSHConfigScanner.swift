public struct SSHConfigScanner: Sendable {
    private let lexer: SSHConfigLexer

    public init(lexer: SSHConfigLexer = SSHConfigLexer()) {
        self.lexer = lexer
    }

    public func discoverAliases(in lines: [String]) -> [String] {
        var aliases: [String] = []
        var seen: Set<String> = []

        for line in lines {
            let tokens = lexer.tokens(in: line)
            guard tokens.first?.caseInsensitiveCompare("Host") == .orderedSame else {
                continue
            }
            for token in tokens.dropFirst() where isExplicitAlias(token) {
                guard seen.insert(token).inserted else { continue }
                aliases.append(token)
            }
        }
        return aliases
    }

    private func isExplicitAlias(_ token: String) -> Bool {
        guard !token.isEmpty, !token.hasPrefix("!") else { return false }
        return !token.contains { "*?[]".contains($0) }
    }
}

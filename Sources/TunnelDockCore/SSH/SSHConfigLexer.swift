import Foundation

public struct SSHConfigLexer: Sendable {
    public init() {}

    public func tokens(in line: String) -> [String] {
        var tokens: [String] = []
        var current = ""
        var quote: Character?
        var escaping = false

        func finishToken() {
            guard !current.isEmpty else { return }
            tokens.append(current)
            current.removeAll(keepingCapacity: true)
        }

        for character in line {
            if escaping {
                current.append(character)
                escaping = false
                continue
            }
            if character == "\\" {
                escaping = true
                continue
            }
            if let activeQuote = quote {
                if character == activeQuote {
                    quote = nil
                } else {
                    current.append(character)
                }
                continue
            }
            if character == "\"" || character == "'" {
                quote = character
            } else if character == "#" {
                break
            } else if character == "=",
                      tokens.isEmpty,
                      current.caseInsensitiveCompare("Host") == .orderedSame
                        || current.caseInsensitiveCompare("Include") == .orderedSame {
                finishToken()
            } else if character.isWhitespace {
                finishToken()
            } else {
                current.append(character)
            }
        }
        if escaping {
            current.append("\\")
        }
        finishToken()
        if let keyword = tokens.first,
           keyword.caseInsensitiveCompare("Host") == .orderedSame
            || keyword.caseInsensitiveCompare("Include") == .orderedSame,
           tokens.count > 1 {
            if tokens[1] == "=" {
                tokens.remove(at: 1)
            } else if tokens[1].hasPrefix("=") {
                tokens[1].removeFirst()
                if tokens[1].isEmpty { tokens.remove(at: 1) }
            }
        }
        return tokens
    }
}

import Darwin
import Foundation

public enum SSHConfigDiagnostic: Sendable, Equatable {
    case includeCycle(URL)
    case unreadableFile(URL, String)
}

public struct ExpandedSSHConfig: Sendable, Equatable {
    public let lines: [String]
    public let watchedFiles: Set<URL>
    public let watchedDirectories: Set<URL>
    public let diagnostics: [SSHConfigDiagnostic]

    public init(
        lines: [String],
        watchedFiles: Set<URL>,
        watchedDirectories: Set<URL>,
        diagnostics: [SSHConfigDiagnostic]
    ) {
        self.lines = lines
        self.watchedFiles = watchedFiles
        self.watchedDirectories = watchedDirectories
        self.diagnostics = diagnostics
    }
}

public struct SSHIncludeResolver: Sendable {
    private let userSSHDirectory: URL
    private let userHomeDirectory: URL
    private let lexer: SSHConfigLexer

    public init(
        userSSHDirectory: URL,
        lexer: SSHConfigLexer = SSHConfigLexer()
    ) {
        self.userSSHDirectory = userSSHDirectory.standardizedFileURL
        self.userHomeDirectory = userSSHDirectory.deletingLastPathComponent().standardizedFileURL
        self.lexer = lexer
    }

    public func resolve(rootURL: URL) throws -> ExpandedSSHConfig {
        var accumulator = Accumulator()
        accumulator.watchedDirectories.insert(userSSHDirectory)
        var activeStack: Set<URL> = []
        try expand(
            fileURL: rootURL.standardizedFileURL,
            isRoot: true,
            activeStack: &activeStack,
            accumulator: &accumulator
        )
        return ExpandedSSHConfig(
            lines: accumulator.lines,
            watchedFiles: accumulator.watchedFiles,
            watchedDirectories: accumulator.watchedDirectories,
            diagnostics: accumulator.diagnostics
        )
    }

    private func expand(
        fileURL: URL,
        isRoot: Bool,
        activeStack: inout Set<URL>,
        accumulator: inout Accumulator
    ) throws {
        let canonicalURL = fileURL.resolvingSymlinksInPath().standardizedFileURL
        guard FileManager.default.fileExists(atPath: canonicalURL.path) else {
            if !isRoot {
                accumulator.diagnostics.append(.unreadableFile(canonicalURL, "File does not exist."))
            }
            return
        }
        guard activeStack.insert(canonicalURL).inserted else {
            accumulator.diagnostics.append(.includeCycle(canonicalURL))
            return
        }
        defer { activeStack.remove(canonicalURL) }

        let contents: String
        do {
            contents = try String(contentsOf: canonicalURL, encoding: .utf8)
        } catch {
            accumulator.diagnostics.append(.unreadableFile(canonicalURL, error.localizedDescription))
            return
        }
        accumulator.watchedFiles.insert(canonicalURL)

        for line in contents.components(separatedBy: .newlines) {
            let tokens = lexer.tokens(in: line)
            guard tokens.first?.caseInsensitiveCompare("Include") == .orderedSame else {
                accumulator.lines.append(line)
                continue
            }
            for pattern in tokens.dropFirst() {
                let expandedPattern = expandPath(pattern)
                accumulator.watchedDirectories.insert(watchedDirectory(for: expandedPattern))
                for includedURL in matchingFiles(for: expandedPattern) {
                    try expand(
                        fileURL: includedURL,
                        isRoot: false,
                        activeStack: &activeStack,
                        accumulator: &accumulator
                    )
                }
            }
        }
    }

    private func expandPath(_ path: String) -> String {
        if path == "~" {
            return userHomeDirectory.path
        }
        if path.hasPrefix("~/") {
            return userHomeDirectory.appending(path: String(path.dropFirst(2))).path
        }
        if path.hasPrefix("/") {
            return URL(fileURLWithPath: path).standardizedFileURL.path
        }
        return userSSHDirectory.appending(path: path).standardizedFileURL.path
    }

    private func matchingFiles(for pattern: String) -> [URL] {
        guard containsGlobCharacter(pattern) else {
            guard FileManager.default.fileExists(atPath: pattern) else { return [] }
            return [URL(fileURLWithPath: pattern).standardizedFileURL]
        }

        var result = glob_t()
        let status = pattern.withCString { glob($0, GLOB_NOSORT, nil, &result) }
        guard status == 0 else {
            globfree(&result)
            return []
        }
        defer { globfree(&result) }

        var paths: [String] = []
        for index in 0..<Int(result.gl_pathc) {
            if let value = result.gl_pathv[index] {
                paths.append(String(cString: value))
            }
        }
        paths.sort { lhs, rhs in
            lhs.utf8.lexicographicallyPrecedes(rhs.utf8)
        }
        return paths.map { URL(fileURLWithPath: $0).standardizedFileURL }
    }

    private func watchedDirectory(for pattern: String) -> URL {
        guard let wildcard = pattern.firstIndex(where: { "*?[".contains($0) }) else {
            return URL(fileURLWithPath: pattern).deletingLastPathComponent().standardizedFileURL
        }
        var prefix = String(pattern[..<wildcard])
        if !prefix.hasSuffix("/") {
            prefix = (prefix as NSString).deletingLastPathComponent
        }
        while prefix.count > 1, prefix.hasSuffix("/") {
            prefix.removeLast()
        }
        return URL(filePath: prefix, directoryHint: .notDirectory).standardizedFileURL
    }

    private func containsGlobCharacter(_ path: String) -> Bool {
        path.contains { "*?[".contains($0) }
    }
}

private struct Accumulator {
    var lines: [String] = []
    var watchedFiles: Set<URL> = []
    var watchedDirectories: Set<URL> = []
    var diagnostics: [SSHConfigDiagnostic] = []
}

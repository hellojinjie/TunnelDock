import Foundation

public protocol SSHHostResolving: Sendable {
    func resolve(alias: String, order: Int) async -> SSHHost
}

public struct SSHHostResolver: SSHHostResolving {
    private let executor: any ProcessExecuting
    private let executableURL = URL(fileURLWithPath: "/usr/bin/ssh")

    public init(executor: any ProcessExecuting = FoundationProcessExecutor()) {
        self.executor = executor
    }

    public func resolve(alias: String, order: Int) async -> SSHHost {
        do {
            let result = try await executor.run(
                executableURL: executableURL,
                arguments: ["-G", alias]
            )
            guard result.exitStatus == 0 else {
                return failedHost(alias: alias, order: order, message: errorText(from: result))
            }
            guard let output = String(data: result.stdout, encoding: .utf8) else {
                return failedHost(alias: alias, order: order, message: "ssh -G returned invalid UTF-8.")
            }
            let values = parse(output)
            guard let hostname = values["hostname"], !hostname.isEmpty,
                  let user = values["user"], !user.isEmpty,
                  let portText = values["port"],
                  let port = UInt16(portText), port > 0
            else {
                return failedHost(alias: alias, order: order, message: "ssh -G omitted a required field.")
            }
            return SSHHost(
                alias: alias,
                hostname: hostname,
                user: user,
                port: port,
                configOrder: order,
                availability: .available
            )
        } catch {
            return failedHost(alias: alias, order: order, message: error.localizedDescription)
        }
    }

    private func parse(_ output: String) -> [String: String] {
        var values: [String: String] = [:]
        for line in output.components(separatedBy: .newlines) {
            let parts = line.split(maxSplits: 1, whereSeparator: \Character.isWhitespace)
            guard parts.count == 2 else { continue }
            let key = parts[0].lowercased()
            if values[key] == nil {
                values[key] = String(parts[1])
            }
        }
        return values
    }

    private func errorText(from result: ProcessResult) -> String {
        let stderr = String(data: result.stderr, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return stderr?.isEmpty == false ? stderr! : "ssh -G exited with status \(result.exitStatus)."
    }

    private func failedHost(alias: String, order: Int, message: String) -> SSHHost {
        SSHHost(
            alias: alias,
            hostname: alias,
            user: "",
            port: 22,
            configOrder: order,
            availability: .configurationError(message)
        )
    }
}

import Foundation

public enum SSHUserError: String, Sendable, Equatable {
    case localPortInUse
    case authenticationFailed
    case hostVerificationRequired
    case hostNotFound
    case configurationError
    case connectionTimedOut
    case couldNotResolveHost
    case connectionRefused
    case processExitedUnexpectedly
    case connectionFailed

    public var title: String {
        switch self {
        case .localPortInUse: "Local port is already in use"
        case .authenticationFailed: "Authentication failed"
        case .hostVerificationRequired: "Host verification required"
        case .hostNotFound: "Host not found"
        case .configurationError: "SSH configuration error"
        case .connectionTimedOut: "Connection timed out"
        case .couldNotResolveHost: "Could not resolve host"
        case .connectionRefused: "Connection refused by SSH server"
        case .processExitedUnexpectedly: "SSH process exited unexpectedly"
        case .connectionFailed: "SSH connection failed"
        }
    }

    public var message: String {
        switch self {
        case .localPortInUse:
            "Choose another local port."
        case .authenticationFailed:
            "Make sure this host can connect noninteractively with the system ssh command."
        case .hostVerificationRequired:
            "This host has not been verified. Connect to it once in Terminal with: ssh <host>"
        case .hostNotFound:
            "The saved SSH host alias is not present in ~/.ssh/config."
        case .configurationError:
            "SSH configuration could not be resolved."
        case .connectionTimedOut:
            "The SSH server did not respond before the connection timed out."
        case .couldNotResolveHost:
            "The SSH host name could not be resolved."
        case .connectionRefused:
            "The SSH server refused the connection."
        case .processExitedUnexpectedly:
            "The SSH process exited unexpectedly. See Log for details."
        case .connectionFailed:
            "SSH connection failed. See Log for details."
        }
    }
}

public struct SSHErrorClassifier: Sendable {
    public init() {}

    public func classify(stderr: String, exitStatus: Int32) -> SSHUserError {
        let value = stderr.lowercased()
        if value.contains("address already in use") || value.contains("cannot listen to port") {
            return .localPortInUse
        }
        if value.contains("permission denied") || value.contains("authentication failed") {
            return .authenticationFailed
        }
        if value.contains("host key verification failed")
            || value.contains("remote host identification has changed") {
            return .hostVerificationRequired
        }
        if value.contains("host not found") {
            return .hostNotFound
        }
        if value.contains("bad configuration option")
            || value.contains("bad configuration options")
            || value.contains("no argument after keyword") {
            return .configurationError
        }
        if value.contains("connection timed out") || value.contains("operation timed out") {
            return .connectionTimedOut
        }
        if value.contains("could not resolve hostname") || value.contains("name or service not known") {
            return .couldNotResolveHost
        }
        if value.contains("connection refused") {
            return .connectionRefused
        }
        if value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty, exitStatus != 0 {
            return .processExitedUnexpectedly
        }
        return .connectionFailed
    }
}

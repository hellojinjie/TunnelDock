import Combine
import Foundation
import TunnelDockCore

public enum SSHHostDraftError: Error, LocalizedError {
    case invalidPort

    public var errorDescription: String? {
        switch self {
        case .invalidPort:
            "Port must be a number between 1 and 65535."
        }
    }
}

@MainActor
public final class SSHHostDraft: ObservableObject {
    @Published public var alias = ""
    @Published public var hostname = "" {
        didSet {
            if !aliasWasEdited { alias = hostname }
        }
    }
    @Published public var user: String
    @Published public var port = "22"

    private var aliasWasEdited = false

    public init(currentUser: String = ProcessInfo.processInfo.userName) {
        user = currentUser
    }

    public func setAlias(_ value: String) {
        alias = value
        aliasWasEdited = true
    }

    public func configuration() throws -> SSHHostConfiguration {
        guard let port = UInt16(port), port > 0 else { throw SSHHostDraftError.invalidPort }
        return SSHHostConfiguration(
            alias: alias.isEmpty ? hostname : alias,
            hostname: hostname,
            user: user,
            port: port
        )
    }
}

import Foundation

public enum ValidationField: String, Sendable, Equatable {
    case hostAlias = "Host Alias"
    case remoteHost = "Remote Host"
    case localAddress = "Local Address"
    case remotePort = "Remote Port"
    case localPort = "Local Port"
    case name = "Name"
}

public enum InputValidationError: Error, Sendable, Equatable, CustomStringConvertible {
    case empty(ValidationField)
    case invalidPort(ValidationField)
    case controlCharacter(ValidationField)

    public var description: String {
        switch self {
        case let .empty(field):
            return "\(field.rawValue) cannot be empty."
        case let .invalidPort(field):
            return "\(field.rawValue) must be a decimal integer from 1 to 65535."
        case let .controlCharacter(field):
            return "\(field.rawValue) contains an invalid control character."
        }
    }
}

public enum InputValidator {
    public static func port(
        _ value: String,
        field: ValidationField = .remotePort
    ) throws -> UInt16 {
        guard !value.isEmpty else {
            throw InputValidationError.empty(field)
        }
        guard value.unicodeScalars.allSatisfy({ $0.value >= 48 && $0.value <= 57 }),
              let parsed = UInt32(value),
              (1...65_535).contains(parsed)
        else {
            throw InputValidationError.invalidPort(field)
        }
        return UInt16(parsed)
    }

    public static func text(_ value: String, field: ValidationField) throws -> String {
        guard !value.unicodeScalars.contains(where: CharacterSet.controlCharacters.contains) else {
            throw InputValidationError.controlCharacter(field)
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw InputValidationError.empty(field)
        }
        return trimmed
    }

    public static func optionalName(_ value: String?) throws -> String? {
        guard let value else { return nil }
        guard !value.unicodeScalars.contains(where: CharacterSet.controlCharacters.contains) else {
            throw InputValidationError.controlCharacter(.name)
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}

public struct TunnelInput: Sendable, Equatable {
    public var name: String?
    public var remoteHost: String
    public var remotePort: String
    public var localAddress: String
    public var localPort: String
    public var webProtocol: TunnelProtocol

    public init(
        name: String? = nil,
        remoteHost: String = "127.0.0.1",
        remotePort: String,
        localAddress: String = "127.0.0.1",
        localPort: String,
        webProtocol: TunnelProtocol = .http
    ) {
        self.name = name
        self.remoteHost = remoteHost
        self.remotePort = remotePort
        self.localAddress = localAddress
        self.localPort = localPort
        self.webProtocol = webProtocol
    }
}

public struct ForwardSpecification: Sendable, Equatable {
    public let localAddress: String
    public let localPort: UInt16
    public let remoteHost: String
    public let remotePort: UInt16

    public init(
        localAddress: String,
        localPort: String,
        remoteHost: String,
        remotePort: String
    ) throws {
        self.localAddress = try InputValidator.text(localAddress, field: .localAddress)
        self.localPort = try InputValidator.port(localPort, field: .localPort)
        self.remoteHost = try InputValidator.text(remoteHost, field: .remoteHost)
        self.remotePort = try InputValidator.port(remotePort, field: .remotePort)
    }

    public var openSSHArgument: String {
        "\(Self.bracket(localAddress)):\(localPort):\(Self.bracket(remoteHost)):\(remotePort)"
    }

    private static func bracket(_ address: String) -> String {
        let unwrapped: String
        if address.hasPrefix("["), address.hasSuffix("]"), address.count >= 2 {
            unwrapped = String(address.dropFirst().dropLast())
        } else {
            unwrapped = address
        }
        return unwrapped.contains(":") ? "[\(unwrapped)]" : unwrapped
    }
}

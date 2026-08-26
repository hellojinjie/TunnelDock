# TunnelDock v1.0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a native macOS 13 SwiftUI application, using SwiftPM only, that discovers OpenSSH hosts and reliably manages independent local SSH forwards exactly as specified for TunnelDock v1.0.

**Architecture:** A SwiftUI executable owns presentation and AppKit lifecycle while a UI-independent `TunnelDockCore` library owns SSH config discovery, validation, persistence, process control, logs, and tunnel state. Protocol-backed operating-system boundaries make the two-stage OpenSSH lifecycle and reconnect state machine deterministic under SwiftPM executable test targets and an in-repository assertion runner.

**Tech Stack:** Swift 6.3, Swift Package Manager, an in-repository test runner, SwiftUI, AppKit, Foundation, Network, Darwin, `/usr/bin/ssh`.

**Spec:** `docs/superpowers/specs/2026-08-25-tunneldock-v1-design.md`

## Global Constraints

- Target macOS 13 Ventura or later and compile in Swift 6 language mode.
- Use SwiftPM only; do not install Xcode and do not create an `.xcodeproj`.
- Do not add third-party dependencies.
- Invoke SSH only as `/usr/bin/ssh` through `Process`; never use a shell.
- Do not enable App Sandbox or set `LSUIElement`; the Dock icon must remain visible.
- Do not persist runtime state or logs and do not store SSH credentials.
- Force `BatchMode=yes` and `StrictHostKeyChecking=yes`; never accept host keys automatically.
- Keep `TunnelDefinition` separate from `TunnelRuntime`.
- Use English UI copy from the product specification.
- This directory is not currently a Git repository. Do not initialize Git implicitly. At each task boundary, retain the verified workspace state; add commits only if the user separately initializes or authorizes a repository.

## File Structure

```text
Package.swift                                      Package products, platform, and targets
Sources/TunnelDockCore/Models/SSHHost.swift        Derived host model and availability
Sources/TunnelDockCore/Models/TunnelDefinition.swift Persistent forward definition
Sources/TunnelDockCore/Models/TunnelRuntime.swift  In-memory lifecycle model
Sources/TunnelDockCore/SSH/SSHConfigLexer.swift    OpenSSH-style token/comment handling
Sources/TunnelDockCore/SSH/SSHIncludeResolver.swift Ordered recursive Include expansion
Sources/TunnelDockCore/SSH/SSHConfigScanner.swift  Explicit alias discovery
Sources/TunnelDockCore/SSH/SSHHostResolver.swift   `/usr/bin/ssh -G` resolution
Sources/TunnelDockCore/SSH/SSHConfigLoader.swift   Complete host snapshot assembly
Sources/TunnelDockCore/SSH/SSHConfigWatcher.swift  File/directory events and debounce
Sources/TunnelDockCore/Process/ProcessExecutor.swift Async Process boundary
Sources/TunnelDockCore/Persistence/TunnelRepository.swift Atomic schema-v1 storage
Sources/TunnelDockCore/Tunnel/ForwardSpecification.swift Input validation and `-L` formatting
Sources/TunnelDockCore/Tunnel/PortAvailabilityChecker.swift Internal and OS collision checks
Sources/TunnelDockCore/Tunnel/TunnelLogBuffer.swift 500-line in-memory log
Sources/TunnelDockCore/Tunnel/SSHErrorClassifier.swift User-facing SSH error categories
Sources/TunnelDockCore/Tunnel/ControlSocketManager.swift Short, private socket paths
Sources/TunnelDockCore/Tunnel/SSHProcessController.swift Two-stage master/forward lifecycle
Sources/TunnelDockCore/Tunnel/TunnelManager.swift Serialized state machine and reconnect
Sources/TunnelDock/App/TunnelDockApp.swift           SwiftUI scenes
Sources/TunnelDock/App/AppDelegate.swift             Reopen and delayed quit
Sources/TunnelDock/App/AppState.swift                Host/definition/runtime join
Sources/TunnelDock/App/SettingsStore.swift           `showMenuBar` UserDefaults value
Sources/TunnelDock/UI/MainWindowView.swift           NavigationSplitView shell
Sources/TunnelDock/UI/HostSidebar.swift              Search, normal, and missing hosts
Sources/TunnelDock/UI/HostDetailView.swift           Host metadata and tunnel lists
Sources/TunnelDock/UI/TunnelRow.swift                State and valid actions
Sources/TunnelDock/UI/QuickForwardView.swift         Quick/advanced form behavior
Sources/TunnelDock/UI/TunnelEditorView.swift         Rename and disconnected edit
Sources/TunnelDock/UI/TunnelLogView.swift            Per-runtime log viewer
Sources/TunnelDock/UI/MenuBarContentView.swift       Saved-tunnel quick controls
Sources/TunnelDock/UI/SettingsView.swift             Single v1 setting
Resources/Info.plist                                 App bundle metadata
Scripts/package-app.sh                               Release build and `.app` assembly
Tests/TestSupport/                                   Dependency-free assertion runner
Tests/TunnelDockCoreTests/                           Core test executable
Tests/TunnelDockAppTests/                            App-state and view-model test executable
docs/manual-acceptance-checklist.md                  Environment-dependent release checks
```

---

### Task 1: SwiftPM Skeleton and Core Models

**Files:**
- Create: `Package.swift`
- Create: `Sources/TunnelDockCore/Models/SSHHost.swift`
- Create: `Sources/TunnelDockCore/Models/TunnelDefinition.swift`
- Create: `Sources/TunnelDockCore/Models/TunnelRuntime.swift`
- Create: `Sources/TunnelDock/main.swift`
- Create: `Tests/TunnelDockCoreTests/Models/TunnelDefinitionTests.swift`

**Interfaces:**
- Produces: `SSHHost`, `SSHHostAvailability`, `TunnelDefinition`, `TunnelState`, `TunnelRuntimeID`, and `TunnelRuntimeSnapshot` as `Sendable` value types.

- [ ] **Step 1: Create the package manifest and a deliberately incomplete model test**

```swift
// Tests/TunnelDockCoreTests/Models/TunnelDefinitionTests.swift
import XCTest
@testable import TunnelDockCore

final class TunnelDefinitionTests: XCTestCase {
    func testUnnamedDisplayNameUsesOnePortWhenPortsMatch() {
        let tunnel = TunnelDefinition.fixture(name: nil, localPort: 8_888, remotePort: 8_888)
        XCTAssertEqual(tunnel.displayName, "8888")
    }

    func testUnnamedDisplayNameShowsMappingWhenPortsDiffer() {
        let tunnel = TunnelDefinition.fixture(name: nil, localPort: 18_888, remotePort: 8_888)
        XCTAssertEqual(tunnel.displayName, "18888 → 8888")
    }
}
```

Use this manifest and add the test-only `fixture` factory shown below the model declarations:

```swift
// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "TunnelDock",
    platforms: [.macOS(.v13)],
    products: [
        .library(name: "TunnelDockCore", targets: ["TunnelDockCore"]),
        .executable(name: "TunnelDock", targets: ["TunnelDock"])
    ],
    targets: [
        .target(name: "TunnelDockCore"),
        .executableTarget(name: "TunnelDock", dependencies: ["TunnelDockCore"]),
        .testTarget(name: "TunnelDockCoreTests", dependencies: ["TunnelDockCore"]),
        .testTarget(name: "TunnelDockAppTests", dependencies: ["TunnelDock"])
    ],
    swiftLanguageModes: [.v6]
)
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `swift test --filter TunnelDefinitionTests`

Expected: compilation fails because `TunnelDefinition` and `displayName` do not exist.

- [ ] **Step 3: Implement the minimal models**

```swift
public enum SSHHostAvailability: Sendable, Equatable {
    case available
    case configurationError(String)
}

public struct SSHHost: Identifiable, Sendable, Equatable {
    public var id: String { alias }
    public let alias: String
    public let hostname: String
    public let user: String
    public let port: UInt16
    public let configOrder: Int
    public let availability: SSHHostAvailability
}

public struct TunnelDefinition: Identifiable, Codable, Sendable, Equatable {
    public let id: UUID
    public var hostAlias: String
    public var name: String?
    public var remoteHost: String
    public var remotePort: UInt16
    public var localAddress: String
    public var localPort: UInt16
    public let createdAt: Date
    public var updatedAt: Date

    public var displayName: String {
        if let name, !name.isEmpty { return name }
        return localPort == remotePort ? "\(remotePort)" : "\(localPort) → \(remotePort)"
    }
}

public enum TunnelState: Sendable, Equatable {
    case disconnected, connecting, connected, reconnecting, failed
}
```

Give each public struct a memberwise public initializer with the same parameter order as its stored properties. Define the runtime identity and snapshot exactly as follows; `TunnelLogEntry` is introduced in Task 6, so keep `logLines` as `[String]` at this boundary:

```swift
public enum TunnelRuntimeID: Hashable, Sendable {
    case saved(UUID)
    case temporary(UUID)
}

public struct TunnelRuntimeSnapshot: Identifiable, Sendable, Equatable {
    public let id: TunnelRuntimeID
    public let hostAlias: String
    public let name: String?
    public let remoteHost: String
    public let remotePort: UInt16
    public let localAddress: String
    public let localPort: UInt16
    public let state: TunnelState
    public let desiredConnection: Bool
    public let lastError: String?
    public let logLines: [String]
}

extension TunnelDefinition {
    static func fixture(name: String?, localPort: UInt16, remotePort: UInt16) -> Self {
        .init(id: UUID(uuidString: "00000000-0000-0000-0000-000000000001")!,
              hostAlias: "gpu", name: name, remoteHost: "127.0.0.1",
              remotePort: remotePort, localAddress: "127.0.0.1",
              localPort: localPort, createdAt: .distantPast, updatedAt: .distantPast)
    }
}
```

- [ ] **Step 4: Verify GREEN and the empty executable build**

Run: `swift test --filter TunnelDefinitionTests && swift build`

Expected: both commands exit 0 and `swift run TunnelDock` prints a temporary skeleton message then exits. This message is removed when the SwiftUI entry point is added in Task 11.

### Task 2: Input Validation and Forward Specification

**Files:**
- Create: `Sources/TunnelDockCore/Tunnel/ForwardSpecification.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/ForwardSpecificationTests.swift`

**Interfaces:**
- Produces: `TunnelInput`, `ForwardSpecification.init(localAddress:localPort:remoteHost:remotePort:) throws`, `openSSHArgument`, `InputValidator.port(_:)`, and `InputValidator.text(_:field:)`.

- [ ] **Step 1: Write validation and formatting tests**

```swift
func testFormatsIPv6EndpointsWithBrackets() throws {
    let value = try ForwardSpecification(
        localAddress: "::1", localPort: "8888",
        remoteHost: "2001:db8::10", remotePort: "8888"
    )
    XCTAssertEqual(value.openSSHArgument, "[::1]:8888:[2001:db8::10]:8888")
}

func testRejectsNonDecimalAndOutOfRangePorts() {
    for value in ["0", "65536", "8_888", " 8888", ""] {
        XCTAssertThrowsError(try InputValidator.port(value))
    }
}

func testRejectsControlCharactersInAddresses() {
    XCTAssertThrowsError(try InputValidator.text("db.internal\n-o ProxyCommand=x", field: .remoteHost))
    XCTAssertThrowsError(try InputValidator.text("127.0.0.1\0", field: .localAddress))
}
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter ForwardSpecificationTests`

Expected: compilation fails because the validation API is absent.

- [ ] **Step 3: Implement the validated value type**

Implement `ValidationField`, `InputValidationError`, and a `ForwardSpecification` whose stored ports are `UInt16`. Parse ports only when every Unicode scalar is ASCII `0...9`, then check `1...65535`. Trim address edges, reject empty values and Unicode control scalars, and format an address containing `:` as `[address]` after removing one valid outer bracket pair.

```swift
public struct TunnelInput: Sendable, Equatable {
    public var name: String?
    public var remoteHost: String
    public var remotePort: String
    public var localAddress: String
    public var localPort: String

    public init(name: String? = nil, remoteHost: String = "127.0.0.1",
                remotePort: String, localAddress: String = "127.0.0.1",
                localPort: String) {
        self.name = name
        self.remoteHost = remoteHost
        self.remotePort = remotePort
        self.localAddress = localAddress
        self.localPort = localPort
    }
}

public var openSSHArgument: String {
    "\(Self.bracket(localAddress)):\(localPort):\(Self.bracket(remoteHost)):\(remotePort)"
}
```

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter ForwardSpecificationTests`

Expected: all validation and IPv6 cases pass.

### Task 3: SSH Config Lexer, Include Resolver, and Alias Scanner

**Files:**
- Create: `Sources/TunnelDockCore/SSH/SSHConfigLexer.swift`
- Create: `Sources/TunnelDockCore/SSH/SSHIncludeResolver.swift`
- Create: `Sources/TunnelDockCore/SSH/SSHConfigScanner.swift`
- Create: `Tests/TunnelDockCoreTests/SSH/SSHIncludeResolverTests.swift`
- Create: `Tests/TunnelDockCoreTests/SSH/SSHConfigScannerTests.swift`

**Interfaces:**
- Produces: `ExpandedSSHConfig(lines:watchedFiles:watchedDirectories:diagnostics:)`, `SSHIncludeResolver.init(userSSHDirectory:)`, `resolve(rootURL:) throws`, and `SSHConfigScanner.discoverAliases(in:)`.

- [ ] **Step 1: Write fixture-backed Include tests**

Create temporary `config`, `config.d/a`, `config.d/b`, and nested files in the test. Assert that `Include config.d/*` expands in bytewise path order at the directive position, nested relative Includes remain based at the test `userSSHDirectory`, a cycle produces one diagnostic without recursion, and the glob parent directory is watched.

```swift
let expanded = try SSHIncludeResolver(userSSHDirectory: sshDirectory)
    .resolve(rootURL: sshDirectory.appending(path: "config"))
XCTAssertEqual(SSHConfigScanner().discoverAliases(in: expanded.lines), ["before", "alpha", "beta", "after"])
XCTAssertTrue(expanded.watchedDirectories.contains(sshDirectory.appending(path: "config.d")))
XCTAssertEqual(expanded.diagnostics.count, 1)
```

- [ ] **Step 2: Write alias-rule tests**

```swift
func testDiscoversOnlyExplicitAliasesOnceInTextOrder() {
    let lines = [
        "Host *", "Host gpu gpu-server lab-gpu", "Host gpu-* lab-? !blocked",
        "Host gpu", "  Host quoted # trailing comment"
    ]
    XCTAssertEqual(SSHConfigScanner().discoverAliases(in: lines),
                   ["gpu", "gpu-server", "lab-gpu", "quoted"])
}
```

- [ ] **Step 3: Verify RED**

Run: `swift test --filter SSHIncludeResolverTests && swift test --filter SSHConfigScannerTests`

Expected: compilation fails because resolver and scanner types are missing.

- [ ] **Step 4: Implement lexer, recursive expansion, and scanner**

`SSHConfigLexer` must strip comments only outside single/double quotes and split whitespace-delimited tokens while removing quote delimiters. The resolver expands `~`, absolute paths, and relative paths against `userSSHDirectory`; uses `glob(3)` with `GLOB_NOSORT`, then bytewise-sorts standardized paths; and tracks canonical URLs in the active recursion stack. It inserts included lines where the Include occurred and records missing matches as nonfatal diagnostics.

The scanner case-insensitively recognizes a first token equal to `Host`, rejects tokens starting with `!` or containing `*`, `?`, `[` or `]`, and preserves the first occurrence.

- [ ] **Step 5: Verify GREEN**

Run: `swift test --filter 'SSHIncludeResolverTests|SSHConfigScannerTests'`

Expected: all Include, cycle, lexer, wildcard, duplicate, and ordering tests pass.

### Task 4: Async Process Executor and Effective Host Resolution

**Files:**
- Create: `Sources/TunnelDockCore/Process/ProcessExecutor.swift`
- Create: `Sources/TunnelDockCore/SSH/SSHHostResolver.swift`
- Create: `Sources/TunnelDockCore/SSH/SSHConfigLoader.swift`
- Create: `Tests/TunnelDockCoreTests/SSH/SSHHostResolverTests.swift`
- Create: `Tests/TunnelDockCoreTests/SSH/SSHConfigLoaderTests.swift`

**Interfaces:**
- Consumes: aliases and expanded config from Task 3.
- Produces: `ProcessResult`, `ProcessExecuting.run(executableURL:arguments:)`, `SSHHostResolving.resolve(alias:order:)`, and `SSHConfigLoading.load(rootURL:)`.

- [ ] **Step 1: Write resolver tests against a recording executor**

```swift
let executor = RecordingProcessExecutor(result: .init(
    exitStatus: 0,
    stdout: Data("hostname 10.0.0.21\nuser researcher\nport 2222\n".utf8),
    stderr: Data()
))
let host = await SSHHostResolver(executor: executor).resolve(alias: "gpu", order: 3)
XCTAssertEqual(host.hostname, "10.0.0.21")
XCTAssertEqual(host.user, "researcher")
XCTAssertEqual(host.port, 2222)
XCTAssertEqual(await executor.calls.first?.arguments, ["-G", "gpu"])
XCTAssertEqual(await executor.calls.first?.executableURL.path, "/usr/bin/ssh")
```

Create `testNonzeroExitMarksConfigurationError`, `testMissingRequiredKeyMarksConfigurationError`, and `testInvalidPortMarksConfigurationError`; each supplies the named malformed `ProcessResult` and pattern-matches `.configurationError` on the returned host. Create `testSearchIsCaseInsensitiveAcrossEveryEffectiveField`, filter `[host]` with `GPU`, `10.0.0`, `RESEARCHER`, and `2222`, and assert every query returns the same host ID.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'SSHHostResolverTests|SSHConfigLoaderTests'`

Expected: compilation fails because process and resolver boundaries are absent.

- [ ] **Step 3: Implement async Process capture and host loading**

Define:

```swift
public struct ProcessResult: Sendable, Equatable {
    public let exitStatus: Int32
    public let stdout: Data
    public let stderr: Data
}

public protocol ProcessExecuting: Sendable {
    func run(executableURL: URL, arguments: [String]) async throws -> ProcessResult
}
```

`FoundationProcessExecutor` configures `Process`, stdout pipe, stderr pipe, and termination handler without shell invocation. It concurrently drains both pipes before returning. `SSHHostResolver` parses the first value for `hostname`, `user`, and `port`; any failure returns the alias with `.configurationError(message)` rather than throwing away the host. `SSHConfigLoader` resolves aliases with a maximum of eight concurrent `ssh -G` calls and restores `configOrder` ordering.

- [ ] **Step 4: Verify GREEN and perform a safe real-process integration check**

Run: `swift test --filter 'SSHHostResolverTests|SSHConfigLoaderTests' && /usr/bin/ssh -G tunneldock-nonexistent-alias >/dev/null`

Expected: tests pass and the real `ssh -G` command exits 0 without making a network connection.

### Task 5: Saved Tunnel Repository and Settings

**Files:**
- Create: `Sources/TunnelDockCore/Persistence/TunnelRepository.swift`
- Create: `Sources/TunnelDock/App/SettingsStore.swift`
- Create: `Tests/TunnelDockCoreTests/Persistence/TunnelRepositoryTests.swift`
- Create: `Tests/TunnelDockAppTests/SettingsStoreTests.swift`

**Interfaces:**
- Produces: actor `TunnelRepository`, `TunnelEnvelope(schemaVersion:tunnels:)`, mutations `save`, `rename`, `update`, `delete`, and `@MainActor SettingsStore.showMenuBar`.

- [ ] **Step 1: Write repository safety tests**

```swift
func testRoundTripsSchemaOneWithoutRuntimeFields() async throws {
    let repository = TunnelRepository(fileURL: fileURL)
    try await repository.replaceAll([definition])
    XCTAssertEqual(try await repository.load(), [definition])
    let json = try String(contentsOf: fileURL, encoding: .utf8)
    XCTAssertTrue(json.contains("\"schemaVersion\" : 1"))
    XCTAssertFalse(json.contains("connected"))
    XCTAssertFalse(json.contains("pid"))
}

func testMalformedFileIsPreservedAndLocksMutations() async throws {
    try Data("not-json".utf8).write(to: fileURL)
    let repository = TunnelRepository(fileURL: fileURL)
    do {
        _ = try await repository.load()
        XCTFail("Malformed JSON must fail to load")
    } catch { }
    do {
        try await repository.replaceAll([])
        XCTFail("A malformed source file must lock mutations")
    } catch { }
    XCTAssertEqual(try Data(contentsOf: fileURL), Data("not-json".utf8))
}
```

Add unsupported-schema, missing-file, timestamp update, rename-during-runtime-independent behavior, and delete tests. Test `SettingsStore` with a uniquely named `UserDefaults` suite and assert the absent key reads as true.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'TunnelRepositoryTests|SettingsStoreTests'`

Expected: compilation fails because repository and settings types do not exist.

- [ ] **Step 3: Implement atomic persistence and the one setting**

Use JSON encoders/decoders configured with `.iso8601`, `Data.write(to:options: .atomic)`, and a repository `isWriteLocked` flag set by malformed or unsupported input. Mutations update `updatedAt`, preserve `createdAt`, reject unknown UUIDs with a typed error, and persist before publishing the new collection.

```swift
@MainActor
final class SettingsStore: ObservableObject {
    @Published var showMenuBar: Bool { didSet { defaults.set(showMenuBar, forKey: key) } }
    private let defaults: UserDefaults
    private let key = "showMenuBar"

    init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
        self.showMenuBar = defaults.object(forKey: key) as? Bool ?? true
    }
}
```

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'TunnelRepositoryTests|SettingsStoreTests'`

Expected: all persistence and settings tests pass.

### Task 6: Log Ring Buffer and SSH Error Classification

**Files:**
- Create: `Sources/TunnelDockCore/Tunnel/TunnelLogBuffer.swift`
- Create: `Sources/TunnelDockCore/Tunnel/SSHErrorClassifier.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/TunnelLogBufferTests.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/SSHErrorClassifierTests.swift`

**Interfaces:**
- Produces: `TunnelLogEntry`, `TunnelLogBuffer.append(_:at:)`, `SSHUserError`, and `SSHErrorClassifier.classify(stderr:exitStatus:)`.

- [ ] **Step 1: Write eviction and classification tests**

```swift
func testKeepsNewestFiveHundredLines() {
    var buffer = TunnelLogBuffer(capacity: 500)
    for value in 0..<505 { buffer.append("line \(value)", at: date) }
    XCTAssertEqual(buffer.entries.count, 500)
    XCTAssertEqual(buffer.entries.first?.message, "line 5")
    XCTAssertEqual(buffer.entries.last?.message, "line 504")
}

func testClassifiesRequiredOpenSSHErrors() {
    XCTAssertEqual(classifier.classify(stderr: "Permission denied (publickey).", exitStatus: 255), .authenticationFailed)
    XCTAssertEqual(classifier.classify(stderr: "Host key verification failed.", exitStatus: 255), .hostVerificationRequired)
    XCTAssertEqual(classifier.classify(stderr: "Could not resolve hostname gpu", exitStatus: 255), .couldNotResolveHost)
}
```

Add fixtures for every required category and the generic fallback.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'TunnelLogBufferTests|SSHErrorClassifierTests'`

Expected: compilation fails because log and classifier APIs are missing.

- [ ] **Step 3: Implement bounded logging and best-effort messages**

Use an array-backed ring with a fixed capacity of 500 and batch-remove overflow. `SSHUserError` owns `title`, `message`, and optional terminal guidance. Match stable stderr fragments case-insensitively, but document in code that state transitions must not consume classifier output.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'TunnelLogBufferTests|SSHErrorClassifierTests'`

Expected: both suites pass.

### Task 7: Port Collision Checking and Control Socket Management

**Files:**
- Create: `Sources/TunnelDockCore/Tunnel/PortAvailabilityChecker.swift`
- Create: `Sources/TunnelDockCore/Tunnel/ControlSocketManager.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/PortAvailabilityCheckerTests.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/ControlSocketManagerTests.swift`

**Interfaces:**
- Produces: `LocalEndpoint`, `PortOwner`, `PortChecking.check(_:)`, reservation methods, `LocalListenerProbing.waitUntilListening(_:timeout:)`, and `ControlSocketManaging.prepareDirectory`, `allocateSocketURL`, `removeSocket`, `removeStaleSockets`.

- [ ] **Step 1: Write real loopback and filesystem-permission tests**

Reserve an ephemeral BSD socket in the test, then assert the checker reports `.occupiedBySystem`. Reserve the same endpoint through the actor and assert `.occupiedByTunnelDock` takes precedence. Assert a fresh loopback endpoint is `.available`.

```swift
let directory = try manager.prepareDirectory()
let mode = try XCTUnwrap(FileManager.default.attributesOfItem(atPath: directory.path)[.posixPermissions] as? NSNumber)
XCTAssertEqual(mode.intValue & 0o777, 0o700)
XCTAssertLessThan(try manager.allocateSocketURL().path.utf8.count, 100)
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'PortAvailabilityCheckerTests|ControlSocketManagerTests'`

Expected: compilation fails because the endpoint and socket manager APIs are absent.

- [ ] **Step 3: Implement Darwin bind preflight and private runtime paths**

`SystemPortAvailabilityChecker` resolves the supplied hostname/address with `getaddrinfo`, creates a socket for each candidate, sets `SO_REUSEADDR` off, and attempts `bind`; it closes every descriptor. The actor tracks successful TunnelDock reservations as `Set<LocalEndpoint>` and never selects a replacement port. `PortOwner` has `.tunnelDock` and `.system` cases so UI errors can distinguish the messages.

`SystemLocalListenerProbe` repeatedly attempts a nonblocking TCP connection to the requested endpoint until the local SSH listener accepts or the supplied timeout expires. It treats a completed local TCP handshake as listener proof and does not wait for remote application data, so an absent remote service cannot turn the tunnel state to Failed.

Build the runtime directory directly as `URL(fileURLWithPath: "/tmp/tunneldock-\(getuid())", isDirectory: true)`. Create it with POSIX permissions `0700`; socket filenames use the first 12 lowercase hexadecimal characters of a UUID. Stale cleanup only removes entries ending in `.sock` inside that exact directory.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'PortAvailabilityCheckerTests|ControlSocketManagerTests'`

Expected: collision, available-port, path-length, permissions, and scoped-cleanup tests pass.

### Task 8: SSH Command Construction and Master Process Controller

**Files:**
- Create: `Sources/TunnelDockCore/Tunnel/SSHProcessController.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/SSHProcessControllerTests.swift`

**Interfaces:**
- Consumes: `ForwardSpecification`, `ProcessExecuting`, and control socket URL.
- Produces: `SSHCommandBuilder`, `SSHMasterHandle`, `SSHMasterEvent`, `SSHProcessControlling.startMaster`, `waitUntilReady`, `addForward`, `requestExit`, `terminate`, and `kill`.

- [ ] **Step 1: Write exact argument-array tests**

```swift
XCTAssertEqual(builder.masterArguments(alias: "gpu", socket: socket), [
    "-M", "-S", socket.path, "-N", "-T", "-n",
    "-o", "ControlPersist=no",
    "-o", "ClearAllForwardings=yes",
    "-o", "ExitOnForwardFailure=yes",
    "-o", "ServerAliveInterval=15",
    "-o", "ServerAliveCountMax=3",
    "-o", "BatchMode=yes",
    "-o", "StrictHostKeyChecking=yes",
    "gpu"
])
XCTAssertEqual(builder.forwardArguments(alias: "gpu", socket: socket, specification: spec),
               ["-S", socket.path, "-O", "forward", "-L", "127.0.0.1:18888:127.0.0.1:8888", "gpu"])
```

Also assert check and exit commands use `-S`, `-O check|exit`, and the alias; stdout/stderr streaming is attributed to the correct handle; no executable other than `/usr/bin/ssh` is requested.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter SSHProcessControllerTests`

Expected: compilation fails because command builder and long-running controller APIs do not exist.

- [ ] **Step 3: Implement two Process boundaries**

Extend the process abstraction with a long-running handle:

```swift
public protocol RunningProcess: Sendable {
    var processIdentifier: Int32 { get }
    func events() -> AsyncStream<ProcessEvent>
    func terminate()
    func interruptWithKill()
}

public protocol ProcessLaunching: Sendable {
    func launch(executableURL: URL, arguments: [String]) throws -> any RunningProcess
}
```

`SSHProcessController` starts the master through `ProcessLaunching`, polls readiness through `ProcessExecuting` with `-O check`, adds the forward through `-O forward`, and requests graceful exit through `-O exit`. Readiness has a fixed 10-second deadline and 100-millisecond poll interval supplied through an injectable clock. The controller never interprets remote-service stderr as a master exit.

`SSHCommandBuilder` is a pure `Sendable` value whose `masterArguments`, `checkArguments`, `forwardArguments`, and `exitArguments` return the exact arrays asserted in Step 1. Both process dependencies receive the fixed executable URL `URL(fileURLWithPath: "/usr/bin/ssh")` from the controller rather than from callers.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter SSHProcessControllerTests`

Expected: exact command, readiness, stream, deadline, exit, terminate, and kill tests pass.

### Task 9: Tunnel Manager Connect, Temporary Save, and Disconnect

**Files:**
- Create: `Sources/TunnelDockCore/Tunnel/TunnelManager.swift`
- Complete: `Sources/TunnelDockCore/Models/TunnelRuntime.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/TunnelManagerConnectionTests.swift`

**Interfaces:**
- Consumes: repository, port checker, socket manager, process controller, host availability provider, and `TunnelInput` from Task 2.
- Produces: `TunnelManagerError`, `@MainActor TunnelManager.connectSaved(id:)`, `connectTemporary(hostAlias:input:)`, `saveTemporary(id:name:)`, `disconnect(id:)`, `rename(id:name:)`, `edit(id:input:)`, and published `[TunnelRuntimeSnapshot]`.

- [ ] **Step 1: Write state-sequence tests with fakes**

```swift
let id = try await manager.connectTemporary(hostAlias: "gpu", input: .valid)
await harness.completeMasterReady(id)
await harness.completeForward(id)
await harness.reportListenerReady(id)
XCTAssertEqual(manager.snapshot(id: id)?.state, .connected)
XCTAssertTrue(manager.snapshot(id: id)?.desiredConnection == true)

let savedID = try await manager.saveTemporary(id: id, name: nil)
XCTAssertEqual(manager.snapshot(id: .saved(savedID))?.state, .connected)
XCTAssertEqual(harness.masterLaunchCount, 1)

await manager.disconnect(id: .saved(savedID))
XCTAssertEqual(manager.snapshot(id: .saved(savedID))?.state, .disconnected)
XCTAssertEqual(harness.requestedExitSockets, [socket])
```

Add tests that invalid input, missing/config-error host, internal collision, and OS collision launch zero processes; a port collision exposes `focusLocalPort = true`; connected tunnels reject forwarding edits and deletion; running tunnels accept rename.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter TunnelManagerConnectionTests`

Expected: compilation fails because `TunnelManager` does not exist.

- [ ] **Step 3: Implement minimal serialized lifecycle**

Represent each mutable runtime in a main-actor-only internal class with `generation: UInt64` and `operationTask: Task<Void, Never>?`. Connect increments generation, reserves the endpoint, allocates the socket, and runs Task 8's master-ready-forward-listener sequence. Every awaited continuation compares generation before mutation. Only after listener confirmation does a temporary runtime enter the published list.

Define `TunnelManagerError` with `.hostNotFound`, `.configurationError(String)`, `.invalidInput(InputValidationError)`, `.localPortInUse(UInt16, owner: PortOwner)`, `.editingActiveTunnel`, `.deletingActiveTunnel`, and `.runtimeNotFound`. Quick Forward maps `.localPortInUse` to its expand-and-focus event; other cases expose concrete English user messages.

Disconnect sets desired false before cancellation, increments generation, cancels tasks, requests exit, applies TERM/KILL escalation through the controller, removes the socket and reservation, then sets disconnected or removes a temporary runtime. Save persists a definition first, then rekeys the same internal runtime from temporary to saved without process restart.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter TunnelManagerConnectionTests`

Expected: connect, collision, save-without-restart, edit restrictions, rename, deletion restriction, and disconnect tests pass.

### Task 10: Reconnect, Missing Hosts, Logs, and Quit

**Files:**
- Modify: `Sources/TunnelDockCore/Tunnel/TunnelManager.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/TunnelManagerRecoveryTests.swift`
- Create: `Tests/TunnelDockCoreTests/Tunnel/TunnelManagerShutdownTests.swift`

**Interfaces:**
- Adds: `updateHosts(_:)`, `shutdownAll() async`, injectable `TunnelClock`, retry sequence, process-termination handling, and runtime log publication.

- [ ] **Step 1: Write reconnect and host-removal tests**

Use a manual clock and assert scheduled sleeps exactly equal `[1, 2, 5, 10, 30, 30]`. Assert a failure before first connected goes to Failed with no sleep. Assert a successful retry resets the next delay to one second. Remove a live host and assert it stays Connected; then terminate its process and assert Failed/Host not found with no sleep. Restore it and assert Connect becomes available but no process starts automatically.

Add a test that a stale generation's termination callback cannot change a newer Connected state and a test that stderr `connect failed: Connection refused` only appends a log line.

- [ ] **Step 2: Write shutdown escalation tests**

```swift
await manager.shutdownAll()
XCTAssertTrue(harness.allDesiredConnectionsAreFalse)
XCTAssertEqual(harness.shutdownEvents, [
    .requestExit(firstSocket), .requestExit(secondSocket),
    .terminate(firstPID), .terminate(secondPID),
    .kill(secondPID),
    .removeSocket(firstSocket), .removeSocket(secondSocket)
])
```

- [ ] **Step 3: Verify RED**

Run: `swift test --filter 'TunnelManagerRecoveryTests|TunnelManagerShutdownTests'`

Expected: tests fail because reconnect, host updates, and shutdown escalation are not implemented.

- [ ] **Step 4: Implement recovery and shutdown**

Use retry delays `[1, 2, 5, 10, 30]` with `min(attempt, 4)`. Schedule only when desired is true, `hasEverConnected` is true, and the alias is available. Preserve the same log over automatic retries and replace it at a new manual Connect. `updateHosts` changes availability without stopping live masters. `shutdownAll` disables every desired connection before awaiting any process work, then performs batched graceful exit, TERM after two seconds, KILL after one further second, and scoped socket cleanup.

- [ ] **Step 5: Verify GREEN**

Run: `swift test --filter 'TunnelManagerRecoveryTests|TunnelManagerShutdownTests'`

Expected: all deterministic recovery, generation, remote-service, host removal/restoration, log, and shutdown tests pass.

### Task 11: SSH Config Watcher and App State Join

**Files:**
- Create: `Sources/TunnelDockCore/SSH/SSHConfigWatcher.swift`
- Create: `Sources/TunnelDock/App/AppState.swift`
- Create: `Tests/TunnelDockCoreTests/SSH/SSHConfigWatcherTests.swift`
- Create: `Tests/TunnelDockAppTests/AppStateTests.swift`

**Interfaces:**
- Consumes: loader snapshots, repository definitions, and manager snapshots.
- Produces: `SSHConfigWatching.events()`, `@MainActor AppState.refreshSSHConfig()`, `normalHosts`, `missingHosts`, `selectedHostAlias`, and filtered views.

- [ ] **Step 1: Write debounce and join tests**

Inject a watcher event source and manual clock. Emit three events within 300 ms and assert one reload request; emit after 301 ms and assert another. Change the watch snapshot and assert old file descriptors/sources are canceled.

```swift
state.apply(hosts: [gpu], definitions: [nasTunnel])
XCTAssertEqual(state.normalHosts.map(\.alias), ["gpu"])
XCTAssertEqual(state.missingHosts.map(\.alias), ["nas"])
state.apply(hosts: [nas, gpu], definitions: [nasTunnel])
XCTAssertEqual(state.normalHosts.map(\.alias), ["nas", "gpu"])
XCTAssertTrue(state.missingHosts.isEmpty)
```

Add search-order tests and missing-config empty-state copy.

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'SSHConfigWatcherTests|AppStateTests'`

Expected: compilation fails because watcher and AppState are missing.

- [ ] **Step 3: Implement DispatchSource-based watching and snapshot joins**

Use `DispatchSource.makeFileSystemObjectSource` for every existing watched file and directory, opened with `O_EVTONLY`. Convert callbacks to an `AsyncStream<Void>`, cancel/close descriptors on watch-set replacement, and debounce using an injected clock. `AppState` retains the last complete host snapshot during refresh, joins absent definition aliases into lightweight missing-host rows, forwards availability to `TunnelManager`, and never reorders filtered hosts.

- [ ] **Step 4: Verify GREEN**

Run: `swift test --filter 'SSHConfigWatcherTests|AppStateTests'`

Expected: watcher debounce, source replacement, config absence, joins, recovery, search, and ordering tests pass.

### Task 12: Main SwiftUI Window and Tunnel Workflows

**Files:**
- Replace: `Sources/TunnelDock/main.swift` with `Sources/TunnelDock/App/TunnelDockApp.swift`
- Create: `Sources/TunnelDock/UI/MainWindowView.swift`
- Create: `Sources/TunnelDock/UI/HostSidebar.swift`
- Create: `Sources/TunnelDock/UI/HostDetailView.swift`
- Create: `Sources/TunnelDock/UI/TunnelRow.swift`
- Create: `Sources/TunnelDock/UI/QuickForwardView.swift`
- Create: `Sources/TunnelDock/UI/TunnelEditorView.swift`
- Create: `Sources/TunnelDock/UI/TunnelLogView.swift`
- Create: `Tests/TunnelDockAppTests/QuickForwardModelTests.swift`

**Interfaces:**
- Consumes: `AppState`, `TunnelManager`, and `SettingsStore`.
- Produces: macOS main window UI and `@MainActor QuickForwardModel` with explicit local-port edit tracking.

- [ ] **Step 1: Write Quick Forward presentation-model tests**

```swift
func testLocalPortFollowsUntilExplicitlyEdited() {
    let model = QuickForwardModel()
    model.setRemotePort("8888")
    XCTAssertEqual(model.localPort, "8888")
    model.setLocalPort("18888", userInitiated: true)
    model.setRemotePort("6006")
    XCTAssertEqual(model.localPort, "18888")
}

func testCollisionExpandsAdvancedAndFocusesLocalPort() {
    let model = QuickForwardModel()
    model.handle(.localPortInUse(8888))
    XCTAssertTrue(model.isAdvancedExpanded)
    XCTAssertEqual(model.focusedField, .localPort)
}
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter QuickForwardModelTests`

Expected: compilation fails because the view model is missing.

- [ ] **Step 3: Implement the main window and workflows**

Use `NavigationSplitView` with a searchable sidebar, `SSH Hosts` and conditional `Missing Hosts` sections. Host detail renders effective connection metadata, saved and temporary rows, then Quick Forward. Bind row buttons only to state-valid actions: Connect, Disconnect, Save, Rename, Edit, Delete, and View Log. Disable Connect for missing/config-error hosts; disable forwarding edit and delete for active states; keep Rename enabled. Delete presents a confirmation dialog and never disconnects automatically.

Quick Forward starts collapsed with Remote Port and Connect; Advanced contains Local Port, Remote Host default `127.0.0.1`, and Local Address default `127.0.0.1`. Bind `@FocusState` to Local Port after collisions. Sheets contain concrete fields and display `SSHUserError.message`; log view displays timestamped entries in a selectable monospaced list.

- [ ] **Step 4: Verify GREEN and compile SwiftUI**

Run: `swift test --filter QuickForwardModelTests && swift build`

Expected: tests pass and the complete SwiftUI executable compiles with no errors.

### Task 13: Menu Bar, Settings, Window Reopen, and Delayed Quit

**Files:**
- Create: `Sources/TunnelDock/App/AppDelegate.swift`
- Modify: `Sources/TunnelDock/App/TunnelDockApp.swift`
- Create: `Sources/TunnelDock/UI/MenuBarContentView.swift`
- Create: `Sources/TunnelDock/UI/SettingsView.swift`
- Create: `Tests/TunnelDockAppTests/MenuBarModelTests.swift`
- Create: `Tests/TunnelDockAppTests/AppTerminationCoordinatorTests.swift`

**Interfaces:**
- Produces: saved-only menu filtering, `AppTerminationCoordinator`, Dock reopen handling, conditional `MenuBarExtra`, and Settings scene.

- [ ] **Step 1: Write menu projection and termination tests**

```swift
XCTAssertEqual(MenuBarModel(saved: [jupyter, webUI], temporary: [scratch]).rows.map(\.id),
               [jupyter.id, webUI.id])
XCTAssertEqual(MenuBarModel(saved: [jupyter], query: "8888").rows.map(\.id), [jupyter.id])

let reply = coordinator.applicationShouldTerminate()
XCTAssertEqual(reply, .terminateLater)
await harness.finishShutdown()
XCTAssertEqual(harness.applicationReplies, [true])
```

- [ ] **Step 2: Verify RED**

Run: `swift test --filter 'MenuBarModelTests|AppTerminationCoordinatorTests'`

Expected: compilation fails because menu and termination coordination are absent.

- [ ] **Step 3: Implement scenes and AppKit lifecycle**

Declare `WindowGroup`, conditional `MenuBarExtra` controlled by `SettingsStore.showMenuBar`, and a Settings scene containing only `Show in Menu Bar`. Menu content groups saved tunnels by host, searches host/name/ports, and exposes only Connect/Disconnect plus Open, Refresh, Settings, and Quit.

Adapt an `NSApplicationDelegate` through `@NSApplicationDelegateAdaptor`. `applicationShouldTerminateAfterLastWindowClosed` returns false. Dock reopen activates the app and opens/raises the main window. Termination returns `.terminateLater`, invokes `TunnelManager.shutdownAll()` exactly once, then calls `NSApplication.shared.reply(toApplicationShouldTerminate: true)` on the main actor.

- [ ] **Step 4: Verify GREEN and compile all scenes**

Run: `swift test --filter 'MenuBarModelTests|AppTerminationCoordinatorTests' && swift build`

Expected: tests pass and SwiftUI/AppKit integration compiles.

### Task 14: App Bundle Packaging

**Files:**
- Create: `Resources/Info.plist`
- Create: `Scripts/package-app.sh`
- Create: `Tests/Packaging/package-app-tests.sh`

**Interfaces:**
- Consumes: release `TunnelDock` executable.
- Produces: `.build/release/TunnelDock.app` with a regular foreground macOS app plist.

- [ ] **Step 1: Write the failing packaging verification script**

```bash
#!/bin/sh
set -eu
APP=".build/release/TunnelDock.app"
test -x "$APP/Contents/MacOS/TunnelDock"
test "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$APP/Contents/Info.plist")" = "com.tunneldock.TunnelDock"
test "$(/usr/libexec/PlistBuddy -c 'Print :LSMinimumSystemVersion' "$APP/Contents/Info.plist")" = "13.0"
if /usr/libexec/PlistBuddy -c 'Print :LSUIElement' "$APP/Contents/Info.plist" >/dev/null 2>&1; then
    exit 1
fi
```

- [ ] **Step 2: Verify RED**

Run: `sh Tests/Packaging/package-app-tests.sh`

Expected: failure because the app bundle does not exist.

- [ ] **Step 3: Implement plist and deterministic packaging**

`Info.plist` defines `CFBundleExecutable=TunnelDock`, `CFBundleIdentifier=com.tunneldock.TunnelDock`, `CFBundleName=TunnelDock`, `CFBundlePackageType=APPL`, `CFBundleShortVersionString=1.0.0`, `CFBundleVersion=1`, `LSMinimumSystemVersion=13.0`, and `NSHighResolutionCapable=true`. It omits `LSUIElement` and sandbox entitlements.

```bash
#!/bin/sh
set -eu
swift build -c release
APP=".build/release/TunnelDock.app"
mkdir -p "$APP/Contents/MacOS"
cp ".build/release/TunnelDock" "$APP/Contents/MacOS/TunnelDock"
cp "Resources/Info.plist" "$APP/Contents/Info.plist"
chmod 755 "$APP/Contents/MacOS/TunnelDock"
```

- [ ] **Step 4: Verify GREEN**

Run: `sh Scripts/package-app.sh && sh Tests/Packaging/package-app-tests.sh`

Expected: release build and all bundle assertions pass.

### Task 15: Full Verification and Manual Acceptance Handoff

**Files:**
- Create: `docs/manual-acceptance-checklist.md`

**Interfaces:**
- Produces: a release-candidate app bundle and a recorded checklist for environment-dependent SSH behavior.

- [ ] **Step 1: Write the manual acceptance checklist from the product spec**

Include explicit unchecked commands and observations for: ordinary/multiple/wildcard Hosts; nested Include and live refresh; `Host *` and pattern inheritance via `ssh -G`; basic and custom-port forwards; remote host and IPv6; externally occupied port; Save/Rename/Edit/Delete/Reconnect; simultaneous independent masters; suppression of config LocalForward/RemoteForward/DynamicForward; password-required failure; unknown and changed host keys; first failure without retry; established disconnect delays; remote service absent while Connected; host removal/restoration; window close; menu-bar toggle; and Quit orphan-process inspection.

For every SSH scenario, require a disposable user-controlled host alias and record `Pass`, `Fail`, or `Not run — no test environment`. Never edit known_hosts automatically.

- [ ] **Step 2: Run the full automated suite fresh**

Run: `swift test`

Expected: all test suites pass with zero failures.

- [ ] **Step 3: Run clean debug and release builds**

Run: `swift package clean && swift build && swift build -c release`

Expected: all three commands exit 0 with no compiler errors.

- [ ] **Step 4: Package and inspect the release artifact**

Run: `sh Scripts/package-app.sh && sh Tests/Packaging/package-app-tests.sh && file .build/release/TunnelDock.app/Contents/MacOS/TunnelDock`

Expected: packaging tests pass and `file` reports an arm64 Mach-O executable for this machine.

- [ ] **Step 5: Perform non-network local checks**

Run: `/usr/bin/ssh -G tunneldock-nonexistent-alias >/dev/null && pgrep -fal '/usr/bin/ssh.*tunneldock-' || true`

Expected: `ssh -G` succeeds without connecting; no orphan TunnelDock-owned SSH master is listed before manual testing.

- [ ] **Step 6: Record environment-dependent results without overstating coverage**

Run the applicable rows in `docs/manual-acceptance-checklist.md`. Mark unavailable server-dependent rows as `Not run — no test environment`; do not convert them to passes. Any observed failure begins a new red-green regression cycle before rerunning Steps 2-5.

## Plan Self-Review Checklist

- Every product-spec area maps to Tasks 1-15 or to an explicit manual acceptance row.
- Every production behavior begins with a focused failing test, except static plist/script assembly which begins with a failing shell assertion.
- Cross-task names are consistent: `TunnelDefinition`, `TunnelRuntimeSnapshot`, `ForwardSpecification`, `ProcessExecuting`, `SSHProcessControlling`, `TunnelManager`, `AppState`, and `SettingsStore`.
- No task introduces remote forwarding, SOCKS, terminal, SFTP, credential storage, automatic host-key acceptance, service health checks, launch at login, notifications, connection restoration, persistent logs, or shared SSH masters.

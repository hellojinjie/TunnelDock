# TunnelDock v1.0 SwiftPM Design

**Date:** 2026-08-25

**Status:** Approved in conversation; awaiting written-spec review

**Source specification:** `docs/TunnelDock v1.0 Product & Technical Specification.md`

## 1. Purpose and Scope

TunnelDock v1.0 is a native SwiftUI macOS application that discovers explicit SSH aliases from the user's `~/.ssh/config`, resolves each alias through `/usr/bin/ssh -G`, and manages independent local SSH port forwards. The implementation follows the source specification as the product authority and does not add post-v1 features.

The project uses Swift Package Manager only. It must build and test without installing Xcode or relying on an Xcode project. A repository script assembles the release executable into a standard `.app` bundle.

Global constraints:

- Swift and SwiftUI, targeting macOS 13 Ventura or later.
- Swift 6 language mode and concurrency checking.
- No third-party dependencies.
- No App Sandbox and no Mac App Store assumptions.
- `/usr/bin/ssh` is the only SSH implementation.
- SSH commands use `Process.executableURL` and `Process.arguments`; a shell never participates.
- The Dock icon remains visible. Menu bar visibility is user-configurable and defaults to on.
- Window close is not application quit.
- No password, passphrase, private-key, or host-key management.

## 2. Package Architecture

The package has a testable core library, a testable app-state support library, and a thin SwiftUI executable:

```text
Package.swift
├── Sources/TunnelDockCore
│   ├── SSH
│   ├── Tunnel
│   └── Persistence
├── Sources/TunnelDockAppSupport
│   └── App
├── Sources/TunnelDock
│   ├── App
│   └── UI
├── Tests/TunnelDockCoreTests
└── Scripts/package-app.sh
```

`TunnelDockCore` contains no SwiftUI. It owns parsing, validation, persistence, process control, logs, and the tunnel state machine. `TunnelDockAppSupport` contains independently testable UI-facing state such as `AppState`, settings, and presentation models. `TunnelDock` contains application lifecycle integration, SwiftUI scenes, and views.

The SwiftPM products are:

- `TunnelDockCore`: a library used by the executable and tests.
- `TunnelDockAppSupport`: a library used by the executable and app-state tests.
- `TunnelDock`: an executable containing the SwiftUI `App` entry point.

The packaging script runs `swift build -c release`, creates `TunnelDock.app/Contents/MacOS`, copies the executable, and writes a fixed `Info.plist`. The plist identifies a regular foreground application and does not set `LSUIElement`.

## 3. SSH Configuration Pipeline

Configuration loading is a snapshot pipeline:

```text
~/.ssh/config and Include directives
    -> SSHIncludeResolver
    -> ordered expanded source lines and watched paths
    -> SSHConfigScanner
    -> ordered, explicit, de-duplicated aliases
    -> SSHHostResolver using /usr/bin/ssh -G
    -> immutable [SSHHost] snapshot
    -> AppState and SwiftUI
```

### 3.1 Include resolution

`SSHIncludeResolver` starts only at `~/.ssh/config`. It expands each `Include` at its textual position and supports tilde paths, absolute paths, paths relative to the user's `~/.ssh` directory, globbing, and recursive includes. Nested files do not change the base directory for a relative user-config Include. Glob results use deterministic bytewise path ordering to mirror OpenSSH expansion as closely as the platform permits.

The resolver canonicalizes included paths for cycle detection. A path already present in the active include stack is skipped and recorded as a nonfatal diagnostic. The same file may still be included again from a separate, noncyclic position because textual inclusion order matters.

The result includes existing files plus parent directories corresponding to glob patterns. These paths become the configuration watch set.

### 3.2 Alias discovery

`SSHConfigScanner` tokenizes expanded logical lines sufficiently to identify `Host` directives without implementing OpenSSH inheritance rules. It emits each Host token that:

- is nonempty;
- does not start with `!`;
- contains none of `*`, `?`, or character-class glob syntax;
- has not previously been emitted.

Multiple explicit aliases on one `Host` line are emitted separately in textual order. Wildcard patterns remain in the source passed conceptually to OpenSSH but do not appear as selectable hosts.

### 3.3 Effective configuration

For every discovered alias, `SSHHostResolver` executes `/usr/bin/ssh -G <alias>` and parses at least `hostname`, `user`, and `port`. Resolution runs outside the main actor with bounded concurrency so a large config does not freeze the UI or spawn an unbounded number of processes.

A failed alias remains in the host snapshot with `configurationError` availability. Failures do not discard successfully resolved aliases. Search performs case-insensitive substring matching over alias, hostname, user, and decimal SSH port while preserving config order.

### 3.4 Refresh and watching

`SSHConfigWatcher` observes the root file, resolved included files, and include-glob directories. It debounces events for 300 milliseconds and then requests a full snapshot reload. The watch set is replaced after each successful resolution so newly matched or removed glob files are reflected.

Manual refresh calls the same pipeline immediately. The UI retains the previous complete snapshot while a refresh is running and swaps in the new complete snapshot on the main actor. Missing `~/.ssh/config` produces an empty successful snapshot rather than an application error.

## 4. Data and Persistence

### 4.1 SSH hosts

`SSHHost` is derived, nonpersistent data containing alias, hostname, user, port, config order, and availability. AppState joins the current host snapshot with saved tunnel aliases. A saved alias absent from the snapshot is represented as a Missing Host; when it reappears, the join automatically restores the normal host.

### 4.2 Saved definitions

`TunnelDefinition` contains:

- `id: UUID`
- `hostAlias: String`
- optional `name: String`
- `remoteHost: String`
- `remotePort: UInt16`
- `localAddress: String`
- `localPort: UInt16`
- `createdAt: Date`
- `updatedAt: Date`

Definitions are stored at `~/Library/Application Support/TunnelDock/saved-tunnels.json` in a Codable envelope containing `schemaVersion: 1` and `tunnels`. Dates use a stable ISO 8601 representation.

`TunnelRepository` is an actor that serializes reads and mutations. Writes use a sibling temporary file followed by atomic replacement. A missing file loads as an empty collection. Malformed JSON or an unsupported schema is surfaced as a persistence error, the original file is preserved, and mutating writes remain disabled until a valid reload prevents accidental overwrite.

Runtime state, PID, control socket, log, connection state, and retry count are never persisted. Every saved tunnel starts disconnected on application launch. `showMenuBar` is the only v1 setting and is stored separately in `UserDefaults`, defaulting to `true`.

### 4.3 Temporary tunnels

A Quick Forward request is created as a runtime-only tunnel. It appears in the selected host's list only after the connection reaches `Connected`. Disconnect removes it and its log. Save creates a `TunnelDefinition` for the same running runtime without restarting SSH and thereafter treats it as a saved tunnel.

## 5. Validation and Forward Formatting

The validation layer rejects input before launching a process:

- ports must be decimal integers in `1...65535`;
- host alias, remote host, and local address must be nonempty after trimming;
- text fields reject NUL, newline, carriage return, and other illegal control characters;
- the optional name may be empty and is normalized to `nil`;
- a non-loopback local address is accepted only when it comes from an explicit user edit.

`ForwardSpecification` formats `localAddress:localPort:remoteHost:remotePort`. Address tokens containing a colon are enclosed in brackets unless already validly bracketed, producing forms such as `[::1]:8888:[2001:db8::10]:8888`.

Local Port follows Remote Port until the user first edits Local Port. Subsequent Remote Port changes preserve the explicit local value.

## 6. Tunnel Ownership and State Machine

`@MainActor TunnelManager` is the sole owner of UI-facing tunnel lifecycles. `TunnelDefinition` and `TunnelRuntime` remain separate. Each runtime contains state, desired connection, process identity, socket path, whether it has ever connected, retry attempt, retry task, last error, and a 500-line in-memory log.

Each tunnel has a serialized operation lane. Every connect lifecycle receives a monotonically increasing generation token; callbacks from a canceled or superseded generation are ignored. This prevents stale process completions and timers from reversing newer user actions.

### 6.1 Connect sequence

Connect performs these steps in order:

1. Validate fields and host availability.
2. Reject a local endpoint reserved by another TunnelDock runtime.
3. Ask `PortAvailabilityChecker` to bind the requested local endpoint temporarily; release it before SSH starts.
4. Allocate a short runtime control socket under `/tmp/tunneldock-<uid>/`, whose directory mode is `0700`.
5. Start an independent SSH master with `-M`, `-S`, `-N`, `-T`, `-n`, `ControlPersist=no`, `ClearAllForwardings=yes`, `ExitOnForwardFailure=yes`, `ServerAliveInterval=15`, `ServerAliveCountMax=3`, `BatchMode=yes`, and `StrictHostKeyChecking=yes`.
6. Poll the control socket with `ssh -S <socket> -O check <control-target>` until ready or the startup deadline/process failure occurs.
7. Add exactly one TunnelDock forward using `ssh -S <socket> -O forward -L <spec> <control-target>`.
8. Confirm that the local endpoint accepts a TCP connection within a bounded interval.
9. Transition to `Connected`, mark `hasEverConnected`, and reset the retry counter.

The preflight port bind reduces obvious failures but does not claim exclusivity; `ExitOnForwardFailure` and the forward command remain authoritative for races.

### 6.2 Disconnect and quit

Disconnect first sets `desiredConnection` to false and increments the generation. It cancels pending retry and connection tasks, requests master exit through the control socket, terminates any remaining child process, cleans its socket, releases the internal endpoint reservation, and transitions to `Disconnected`. A temporary runtime is then removed.

Application quit first disables reconnect for every tunnel. It requests graceful master exit, then sends SIGTERM after a short grace period, and finally SIGKILL only for remaining owned children. Socket cleanup completes before AppKit is told to finish terminating.

### 6.3 Failure and reconnect

A tunnel that has never reached `Connected` transitions from `Connecting` to `Failed` without automatic retry.

If a previously connected master exits unexpectedly while `desiredConnection` is true and its host is available, state becomes `Reconnecting`. Retry delays are `1, 2, 5, 10, 30` seconds and then 30 seconds indefinitely. A successful reconnect resets the attempt count.

If a host disappears, an active master remains running. If that master later exits while the host is missing, state becomes `Failed` with Host not found and no retry is scheduled. Reappearance before disconnection preserves reconnect eligibility; reappearance after failure re-enables the Connect action but does not connect automatically.

OpenSSH stderr indicating a remote application connection refusal is logged but does not change an established tunnel's state. `Connected` asserts only that the SSH master lives, the forward request succeeded, and the local listener was observed.

## 7. Process and Socket Boundaries

`SSHProcessController` owns the concrete `Process` interactions behind protocols used by the core state machine. It captures stdout and stderr asynchronously without blocking the main actor. Production always uses `/usr/bin/ssh`; tests substitute a deterministic process runner without invoking a shell.

The Stage 1 master deliberately clears config-defined LocalForward, RemoteForward, and DynamicForward values. Stage 2 adds the app's sole forwarding request. Each tunnel has its own master and socket.

`ControlSocketManager` creates `/tmp/tunneldock-<uid>/` with mode `0700`, uses short random runtime identifiers for socket names, and removes sockets owned by the current lifecycle. At launch it removes stale entries only inside this application-specific directory, never arbitrary `/tmp` content.

## 8. Error Handling and Logs

Core transitions depend on structured events such as process exit, control-command result, timeout, listener observation, host availability, and user intent. They do not depend on complete parsing of unstable OpenSSH prose.

`SSHErrorClassifier` performs a best-effort mapping of captured stderr to these user categories:

- Local port is already in use
- Authentication failed
- Host verification required
- Host not found
- SSH configuration error
- Connection timed out
- Could not resolve host
- Connection refused by SSH server
- SSH process exited unexpectedly
- SSH connection failed; see Log for details

Unknown host keys never trigger automatic acceptance. The Host verification error instructs the user to connect once with `ssh <alias>` in Terminal. Authentication errors explain that TunnelDock requires the system SSH environment to work noninteractively.

Each runtime owns a ring buffer of at most 500 timestamped lines. It includes lifecycle messages, stdout, stderr, reconnect scheduling, process status, and forward results. Logs never reach disk. A saved tunnel begins a new log on each new manual Connect; automatic reconnect continues the same log.

## 9. SwiftUI Design

The executable uses macOS 13-compatible `ObservableObject` and `@Published` state at the UI boundary. Core services remain independent of SwiftUI.

### 9.1 Main window

The main window uses `NavigationSplitView`. Its sidebar contains a search field, an SSH Hosts section in config order, and a Missing Hosts section. Selecting a host shows alias, effective `user @ hostname:port`, Saved and Temporary Tunnels, and Quick Forward.

Tunnel rows show display name, port mapping, state, and valid actions. An unnamed equal-port tunnel displays the port; an unnamed differing-port tunnel displays `local -> remote`. Connected, Connecting, and Reconnecting tunnels cannot edit forwarding fields. Rename remains available while running. Delete is disabled while running and requires confirmation when enabled.

Quick Forward initially shows only Remote Port and Connect. Advanced is collapsed and contains Local Port, Remote Host, and Local Address. Port collision automatically expands Advanced and focuses Local Port. Validation messages are shown inline and no SSH process starts for invalid input.

Edit, rename, and log presentation use focused sheets or auxiliary windows appropriate to macOS. The interface uses the English labels and messages defined by the source specification.

### 9.2 Menu bar and settings

`MenuBarExtra` is conditionally present according to `showMenuBar`. It contains only saved tunnels, grouped by host, with search over host, tunnel name, and ports. It offers Connect or Disconnect plus Open TunnelDock, Refresh SSH Config, Settings, and Quit. It excludes temporary tunnels, Quick Forward, edit, delete, save, and advanced settings.

Settings v1 contains only General -> Show in Menu Bar. Turning it off removes the status item immediately while the Dock application remains available.

### 9.3 Application lifecycle

Closing the main window does not terminate the application or change runtime state. The AppKit delegate handles Dock reopen by bringing forward or recreating the main window. Application termination returns a delayed response while asynchronous tunnel shutdown completes, then replies to AppKit.

## 10. Testing Strategy

Implementation follows red-green-refactor. Core behavior is compiled and run through SwiftPM executable test targets before production implementation. The installed standalone Command Line Tools expose neither a compatible XCTest nor Swift Testing module, and Xcode must not be installed for this project, so a small in-repository assertion runner supplies filtering and failure exit codes without third-party dependencies.

### 10.1 Unit tests

- Include resolution: tilde, absolute and relative paths; bytewise glob order; nested includes; active-stack cycles; repeated noncyclic inclusion; missing files; watched glob directories.
- Alias scanning: ordinary hosts, multiple aliases, `Host *`, `*`, `?`, character classes, negative patterns, duplicate aliases, and textual order.
- Effective configuration: `ssh -G` output parsing, per-alias failure, and host search across all fields.
- Validation and forwarding: port boundaries, decimal-only ports, empty/control-character rejection, local-port follow behavior, IPv4, hostname, and bracketed IPv6 output.
- Persistence: schema v1 round trips, ISO 8601 dates, atomic replacement, missing file, malformed JSON preservation, unsupported schema, and serialized mutations.
- Port availability: an available loopback port, OS-owned collisions, and TunnelDock-owned endpoint collisions.
- Logs: timestamped append and eviction down to the newest 500 lines.
- Error classification: every required category and unknown fallback.

### 10.2 State-machine tests

Tests use fake process, socket, port, and clock dependencies to verify:

- exact Stage 1 and Stage 2 argument arrays;
- one master per tunnel and no shell invocation;
- first failure transitions to Failed without retry;
- established failure follows `1, 2, 5, 10, 30, 30` retry delays;
- success resets retry attempts;
- Disconnect cancels connect/retry work and prevents reconnection;
- stale generation callbacks cannot change current state;
- remote-service refusal text does not fail an established tunnel;
- host removal preserves a live connection but prevents retry after exit;
- host recovery restores availability without implicit connection;
- temporary save preserves the active runtime;
- quit performs graceful request, SIGTERM, SIGKILL, and cleanup in order.

### 10.3 Build and packaging verification

- `swift test` runs all core tests.
- `swift build` compiles the SwiftUI executable in debug configuration.
- `swift build -c release` compiles the release executable.
- `Scripts/package-app.sh` assembles the app bundle.
- Automated checks verify the app directory, executable permission, bundle identifier, minimum macOS version, regular-app activation policy, and absence of sandbox entitlements.

Real remote-server behavior such as ProxyJump, unknown host keys, authentication prompts, network loss, remote-service absence, and multiple simultaneous tunnels requires the release manual acceptance checklist because the development environment has no designated SSH test servers. The checklist uses user-controlled disposable hosts and never changes host-key policy.

## 11. Acceptance Mapping

The source specification's minimum scenarios map to the following implementation boundaries:

- Host discovery and refresh: SSH configuration pipeline and watcher tests.
- Effective OpenSSH inheritance: `/usr/bin/ssh -G` integration plus manual config fixtures.
- Basic, custom-port, remote-host, and IPv6 forwards: validation/formatting tests and manual tunnel checks.
- Port collision: checker tests plus state-machine proof that SSH is not launched.
- Saved, renamed, edited, deleted, and reconnected tunnels: repository and manager tests.
- Multiple tunnels and config-defined forwarding suppression: exact process-argument tests and manual process inspection.
- Authentication and host-key failures: classifier tests plus manual noninteractive connections.
- First failure and established failure: deterministic state-machine tests.
- Missing and restored hosts: AppState join and manager tests.
- Quit cleanup: manager ordering tests plus manual orphan-process inspection.

The v1 release is complete only when automated verification passes and the manual acceptance checklist has recorded results for environment-dependent scenarios. Lack of access to a suitable remote host must be reported as unverified rather than treated as a pass.

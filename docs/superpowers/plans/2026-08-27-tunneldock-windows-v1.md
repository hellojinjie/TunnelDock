# TunnelDock Windows v1.0 Implementation Plan

> **For agentic workers:** Execute this plan task-by-task. If Superpowers skills are available, use `superpowers:using-git-worktrees` before implementation, then `superpowers:subagent-driven-development` or `superpowers:executing-plans`, use TDD for core logic, and run `superpowers:verification-before-completion` before claiming completion.
>
> Do not redesign the product unnecessarily. The existing macOS implementation is the behavioral reference implementation. Preserve the existing macOS code and build.

**Goal:** Add a native, lightweight Windows version of TunnelDock using Go + `github.com/tailscale/walk` + Windows OpenSSH while preserving the existing TunnelDock product model and behavior as closely as Windows permits.

**Architecture:** Keep Windows implementation isolated under `Windows/` as an independent Go module. Port the existing domain model and SSH-config behavior to Go, but implement Windows tunnel lifetime using one independent `ssh.exe` process per tunnel instead of OpenSSH ControlMaster. Walk must remain a thin UI layer over testable application/core packages.

**Tech Stack:**

- Go 1.27
- `github.com/tailscale/walk`
- `golang.org/x/sys/windows`
- Windows built-in OpenSSH client
- Win32 Job Objects
- Win32 filesystem notifications where practical
- Standard Go `testing`
- No CGO
- No WebView
- No Electron/Wails/Fyne
- No shell invocation

**Primary spec:**

`docs/TunnelDock v1.0 Product & Technical Specification.md`

**Behavioral reference implementation:**

`Sources/TunnelDockCore/`

`Sources/TunnelDockAppSupport/`

`Sources/TunnelDock/`

**Plan location:**

Save a copy of this plan to:

`docs/superpowers/plans/2026-08-27-tunneldock-windows-v1.md`

---

# Global Constraints

1. Do not modify existing macOS behavior unless a shared documentation change is explicitly required.
2. Put all Windows code under `Windows/`.
3. Windows minimum target: Windows 10 1809 or newer; Windows 11 is the primary target.
4. Use Go 1.27.
5. Use `github.com/tailscale/walk`, not `github.com/lxn/walk`.
6. TunnelDock continues to use OpenSSH rather than implementing the SSH protocol itself.
7. Never invoke `cmd.exe`, PowerShell, `sh`, or another shell to construct SSH commands.
8. Invoke `ssh.exe` through `os/exec` with an explicit argument array.
9. Never store passwords, private-key contents, passphrases, known-host decisions, process logs, or runtime PIDs.
10. Force `BatchMode=yes` and `StrictHostKeyChecking=yes`.
11. Default local bind address is always `127.0.0.1`.
12. A user must explicitly choose a non-loopback bind address.
13. Saved tunnel JSON must remain schema version 1 and be compatible in structure with the macOS implementation.
14. Runtime state must never be persisted.
15. Every tunnel owns one independent `ssh.exe` process.
16. Windows implementation must not use OpenSSH ControlMaster, ControlPath, or ControlPersist.
17. All TunnelDock-created SSH processes must belong to a Windows Job Object configured with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`.
18. Closing the main window must not unexpectedly terminate active tunnels.
19. Explicit Quit must terminate all TunnelDock-created SSH processes.
20. Keep dependencies minimal.
21. Core packages must not import Walk.
22. UI-thread synchronization must happen only at the Walk/application boundary.
23. Run `go test ./...` and `go vet ./...` before every milestone is considered complete.
24. Use frequent, logically scoped commits.

---

# Target File Structure

Create:

```text
Windows/
├── go.mod
├── go.sum
├── cmd/
│   └── tunneldock/
│       └── main.go
│
├── internal/
│   ├── model/
│   │   ├── ssh_host.go
│   │   ├── tunnel_definition.go
│   │   ├── tunnel_runtime.go
│   │   └── tunnel_definition_test.go
│   │
│   ├── sshconfig/
│   │   ├── lexer.go
│   │   ├── scanner.go
│   │   ├── include_resolver.go
│   │   ├── host_resolver.go
│   │   ├── sanitizer.go
│   │   ├── watcher_windows.go
│   │   ├── scanner_test.go
│   │   ├── include_resolver_test.go
│   │   ├── host_resolver_test.go
│   │   └── sanitizer_test.go
│   │
│   ├── sshclient/
│   │   ├── locator_windows.go
│   │   ├── command.go
│   │   ├── process_windows.go
│   │   ├── job_windows.go
│   │   ├── error_classifier.go
│   │   ├── command_test.go
│   │   └── error_classifier_test.go
│   │
│   ├── tunnel/
│   │   ├── forward_spec.go
│   │   ├── port_checker.go
│   │   ├── log_buffer.go
│   │   ├── manager.go
│   │   ├── forward_spec_test.go
│   │   ├── port_checker_test.go
│   │   ├── log_buffer_test.go
│   │   └── manager_test.go
│   │
│   ├── persistence/
│   │   ├── tunnel_repository.go
│   │   ├── settings_store.go
│   │   ├── tunnel_repository_test.go
│   │   └── testdata/
│   │       └── macos-saved-tunnels.json
│   │
│   ├── app/
│   │   ├── model.go
│   │   ├── quick_forward.go
│   │   ├── host_filter.go
│   │   └── host_filter_test.go
│   │
│   ├── ui/
│   │   ├── main_window.go
│   │   ├── host_sidebar.go
│   │   ├── host_detail.go
│   │   ├── quick_forward.go
│   │   ├── tunnel_list.go
│   │   ├── tunnel_editor.go
│   │   ├── tunnel_log.go
│   │   ├── tray.go
│   │   └── settings.go
│   │
│   └── winapp/
│       ├── single_instance_windows.go
│       └── paths_windows.go
│
├── assets/
│   ├── TunnelDock.ico
│   └── winres/
│
├── scripts/
│   ├── build.ps1
│   └── test.ps1
│
└── README.md
```

Do not create large catch-all files. Keep domain logic independent from Walk.

---

# Task 1 — Establish Windows module and baseline

**Files:**

Create:

`Windows/go.mod`

`Windows/cmd/tunneldock/main.go`

`Windows/scripts/test.ps1`

`Windows/README.md`

## Steps

- [ ] Create an isolated worktree/branch named `windows-v1`.
- [ ] Verify current macOS tree is clean before adding Windows files.
- [ ] Create `Windows/go.mod`.

Use:

```text
module github.com/hellojinjie/TunnelDock/Windows

go 1.27
```

- [ ] Add dependencies:

```powershell
cd Windows
go get github.com/tailscale/walk
go get golang.org/x/sys/windows
```

- [ ] Create a minimal `main.go` which opens a Walk window titled `TunnelDock`.
- [ ] Do not implement product UI yet.
- [ ] Add `scripts/test.ps1`:

```powershell
$ErrorActionPreference = "Stop"

go test ./...
go vet ./...
```

- [ ] Verify:

```powershell
cd Windows
.\scripts\test.ps1
go build ./cmd/tunneldock
```

Expected: successful build and tests.

- [ ] Commit:

```text
build(windows): initialize Go and Walk application
```

---

# Task 2 — Port cross-platform data model and JSON compatibility

Use the existing Swift definitions as the source of truth.

The Windows `TunnelDefinition` must contain:

```go
type TunnelProtocol string

const (
    TunnelProtocolHTTP  TunnelProtocol = "http"
    TunnelProtocolHTTPS TunnelProtocol = "https"
)

type TunnelDefinition struct {
    ID              string         `json:"id"`
    HostAlias       string         `json:"hostAlias"`
    Name            *string        `json:"name,omitempty"`
    RemoteHost      string         `json:"remoteHost"`
    RemotePort      uint16         `json:"remotePort"`
    LocalAddress    string         `json:"localAddress"`
    LocalPort       uint16         `json:"localPort"`
    WebProtocol     TunnelProtocol `json:"webProtocol"`
    CreatedAt       time.Time      `json:"createdAt"`
    UpdatedAt       time.Time      `json:"updatedAt"`
    LastConnectedAt *time.Time     `json:"lastConnectedAt,omitempty"`
}
```

Runtime states:

```go
type TunnelState int

const (
    StateDisconnected TunnelState = iota
    StateConnecting
    StateConnected
    StateReconnecting
    StateFailed
)
```

Runtime data must be separate from `TunnelDefinition`.

- [ ] Write tests first for display-name behavior.
- [ ] Test unnamed `8888 -> 8888` displays `8888`.
- [ ] Test unnamed `18888 -> 8888` displays `18888 → 8888`.
- [ ] Test explicit name overrides generated display name.
- [ ] Implement input validation:
  - port range `1..65535`
  - required alias/host/address
  - reject NUL/newline/control characters.
- [ ] Implement UUIDv4 generation using `crypto/rand`; do not add a UUID dependency just for generation.
- [ ] Preserve canonical hyphenated UUID text.
- [ ] Commit:

```text
feat(windows): add TunnelDock domain models
```

---

# Task 3 — Implement schema-compatible persistence

**Windows storage path:**

```text
%LOCALAPPDATA%\TunnelDock\saved-tunnels.json
```

Envelope:

```go
type TunnelEnvelope struct {
    SchemaVersion int                `json:"schemaVersion"`
    Tunnels       []TunnelDefinition `json:"tunnels"`
}
```

Requirements:

- schema version exactly `1`;
- atomic replacement;
- malformed files must not be silently overwritten;
- malformed/unsupported files place repository in write-locked state until corrected/reloaded;
- runtime state is never written;
- ISO-8601/RFC3339 timestamps;
- pretty JSON is preferred.

- [ ] Add a golden JSON fixture based on an actual macOS `saved-tunnels.json` structure.
- [ ] Write tests proving Go can read it.
- [ ] Write a round-trip test proving fields survive save/reload.
- [ ] Write unsupported-schema test.
- [ ] Write malformed-file write-lock test.
- [ ] Implement atomic write as:
  1. create temp file in same directory;
  2. write;
  3. `Sync`;
  4. close;
  5. rename over destination.
- [ ] Do not write PID/state/logs.
- [ ] Settings go to:

```text
%LOCALAPPDATA%\TunnelDock\settings.json
```

Initial setting:

```json
{
  "showTrayIcon": true
}
```

- [ ] Commit:

```text
feat(windows): add tunnel and settings persistence
```

---

# Task 4 — Port SSH config discovery and Include semantics

The existing macOS SSH-config behavior is authoritative.

Default source:

```text
%USERPROFILE%\.ssh\config
```

Support:

- explicit Host aliases;
- multiple aliases on a `Host` line;
- ignore wildcard aliases;
- ignore negated aliases;
- recursive Include;
- glob expansion;
- relative Include;
- absolute Include;
- `~`;
- cycle prevention;
- first explicit alias determines display ordering.

Do not reimplement full OpenSSH option inheritance.

- [ ] Port the lexer behavior from `SSHConfigLexer.swift`.
- [ ] Port scanner behavior.
- [ ] Port Include resolver.
- [ ] Represent expanded configuration as ordered source lines with original source path metadata.

Use a structure similar to:

```go
type ExpandedLine struct {
    SourcePath string
    LineNumber int
    Text       string
}
```

- [ ] Tests must cover:
  - normal Host;
  - multiple aliases;
  - `Host *`;
  - `Host gpu-*`;
  - `Host !foo`;
  - nested Include;
  - glob Include;
  - recursive Include cycle;
  - duplicate explicit alias;
  - missing config.

- [ ] Commit:

```text
feat(windows): add OpenSSH config discovery
```

---

# Task 5 — Resolve effective Host configuration with Windows OpenSSH

Locate SSH in this order:

```text
%WINDIR%\System32\OpenSSH\ssh.exe
PATH lookup using exec.LookPath("ssh.exe")
```

If unavailable, report a clear application-level error:

```text
OpenSSH Client is not installed.
Install the Windows OpenSSH Client feature and restart TunnelDock.
```

For each explicit alias use:

```text
ssh.exe -G <alias>
```

Parse at least:

```text
hostname
user
port
```

Create:

```go
type SSHHost struct {
    Alias       string
    Hostname    string
    User        string
    Port        uint16
    ConfigOrder int
    Availability HostAvailability
}
```

Availability must distinguish at least:

```text
available
configuration-error
missing
```

- [ ] Unit-test parser with captured `ssh -G` output.
- [ ] Keep command execution behind an interface so tests never require real SSH.
- [ ] Search behavior later must match alias, hostname, user and port.
- [ ] Commit:

```text
feat(windows): resolve effective OpenSSH host configuration
```

---

# Task 6 — Implement the Windows SSH config sanitizer

This is a critical architecture task.

Windows OpenSSH does not provide usable Client ControlMaster support.

Do **not** attempt:

```text
-M
-S
-O forward
-O check
-O exit
```

Do **not** use:

```text
ClearAllForwardings=yes
```

on the same direct SSH process as TunnelDock's `-L`, because that also clears the command-line forwarding.

Instead, create an ephemeral flattened runtime SSH config.

## Sanitizer algorithm

Input:

```go
[]ExpandedLine
```

Output:

```text
%LOCALAPPDATA%\TunnelDock\runtime\<runtime-id>\ssh_config
```

Process every expanded line in original textual order.

Remove directives whose keyword, case-insensitively, is:

```text
Include
LocalForward
RemoteForward
DynamicForward
ClearAllForwardings
ControlMaster
ControlPath
ControlPersist
```

`Include` lines disappear because their contents have already been expanded inline.

Preserve all other lines verbatim.

This allows existing:

```text
Host
Match
HostName
User
Port
IdentityFile
CertificateFile
ProxyJump
ProxyCommand
IdentityAgent
PreferredAuthentications
HostKeyAlgorithms
KexAlgorithms
MACs
Ciphers
UserKnownHostsFile
```

and other OpenSSH configuration behavior to remain controlled by OpenSSH rather than TunnelDock.

- [ ] Write sanitizer tests using nested Include fixtures.
- [ ] Verify forwarded directives are absent.
- [ ] Verify ordinary directives remain byte-for-byte.
- [ ] Verify comments and ordering remain.
- [ ] Verify Match blocks remain structurally intact.
- [ ] Add an integration test, when `ssh.exe` exists:

```text
ssh.exe -F <sanitized-config> -G <alias>
```

Compare at minimum the original and sanitized effective values for:

```text
hostname
user
port
proxyjump
identityfile
```

where they are defined.

- [ ] Ensure runtime config directories are deleted after tunnel termination.
- [ ] Delete stale TunnelDock runtime directories during app startup.
- [ ] Runtime config must never contain password/passphrase/private-key contents; it only contains SSH configuration text already supplied by the user.
- [ ] Commit:

```text
feat(windows): sanitize SSH config for isolated forwarding
```

---

# Task 7 — Build SSH command generation

Create a pure/testable command builder.

Expected conceptual invocation:

```text
ssh.exe
-F <runtime-config>
-N
-T
-n
-L <forward-spec>
-o ExitOnForwardFailure=yes
-o ServerAliveInterval=15
-o ServerAliveCountMax=3
-o BatchMode=yes
-o StrictHostKeyChecking=yes
<hostAlias>
```

Do not use `ClearAllForwardings`.

Do not use a shell.

Implement IPv4/IPv6 formatting.

Examples:

```text
127.0.0.1:8888:127.0.0.1:8888
127.0.0.1:18888:192.168.10.50:8888
[::1]:8888:[2001:db8::10]:8888
```

- [ ] Write forward-spec tests first.
- [ ] Test hostname.
- [ ] Test IPv4.
- [ ] Test IPv6 local.
- [ ] Test IPv6 remote.
- [ ] Test both IPv6.
- [ ] Write exact command-array tests.
- [ ] Verify argument values are separate `exec.Command` arguments.
- [ ] Commit:

```text
feat(windows): build isolated SSH forwarding commands
```

---

# Task 8 — Implement port availability checking

Before starting SSH, test:

```text
LocalAddress + LocalPort
```

Requirements:

- distinguish a collision with another TunnelDock runtime from external system occupancy;
- never auto-increment;
- never choose a random port;
- SSH's own bind result remains authoritative.

Provide:

```go
type PortStatus int

const (
    PortAvailable PortStatus = iota
    PortUsedByTunnelDock
    PortUsedExternally
)
```

- [ ] Use `net.Listen` for preflight bind testing.
- [ ] Close the temporary listener immediately after the check.
- [ ] Maintain an in-memory reservation map in `TunnelManager` to prevent two TunnelDock connects racing for the same local endpoint.
- [ ] Normalize equivalent loopback endpoint keys where appropriate.
- [ ] Unit-test collision logic.
- [ ] Commit:

```text
feat(windows): add local forwarding port preflight
```

---

# Task 9 — Implement Windows Job Object process ownership

Create one Job Object owned by the application.

Configure:

```text
JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
```

Every `ssh.exe` started by TunnelDock must be assigned to this Job immediately after process creation.

Use `golang.org/x/sys/windows`.

Required API behavior:

```go
type Job struct { ... }

func NewJob() (*Job, error)
func (j *Job) Assign(process *os.Process) error
func (j *Job) Close() error
```

- [ ] Unit-test pure configuration helpers where possible.
- [ ] Add Windows integration test spawning a harmless child process into a Job and confirming it terminates after Job close.
- [ ] Never invoke `taskkill.exe`.
- [ ] Never rely on process-name matching.
- [ ] Commit:

```text
feat(windows): own SSH processes with Win32 job object
```

---

# Task 10 — Implement SSH process controller

Create:

```go
type ProcessHandle struct {
    Cmd       *exec.Cmd
    RuntimeID string
}
```

The controller must:

- create runtime config;
- construct `ssh.exe` arguments;
- redirect stdin away from UI;
- capture stdout/stderr asynchronously;
- hide console windows;
- assign process to Job Object;
- expose process termination;
- delete runtime config when complete.

On Windows set process attributes so no console window flashes.

Do not expose `exec.Cmd` directly to UI code.

Connected readiness:

A tunnel becomes `Connected` only when:

1. SSH process is still alive;
2. local listener is observable;
3. startup has not produced an SSH failure.

Poll readiness approximately every 100 ms with a bounded initial startup timeout.

Do not probe the remote application.

- [ ] Create fake launcher interfaces for manager tests.
- [ ] Test stdout/stderr forwarding.
- [ ] Test early exit.
- [ ] Test cancellation.
- [ ] Test cleanup.
- [ ] Commit:

```text
feat(windows): manage dedicated OpenSSH tunnel processes
```

---

# Task 11 — Port SSH error classification

Map common errors to application categories without making stderr parsing responsible for the state machine.

At minimum support:

```text
Local port is already in use
Authentication failed
Host verification required
Host not found
SSH configuration error
Connection timed out
Could not resolve host
Connection refused by SSH server
SSH process exited unexpectedly
OpenSSH not installed
```

Preserve raw stderr in runtime log.

Do not display only:

```text
Error 255
```

- [ ] Port representative classifier tests from the Swift implementation.
- [ ] Add Windows-specific OpenSSH stderr fixtures.
- [ ] Keep unknown fallback:

```text
SSH connection failed.
See Log for details.
```

- [ ] Commit:

```text
feat(windows): classify OpenSSH connection failures
```

---

# Task 12 — Implement runtime log buffer

Each runtime gets an in-memory ring buffer of exactly 500 lines.

Log examples:

```text
[11:30:02] Connecting...
[11:30:03] SSH transport established.
[11:30:03] Connected.
[11:45:20] SSH process exited: 255
[11:45:20] Reconnecting in 1 second...
```

Requirements:

- stdout;
- stderr;
- lifecycle events;
- reconnect attempts;
- exit status;
- no disk logging.

- [ ] Test 500-line capacity.
- [ ] Test oldest-line eviction.
- [ ] Test concurrency.
- [ ] Commit:

```text
feat(windows): add bounded in-memory tunnel logs
```

---

# Task 13 — Port TunnelManager state machine

`TunnelManager` is the sole owner of tunnel lifecycle.

States:

```text
Disconnected
Connecting
Connected
Reconnecting
Failed
```

Rules:

Initial connect:

```text
Disconnected
→ Connecting
→ Connected
```

Initial failure:

```text
Connecting
→ Failed
```

No automatic reconnect after a tunnel that has never reached `Connected`.

After a previously connected process dies unexpectedly:

```text
Connected
→ Reconnecting
```

Retry delays:

```text
1s
2s
5s
10s
30s
30s
...
```

After successful reconnect:

```text
retryAttempt = 0
```

User Disconnect:

```text
desiredConnection = false
cancel retry
terminate SSH process
cleanup runtime config
release endpoint reservation
state = Disconnected
```

Missing Host:

- saved tunnel remains;
- running tunnel remains alive;
- no new connect while missing;
- if running tunnel dies while Host missing, do not reconnect;
- when Host returns, reconnect eligibility returns.

Remote-service connection errors such as:

```text
connect failed: Connection refused
```

must **not** mark an established SSH tunnel as failed.

- [ ] Port state-machine tests before implementation.
- [ ] Use fake clock.
- [ ] Use fake SSH process controller.
- [ ] Test double-connect suppression.
- [ ] Test disconnect during reconnect delay.
- [ ] Test host removal and restoration.
- [ ] Test temporary tunnel behavior.
- [ ] Test save conversion.
- [ ] Test multiple independent tunnels.
- [ ] Commit:

```text
feat(windows): port tunnel lifecycle state machine
```

---

# Task 14 — Implement app model, filtering and Quick Forward behavior

Walk should bind to a testable application model.

Host filtering:

case-insensitive substring search over:

```text
Alias
Hostname
User
Port
```

Quick Forward defaults:

```text
RemoteHost  = 127.0.0.1
RemotePort  = user input
LocalAddress = 127.0.0.1
LocalPort    = RemotePort
WebProtocol  = http
```

Local Port follows Remote Port until user manually edits Local Port.

After manual edit it stops auto-following.

On port conflict:

- do not launch SSH;
- expand Advanced;
- focus Local Port;
- preserve user's entered values.

- [ ] Unit-test filter ordering.
- [ ] Unit-test Local Port follow/detach behavior.
- [ ] Unit-test reset behavior when switching host if macOS behavior indicates reset.
- [ ] Commit:

```text
feat(windows): add application and quick-forward model
```

---

# Task 15 — Implement SSH config watching

Watch:

```text
%USERPROFILE%\.ssh\config
Include files
directories containing Include globs
```

Use Windows filesystem notification APIs through `x/sys/windows`; avoid adding a separate watcher dependency unless the native approach proves unmaintainable.

Debounce reloads by approximately:

```text
300 ms
```

Requirements:

- config save does not cause repeated visible reload churn;
- re-resolve Include graph after each reload;
- watcher subscriptions update if Include graph changes;
- manual Refresh remains available.

- [ ] Unit-test debounce logic separately from Win32 event source.
- [ ] Integration-test a temporary watched directory on Windows.
- [ ] Commit:

```text
feat(windows): watch OpenSSH configuration changes
```

---

# Task 16 — Build the main Walk window shell

Implement native Win32 UI using `tailscale/walk`.

Do not recreate macOS visual styling pixel-for-pixel.

Use normal Windows controls.

Conceptual layout:

```text
┌──────────────────────┬──────────────────────────────────────┐
│ Search Hosts...      │ gpu-server                           │
│                      │ ubuntu @ 192.168.1.20:22             │
│ SSH Hosts            │                                      │
│  gpu-server          │ Saved Tunnels                        │
│  nas                 │                                      │
│  dev                 │ Jupyter       ● Connected            │
│                      │ 8888 → 8888                           │
│ Missing Hosts        │                                      │
│                      │ Quick Forward                        │
│                      │ Remote Port [8888]       [Connect]    │
└──────────────────────┴──────────────────────────────────────┘
```

Requirements:

- resizable;
- sensible minimum size;
- sidebar width stable;
- keyboard navigation;
- Per-Monitor-V2 DPI awareness;
- no custom-rendered UI unless required;
- no embedded HTML/WebView.

- [ ] Implement sidebar.
- [ ] Implement search.
- [ ] Implement host-detail header.
- [ ] Connect UI to app model.
- [ ] Keep runtime operations off UI thread.
- [ ] Use Walk synchronization primitives when updating widgets.
- [ ] Commit:

```text
feat(windows): add native TunnelDock main window
```

---

# Task 17 — Implement Quick Forward and Advanced UI

Default visible UI:

```text
Remote Port [      ] [Connect]
```

Advanced collapsed by default.

Advanced contains:

```text
Local Port
Remote Host
Local Address
HTTP/HTTPS browser protocol
```

Requirements:

- Remote Port primary input;
- Local Port follows until manually changed;
- validation shown near action;
- Connect disabled for unavailable/config-error Host;
- conflict auto-expands Advanced and focuses Local Port.

- [ ] Connect UI action to `TunnelManager`.
- [ ] Temporary tunnel appears only after successful connection.
- [ ] Commit:

```text
feat(windows): add quick forwarding workflow
```

---

# Task 18 — Implement Saved/Temporary Tunnel controls

Temporary tunnel actions:

```text
Disconnect
Save
View Log
Open in Browser
```

Saved tunnel actions:

```text
Connect / Disconnect
Rename
Edit
Delete
View Log
Open in Browser
```

Rules:

- connected/connecting/reconnecting forward parameters cannot be edited;
- rename is allowed while running;
- running tunnel cannot be deleted;
- delete requires confirmation;
- temporary tunnel disappears after Disconnect;
- Save converts temporary runtime to a persisted definition;
- successful connect updates `lastConnectedAt`.

Browser URL:

```text
http://127.0.0.1:<localPort>
https://127.0.0.1:<localPort>
```

Use Windows shell-open API; do not invoke `cmd /c start`.

- [ ] Commit:

```text
feat(windows): add tunnel management controls
```

---

# Task 19 — Implement tunnel log window

Create native Walk log window/dialog.

Requirements:

- read-only;
- independently scrollable;
- live updates;
- raw SSH stderr visible;
- no disk writes;
- closing log window does not affect tunnel.

- [ ] Commit:

```text
feat(windows): add tunnel log viewer
```

---

# Task 20 — Implement tray behavior and settings

Use Walk's notification icon support.

Default:

```text
Show tray icon = ON
```

Tray contains only saved tunnel quick controls plus:

```text
TunnelDock
Open TunnelDock
Refresh SSH Config
Settings
Quit TunnelDock
```

Temporary tunnels do not appear in tray tunnel list.

Saved tunnel tray actions:

```text
Connected    → Disconnect
Disconnected → Connect
```

Windows-specific close behavior:

- tray ON: closing main window hides it; app/tunnels continue;
- tray OFF: closing main window must keep a visible taskbar re-entry point, e.g. minimize rather than leave the running application completely inaccessible;
- explicit Quit terminates tunnels and application.

- [ ] Apply setting immediately.
- [ ] Persist setting.
- [ ] Commit:

```text
feat(windows): add tray tunnel controls and settings
```

---

# Task 21 — Add single-instance behavior

TunnelDock must not accidentally create two independent managers controlling the same ports.

Use a named Windows mutex such as:

```text
Local\TunnelDock.Windows.Singleton
```

On second launch:

1. detect existing instance;
2. find existing TunnelDock main window;
3. restore/show it;
4. bring it to foreground when Windows permits;
5. exit second process.

Do not terminate the existing instance.

- [ ] Test mutex behavior.
- [ ] Add a manual smoke test for hidden/minimized main window.
- [ ] Commit:

```text
feat(windows): enforce single TunnelDock instance
```

---

# Task 22 — Implement clean application shutdown

Explicit Quit sequence:

```text
disable new reconnects
→ cancel reconnect timers
→ terminate each SSH process
→ wait for process completion
→ clean runtime config
→ close Job Object
→ exit UI
```

Job Object close is the final orphan-process safety net.

Window close is not equivalent to Quit.

- [ ] Test manager-wide shutdown.
- [ ] Integration-test that no spawned child survives process cleanup.
- [ ] Commit:

```text
feat(windows): implement deterministic application shutdown
```

---

# Task 23 — Windows resources and portable packaging

Generate a Windows `.ico` from the existing TunnelDock icon assets.

Embed:

- application icon;
- version information;
- Common Controls v6 manifest;
- PerMonitorV2 DPI awareness.

Use a Go-based resource tool rather than requiring Visual Studio or a MinGW development environment.

Build must result in:

```text
Windows\dist\TunnelDock.exe
```

Build command should use approximately:

```powershell
go build `
  -trimpath `
  -ldflags="-H=windowsgui -s -w" `
  -o dist\TunnelDock.exe `
  ./cmd/tunneldock
```

Requirements:

- no console window;
- no external runtime beside normal Windows system components and OpenSSH Client;
- no installer required for v1;
- portable `.exe` distribution is sufficient.

Create:

```text
Windows/scripts/build.ps1
```

Build script must:

1. run tests;
2. run vet;
3. generate/update resources;
4. build GUI executable;
5. fail on any command error.

- [ ] Confirm executable starts on a clean Windows machine with Go not installed.
- [ ] Commit:

```text
build(windows): package portable TunnelDock executable
```

---

# Task 24 — Add CI

Create a Windows GitHub Actions workflow.

Required steps:

```text
checkout
setup-go 1.27
go mod download
go test ./...
go vet ./...
build TunnelDock.exe
```

Run on:

```text
push affecting Windows/**
pull_request affecting Windows/**
```

Optionally upload the built `.exe` as workflow artifact.

Do not modify macOS CI behavior.

- [ ] Commit:

```text
ci(windows): test and build TunnelDock on Windows
```

---

# Task 25 — Manual acceptance matrix

Before declaring Windows v1 complete, manually verify all of the following.

## Host discovery

```text
normal Host
multiple aliases
Host *
Host wildcard
Include
nested Include
Include glob
config auto-refresh
manual refresh
missing config
configuration error
```

## Effective config

Verify `Host *`, patterns, User, HostName, Port and ProxyJump are reflected through `ssh -G`.

## Sanitized runtime config

Create a test SSH config containing:

```text
LocalForward
RemoteForward
DynamicForward
```

Verify TunnelDock's runtime `ssh.exe` does **not** create those forwardings and creates only the TunnelDock forwarding.

Also verify:

```text
IdentityFile
ProxyJump
ProxyCommand
Match
```

remain functional.

## Forwarding

Verify:

```text
remote 8888 → local 8888
remote 8888 → local 18888
custom remote host
IPv4
IPv6
```

## Port collision

Verify:

```text
occupied local port
```

does not launch a successful tunnel and does not silently choose another port.

## Authentication

Verify password-required connection fails without a TunnelDock password dialog.

## Host key

Verify unknown host key fails and is never automatically accepted.

## Lifecycle

Verify:

```text
initial failure → Failed, no auto-retry
successful connection then network failure → Reconnecting
1/2/5/10/30 second backoff
manual Disconnect stops reconnect
reconnect resets retry count
```

## Missing Host

Verify:

```text
saved tunnel preserved
active connection remains active
no reconnect while missing
host restoration re-associates tunnel
```

## Tunnel operations

Verify:

```text
Save
Rename
Edit while disconnected
reject edit while running
Delete while disconnected
reject delete while running
View Log
Open in Browser
```

## Multi-tunnel

Run at least three independent tunnels to the same Host and verify each has a distinct `ssh.exe` process.

## Tray/window

Verify:

```text
close main window
restore from tray
tray setting ON/OFF
Quit
```

## Process cleanup

After explicit Quit, verify no TunnelDock-created `ssh.exe` survives.

Also kill TunnelDock forcibly during a test and verify the Job Object prevents orphan SSH processes.

---

# Task 26 — Documentation and final verification

Update:

```text
README.md
README.en.md
```

only enough to point to the Windows implementation.

Create:

```text
Windows/README.md
```

Document:

```text
Windows requirements
OpenSSH Client requirement
build commands
test commands
portable executable location
data location
SSH config location
security model
Windows architecture differences
```

Explicitly document this Windows difference:

```text
macOS:
independent OpenSSH ControlMaster per Tunnel

Windows:
independent ssh.exe process per Tunnel
using a temporary forwarding-sanitized OpenSSH config
```

Do not claim feature parity until acceptance tests pass.

Run final verification:

```powershell
cd Windows

go version
go mod tidy
go test ./...
go test -race ./...
go vet ./...
.\scripts\build.ps1
```

If `-race` is unsupported by the chosen Windows build configuration, document that fact and run all non-race tests; do not silently skip it.

Then inspect:

```text
git status
git diff --check
```

There must be:

- no generated runtime SSH config committed;
- no `.exe` accidentally committed unless distribution policy explicitly requires it;
- no user SSH config;
- no credentials;
- no test private keys;
- no logs;
- no unrelated macOS changes.

Finally perform manual acceptance on Windows 11.

Commit:

```text
docs(windows): document Windows TunnelDock implementation
```

---

# Definition of Done

Windows TunnelDock v1 is complete only when all of these are true:

- `Windows/` is independently buildable with Go 1.27.
- Existing macOS build remains unaffected.
- UI is native Walk/Win32 with no WebView.
- `%USERPROFILE%\.ssh\config` Hosts and Include files are discovered.
- Effective HostName/User/Port come from Windows OpenSSH.
- User-configured LocalForward/RemoteForward/DynamicForward are suppressed for TunnelDock runtime connections.
- TunnelDock's own `-L` forwarding works.
- Each tunnel uses an independent `ssh.exe`.
- No ControlMaster is used.
- No shell is used.
- Host key verification remains strict.
- Interactive password/passphrase handling is not implemented.
- Temporary and saved tunnels behave according to the macOS product model.
- Automatic reconnect follows `1s, 2s, 5s, 10s, 30s...`.
- Runtime logs remain memory-only and capped at 500 lines.
- Saved tunnel JSON remains schema version 1.
- Tray behavior works.
- Explicit Quit leaves no TunnelDock-created SSH process.
- Portable `TunnelDock.exe` builds without Visual Studio.
- `go test ./...` passes.
- `go vet ./...` passes.
- CI passes.
- Manual acceptance matrix passes.

---

# Execution Rule for Codex

Do not implement all tasks in one giant change.

Execute tasks sequentially.

For each task:

1. inspect the corresponding existing Swift implementation before coding;
2. write or port tests first for core logic;
3. implement the minimum code needed;
4. run targeted tests;
5. run `go test ./...`;
6. run `go vet ./...`;
7. review the diff;
8. commit the completed task;
9. continue to the next task.

If behavior is ambiguous, prefer the existing Swift/macOS implementation and existing product specification over inventing new behavior.

If Windows makes exact parity impossible, isolate the difference behind the narrowest Windows-specific interface and document the difference. Do not weaken security behavior merely to simplify implementation.

The most important architectural invariants are:

```text
Walk UI
   ↓
App Model
   ↓
TunnelManager
   ↓
SSHProcessController
   ↓
one ssh.exe per tunnel
   ↓
Windows Job Object
```

and:

```text
User SSH Config
   ↓
recursive Include expansion
   ↓
remove forwarding/control directives
   ↓
temporary runtime ssh_config
   ↓
ssh.exe -F runtime-config -L TunnelDockForward alias
```

Preserve these boundaries throughout implementation.
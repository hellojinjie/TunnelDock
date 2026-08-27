# TunnelDock for Windows

TunnelDock's Windows client is a native Go application built with
[`github.com/tailscale/walk`](https://github.com/tailscale/walk). It is kept in
an independent Go module. This document describes the Windows client only.

TunnelDock uses one dedicated `ssh.exe` process per tunnel. It writes a
temporary flattened OpenSSH configuration for each runtime after removing
forwarding and ControlMaster directives, so TunnelDock owns only its own local
forwarding rules.

## Requirements

- Windows 10 version 1809 or newer (Windows 11 recommended)
- Go 1.27
- Windows OpenSSH Client

Install OpenSSH Client in **Settings → Optional features** if `ssh.exe` is not
available. TunnelDock reports this at startup rather than silently falling back
to another transport.

## Test

```powershell
cd Windows
.\scripts\test.ps1
```

`go test -race ./...` additionally requires a Windows C compiler. The current
portable Go-only toolchain has CGO disabled and no `gcc`, so the normal test
suite is run here; enable CGO with a supported compiler on a release-validation
machine before relying on race detection.

## Build

```powershell
cd Windows
go build ./cmd/tunneldock
```

Or create the portable executable (including tests and vet):

```powershell
cd Windows
.\scripts\build.ps1
```

The output is `Windows\dist\TunnelDock.exe`. No installer or non-Windows
runtime is required.

## Data and SSH configuration

- Saved tunnels: `%LOCALAPPDATA%\TunnelDock\saved-tunnels.json`
- Settings: `%LOCALAPPDATA%\TunnelDock\settings.json`
- Temporary runtime configs: `%LOCALAPPDATA%\TunnelDock\runtime\`
- SSH source configuration: `%USERPROFILE%\.ssh\config` and its `Include`
  graph

Runtime configs and process output are not persisted. Tunnel logs are kept in
memory only. Local ports are preflight-checked and never auto-incremented.

## Windows architecture

Windows does not use shared OpenSSH ControlMaster sessions. Each tunnel starts
an independent, Job-Object-owned `ssh.exe` with an isolated sanitized config.
Closing the application terminates managed processes; the Job Object is the
final orphan-process safety net.

## Security model and limitations

- TunnelDock invokes `ssh.exe` directly; it never starts a shell.
- It does not store passwords, passphrases, private keys, `known_hosts`, or
  runtime logs. Unknown or changed host keys are not accepted automatically.
- Interactive password and passphrase prompts are intentionally unsupported.
- The Windows client does not use OpenSSH ControlMaster. Each tunnel is a
  separate `ssh.exe` process with its own temporary, forwarding-sanitized
  configuration.

## Acceptance

Run the Windows 11 checklist in
[docs/manual-acceptance.md](docs/manual-acceptance.md) against a disposable SSH
environment before claiming production readiness.

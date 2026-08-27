# TunnelDock for Windows

TunnelDock's Windows client is a native Go application built with
[`github.com/tailscale/walk`](https://github.com/tailscale/walk). It is kept in
an independent Go module so the existing macOS application remains unchanged.

This directory currently contains the Windows application baseline. Product
features and packaging are implemented incrementally by the Windows v1 plan in
`docs/superpowers/plans/2026-08-27-tunneldock-windows-v1.md`.

## Requirements

- Windows 10 version 1809 or newer (Windows 11 recommended)
- Go 1.27
- Windows OpenSSH Client

## Test

```powershell
cd Windows
.\scripts\test.ps1
```

## Build

```powershell
cd Windows
go build ./cmd/tunneldock
```

# TunnelDock

> Native macOS management for OpenSSH local port forwards.

[简体中文](README.md) | English

TunnelDock discovers the connectable hosts in your existing `~/.ssh/config` and turns a remote port into a local SSH tunnel with a few clicks. It is a graphical companion to the system OpenSSH client—not a replacement for SSH.

![TunnelDock application icon](Resources/TunnelDockIconLight-v2.png)

## Features

- **SSH configuration discovery**: Reads `~/.ssh/config` and recursively follows `Include` directives, including relative paths and glob patterns. Only explicit, connectable host aliases are listed.
- **Quick Forward**: Select a host, enter a remote port, and connect. The default creates `127.0.0.1:<port>` → `<host>:127.0.0.1:<port>`.
- **Recent Tunnels**: Successfully connected Quick Forwards are saved automatically. Rename, edit, reconnect, or remove them after they are disconnected.
- **Independent connections**: Each tunnel uses its own OpenSSH control socket and lifecycle, so multiple forwards can run independently.
- **Menu bar and Dock**: The app remains a regular Dock app and can optionally expose saved tunnels from the menu bar.
- **Resilience**: An established connection that drops retries with a capped backoff; the interface exposes connection state and per-tunnel logs.
- **Open in Browser**: Connected tunnels can open their local URL with an HTTP or HTTPS scheme.

## Requirements

- macOS 13 Ventura or later
- Swift Command Line Tools with Swift Package Manager
- A working system OpenSSH client at `/usr/bin/ssh`

Xcode is not required. This project intentionally uses SwiftPM only and does not include an `.xcodeproj`.

## Quick start

From the repository root, build a signed local `.app` bundle:

```sh
sh Scripts/package-app.sh
```

The resulting universal app is located at:

```text
.build/release/TunnelDock.app
```

Launch it from Finder or run:

```sh
open .build/release/TunnelDock.app
```

### Create a tunnel

1. Define a host in `~/.ssh/config`, for example:

   ```ssh-config
   Host gpu-server
       HostName gpu.example.com
       User alice
       Port 22
   ```

2. Open TunnelDock and select **gpu-server** from the sidebar.
3. Under **Quick Forward**, enter the remote port (for example, `8888`) and click **Connect**.
4. Use `http://127.0.0.1:8888` locally, or select **Open in Browser** after the tunnel is connected.

Use **Advanced** to choose a different local port, remote host, local listening address, or the HTTP/HTTPS scheme used by **Open in Browser**. These settings do not change SSH forwarding semantics.

## Security and privacy

- TunnelDock launches `/usr/bin/ssh` through `Process`; it never invokes a shell and does not implement the SSH protocol itself.
- It does not store SSH passwords, passphrases, private keys, `known_hosts`, process IDs, control sockets, or runtime logs.
- SSH runs in batch mode and keeps strict host-key verification enabled. Unknown or changed host keys are not accepted automatically.
- Only saved tunnel definitions are persisted at `~/Library/Application Support/TunnelDock/saved-tunnels.json`.

## Development

Run the dependency-free executable test suites:

```sh
sh Scripts/test.sh
```

Run the packaging checks after building the app:

```sh
sh Scripts/package-app.sh
sh Tests/Packaging/package-app-tests.sh
```

The scripts select the Command Line Tools SDK when available and keep SwiftPM and Clang build caches inside `.build/`, so no global Xcode installation or cache setup is needed.

## Project layout

```text
Sources/TunnelDockCore/       SSH configuration, persistence, tunnel lifecycle
Sources/TunnelDockAppSupport/ UI-facing application state and helpers
Sources/TunnelDock/           SwiftUI and AppKit application interface
Resources/                    Application metadata and icon assets
Scripts/                      Test, icon-generation, and app-packaging scripts
Tests/                        Dependency-free executable and packaging tests
docs/                         Product specification and manual acceptance checklist
```

## Limitations

- TunnelDock targets local port forwarding; it does not replace the full OpenSSH command-line feature set.
- v1 reads the user SSH configuration and its `Include` files only; it does not offer alternate SSH configuration profiles or a custom `ssh -F` source.
- The app is distributed outside the Mac App Store and does not include login-item support or system notifications.
- Use a disposable SSH environment for manual acceptance testing; do not use production hosts or credentials for test scenarios.

## Documentation

- [Product and technical specification (Chinese)](docs/TunnelDock%20v1.0%20Product%20%26%20Technical%20Specification.md)
- [Manual acceptance checklist (Chinese)](docs/manual-acceptance-checklist.md)

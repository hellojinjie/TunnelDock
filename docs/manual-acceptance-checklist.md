# TunnelDock v1.0 Manual Acceptance Checklist

This checklist covers behavior that requires a real macOS UI session, disposable SSH configuration, or a user-controlled SSH server. For every item, record exactly one result: `Pass`, `Fail — <observation>`, or `Not run — no test environment`.

## Test environment

- macOS version:
- TunnelDock build/version:
- Disposable SSH host aliases:
- Tester/date:
- Do not use production hosts or credentials.
- Do not allow TunnelDock or the test procedure to edit `known_hosts` automatically.

## Host discovery and refresh

- [ ] Add one ordinary `Host` block; confirm the alias appears once and effective host/user/port are shown.
- [ ] Add multiple aliases to one `Host` line; confirm each concrete alias appears in declaration order.
- [ ] Add `Host *`; confirm `*` is not displayed as a selectable host.
- [ ] Add `Host gpu-*`; confirm the wildcard pattern is not displayed as a selectable host.
- [ ] Add an `Include` file with a concrete alias; confirm the alias appears.
- [ ] Add a nested `Include`; confirm its concrete aliases appear without duplicates.
- [ ] Modify the root config and an included config while TunnelDock is running; confirm automatic refresh updates the sidebar.
- [ ] Use **Refresh SSH Config**; confirm changes are applied immediately and the UI remains responsive.
- [ ] Save a tunnel, remove its host alias, refresh, and confirm it moves to **Missing Hosts** and cannot connect.
- [ ] Restore the alias and refresh; confirm it returns to the normal host list without connecting automatically.

## Effective SSH configuration

- [ ] Put `User`, `Port`, or `Hostname` in `Host *`; compare TunnelDock with `/usr/bin/ssh -G <alias>`.
- [ ] Combine wildcard and concrete blocks; confirm first-value-wins inheritance matches `/usr/bin/ssh -G <alias>`.
- [ ] Add `LocalForward`, `RemoteForward`, and `DynamicForward` to the alias; connect through TunnelDock and confirm none of those config forwards are opened (TunnelDock passes forwarding suppression options).

## Forwarding

- [ ] Create a basic loopback forward where local and remote ports match; confirm traffic reaches the remote service.
- [ ] Create a forward with a different local port; confirm only that local port listens.
- [ ] Set a non-default remote host; confirm traffic reaches that host from the SSH server.
- [ ] Use a valid IPv6 remote host; confirm bracket/colon handling produces a working forward.
- [ ] Occupy the chosen local port in another process; confirm TunnelDock rejects the connection and focuses **Local Port** in Advanced.
- [ ] Confirm the default local address is always `127.0.0.1` for a new Quick Forward.
- [ ] Explicitly choose a non-loopback local address; confirm it is used only after that explicit edit.
- [ ] Start two independent tunnels simultaneously; confirm they use distinct SSH master processes/control sockets and operate independently.
- [ ] Stop the remote service while SSH remains connected; confirm TunnelDock still reports the tunnel as Connected and connection attempts fail at the service layer.

## Saved tunnel lifecycle

- [ ] Connect a Quick Forward, save it, restart TunnelDock, and confirm the definition persists but runtime state does not.
- [ ] Rename a disconnected saved tunnel; confirm the new name persists.
- [ ] Edit all forwarding fields while disconnected; reconnect and confirm the new values are used.
- [ ] Confirm edit and delete are disabled while connecting, connected, or reconnecting.
- [ ] Delete a disconnected saved tunnel; restart and confirm it remains deleted.
- [ ] Disconnect and reconnect a saved tunnel; confirm logs and state transition correctly.

## Authentication, host keys, failure, and retry

- [ ] Connect to a host that requires an interactive password/passphrase; confirm TunnelDock fails clearly and stores no credential.
- [ ] Connect to a host with an unknown host key; confirm SSH verification is not bypassed and TunnelDock does not edit `known_hosts`.
- [ ] Change a disposable server's host key; confirm SSH rejects it and TunnelDock surfaces a host-key error.
- [ ] Cause the first connection attempt to fail; confirm it enters Failed without automatic retry.
- [ ] Break an established SSH connection; observe retry delays of approximately 1, 2, 5, 10, then 30 seconds capped at 30.
- [ ] Restore connectivity during retry; confirm it reconnects and the next future retry sequence resets to 1 second.

## Window, menu bar, and quit

- [ ] Close the main window; confirm the app and active tunnels continue running.
- [ ] Click the Dock icon after closing the window; confirm the main window reopens and becomes key.
- [ ] With **Show in Menu Bar** enabled, confirm the menu lists only saved tunnels, grouped by host.
- [ ] Search the menu by host, saved name, local port, and remote port; confirm matching rows remain.
- [ ] Connect and disconnect a saved tunnel from the menu; confirm main-window state follows.
- [ ] Disable **Show in Menu Bar** in Settings; confirm the item disappears. Re-enable it from Settings and confirm it returns.
- [ ] Choose Quit with active tunnels; wait for the app to exit, then run `pgrep -afil 'ssh.*TunnelDock|ssh.*ControlPath'` and confirm no TunnelDock-owned SSH child remains.

## Release bundle

- [ ] Run `sh Scripts/package-app.sh` and `sh Tests/Packaging/package-app-tests.sh` from the repository root.
- [ ] Launch `.build/release/TunnelDock.app`; confirm it is a regular foreground Dock application with a main window.
- [ ] Confirm no Xcode project, Xcode installation, sandbox entitlement, privileged helper, or login item is required.

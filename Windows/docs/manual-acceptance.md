# TunnelDock Windows v1 manual acceptance

Run this on Windows 11 with a disposable SSH server and test account. Do not
use production hosts, credentials, or private keys. Mark each row Pass, Fail,
or Not run and retain the SSH test configuration with the test record.

| Area | Checks | Status |
| --- | --- | --- |
| Host discovery | normal `Host`; multiple aliases; `Host *`; wildcard; `Include`; nested Include; Include glob; missing config; config syntax error | Not run |
| Refresh | edit the root config and an included file; confirm one debounced UI refresh; add/remove an Include target; use **Refresh SSH Config** | Not run |
| Effective values | verify `Host *`, patterns, `User`, `HostName`, `Port`, and `ProxyJump` through the displayed `ssh -G` result | Not run |
| Sanitized config | source config has `LocalForward`, `RemoteForward`, and `DynamicForward`; verify only TunnelDock's `-L` exists while `IdentityFile`, `ProxyJump`, `ProxyCommand`, and `Match` still work | Not run |
| Forwarding | remote `8888` to local `8888`; remote `8888` to local `18888`; custom remote host; IPv4 and IPv6 local addresses | Not run |
| Port collision | occupy the requested local port; Connect must fail, preserve values, expand Advanced, focus Local Port, and never choose another port | Not run |
| Authentication | password-required SSH fails without a TunnelDock password dialog | Not run |
| Host key | an unknown host key fails and is never accepted automatically | Not run |
| Lifecycle | first connection failure is Failed without retry; established tunnel loss retries at 1/2/5/10/30 seconds; Disconnect stops retry; success resets delay | Not run |
| Missing host | saved definition remains; active forwarding stays alive; no reconnect while missing; returning Host restores eligibility | Not run |
| Tunnel operations | Save; Rename while connected; edit only while disconnected; reject edit/delete while running; confirmation before delete; View Log; Open in Browser | Not run |
| Multi-tunnel | start three saved tunnels to one Host; confirm three distinct owned `ssh.exe` processes | Not run |
| Tray and window | close with tray on hides window and leaves tunnels alive; restore from tray; turn tray off then close/minimize and restore from taskbar; explicit Quit stops application | Not run |
| Process cleanup | after Quit, no TunnelDock `ssh.exe` survives; force-kill TunnelDock and confirm Job Object terminates its children | Not run |
| Portable package | copy `dist\\TunnelDock.exe` to a clean Windows machine without Go; with OpenSSH Client installed, start it and repeat smoke checks | Not run |

Automated tests cover parsing, persistence, state transitions, process cleanup,
and the Job Object child-process guarantee. They do not replace this matrix:
the network, Windows shell, tray, and real OpenSSH scenarios above require a
manual run in the stated environment.

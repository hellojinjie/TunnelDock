# TunnelDock

> Native macOS management for OpenSSH local port forwards.
> 原生 macOS OpenSSH 本地端口转发管理器。

TunnelDock discovers the connectable hosts in your existing `~/.ssh/config` and turns a remote port into a local SSH tunnel with a few clicks. It is a graphical companion to the system OpenSSH client—not a replacement for SSH.

TunnelDock 会从现有的 `~/.ssh/config` 发现可连接的 Host，让你通过图形界面快速建立本地 SSH 隧道。它是系统 OpenSSH 的图形化管理工具，并不自行实现 SSH 协议。

![TunnelDock application icon](Resources/TunnelDockIconLight-v2.png)

## Features / 功能

- **SSH config discovery / SSH 配置发现** — Reads `~/.ssh/config` and recursively follows `Include` directives, including relative paths and glob patterns. Only explicit, connectable host aliases are listed.
- **Quick Forward / 快速转发** — Select a host, enter a remote port, and connect. The default creates `127.0.0.1:<port>` → `<host>:127.0.0.1:<port>`.
- **Recent Tunnels / 最近隧道** — Successfully connected Quick Forwards are saved automatically. Rename, edit, reconnect, or remove them after they are disconnected.
- **Independent connections / 独立连接** — Each tunnel uses its own OpenSSH control socket and lifecycle, so multiple forwards can run independently.
- **Menu bar and Dock / 菜单栏与 Dock** — The app remains a regular Dock app and can optionally expose saved tunnels from the menu bar.
- **Resilience / 连接恢复** — An established connection that drops retries with a capped backoff; the interface exposes connection state and per-tunnel logs.
- **Open in Browser / 在浏览器打开** — Connected tunnels can open their local URL with an HTTP or HTTPS scheme.

## Requirements / 环境要求

- macOS 13 Ventura or later / macOS 13 Ventura 或更高版本
- Swift Command Line Tools with Swift Package Manager / 含 SwiftPM 的 Command Line Tools
- A working system OpenSSH client at `/usr/bin/ssh` / 系统 `/usr/bin/ssh` 可用

Xcode is not required. This project intentionally uses SwiftPM only and does not include an `.xcodeproj`.

无需安装 Xcode。项目只使用 SwiftPM，且不提供 `.xcodeproj`。

## Quick start / 快速开始

From the repository root, build a signed local `.app` bundle:

在仓库根目录执行以下命令，构建本地签名的 `.app`：

```sh
sh Scripts/package-app.sh
```

The resulting universal app is located at:

构建出的通用二进制应用位于：

```text
.build/release/TunnelDock.app
```

Launch it from Finder or run:

可从 Finder 打开，或执行：

```sh
open .build/release/TunnelDock.app
```

### Create a tunnel / 创建隧道

1. Define a host in `~/.ssh/config`, for example:

   在 `~/.ssh/config` 中定义 Host，例如：

   ```ssh-config
   Host gpu-server
       HostName gpu.example.com
       User alice
       Port 22
   ```

2. Open TunnelDock and select **gpu-server** from the sidebar.
   打开 TunnelDock，然后在侧栏中选择 **gpu-server**。
3. Under **Quick Forward**, enter the remote port (for example, `8888`) and click **Connect**.
   在 **Quick Forward** 中输入远端端口（例如 `8888`），然后点击 **Connect**。
4. Use `http://127.0.0.1:8888` locally, or select **Open in Browser** after the tunnel is connected.
   隧道连通后，访问本机 `http://127.0.0.1:8888`，或点击 **Open in Browser**。

Use **Advanced** to choose a different local port, remote host, local listening address, or the HTTP/HTTPS scheme used by **Open in Browser**. These settings do not change SSH forwarding semantics.

可通过 **Advanced** 修改本地端口、远端主机、本地监听地址，以及 **Open in Browser** 使用的 HTTP/HTTPS 协议；这些选项不会改变 SSH 转发本身的语义。

## Security and privacy / 安全与隐私

- TunnelDock launches `/usr/bin/ssh` through `Process`; it never invokes a shell and does not implement the SSH protocol itself.
- It does not store SSH passwords, passphrases, private keys, `known_hosts`, process IDs, control sockets, or runtime logs.
- SSH runs in batch mode and keeps strict host-key verification enabled. Unknown or changed host keys are not accepted automatically.
- Only saved tunnel definitions are persisted at `~/Library/Application Support/TunnelDock/saved-tunnels.json`.

---

- TunnelDock 通过 `Process` 调用 `/usr/bin/ssh`，不经由 shell，也不会自行实现 SSH 协议。
- 不会保存 SSH 密码、私钥口令、私钥、`known_hosts`、进程 ID、控制套接字或运行日志。
- SSH 使用 batch 模式并保持严格的主机密钥校验；未知或变更的主机密钥不会被自动接受。
- 仅已保存的隧道定义会写入 `~/Library/Application Support/TunnelDock/saved-tunnels.json`。

## Development / 开发

Run the dependency-free executable test suites:

运行无第三方依赖的可执行测试套件：

```sh
sh Scripts/test.sh
```

Run the packaging checks after building the app:

构建应用后，运行打包检查：

```sh
sh Scripts/package-app.sh
sh Tests/Packaging/package-app-tests.sh
```

The scripts select the Command Line Tools SDK when available and keep SwiftPM and Clang build caches inside `.build/`, so no global Xcode installation or cache setup is needed.

脚本会优先选择 Command Line Tools SDK，并把 SwiftPM 与 Clang 的构建缓存保留在 `.build/` 中，因此不需要全局 Xcode 安装或额外的缓存配置。

## Project layout / 项目结构

```text
Sources/TunnelDockCore/       SSH configuration, persistence, tunnel lifecycle
Sources/TunnelDockAppSupport/ UI-facing application state and helpers
Sources/TunnelDock/           SwiftUI and AppKit application interface
Resources/                    Application metadata and icon assets
Scripts/                      Test, icon-generation, and app-packaging scripts
Tests/                        Dependency-free executable and packaging tests
docs/                         Product specification and manual acceptance checklist
```

## Limitations / 当前限制

- TunnelDock targets local port forwarding; it does not replace the full OpenSSH command-line feature set.
- v1 reads the user SSH config and its `Include` files only; it does not offer alternate SSH config profiles or a custom `ssh -F` source.
- The app is distributed outside the Mac App Store and does not include login-item support or system notifications.
- Use a disposable SSH environment for manual acceptance testing; do not use production hosts or credentials for test scenarios.

TunnelDock 聚焦于本地端口转发，并非完整 OpenSSH 命令行功能的替代品。v1 只读取用户 SSH 配置及其 `Include` 文件，不支持自定义 `ssh -F` 配置来源、多配置档案、登录项或系统通知。进行手动验收时请使用可丢弃的 SSH 测试环境，避免使用生产主机或凭据。

## Documentation / 文档

- [Product and technical specification](docs/TunnelDock%20v1.0%20Product%20%26%20Technical%20Specification.md) / 产品与技术规格
- [Manual acceptance checklist](docs/manual-acceptance-checklist.md) / 手动验收清单

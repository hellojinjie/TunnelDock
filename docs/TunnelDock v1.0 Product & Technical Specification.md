# TunnelDock v1.0 Product & Technical Specification

## 1. 产品定义

**TunnelDock** 是一个使用 Swift 编写的原生 macOS SSH 本地端口转发管理工具。

核心目标是：

> 从用户现有的 `~/.ssh/config` 自动发现 SSH Host，让用户只输入一个远端端口，就能快速建立本地 SSH Tunnel。

典型操作：

```text
gpu-server
Remote Port: 8888
Connect
```

默认建立：

```text
127.0.0.1:8888
        ↓
SSH: gpu-server
        ↓
127.0.0.1:8888
```

用户不需要手写：

```bash
ssh -N -L 127.0.0.1:8888:127.0.0.1:8888 gpu-server
```

TunnelDock 不替代 OpenSSH，而是作为系统 `/usr/bin/ssh` 的图形化 Tunnel 管理器。

---

# 2. 平台与技术约束

* 开发语言：Swift
* UI：SwiftUI
* 最低系统：macOS 13 Ventura
* App 类型：普通 macOS GUI App
* 不进入 Mac App Store
* 不启用 App Sandbox
* Dock 图标始终显示
* 支持菜单栏模式
* 第一版不提供 Launch at Login
* 第一版不发送 macOS 系统通知
* 第一版尽量不引入第三方依赖

主要系统能力：

```text
SwiftUI
Foundation
Process
Network
FileManager
FSEvents / 文件监听机制
```

SSH 必须调用：

```text
/usr/bin/ssh
```

不得自行实现 SSH 协议。

---

# 3. 应用形态

TunnelDock 同时具有：

1. 标准 macOS 主窗口
2. macOS 菜单栏入口

Dock 图标始终存在。

默认：

```text
Show in Menu Bar = ON
```

Settings 中提供：

```text
Show in Menu Bar    [ON/OFF]
```

关闭后：

* 菜单栏 TunnelDock 图标立即消失；
* Dock 图标仍然存在；
* 用户仍可通过 Dock、Launchpad、Spotlight 打开 TunnelDock。

关闭主窗口：

```text
≠ Quit
```

即：

* Window Close 只关闭/隐藏窗口；
* TunnelDock 继续运行；
* 已连接 Tunnel 继续运行。

只有明确执行：

```text
Quit TunnelDock
```

才真正退出程序。

---

# 4. SSH Config 数据源

v1 只读取：

```text
~/.ssh/config
```

以及其中通过：

```ssh
Include ...
```

引用的配置。

不支持：

```text
自定义 -F config
额外 SSH config source
多个独立 Config Profile
```

---

# 5. Include 处理

TunnelDock 必须递归处理 `Include`。

例如：

```ssh
Include ~/.ssh/config.d/*
```

Included 文件按照 OpenSSH 对 glob 的顺序展开，并插入到 Include 出现的位置。

需要：

* 支持 `~`
* 支持绝对路径
* 支持相对用户 SSH Config 的路径
* 支持 glob
* 支持递归 Include
* 防止 Include 循环导致无限递归

Host 的显示顺序必须尽可能保持：

> OpenSSH Config 实际文本展开后的出现顺序。

---

# 6. Host 发现规则

只展示**明确可连接的 Host alias**。

例如：

```ssh
Host gpu
```

显示。

```ssh
Host gpu gpu-server lab-gpu
```

显示为三个独立 Host：

```text
gpu
gpu-server
lab-gpu
```

以下不作为 Host 显示：

```ssh
Host *
Host gpu-*
Host lab-?
```

带 wildcard 的 pattern 只作为 SSH 配置匹配规则使用。

否定 pattern：

```text
!xxx
```

不作为 Host。

如果同一个明确 alias 在多个位置重复出现：

> Sidebar 中只显示一次，以第一次明确出现的位置作为排序位置。

---

# 7. SSH 有效配置解析

TunnelDock 不自己重新实现 OpenSSH 的：

```text
Host *
Host pattern
Match
User
Hostname
Port
ProxyJump
IdentityFile
Canonicalization
```

等复杂配置继承语义。

对于每个发现的明确 Host alias，调用：

```bash
/usr/bin/ssh -G <alias>
```

获取 OpenSSH 最终生效配置。

至少解析：

```text
hostname
user
port
```

例如：

```text
gpu-01
researcher @ 10.0.0.21:22
```

UI 显示的是最终有效配置，而不是仅显示某个 Host block 中直接出现的字段。

---

# 8. Host 搜索

主窗口提供搜索。

搜索至少匹配：

```text
Host alias
HostName
User
SSH Port
```

使用：

* case-insensitive
* substring matching

例如搜索：

```text
root
```

匹配：

```text
User root
```

搜索：

```text
192.168.1
```

匹配 HostName。

搜索只过滤列表：

> 不改变 SSH Config 原本顺序。

---

# 9. SSH Config 刷新

支持两种方式同时存在：

## 自动刷新

监听：

```text
~/.ssh/config
Include 文件
Include glob 相关目录
```

发生变化后重新解析。

文件事件需要 debounce，避免保存文件时连续触发大量 reload。

推荐：

```text
约 300 ms debounce
```

## 手动刷新

UI 始终提供：

```text
Refresh SSH Config
```

---

# 10. 主窗口结构

使用 macOS 标准两栏布局：

```text
┌──────────────────────┬─────────────────────────────────────┐
│ Search Hosts…        │ gpu-server                          │
│                      │ ubuntu @ 192.168.1.20:22             │
│ gpu-server           │                                     │
│ nas                  │ Saved Tunnels                       │
│ dev                  │                                     │
│                      │ Jupyter      ● Connected             │
│                      │ 8888 → 8888                          │
│                      │                                     │
│                      │ TensorBoard  ○ Disconnected          │
│                      │ 6006 → 6006                          │
│                      │                                     │
│                      │ Quick Forward                       │
│                      │ Remote Port [ 8888 ]       [Connect] │
└──────────────────────┴─────────────────────────────────────┘
```

SwiftUI 推荐使用：

```text
NavigationSplitView
```

---

# 11. Sidebar

Sidebar 显示：

```text
Search
SSH Hosts
Missing Hosts
```

正常 Host：

> 按 SSH Config 原始顺序排列。

如果某 Host 已从 SSH Config 删除，但存在 Saved Tunnel，则放入：

```text
Missing Hosts
```

区域。

Missing Host 不参与正常 Config 排序。

当同名 Host 重新出现时：

* 自动离开 Missing Hosts；
* 回归正常 SSH Config 位置；
* Saved Tunnel 自动重新关联。

---

# 12. Host Detail

选择某 Host 后，右侧显示：

```text
Alias
User
HostName
SSH Port
```

例如：

```text
gpu-server
ubuntu @ 192.168.1.20:22
```

随后显示：

```text
Saved Tunnels
Quick Forward
```

---

# 13. Quick Forward

Quick Forward 是 TunnelDock 的核心快捷功能。

默认界面只显示：

```text
Remote Port [      ] [Connect]
```

用户输入：

```text
8888
```

默认自动得到：

```text
Remote Host   = 127.0.0.1
Remote Port   = 8888

Local Address = 127.0.0.1
Local Port    = 8888
```

即：

```text
127.0.0.1:8888
    ↓
gpu-server
    ↓
127.0.0.1:8888
```

---

# 14. Advanced Forward Settings

Quick Forward 提供折叠的 Advanced 区域。

默认折叠。

展开后：

```text
Remote Port      8888

Local Port       8888
Remote Host      127.0.0.1
Local Address    127.0.0.1
```

其中：

```text
Remote Port
```

是主要输入。

Local Port 默认自动跟随 Remote Port。

例如 Remote Port：

```text
8888
```

Local Port 默认自动变成：

```text
8888
```

如果用户手动修改过 Local Port：

```text
18888
```

以后 Remote Port 的变化不应再覆盖用户手动设置。

---

# 15. 地址支持

Remote Host 支持：

```text
hostname
IPv4
IPv6
```

例如：

```text
127.0.0.1
192.168.10.50
db.internal
::1
2001:db8::10
```

Local Address 同样支持：

```text
hostname
IPv4
IPv6
```

默认：

```text
127.0.0.1
```

允许：

```text
0.0.0.0
::
指定本地接口地址
```

IPv6 在生成 OpenSSH Forwarding 参数时必须正确进行 `[]` 格式处理。

例如：

```text
[::1]:8888:[2001:db8::10]:8888
```

---

# 16. Local Port 冲突

启动 Tunnel 前必须主动检测：

```text
Local Address + Local Port
```

是否可绑定。

如果已经被 TunnelDock 自身另一个 Tunnel 使用：

```text
Local port is already in use.
```

如果系统其他程序占用：

```text
Local port 8888 is already in use.
Choose another local port.
```

随后：

* 不自动修改端口；
* 不自动寻找 8889；
* 不随机选择端口；
* 自动展开 Advanced；
* 将焦点放到 Local Port。

SSH 本身的绑定结果仍然是最终权威。

---

# 17. Saved Tunnel

Quick Forward 成功连接后，会作为 Temporary Tunnel 出现在当前 Host 下。

例如：

```text
● 8888 → 8888       Disconnect   Save
```

Disconnect：

> 临时 Tunnel 消失。

Save：

> 转换为 Saved Tunnel。

Saved Tunnel 字段：

```text
id
name
hostAlias

remoteHost
remotePort

localAddress
localPort

createdAt
updatedAt
```

---

# 18. Tunnel 名称

名称可选。

例如用户指定：

```text
Jupyter
TensorBoard
Web UI
```

如果没有名称：

### 本地端口等于远端端口

显示：

```text
8888
```

### 本地端口不等于远端端口

显示：

```text
18888 → 8888
```

保存时不得强迫用户填写名称。

之后可以 Rename。

---

# 19. Saved Tunnel 数量

同一 Host 可以保存和运行多个 Tunnel。

例如：

```text
gpu-server

Jupyter
8888 → 8888

TensorBoard
6006 → 6006

WebUI
17860 → 7860
```

每个 Tunnel：

> 拥有独立 SSH transport 生命周期。

不同 Tunnel 之间不共享长期 SSH Master。

---

# 20. Tunnel 数据存储

Saved Tunnel 存储在：

```text
~/Library/Application Support/TunnelDock/
```

推荐文件：

```text
saved-tunnels.json
```

结构：

```text
schemaVersion: 1
tunnels: [...]
```

使用 Swift：

```text
Codable
```

保存时：

* atomic write；
* 防止半写入导致文件损坏。

Runtime 状态不保存。

例如：

```text
Connected
Reconnecting
Failed
PID
Control Socket
Log
```

全部只存在内存。

---

# 21. UserDefaults

UserDefaults 只保存轻量设置，例如：

```text
showMenuBar = true
```

不得将全部 Saved Tunnel 数据存入 UserDefaults。

---

# 22. App 启动行为

启动 TunnelDock 后：

* 加载 Saved Tunnel；
* 加载 SSH Hosts；
* 所有 Tunnel 初始状态为：

```text
Disconnected
```

不得自动恢复上次运行状态。

不得因为上次 App 关闭时某 Tunnel 是 Connected 就自动连接。

---

# 23. SSH 认证原则

TunnelDock 不管理 SSH Credential。

不提供：

```text
SSH Password 输入框
Private Key Passphrase 输入框
密码存储
自己的 Keychain Credential
```

完全复用系统 OpenSSH 认证体系：

```text
~/.ssh/config
IdentityFile
ssh-agent
macOS Keychain
ProxyJump
ProxyCommand
known_hosts
```

连接必须能够以非交互方式成功。

TunnelDock 启动 SSH 时使用：

```text
BatchMode=yes
```

因此需要密码、passphrase 等人工交互时：

> 连接失败。

UI 提示用户先确保：

```bash
ssh <host>
```

可以在现有系统 SSH 环境中正常使用。

---

# 24. Host Key

TunnelDock 不自动接受未知 Host Key。

不得使用：

```text
StrictHostKeyChecking=no
StrictHostKeyChecking=accept-new
```

TunnelDock 强制：

```text
StrictHostKeyChecking=yes
```

未知 Host：

```text
Host verification failed.
```

UI 给出明确提示：

```text
This host has not been verified.

Connect to it once in Terminal:

ssh gpu-server
```

服务器 Host Key 改变同样必须失败，不得自动覆盖。

---

# 25. SSH Tunnel 固定参数

TunnelDock 强制使用：

```text
ClearAllForwardings = yes
ExitOnForwardFailure = yes

ServerAliveInterval = 15
ServerAliveCountMax = 3

BatchMode = yes
StrictHostKeyChecking = yes
```

并使用：

```text
-N
-T
-n
```

含义：

```text
-N   不执行远端 command
-T   不申请 pseudo-terminal
-n   stdin 不参与 SSH 交互
```

---

# 26. ClearAllForwardings 实现

不能使用简单的：

```bash
ssh \
  -o ClearAllForwardings=yes \
  -L ... \
  host
```

因为 ClearAllForwardings 会同时清除命令行 Forwarding。

v1 必须采用：

## Stage 1 — 建立独立 SSH Master

每个 Tunnel 创建独立 Control Socket。

概念命令：

```bash
/usr/bin/ssh \
  -M \
  -S <socket> \
  -N \
  -T \
  -n \
  -o ControlPersist=no \
  -o ClearAllForwardings=yes \
  -o ExitOnForwardFailure=yes \
  -o ServerAliveInterval=15 \
  -o ServerAliveCountMax=3 \
  -o BatchMode=yes \
  -o StrictHostKeyChecking=yes \
  <hostAlias>
```

这条 SSH transport：

* 使用用户原来的 SSH Config；
* 使用 HostName / User / IdentityFile / ProxyJump 等；
* 不应用 Config 中已有的 LocalForward；
* 不应用 Config 中已有的 RemoteForward；
* 不应用 Config 中已有的 DynamicForward。

## Stage 2 — 给该 Master 增加 TunnelDock Forward

Master 建立后，通过：

```text
ssh -O forward
```

只添加当前 TunnelDock Tunnel。

概念：

```bash
/usr/bin/ssh \
  -S <socket> \
  -O forward \
  -L <TunnelDockForwardSpec> \
  <control-target>
```

因此最终：

> 每个 Tunnel 一个独立 SSH Master + 一个 TunnelDock Local Forward。

不同 Saved Tunnel 不共享这个 Master。

---

# 27. Control Socket

Runtime Control Socket 不持久化。

必须使用较短路径，避免 Unix Domain Socket 路径长度问题。

推荐：

```text
/tmp/tunneldock-<uid>/
```

目录权限：

```text
0700
```

Socket 名：

```text
<short-runtime-id>.sock
```

App 正常退出时清理。

App 异常退出后：

> 下次启动清除自己的 stale socket。

---

# 28. Forwarding 参数格式

逻辑模型：

```text
LocalAddress:LocalPort
        ↓
SSH Host
        ↓
RemoteHost:RemotePort
```

例如：

```text
Local Address = 127.0.0.1
Local Port    = 18888

Remote Host   = 127.0.0.1
Remote Port   = 8888

SSH Host      = gpu-server
```

最终 Forward：

```text
127.0.0.1:18888:127.0.0.1:8888
```

Shell 不得参与构造命令。

必须：

```swift
Process.executableURL
Process.arguments
```

不得：

```text
/bin/sh -c "ssh ..."
```

避免参数注入和 quoting 问题。

---

# 29. 输入校验

Port：

```text
1...65535
```

只接受十进制整数。

以下不能为空：

```text
Host Alias
Remote Host
Local Address
Remote Port
Local Port
```

名称可以为空。

拒绝：

```text
换行符
NUL
非法控制字符
```

---

# 30. Tunnel 状态机

Runtime 状态：

```text
Disconnected
Connecting
Connected
Reconnecting
Failed
```

典型流程：

```text
Disconnected
     ↓ Connect
Connecting
     ↓ success
Connected
```

首次失败：

```text
Connecting
     ↓ failure
Failed
```

用户必须再次手动 Connect。

---

# 31. Connected 判定

`Connected` 不表示 Remote Service 一定正常。

只表示：

1. SSH master 仍然存活；
2. Forward 请求成功；
3. Local Address + Local Port 已实际进入监听状态。

因此：

```text
● Connected
```

意味着：

> SSH Tunnel established.

不意味着：

```text
RemoteHost:RemotePort application is healthy.
```

TunnelDock 不主动探测远端服务。

---

# 32. Remote Service Failure

例如 SSH Tunnel 已连接：

```text
localhost:8888
```

但远端：

```text
127.0.0.1:8888
```

没有程序监听。

TunnelDock 仍显示：

```text
Connected
```

OpenSSH stderr 中可能出现：

```text
connect failed: Connection refused
```

这属于：

> Remote service connection error

而不是：

> SSH Tunnel disconnected

不得因此把 Tunnel 状态改成 Failed。

---

# 33. 首次连接失败

只有曾经真正进入过：

```text
Connected
```

状态的 Tunnel 才具有自动重连资格。

首次 Connect 如果发生：

```text
Permission denied
Host key verification failed
DNS error
SSH configuration error
Host unreachable
```

直接：

```text
Failed
```

不得无限重试。

---

# 34. 自动重连

如果某 Tunnel 已成功进入过：

```text
Connected
```

之后 SSH master 意外退出，则：

```text
Connected
→ Reconnecting
```

自动无限重连。

重连延迟：

```text
1s
2s
5s
10s
30s
30s
30s
...
```

不设置最大次数。

只要：

```text
desiredConnection = true
```

就持续尝试。

一旦重新 Connected：

```text
retry counter = reset
```

---

# 35. 主动 Disconnect

用户点击 Disconnect 后：

```text
desiredConnection = false
```

必须：

* 停止 reconnect timer；
* 关闭 SSH master；
* 清理 Control Socket；
* 状态回到 Disconnected。

不得再次自动连接。

---

# 36. 网络失联判定

强制：

```text
ServerAliveInterval=15
ServerAliveCountMax=3
```

因此长期无响应的 SSH 会话应由 OpenSSH 自己退出。

TunnelDock 检测 Child Process termination 后：

```text
Connected
→ Reconnecting
```

而不是自己重新实现 SSH 心跳协议。

---

# 37. Host 被删除

如果：

```text
gpu-server
```

从 SSH Config 删除：

### 没有运行的 Tunnel

Saved Tunnel 保留。

Host 显示到：

```text
Missing Hosts
```

状态：

```text
Host not found
```

Connect disabled。

### 已运行 Tunnel

当前 SSH Tunnel：

> 不主动断开。

例如：

```text
⚠ Host not found
● Connected
```

现有 connection 继续运行。

但：

> Host 不存在期间禁止自动重连。

如果当前 connection 随后断开：

```text
Failed
Host not found
```

如果同名 Host 在断开之前重新出现在 SSH Config：

> 自动恢复正常重连资格。

---

# 38. Host 恢复

Missing Host 如果重新出现在 SSH Config：

```text
gpu-server
```

Saved Tunnel 自动重新关联。

无需用户重新保存。

Host 恢复：

```text
Host not found
→ available
```

---

# 39. 编辑 Saved Tunnel

Disconnected Saved Tunnel 可以编辑：

```text
Name
Remote Host
Remote Port
Local Address
Local Port
```

Connected / Connecting / Reconnecting Tunnel：

> 禁止编辑 Forwarding 参数。

运行期间可以 Rename。

如果要修改连接参数：

```text
Disconnect
→ Edit
→ Connect
```

不得自动隐式重启。

---

# 40. 删除 Saved Tunnel

运行中的 Tunnel：

> 禁止 Delete。

用户必须：

```text
Disconnect
→ Delete
```

Delete 不允许自动替用户 Disconnect。

删除操作应该有标准确认，防止误删 Saved Tunnel。

---

# 41. Temporary Tunnel

Quick Forward 成功后：

```text
Temporary Tunnel
```

显示在当前 Host 的 Tunnel 列表中。

例如：

```text
● 8888 → 8888
  Disconnect
  Save
```

如果 Disconnect：

> Runtime 对象删除。

如果 Save：

> 转成 Saved Tunnel。

---

# 42. Menu Bar

菜单栏只负责：

> Saved Tunnel 快速控制。

不提供：

```text
Quick Forward
Edit
Delete
Save
高级配置
```

建议布局：

```text
TunnelDock

Search…

gpu-server
  Jupyter        ●                 Disconnect
  TensorBoard    ○                 Connect

nas
  Web UI         ○                 Connect

──────────────────────────
Open TunnelDock
Refresh SSH Config
Settings
Quit TunnelDock
```

搜索可以匹配：

```text
Host
Tunnel Name
Port
```

Temporary Tunnel 不进入菜单栏 Saved Tunnel 列表。

---

# 43. Tunnel Log

每个 Runtime Tunnel 有独立内存日志。

入口：

```text
View Log
```

内容包括：

```text
TunnelDock lifecycle messages
ssh stdout
ssh stderr
reconnect attempts
process exit status
forward command result
```

例如：

```text
[11:30:02] Connecting...
[11:30:03] SSH transport established.
[11:30:03] Forward established.
[11:30:03] Connected.
[11:45:20] SSH process exited: 255
[11:45:20] Reconnecting in 1 second...
```

---

# 44. Log 保留策略

每 Tunnel：

```text
最多保留最后 500 行
```

使用 ring buffer。

日志：

```text
仅内存
不写磁盘
```

App Quit：

> 全部日志消失。

Temporary Tunnel 被删除：

> 日志消失。

Saved Tunnel 下一次新的手动 Connect：

> 开始新的 Runtime Log。

自动 reconnect：

> 属于同一次 Runtime 生命周期，因此继续写入同一个 Log。

---

# 45. 错误显示

UI 不能只显示：

```text
Error 255
```

应该转换成用户可以理解的类别，同时保留原始 stderr 在 View Log。

至少包括：

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
```

无法识别的错误：

```text
SSH connection failed.
See Log for details.
```

不得依赖完整解析 OpenSSH stderr 来决定核心状态机。

---

# 46. 主机配置错误

如果：

```bash
ssh -G alias
```

失败：

Sidebar 仍可显示已扫描到的 alias，但标记：

```text
Configuration Error
```

该 Host 禁止 Connect。

View / UI 应展示：

```text
SSH configuration could not be resolved.
```

Refresh 后重新评估。

---

# 47. SSH Config 不存在

如果：

```text
~/.ssh/config
```

不存在：

App 正常启动。

Sidebar 显示：

```text
No SSH hosts found.

TunnelDock reads hosts from:
~/.ssh/config
```

不得 crash。

---

# 48. Quit 行为

Quit TunnelDock 时：

> 自动关闭全部由 TunnelDock 创建的 SSH Tunnel。

流程：

```text
disable reconnect
→ request SSH master exit
→ terminate remaining child process
→ cleanup control sockets
→ quit app
```

不得留下 TunnelDock 创建的孤立 SSH Process。

如果 graceful shutdown 失败：

1. SIGTERM
2. 短暂 grace period
3. 必要时 SIGKILL

---

# 49. 主窗口关闭行为

点击窗口红色关闭按钮：

```text
不关闭 Tunnel
不退出 App
不停止 reconnect
```

如果：

```text
Show in Menu Bar = OFF
```

仍可通过 Dock 再次打开窗口。

---

# 50. 非目标功能

TunnelDock v1.0 明确不实现：

```text
Remote Forward (-R)
Dynamic Forward / SOCKS (-D)
SSH Terminal
SFTP
文件传输
SSH Password 管理
Private Key 管理
Host Key 自动接受
远程服务健康检查
App Store
App Sandbox
Launch at Login
macOS 系统通知
自定义 SSH Config 文件
自动恢复上次连接状态
Tunnel 历史日志
多个 Tunnel 共用 SSH Connection
```

这些均不是 v1 范围。

---

# 51. 推荐代码模块

建议拆分：

```text
TunnelDockApp
│
├── AppState
│
├── SSH
│   ├── SSHConfigScanner
│   ├── SSHIncludeResolver
│   ├── SSHHostResolver
│   └── SSHConfigWatcher
│
├── Tunnel
│   ├── TunnelDefinition
│   ├── TunnelRuntime
│   ├── TunnelManager
│   ├── SSHProcessController
│   ├── ControlSocketManager
│   ├── PortAvailabilityChecker
│   └── TunnelLogBuffer
│
├── Persistence
│   ├── TunnelRepository
│   └── SettingsStore
│
└── UI
    ├── MainWindow
    ├── HostSidebar
    ├── HostDetailView
    ├── SavedTunnelRow
    ├── QuickForwardView
    ├── TunnelEditor
    ├── TunnelLogView
    ├── MenuBarView
    └── SettingsView
```

各模块职责必须分离。

---

# 52. 核心数据模型

## SSHHost

```text
alias
hostname
user
port
configOrder
availability
```

SSHHost 是：

> SSH Config 派生数据。

不作为主要持久化业务数据。

## TunnelDefinition

```text
id: UUID

hostAlias: String
name: String?

remoteHost: String
remotePort: UInt16

localAddress: String
localPort: UInt16

createdAt
updatedAt
```

## TunnelRuntime

```text
definitionID / temporaryID

state
desiredConnection

sshProcess
controlSocket

hasEverConnected

retryAttempt
retryTask

lastError
logBuffer
```

Runtime 不持久化。

---

# 53. 状态与配置分离

必须严格区分：

```text
TunnelDefinition
```

与：

```text
TunnelRuntime
```

例如 Saved Tunnel：

```text
Jupyter
127.0.0.1:8888 → 127.0.0.1:8888
```

是 Definition。

而：

```text
Connected
PID 1234
retry 0
```

是 Runtime。

不能把 Runtime 状态写进 Saved Tunnel JSON。

---

# 54. 并发原则

TunnelManager 是 Tunnel 生命周期的唯一 owner。

建议：

```text
@MainActor
```

管理 UI-facing 状态。

Process I/O、配置解析和文件监听：

> 异步执行，不阻塞主线程。

同一 Tunnel 的 Connect / Disconnect / Reconnect 操作必须串行化，避免：

```text
Connect
Connect
Disconnect
Reconnect
```

产生竞争条件。

---

# 55. 安全要求

TunnelDock：

* 不读取 Private Key 内容；
* 不复制 SSH 密钥；
* 不存 SSH Password；
* 不存 Key Passphrase；
* 不绕过 Host Key 校验；
* 不使用 shell 拼接 SSH 命令；
* 不在磁盘保存 SSH stderr history；
* 不自动开放 `0.0.0.0`。

只有用户明确修改：

```text
Local Address
```

后，才允许监听非 localhost 地址。

默认必须始终：

```text
127.0.0.1
```

---

# 56. 第一版设置页

Settings v1 只需要很少设置。

至少：

```text
General

Show in Menu Bar      [ON]
```

不要为了“设置页看起来丰富”加入无实际必要的选项。

---

# 57. 最低验收场景

TunnelDock v1.0 发布前至少通过以下场景：

### Host discovery

```text
普通 Host
多个 Alias
Host *
Host gpu-*
Include
嵌套 Include
Config 修改自动刷新
手动 Refresh
```

### Effective configuration

确认：

```text
Host *
User ...
```

以及：

```text
Host gpu-*
```

等规则最终通过 `ssh -G` 正确反映到：

```text
HostName
User
Port
```

### Basic Forward

```text
Remote 8888
→ Local 8888
```

成功。

### Custom Local Port

```text
Remote 8888
Local 18888
```

成功。

### Remote Host

```text
Remote Host = 192.168.10.50
```

成功。

### IPv6

IPv6 Local Address / Remote Host 正确生成 OpenSSH Forward 语法。

### Port collision

本地端口被其他程序占用：

```text
不得启动 SSH
必须提示冲突
不得自动换端口
```

### Saved Tunnel

```text
Save
Rename
Edit
Delete
Reconnect
```

符合 Spec。

### Multiple Tunnel

同一个 Host 同时运行多个独立 Tunnel。

### Existing SSH Config Forwarding

即使 SSH Config 中存在：

```text
LocalForward
RemoteForward
DynamicForward
```

TunnelDock 连接也只能建立自己指定的 Forward。

### Authentication failure

需要 Password 时：

```text
Failed
```

不得弹 TunnelDock 密码框。

### Unknown Host Key

```text
Failed
```

不得自动接受。

### First connection failure

不得自动无限 retry。

### Established connection failure

必须：

```text
1 → 2 → 5 → 10 → 30 秒
```

无限自动重连。

### Remote service absent

SSH Tunnel 仍正常：

```text
Connected
```

### Host removed

已经运行的 Tunnel：

```text
继续运行
```

断线后：

```text
不重连
```

Saved Tunnel：

```text
不删除
```

Host 恢复：

```text
自动重新关联
```

### Quit

所有 Tunnel SSH Process 均退出。

不得留下孤儿进程。

---

# 58. v1.0 成功标准

TunnelDock v1.0 的成功标准不是功能多，而是：

> 用户打开 App，从 SSH Config 选择一台机器，输入 `8888`，点击一次 Connect，就可靠得到一个 `localhost:8888 → remote:localhost:8888` 的 SSH Tunnel。

与此同时：

* 用户现有 SSH Config 正常生效；
* 不要求重新管理 SSH Key；
* 不要求输入密码到 TunnelDock；
* 多 Tunnel 可以独立启停；
* Tunnel 可以保存；
* 网络中断后自动恢复；
* SSH Config 修改自动反映；
* 应用退出时环境干净；
* 整个工具保持轻量、原生、可预测。

这就是 **TunnelDock v1.0 的功能边界**。

任何超出本 Spec 的功能默认视为：

> v1.1 或之后版本需求。

开发阶段不得自行扩展 v1.0 Scope。

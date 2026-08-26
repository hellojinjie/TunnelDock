# TunnelDock 会话交接与审阅说明

日期：2026-08-26  
范围：本文件汇总本会话中针对 macOS SwiftPM 应用 TunnelDock 所做的代码、打包和验证工作，供后续模型进行静态审阅。仓库当前不是 Git 仓库，因而没有提交号或可供比较的分支基线。

## 当前架构与运行约束

- 工程使用 SwiftPM，不应安装 Xcode 或创建 `.xcodeproj`。
- 最低部署目标是 macOS 13；`Scripts/package-app.sh` 构建 arm64 和 `x86_64-apple-macosx13.0`，再用 `lipo` 合成为通用包。
- 数据分层：`TunnelDockCore` 负责 SSH、持久化、运行时；`TunnelDockAppSupport` 承载可观察 UI 状态；`TunnelDock` 是 SwiftUI 界面。
- 保存的 tunnel 数据写入 `~/Library/Application Support/TunnelDock/saved-tunnels.json`，schema 仍为 `1`。
- 在当前命令行工具环境中，构建脚本会设置 `SDKROOT=/Library/Developer/CommandLineTools/SDKs/MacOSX15.4.sdk` 和工作区内的 Clang 模块缓存。

## 本会话已完成的改动

### 1. Settings 与菜单栏

- Settings 窗口仅包含 `Show in Menu Bar` 开关。
- 菜单栏可见性现在由 `TunnelDockApp` 的 `@AppStorage("showMenuBar")` 直接驱动 `MenuBarExtra(isInserted:)`，避免此前将 `@StateObject` 绑定到 `MenuBarExtra` 造成的 SwiftUI 工具栏/导航更新循环。
- 主窗口工具栏新增 Settings 入口：macOS 14 及以上使用 `SettingsLink`，macOS 13 回退为 AppKit `showSettingsWindow:` action。
- 相关文件：
  - `Sources/TunnelDock/App/TunnelDockApp.swift`
  - `Sources/TunnelDock/UI/SettingsView.swift`
  - `Sources/TunnelDock/UI/MainWindowView.swift`
  - `Sources/TunnelDock/UI/MenuBarContentView.swift`

### 2. 运行无响应（ANR）诊断与修复

- 之前的运行样本显示进程长时间处于 SwiftUI `ToolbarStorage` / `NavigationPane` 图更新循环并占用约 93–99% CPU。
- 先前尝试过移动 Settings 按钮位置、使用普通按钮等，均不能消除循环；最终去除 `SettingsStore` 对 `MenuBarExtra` 的 `@StateObject` 绑定，改为上述 `@AppStorage` 绑定。
- 修复后曾实际启动 release `.app` 并观察到进程 CPU 为 0%。本次后续功能改动没有重新进行同样的运行时采样。

### 3. Intel 兼容性与通用应用包

- 保持 `Package.swift` 的 `.macOS(.v13)`，以覆盖较广的 Intel Mac 版本。
- 打包脚本同时构建 Apple Silicon 与 Intel 切片，产物为 `.build/release/TunnelDock.app`。
- `Tests/Packaging/package-app-tests.sh` 已检查签名和通用二进制架构；最近一次人工 `lipo -archs` 输出为 `x86_64 arm64`，`codesign --verify --deep --strict` 退出成功。
- 相关文件：
  - `Scripts/package-app.sh`
  - `Tests/Packaging/package-app-tests.sh`

### 4. Host 列表的活跃 tunnel 标记

- `AppState` 订阅 `TunnelManager.runtimes`，派生活跃 host alias 集合。
- `connecting`、`connected`、`reconnecting` 状态均视为活跃；`disconnected` 与 `failed` 不显示。
- 左侧 Host 列表对应行显示绿色 `point.3.connected.trianglepath.dotted` 图标。
- 相关文件：
  - `Sources/TunnelDockAppSupport/App/AppState.swift`
  - `Sources/TunnelDock/UI/HostSidebar.swift`
  - `Tests/TunnelDockAppTests/AppStateTests.swift`

### 5. tunnel 协议与浏览器入口

- 新增 `TunnelProtocol`：`http`、`https`，默认 `http`。
- `TunnelDefinition` 保存 `webProtocol`。自定义解码使用 `decodeIfPresent`，使既有 JSON 缺少该字段时仍默认 HTTP；编码会写入该字段，schema 版本没有变化。
- `TunnelRuntimeSnapshot.browserURL` 根据协议、本地地址和端口生成 URL；本地监听地址是 `0.0.0.0` 时，浏览器 URL 使用 `127.0.0.1`。
- Quick Forward 的 Advanced 中新增 Protocol 选择器；已保存 tunnel 的编辑表单也可修改协议。
- 仅 `.connected` 的 tunnel 行显示地球图标按钮，调用 `NSWorkspace.shared.open(url)` 交给系统默认浏览器。
- 注意：macOS 的通用 URL 打开 API 不能跨浏览器强制指定“当前活动窗口”；默认浏览器决定是在当前窗口、新标签页或新窗口中处理 URL。应用本身不主动创建浏览器窗口。
- 相关文件：
  - `Sources/TunnelDockCore/Models/TunnelDefinition.swift`
  - `Sources/TunnelDockCore/Models/TunnelRuntime.swift`
  - `Sources/TunnelDockCore/Tunnel/ForwardSpecification.swift`
  - `Sources/TunnelDockCore/Tunnel/TunnelManager.swift`
  - `Sources/TunnelDockCore/Persistence/TunnelRepository.swift`
  - `Sources/TunnelDock/UI/QuickForwardView.swift`
  - `Sources/TunnelDock/UI/TunnelEditorView.swift`
  - `Sources/TunnelDock/UI/TunnelRow.swift`

### 6. Recent tunnel 自动保存

产品确认的行为：Quick Forward 只有在连接成功后才自动保存为 Recent；Recent 可命名、可在断开/失败后删除；连接中、连接中或重连中均不能删除。

- 新增 `TunnelManager.connectRecent(hostAlias:input:)`：先通过原有临时连接流程建立 tunnel，连接成功后调用 `saveTemporary` 写入 repository 并转换运行时 ID 为 `.saved`。
- `QuickForwardView` 已改用 `connectRecent`，因此正常 GUI 流程不再产生短暂且断开即消失的 tunnel。
- 若自动持久化步骤失败，`connectRecent` 会断开并移除临时运行时，然后向 UI 抛出错误；不会留下未保存的活跃 tunnel。
- `HostDetailView` 的区块标题从 `Tunnels` 改为 `Recent Tunnels`。
- `TunnelRow` 移除手动 `Save…` 菜单项；保存后的 Recent 有 `Rename…`、`Edit…`、`Delete`。删除 UI 根据 `canModify` 禁用，`TunnelManager.delete` 也在核心层拒绝活动状态，形成双重保护。
- 相关文件：
  - `Sources/TunnelDockCore/Tunnel/TunnelManager.swift`
  - `Sources/TunnelDock/UI/QuickForwardView.swift`
  - `Sources/TunnelDock/UI/HostDetailView.swift`
  - `Sources/TunnelDock/UI/TunnelRow.swift`
  - `Tests/TunnelDockCoreTests/Tunnel/TunnelManagerConnectionTests.swift`

## 关键自动化测试

新增或更新的测试包括：

- `AppStateTests.activeTunnelHostsIncludeConnectingConnectedAndReconnecting`
- `TunnelDefinitionTests.decodesMissingWebProtocolAsHTTP`
- `TunnelDefinitionTests.browserURLUsesSelectedProtocolAndLoopbackForWildcardAddress`
- `QuickForwardModelTests.protocolDefaultsToHTTPAndResetsToHTTP`
- `TunnelRepositoryTests.roundTripsSelectedWebProtocol`
- `TunnelManagerConnectionTests.connectedTunnelIsAutomaticallySavedAsRecentAndRetainedAfterDisconnect`
- `TunnelManagerConnectionTests.runningTunnelCanRenameButCannotEditOrDelete`（已切到 Recent 连接入口，并断言命名写入 repository）
- `TunnelManagerConnectionTests.concurrentDisconnectsShareOneCleanupOperation`（已切到 Recent 连接入口）

最近一次完整验证（本会话最后执行）：

```sh
sh Scripts/test.sh
sh Scripts/package-app.sh
sh Tests/Packaging/package-app-tests.sh
lipo -archs .build/release/TunnelDock.app/Contents/MacOS/TunnelDock
codesign --verify --deep --strict .build/release/TunnelDock.app
```

结果：核心测试 68/68 通过，应用测试 14/14 通过，总计 82；release 构建和包测试通过；通用二进制输出为 `x86_64 arm64`；签名校验通过。

## 建议审阅重点

1. `connectRecent` 当前复用公开的 `connectTemporary` 与 `saveTemporary`。正常 GUI 不调用这两个旧入口，但它们仍作为公开 API 和底层测试入口存在。若目标是严格保证任何未来调用方都不可能创建非 Recent 的连接，可考虑将临时流程收窄为 `private` 或重命名为内部实现；这会要求同步重写部分底层恢复测试。
2. Recent 目前按现有 `TunnelManager.publish()` 的 host alias 与本地端口排序，不按最近连接时间排序，也不会去重。用户要求是“连接过即保存”，没有要求最大数量、去重规则或按时间排序；后续若要实现真正的历史列表，应明确这些产品规则并增加 `lastConnectedAt` 等持久化字段。
3. 自动保存失败时目前选择断开已经建立的连接以保持“已连接的 tunnel 必为 Recent”的一致性。该选择避免隐藏的、无法从 Recent 管理的活跃 tunnel，但会使存储目录不可写时连接失败；请确认该 UX 是否符合预期。
4. 浏览器按钮仅对 `connected` 显示；`connecting` 和 `reconnecting` 不显示，避免连接尚未就绪时访问本地端口。请核对这是否符合产品期望。
5. 未在物理 Intel Mac 上运行应用；Intel 切片由交叉编译、通用包和签名检查验证。

## 明确未处理项

- 用户曾指出 Finder 中显示默认图标而非自定义图标，并明确表示“不用解决”，本会话没有处理该问题。
- 没有进行外部发布、签名公证、推送远程仓库或创建提交。

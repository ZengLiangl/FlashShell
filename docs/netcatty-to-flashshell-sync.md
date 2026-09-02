# 从 Netcatty 代码对照：可同步到 FlashShell 的功能

> 依据：`first-cmd`（FlashShell / Go 模块 FlashDock）与 `Netcatty` 仓库源码（类型、Wails 绑定、Electron bridges、UI 组件）。不引用 FlashShell 既有产品文档。
>
> 日期：2026-09-02

## 1. 对照方法

两边都是「SSH 工作台」：主机清单 + 终端 + SFTP + 隧道。实现栈不同（Wails/Go+Vue vs Electron/React），因此「同步」指**产品能力与交互契约**，不是搬 React 组件。

判定规则：

- **已覆盖**：FlashShell 已有同职责 API/UI，不必再抄。
- **可同步**：Netcatty 有独立实现、FlashShell 明显缺失或明显弱，且与现有 Go/MCP 路线兼容。
- **不原样搬**：与 FlashShell 已有差异化能力冲突，或成本/平台不匹配。

## 2. 代码层面的产品差

| | FlashShell | Netcatty |
|---|---|---|
| 连接 | SSH + 本地 Shell | SSH / Telnet / Mosh / EternalTerminal / Serial / 插件协议 |
| AI | 对外 MCP 服务（策略、审批、审计、部署类工具） | 应用内 Agent（Catty）+ 外部 SDK（Codex/Claude/Copilot 等）+ 内置 MCP/CLI |
| 编排 | 项目 / 子项目 / 步骤流水线 | 无等价任务引擎；有 JS/Python 脚本片段 |
| 工作区 | 标签 + 最多 4 路网格分屏 + 广播 | 嵌套分屏树 + 侧栏再分屏 |
| 主机库 | 扁平分组、卡片首页、全局帐号 | 嵌套分组树、网格/列表/树、Keychain（密钥/证书/身份拆开） |
| 同步 | 本机 YAML / 多配置文件 | Gist / S3 / WebDAV / OneDrive 等 vault 云同步 |

FlashShell 已明显强于 Netcatty、**不要用 Netcatty 替换**的部分：任务流水线、MCP 策略/审批/审计、跨机复制与目录镜像、多客户端导入（Xshell/Moba/PuTTY/FinalShell/SecureCRT/OpenSSH）、主密码保险库、部署/装服务类 MCP 工具。

---

## 3. 已高度重叠（不必当缺口）

从绑定与组件可见，下列能力 FlashShell 已具备同级或接近实现：

- SSH 跳板 / 跳板链、HTTP/SOCKS 代理覆盖、旧算法与跳过 ECDSA host key
- Agent 转发、启动命令、本机回显、每主机终端配色
- SFTP / SCP 回退、sudo SFTP、文件名编码、本地+远端双栏、收藏路径、跟随 CWD
- 内置 Monaco、系统默认打开、外部应用关联、传输暂停/续传/冲突、剪贴板图上传
- 端口转发（本机/远程/动态）+ 会话内临时隧道
- 命令片段 + 快捷键 + 连接后自动执行 + 命令面板
- 会话恢复、标签休眠、WebGL、自动重连
- 监控：概览 / 磁盘 / 进程 / 监听端口
- Known Hosts 导入导出、CSV/模板导入导出、置顶/标签/备注/图标
- 全局帐号（身份复用）、保险库锁定、主题/字体/日志高亮、自动更新

---

## 4. 建议同步清单（按优先级）

### P0 — 终端工作区（用户每天碰到）

#### 4.1 嵌套分屏树

- **Netcatty**：`domain/workspace.ts` 任意水平/垂直嵌套、比例记忆、拖入插入、关闭塌缩。
- **FlashShell**：`ShellTerminalTabs.vue` 固定 2×2 网格，`MAX_SPLIT = 4`，无嵌套、无自由比例。
- **建议**：引入与 Netcatty 同构的 pane/split 树（可落在前端 + `SaveShellSessionRestore`）。保留现有广播、分屏快捷键。
- **不做**：一次搬侧栏再分屏（`useTerminalSidePanelLayoutState`），那是第二阶段。

#### 4.2 连接协议扩展：Telnet / 串口（优先），Mosh（其次）

- **Netcatty**：`HostProtocol = ssh | telnet | mosh | et | local | serial`，串口有波特率/校验/YMODEM。
- **FlashShell**：`ConnectShell` / `ConnectLocalShell`；导入器能识别 telnet/serial 但运行时当 SSH。
- **建议**：
  1. Telnet（网络设备、实验室）。
  2. Serial（COM / ttyUSB，配 `SerialConfig`）。
  3. Mosh 按需（需捆绑或探测 `mosh-server`，Go 侧工作量大）。
- **不做**：EternalTerminal、插件协议宿主（见第 6 节）。

#### 4.3 认证补齐：键盘交互 MFA、系统 Agent、证书、PPK

- **Netcatty**：`requiresMfa`、`useSshAgent` / `identityAgent`、`HostAuthMethod` 含 `certificate`、`ppkConverter`。
- **FlashShell**：密码 + 密钥文件 + 密钥口令 + 全局帐号；未见 keyboard-interactive 循环、证书登录、PuTTY PPK 转换、本机 ssh-agent 作为登录源（仅有远程 Agent 转发）。
- **建议**：优先 keyboard-interactive（跳板/堡垒 MFA）；其次系统 ssh-agent；再次证书 + PPK 导入到现有密钥文件模型。可挂在「密钥库」而不是新做一套 Keychain UI。

---

### P1 — 主机库与终端侧能力

#### 4.4 嵌套分组 + 列表/树视图

- **Netcatty**：`group` 为路径树，Vault 网格/列表/树，组默认值 `GroupConfig`。
- **FlashShell**：单层 `Machine.Group`，首页卡片；配置对话框有 table/board，Shell 侧有可折叠分组。
- **建议**：分组改为 `a/b/c` 路径（兼容旧扁平名）；首页增加列表/树；组默认用户名/端口/代理（对齐 `GroupConfig` 子集）。
- **可借鉴 UI**：最近连接时间 `lastConnectedAt`（Netcatty Host 字段；FlashShell 有连接历史组件，可把「最近」做成首页区）。

#### 4.5 发行版/设备图标与每主机字体

- **Netcatty**：`distro` 检测 + 手动覆盖、网络设备 `deviceType`、每主机 fontFamily/size/weight。
- **FlashShell**：emoji/预设 `Icon`，全局 Shell 字体，每主机仅 `TerminalPreset`。
- **建议**：连接后探测 `/etc/os-release` 写 `distro`；网络设备标记（影响后续「原始命令、不包 shell」——对 MCP/广播有用）；每主机字体覆盖。

#### 4.6 主机级 SSH 细节

从 `domain/models/connection.ts` 有、`define.Machine` 没有的字段，建议按需加：

| 字段 | 用途 |
|---|---|
| `environmentVariables` | 会话环境变量 |
| `charset` | 远端字符集（与 SFTP 编码分开） |
| `keepaliveInterval` / `keepaliveCountMax` / `keepaliveOverride` | 老设备关 keepalive |
| `sshTcpConnectTimeoutSeconds` / `sshAuthReadyTimeoutSeconds` | 超时 |
| `algorithms` | 按类覆盖 KEX/cipher/hmac（比布尔 `LegacyAlgorithms` 更细） |
| `showLineTimestamps` | 行时间戳 |
| `backspaceBehavior` | Backspace 发 `^H` |
| `disableDynamicTabTitle` | 禁 OSC 标题 |
| `startupCommandRunMode` | 多行启动：逐行延迟 vs 粘贴 |
| `proxy.type = command` | 命令式代理（`ProxyCommand`） |

#### 4.7 X11 转发

- Netcatty：`electron/bridges/x11Forwarding.cjs`。
- FlashShell：明确把 LocalEcho 写成「不是 X11」。
- **建议**：Windows 走 VcXsrv/X410 探测，Linux/macOS 走 `$DISPLAY`；作为主机开关，默认关。

#### 4.8 终端图形与键盘协议

- Netcatty：Kitty 图形 / SIXEL / iTerm 内联图、Kitty keyboard protocol。
- FlashShell：`ShellTerminal.vue` 无 ImageAddon / sixel。
- **建议**：先加 xterm 图像插件（看图、远程 TUI）；Kitty 键盘协议可后置。

#### 4.9 ZMODEM / YMODEM

- Netcatty：会话内 zmodem sentry、串口 ymodem。
- FlashShell：文件走 SFTP/SCP，无 rz/sz。
- **建议**：SSH 会话检测 ZMODEM 并走现有传输面板（网络设备、无 SFTP 场景）；YMODEM 跟串口一起做。

#### 4.10 压缩上传

- Netcatty：`compressUploadBridge.cjs` + `lib/uploadCompressed.ts`。
- **建议**：大目录上传先在远端 `tar`/`gzip` 再解压，接到现有 `StartShellUpload`。

---

### P1 — 监控、笔记、日志、片段

#### 4.11 系统管理侧栏增强

FlashShell `ShellMonitorPanel`：概览/磁盘/进程/端口。

Netcatty `components/systemManager/`：**服务（systemd）**、**Docker 容器/镜像/inspect**、**tmux 会话**、**GPU**。

- **建议**：进程/端口之后优先 Docker 与 systemd；tmux 次之。实现可复用 MCP 侧已有 docker 脚本思路，但要做成**交互面板**（启停/日志），不要只给 AI。

#### 4.12 笔记从「主机字段」升级为工作台

- FlashShell：`Machine.Notes` 纯文本。
- Netcatty：主机 Markdown 笔记 + 独立 `VaultNote`（标签、关联主机、大纲编辑器）。
- **建议**：主机备注改为 Markdown 预览（终端侧栏一页即可）；独立笔记库可放 P2。

#### 4.13 会话连接日志

- Netcatty：`domain/connectionLog.ts` + `ConnectionLogsManager`（可收藏、按会话回放）。
- FlashShell：命令历史、Shell 连接历史，不是「整段 PTY 录音」。
- **建议**：可选会话录像（敏感输入过滤可参考 Netcatty `terminalSensitiveInputRegistry` / log sanitizer）。默认关，避免磁盘与泄密。

#### 4.14 片段/脚本引擎

FlashShell `ShellSnippet`：文本、作用域、快捷键、是否执行、OnConnect。

Netcatty `Snippet`：变量占位、目标主机/组、包路径、`kind=script`（JS/Python）、`onOutput` 正则触发、`HostOutputTrigger`、自动完成。

- **建议**：先做 **变量模板** 与 **多主机目标**（广播已有，片段缺「指定组」）；脚本引擎与输出触发放到 P2（和任务流水线重叠，避免两套自动化）。

---

### P2 — 平台、同步、设置体验

#### 4.15 界面语言

- Netcatty：`UILanguage` + 全应用 i18n。
- FlashShell：用户可见文案固定中文。
- **建议**：若有海外用户再抽 i18n；否则维持中文。

#### 4.16 Vault 云同步

- Netcatty：`cloudSyncBridge.cjs`（S3/WebDAV）、Gist、OneDrive、`convergentSync`。
- FlashShell：本机 YAML + 多配置文件切换。
- **建议**：不要照搬 JSON vault。若要同步，应对现有加密 YAML 做「整文件加密上传 + 冲突策略」，且必须过主密码保险库。优先 WebDAV/S3，Gist 不适合放密文主机库。

#### 4.17 实时托管 `~/.ssh/config`

- Netcatty：`managedSourceId`，配置源与 vault 并存。
- FlashShell：一次性 `ImportOpenSSHConfig*`。
- **建议**：可选「监视 OpenSSH 配置并刷新列表」（只读托管，编辑仍走 FlashShell 机器）。

#### 4.18 设置内搜索、文件关联页、窗口透明度、托盘、全局快捷键

Netcatty `settingsSearchCatalog.ts`、`file-associations`、`windowOpacitySync`、tray、`globalShortcutBridge`。

FlashShell 已有文件关联、快捷键面板；无设置全文检索、透明度、托盘常驻。

- **建议**：设置搜索成本低、收益高；托盘/透明度按桌面习惯选做。

#### 4.19 Deep link / 一次性主机

- Netcatty：`ephemeral` 主机、JumpServer SFTP 自动开面板。
- **建议**：仅在有明确集成方时做；不要为了对齐而做。

#### 4.20 插件宿主

- Netcatty：`electron/plugins/`（`.ncpkg`、沙箱、RPC），协议可扩展。
- **建议：不同步。** Wails 无现成插件运行时；FlashShell 的扩展面应继续走 MCP 工具，而不是第二套插件 ABI。

---

## 5. AI：不要搬 Catty，可搬「上下文」

Netcatty 应用内对话（`infrastructure/ai/harness/`、侧栏 `ai` tab、多供应商）与 FlashShell「把能力暴露给 Cursor 等外部 Agent」是两条产品线。

**不要同步：**

- Catty / streamText 编排、compaction、Codex app-server
- 应用内聊天 UI 作为主 AI 入口

**可以同步的「给 MCP 用的上下文」**（Netcatty `buildAITerminalSessionInfo` / `hostChain` / `activePortForwards`）：

- 当前标签 CWD、跳板链、已开隧道，写入 MCP `system_info` / guidance
- Observer / Confirm / Auto 与 FlashShell `AIPolicy` 语义对齐（readonly / approval / trusted），避免两套权限语言
- 工具输出过大时的 handle（Netcatty `tool_output_read`）——若 MCP 返回截断，可同样落盘再读

Netcatty 的 **External MCP 客户端**（连别人的 MCP）对 FlashShell 价值低：FlashShell 自己就是 MCP 服务端。

---

## 6. 明确不建议原样同步

| Netcatty 能力 | 原因 |
|---|---|
| 完整插件协议与 `.ncpkg` | 运行时不匹配；扩展应走 MCP |
| 应用内多 Agent 供应商 | 与对外 MCP 定位重复，维护面巨大 |
| 密钥/证书/身份三套 Keychain | FlashShell 全局帐号 + 每机密钥已够；先补证书/Agent 即可 |
| 独立脚本语言引擎 | 与 SubProject/Command/Steps 重叠 |
| EternalTerminal | 使用面窄、捆绑成本高 |
| 收敛云同步的完整 CRDT | 实现与测试成本远高于「加密备份」 |

---

## 7. 推荐落地顺序

1. **分屏树**（工作区体感最接近 Netcatty）
2. **keyboard-interactive + ssh-agent 登录**
3. **Telnet**
4. **嵌套分组 + 列表视图 + lastConnectedAt**
5. **Docker / systemd 监控页**
6. **主机环境变量、keepalive、charset、命令代理**
7. **ZMODEM + 压缩上传**
8. **内联图像**
9. **串口（+ YMODEM）**
10. **X11、Mosh、会话录像、片段变量、加密云备份** —— 按真实用户需求抽

每项应落到 `define.Machine` / `data.GlobalConfig` / `app` 导出方法，前端只走 `wailsjs/go/app/App`，事件名保持 `域:动作`。

---

## 8. 源码锚点（便于开工）

**Netcatty**

- 主机模型：`domain/models/connection.ts`
- 分屏：`domain/workspace.ts`、`application/state/useSessionState*`
- 侧栏工具：`application/state/terminalSidePanelTabs.ts`
- 系统管理：`components/systemManager/`
- 协议/传输：`electron/bridges/terminalBridge.cjs`、`sftpBridge.cjs`、`portForwardingBridge.cjs`、`compressUploadBridge.cjs`
- 能力目录：`electron/capabilities/catalog/*.cjs`

**FlashShell**

- 主机模型：`define/types.go` `Machine`
- 前端绑定：`frontend/wailsjs/go/app/App.d.ts`
- 分屏：`frontend/src/components/shell/ShellTerminalTabs.vue`
- 监控：`frontend/src/components/shell/ShellMonitorPanel.vue`
- 主机编辑：`frontend/src/components/MachineConfigDialog.vue`
- 片段：`data/shortcuts.go` `ShellSnippet`
- MCP（已有、勿削弱）：`mcp/`

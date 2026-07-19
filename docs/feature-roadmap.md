# FlashDock 功能与优化路线图

> 定位：桌面运维工作台 —— **任务流水线**（YAML 驱动）+ **多会话 Shell/SSH/SFTP**。  
> 本文档基于 v1.0.8 代码现状整理，与 `memory-optimization.md` 互补：后者聚焦内存，本文聚焦功能补齐与产品演进。

---

## 一、现状速览

### 已有能力

| 域 | 能力 |
|----|------|
| 任务 | Project → SubProject → Command（batch/remote）→ Step；retry / on_fail；变量 `${KEY}`；配置 GUI 编辑 |
| Shell | 多标签 PTY、分屏（≤4）、广播、SFTP 传输、监控、隧道、连接历史、Xshell/FinalShell 导入 |
| 基础设施 | 主题、代理、快捷键、多窗口会话、任务与 Shell 并行（`App.vue` v-show） |
| 内存优化 | 输出缓冲 FIFO、xterm 懒初始化、Shell 懒挂载、部分按需加载（见 `memory-optimization.md`） |

### 已知缺口（代码/文档不一致）

| 缺口 | 说明 |
|------|------|
| 执行历史 UI 未接入 | `ExecutionHistoryDialog.vue` 与后端 API 存在，但未挂载到设置或菜单 |
| 日志设置无前端 | `logSettings.enabled/path` 仅 `global_config.yaml` 可配 |
| `Step.when` 未实现 | schema 有字段，`step_runner.go` 未求值 |
| 单 Command 执行缺失 | `ExecuteCommand` 实际跑整个 SubProject |
| 远程步骤不可中断 | `USAGE.md` 已注明，前端无法强制终止 remote SSH 步骤 | **已实现**：`SSHClient.Stop()` 关闭活跃 session + `executeSteps` 轮询 stopChannel |
| Host Key 未校验 | `ssh.InsecureIgnoreHostKey()`，无 known_hosts 管理 | **已实现**：`~/.flashdock/known_hosts.json` + 信任对话框 |

---

## 二、高价值补齐（已有雏形，优先落地）

这些改动投入小、收益高，且与现有架构一致。

| 优先级 | 项 | 涉及路径（参考） |
|--------|----|------------------|
| P0 | 挂载执行历史对话框 | `ExecutionHistoryDialog.vue`、`SettingsHubDialog.vue` |
| P0 | 系统设置暴露日志开关与路径 | `SystemSettingsDialog.vue`、`data/global_config.go` |
| P0 | 实现 `Step.when` 条件执行 | `define/step.go`、`machine/step_runner.go` |
| P1 | SubProject 列表支持「仅运行此 Command」 | `SubProjectList.vue`、`app/app.go` |
| P1 | 远程任务步骤可中断 | `machine/step_runner.go`、`machine/ssh.go`、任务 UI 停止按钮 |
| P1 | Host Key 信任管理 | `machine/ssh.go`、`MachineConfigDialog.vue` |

---

## 三、任务工具：功能建议

### 3.1 执行与编排

- **并行 Command**：同一 SubProject 内支持 `parallel: true` 的命令组并发执行（多服务部署场景）。
- **步骤依赖**：YAML 层 `depends_on: [step-id]`，无需可视化 DAG 编辑器。
- **执行预设（Profile）**：同一 SubProject 绑定不同 `${ENV}` 组合，一键切换 dev/staging/prod。
- **干跑（dry-run）**：展开变量并打印将执行的命令，不真正执行。
- **执行前检查（pre_check）**：磁盘、端口、依赖服务等检查失败则整组 abort。

### 3.2 变量与模板

- **步骤间传值**：上一步 `export` 或捕获 stdout 末行，下一步通过 `${VAR}` 引用。
- **机器级变量**：`Machine.vars` 在 remote 步骤中自动合并。
- **内置变量**：`${TIMESTAMP}`、`${GIT_BRANCH}`、`${USER}`、`${SESSION_ID}` 等。

### 3.3 自动化触发（保持桌面工具定位）

- **配置变更提示**：`config.yaml` 保存后提示是否重跑相关 SubProject。
- **定时任务（可选）**：用户显式开启的本机定时，适合备份类任务。
- **本地 Webhook**：内网 CI 回调触发指定 SubProject。

### 3.4 任务 ↔ Shell 联动（差异化优势）

任务模式与 Shell 模式已可并行，可进一步深化：

```
任务 remote 步骤失败 → 一键「在 Shell 中打开该机器」并 cd 到 workdir
Shell 选中命令 → 「保存为任务 Step」
任务执行复用已连接的 Shell session（减少重复 SSH 握手）
```

| 能力 | 价值 |
|------|------|
| 失败跳转 Shell | 缩短「看日志 → 登录排查」路径 |
| Shell → Step | 降低 YAML 编写门槛 |
| Session 复用 | 提速 + 减少连接数 |

### 3.5 可观测性

- 步骤级耗时与成功/失败统计（写入 execution log）。
- 输出按 Command/Step **折叠**，长日志可读性更好。
- 失败步骤高亮 + **一键复制错误上下文**（便于协作或 AI 辅助排查）。

---

## 四、Shell 工具：功能建议

### 4.1 安全与连接

| 功能 | 说明 |
|------|------|
| Host Key 信任 | 首次连接展示指纹，支持 known_hosts 导入/导出 |
| Jump Host | `Machine.proxyJump` 堡垒机跳转 |
| 连接模板 | 机器组导出/导入（密码脱敏），团队共享 |
| 代理变更批量重连 | 修改代理后「一键重连所有会话」 |

### 4.2 终端体验

- **跨会话命令历史**：按机器或全局合并，可搜索。
- **Snippets / 命令片段**：与 `shortcuts.json` 打通，终端内快速插入。
- **端口转发可视化**：隧道列表展示 `local:port → remote:port`，支持临时添加。
- **会话软恢复**：软关闭标签后保持连接一段时间，重开可 buffer 回放。

### 4.3 SFTP / 文件

- 文件夹双向同步（rsync 风格或简单 diff）。
- 远程小文件内置编辑，大文件用系统默认应用打开。
- 任务 `upload` 步骤与 `ShellTransferPanel` **共用传输队列**。

### 4.4 监控与诊断

- Monitor **告警阈值**（如 CPU > 90% 持续 1 分钟 → 系统通知）。
- **多机监控仪表盘**：选中机器组，一屏查看核心指标。
- SSH 通道 **RTT/质量** 指标（不稳定链路诊断）。

### 4.5 本地 Shell 局限（待明确产品边界）

- 本地终端暂无 SFTP/Monitor 面板（仅 remote）。
- 若需对齐体验，可考虑「本地文件侧栏」或明确文档说明差异。

---

## 五、优化细节

与 `memory-optimization.md` 衔接；未标注「已完成」的项仍可作为实施项。

### 5.1 内存 / 性能（P0–P1）

| 项 | 预期效果 | 状态 |
|----|----------|------|
| 非活跃 tab 销毁 xterm，buffer 回放 | N 标签仅 1（分屏最多 4）个 live xterm | 待实施 |
| xterm `scrollback: 500` | 单终端 scrollback 内存减半 | 部分已有 |
| 后台 tab buffer 512KB / 活跃 2MB | 10 后台标签 buffer ~20MB → ~5MB | 部分已有 |
| 机器列表与凭证解密分离（ListHost hint） | 列表不触发全量解密 | 待实施 |
| `GetProjectSummaries` 懒加载 | 首页不持有完整 steps 树 | 部分已有 |
| shell buffer `Uint8Array` 存储 | 缓冲路径内存降 ~25–33% | 待实施 |
| `EventsOn` 使用 unsubscribe | 避免误删全局监听 | 待实施 |
| 合并机器列表请求（单一 `shellMachines`） | 减少 IPC + 一份 JS 数组 | 待实施 |
| 延迟 `setupShellEvents` 至首次进 Shell | 首页无 shell 输出写入路径 | 待实施 |

### 5.2 启动与包体

- 路由级 **code splitting**：`ShellWorkspace`、`SettingsHub`、`About`、`ConfigEditor` 异步加载。
- Vite `manualChunks` 拆分 `element-plus` / `xterm`。
- Element Plus **Icons 按需注册**（组件按需引入已完成）。

### 5.3 后端

- Shell 输出 channel **背压策略可配置**（高流量：丢弃 vs 合并）。
- Session pool **空闲超时**：长时间无 I/O 的 Aux（SFTP/Monitor）自动释放。
- 凭证解密 **LRU 缓存**：大量机器时避免堆上长期明文。

### 5.4 体验型「省内存」

- **省内存模式**（`shellMemorySaver`）：离开 Shell 卸载 `ShellWorkspace`，Go 端会话保持，返回时 buffer 回放。适合「主要用任务、偶尔 Shell」用户。

### 5.5 验证清单（功能 + 内存回归）

1. 启动 → 首页 30s → 记录 JS Heap + Go RSS。
2. 开 8 个 SSH 标签，仅 1 活跃 → 确认 xterm 实例数。
3. 后台 tab 有输出 → 切回 → 缓冲完整可见。
4. 分屏 4 pane fit/resize 正常。
5. 列表不解密后：连接 / 编辑 / 复制机器正常。
6. 广播、软断开重连、SFTP、Monitor、任务执行与停止回归。

---

## 六、产品 / UX 快赢

| 项 | 说明 |
|----|------|
| 首页最近使用 | 最近 SubProject + 最近连接机器 |
| 全局命令面板（`Cmd+K`） | 跳转项目、连机器、开设置、跑任务 |
| 任务执行状态栏 | 当前 Step 名 + 已用时间 + 取消 |
| 机器健康徽章 | 上次连接成功/失败、延迟 |
| 配置校验 | 保存时检查 machine 引用、`when` 语法等 |
| 多窗口协同提示 | 「另一窗口正在执行 xxx」 |

---

## 七、实施路线图

```
第一批（1–2 周）— 补齐缺口 + 安全基线
├── 执行历史 UI 接入
├── 日志设置 UI
├── 单 Command 执行 + Step.when
├── 非活跃 xterm 销毁 + 后台 buffer 降档
└── Host key 信任管理

第二批（2–4 周）— 联动与编排
├── 任务 ↔ Shell 联动（失败跳转、session 复用）
├── Jump Host
├── 步骤间变量 / dry-run
├── GetProjectSummaries + 机器列表 hint 完善
└── 路由级 code splitting

第三批（按需）— 进阶能力
├── 并行 Command / depends_on
├── Snippets + 命令面板
├── 定时 / Webhook 触发
├── 多机监控仪表盘
└── SFTP 同步与远程编辑
```

---

## 八、不建议过早投入

| 方向 | 原因 |
|------|------|
| 全可视化 DAG 编辑器 | 维护成本高；YAML + 表单对运维人群通常足够 |
| 内置 Git/CI | 用任务步骤调用 `git` / `gh` 更轻量 |
| 云同步配置 | 先做本地导出/导入 + 脱敏分享更稳妥 |
| 重型工作流引擎 | 与「轻量桌面工具」定位冲突 |

---

## 九、关键文件索引

| Concern | Path |
|---------|------|
| 任务执行 | `machine/subproject.go`、`machine/step_runner.go` |
| 任务 UI | `frontend/src/components/SubProjectList.vue`、`TerminalOutput.vue` |
| 执行历史（待接入） | `frontend/src/components/ExecutionHistoryDialog.vue` |
| Shell 状态 | `frontend/src/composables/useShell.js` |
| Shell 工作区 | `frontend/src/views/ShellWorkspace.vue` |
| 设置中心 | `frontend/src/components/SettingsHubDialog.vue` |
| 全局配置 | `data/global_config.go` |
| 内存优化详案 | `docs/memory-optimization.md` |

---

*文档版本：2026-07-17 · 与代码实现同步维护*

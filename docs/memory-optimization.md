# FlashDock 内存优化方案

> 目标：减少应用启动后的常驻内存占用，同时保持现有功能与用户体验。  
> 技术栈：Go + Wails v2 后端，Vue 3 + Element Plus + xterm.js 前端。

---

## 一、当前内存结构概览

```
Go 后端（常驻）
├── GlobalConfig + 全部 Machine（含加密块）
├── ProjectConfig 完整树
├── ShellHistory 全量记录
├── Session Pools（Shell / Local / Aux）
└── sensitiveData 解密缓存

前端 - 仅首页
├── Element Plus + Icons 全量注册
├── 静态打包全部 SFC（无 code splitting）
├── useShell 状态 + shell 事件监听（首页即激活）
└── 机器列表重复副本（useShell + HomePage）

前端 - Shell 模式
├── ShellWorkspace（v-show 常驻）
├── xterm × 已激活过的标签数
├── shellOutputBuffer（最多 2MB/会话）
└── SSH 连接 + Aux SFTP/Monitor
```

### 已有优化（保留）

| 机制 | 位置 |
|------|------|
| 任务输出上限 2000 行 | `App.vue` |
| Shell 输出缓冲 2MB/会话 FIFO | `shellOutputBuffer.js` |
| xterm 懒初始化 | `ShellTerminal.vue` |
| Shell UI 首次懒挂载 | `App.vue` `shellMounted` |
| 设置面板 destroy-on-close | `SettingsHubDialog.vue` |
| Monitor 图表历史 cap（24 点） | `ShellMonitorPanel.vue` |
| 分屏上限 4 pane | `ShellTerminalTabs.vue` |
| 输出 channel 溢出丢弃 | `app.go` |
| SFTP 复制 sync.Pool | `utils/iocopy.go` |

---

## 二、内存热点排序

| 优先级 | 热点 | 典型占用 | 触发条件 |
|--------|------|----------|----------|
| P0 | 非活跃标签保留 xterm + writer | 每实例 5–20MB+ | 切换过多个 Shell 标签 |
| P0 | shellOutputBuffer 模块级 Map | 最多 2MB × 打开标签数 | 任意已连接会话 |
| P1 | SSH PTY + Aux 连接 | 每连接数 MB | 连接远程机器 |
| P1 | GetMachines 全量解密凭证 | 机器数 × 凭证大小 | 启动时重复调用 |
| P2 | Shell 基础设施首页常驻 | 事件监听 + 机器列表 | 从未进 Shell |
| P2 | 前端无代码分割 | Element Plus 全量 bundle | 启动即有 |
| P3 | base64 shell:data 双份存储 | ~33% 编码开销 | 活跃 Shell 会话 |
| P3 | Shell 连接历史无上限 | 随时间增长 | 长期用户 |

---

## 三、分阶段实施方案

### 阶段 1：Quick Wins（1–2 天，低风险）

#### 1.1 非活跃标签销毁 xterm，保留缓冲回放

- **文件：** `ShellTerminal.vue`
- **做法：** `active === false` 或 `viewVisible === false` 时调用 `destroyTerminal()`；切回时通过 `registerShellWriter({ replay: true })` 从 buffer 回放
- **预期：** N 标签仅 1（分屏最多 4）个 live xterm

#### 1.2 设置 xterm scrollback 上限

- **文件：** `ShellTerminal.vue`
- **做法：** `new Terminal({ scrollback: 500 })`
- **预期：** 单终端 scrollback 内存减半

#### 1.3 合并机器列表请求

- **文件：** `useShell.js`、`HomePage.vue`、`App.vue`
- **做法：** 单一数据源 `shellMachines`，HomePage 通过 props 接收
- **预期：** 减少 1 次 IPC + 1 份 JS 数组

#### 1.4 延迟 Shell 事件监听

- **文件：** `useShell.js`、`App.vue`
- **做法：** `setupShellEvents()` 移至首次 `enterShellMode()`；首页仅 `loadMachines()`
- **预期：** 首页空闲时无 shell 输出写入路径

#### 1.5 Element Plus Icons 按需注册

- **文件：** `main.js`、`utils/registerIcons.js`
- **做法：** 仅注册实际使用的图标组件

#### 1.6 Element Plus 组件按需引入（已完成）

- **文件：** `vite.config.js`、`main.js`
- **做法：** `unplugin-vue-components` + `unplugin-auto-import` + `ElementPlusResolver`
- **效果：** 构建产物 element-plus JS ~1018KB → ~533KB；CSS ~360KB → ~156KB

#### 1.7 默认窗口 Normal 启动（已完成）

- **文件：** `main.go`
- **做法：** `WindowStartState: options.Normal`（默认 1200×768）

---

### 阶段 2：结构性优化（3–5 天）

#### 2.1 机器列表与凭证解密分离

- **文件：** `define/types.go`、`app/app.go`
- **做法：** 新增 `ListHost/ListPort/ListUser` 明文 hint（`SetSensitiveData` 时写入）；`GetMachines` 优先读 hint，legacy 机器一次性迁移
- **预期：** 列表不再触发全量解密；Go 堆上少持有明文密码

#### 2.2 前端路由级代码分割

- **文件：** `App.vue`、`vite.config.js`
- **做法：** `defineAsyncComponent` 懒加载 ShellWorkspace、SettingsHub、About、ConfigEditor；manualChunks 拆分 element-plus / xterm
- **预期：** 首页基线 JS 内存降 2–5MB

#### 2.3 后台标签降低 buffer 上限

- **文件：** `shellOutputBuffer.js`、`ShellTerminal.vue`
- **做法：** 活跃标签 2MB，后台标签 512KB
- **预期：** 10 个后台标签 buffer 从 ~20MB 降至 ~5MB

#### 2.4 Shell 连接历史上限

- **文件：** `data/shell_history.go`
- **做法：** 保留最近 500 条记录
- **预期：** 历史文件与内存有界

---

### 阶段 3：深度优化（1–2 周）

#### 3.1 shell 输出缓冲二进制存储

- **文件：** `shellOutputBuffer.js`、`useShell.js`、`ShellTerminal.vue`
- **说明：** Wails 事件 JSON 传输下 `[]byte` 仍为 base64；前端收到后立即解码为 `Uint8Array` 存储，避免 buffer 内重复 base64 字符串
- **预期：** 缓冲路径内存降 ~25–33%

#### 3.2 Shell 省内存模式（可选）

- **文件：** `ThemeSettings`、`SystemSettingsDialog.vue`、`App.vue`
- **做法：** 开启后离开 Shell 时 `shellMounted = false` 卸载工作区；Go 端会话保持，回 Shell 时重建 UI + buffer 回放
- **预期：** 回首页后释放全部 xterm/DOM

#### 3.3 首页项目列表轻量加载

- **文件：** `app/app.go`、`App.vue`
- **做法：** `GetProjectSummaries()` 仅返回 name/description/subProjectCount；进入任务详情时 `GetProject(name)` 加载完整树
- **预期：** 前端少持有完整 steps 树

#### 3.4 修复 EventsOn 清理

- **文件：** `ShellTerminal.vue` 等
- **做法：** 使用 `EventsOn` 返回的 unsubscribe 函数，避免 `EventsOff(eventName)` 误删全局监听

---

## 四、实施路线图

```
阶段 1（立即见效）
├── 非活跃 tab 销毁 xterm
├── scrollback 上限
├── 机器列表去重
├── 延迟 setupShellEvents
└── Icons 按需注册

阶段 2（结构性）
├── ListHost hint + GetMachines 不解密
├── 异步组件 + vite chunks
├── 后台 tab 低 buffer
└── 连接历史上限 500

阶段 3（深度）
├── buffer Uint8Array 存储
├── 省内存模式开关
├── GetProjectSummaries 懒加载
└── EventsOn unsubscribe 修复
```

---

## 五、预期效果（估算）

| 场景 | 优化前 | 阶段 1 后 | 阶段 1+2 后 |
|------|--------|-----------|-------------|
| 启动后仅首页 | ~80–120MB | ~60–90MB | ~50–70MB |
| Shell 5 标签（1 活跃） | ~150–250MB | ~100–150MB | ~80–120MB |
| Shell 10 标签 + 分屏 | ~300MB+ | ~180–220MB | ~140–180MB |
| 500 台机器配置 | Go +30–50MB | 略减 | Go 端显著下降 |

---

## 六、验证清单

1. 启动 → 首页 30s → 记录 JS Heap + Go RSS
2. 开 8 个 SSH 标签，切换 1 个活跃 → 仅 1 个 xterm 实例
3. 后台 tab 有输出 → 切回 → 缓冲内容完整可见
4. 分屏 4 pane fit/resize 正常
5. 列表不解密后：连接 / 编辑 / 复制机器正常
6. 广播、软断开重连、SFTP、Monitor 回归

---

## 七、关键文件索引

| Concern | Path |
|---------|------|
| 后端 init / pools | `app/app.go` |
| 前端 bootstrap | `frontend/src/main.js` |
| Shell 状态 | `frontend/src/composables/useShell.js` |
| 输出缓冲 | `frontend/src/utils/shellOutputBuffer.js` |
| xterm 生命周期 | `frontend/src/components/shell/ShellTerminal.vue` |
| 凭证与 hint | `define/types.go` |
| 连接历史 | `data/shell_history.go` |

---

*文档版本：2026-07-17 · 与代码实现同步维护*

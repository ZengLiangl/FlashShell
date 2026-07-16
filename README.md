<p align="center">
  <img src="build/appicon.png" alt="FlashDock" width="128" height="128" />
</p>

<h1 align="center">FlashDock · 闪舵</h1>

<p align="center">
  <strong>一次停靠 · 本地任务与远程 Shell 同港出海</strong>
</p>

<p align="center">
  跨平台桌面运维工作台 — YAML 驱动流水线 × 多会话 SSH / SFTP × 实时终端
</p>

<p align="center">
  <a href="#功能矩阵"><img src="https://img.shields.io/badge/平台-Windows%20%7C%20macOS%20%7C%20Linux-2f9e6a?style=flat-square" alt="platforms" /></a>
  <a href="#技术栈"><img src="https://img.shields.io/badge/Backend-Go%201.23%20%2B%20Wails%20v2-00ADD8?style=flat-square&logo=go" alt="go" /></a>
  <a href="#技术栈"><img src="https://img.shields.io/badge/Frontend-Vue%203%20%2B%20xterm.js-42b883?style=flat-square&logo=vue.js" alt="vue" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow?style=flat-square" alt="license" /></a>
  <a href="https://github.com/ZengLiangl/first-cmd/releases"><img src="https://img.shields.io/github/v/release/ZengLiangl/first-cmd?style=flat-square&label=release" alt="release" /></a>
</p>

<p align="center">
  <a href="USAGE.md"><b>使用手册</b></a>
  ·
  <a href="#快速启航">快速启航</a>
  ·
  <a href="#典型航线">典型航线</a>
  ·
  <a href="https://github.com/ZengLiangl/first-cmd/releases">下载 Release</a>
</p>

---

```text
                    ╭────────── FlashDock ──────────╮
                    │                               │
     YAML 流水线 ──►│  任务模式   ║   Shell 模式   │◄── SSH / SFTP / PTY
     本地 & 远程    │  一键执行   ║  多会话并行     │    本机终端
                    │                               │
                    ╰──────────── 同港出海 ─────────╯
```

把重复发布、联调、登舰运维收进**同一个桌面港**——左边跑任务，右边开 Shell，互不打断、随时切换。

## 为什么叫「闪舵」

| | |
|:---:|:---|
| **闪** | 配置即执行。点一下，流水线起飞；ANSI 彩色日志实时回港 |
| **舵** | 任务与 Shell 同舵掌控。多机、多 Tab、监控与文件同屏 |

## 功能矩阵

<table>
<tr>
<td width="50%" valign="top">

### 任务模式
- 项目 → 子项目 → 命令步骤，图形化一键执行
- 本地命令 + 远程 SSH 步骤混排
- 实时进度、停止、输出回传
- YAML 驱动，`${ENV}` 全局变量替换

</td>
<td width="50%" valign="top">

### Shell 模式
- 多 Tab SSH / **本机终端**
- 交互式 xterm、搜索、粘底跟随
- 机器监控 + SFTP 文件面板
- 连接历史 / 连接管理器一键回连
- 导入 **Xshell / FinalShell**

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 体验与掌控
- 任务与 Shell **并行不互斥**
- 主题 · 终端配色 · 系统字体
- 平台快捷键（⌘ / Ctrl）可自定义
- 应用内检查更新 / 下载安装包

</td>
<td width="50%" valign="top">

### 安全与配置
- SSH 密钥 / 密码，敏感信息**加密落盘**
- 业务 YAML + `~/.flashdock/` 全局配置分离
- 机器分组、全局账号、环境变量
- 系统设置中心统一入口

</td>
</tr>
</table>

## 技术栈

| 层 | 选型 |
|:---|:---|
| 船体 | **Go 1.23+** · [Wails v2](https://wails.io) |
| 舰桥 | **Vue 3** · Element Plus · Vite · **xterm.js** |
| 航线 | SSH · SFTP · PTY（本机 / 远程） |

## 环境要求

- Go `1.23+`
- Node.js `16+`
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 快速启航

```bash
git clone https://github.com/ZengLiangl/first-cmd.git
cd first-cmd

cd frontend && npm install && cd ..
go mod tidy

wails dev      # 开发 · 热重载
wails build    # 发布构建
```

构建产物在 `build/bin/`（macOS 为 `FlashDock.app`）。打 tag 发布时，CI 产物命名为 `FlashDock-<tag>-…`。

首次启动若当前目录没有 `config.yaml`，会自动生成示例业务配置。全局数据落在：

```text
~/.flashdock/
├── global_config.yaml   # 机器、主题、环境变量…
├── shortcuts.json       # 快捷键
└── shell_history.json   # 连接历史
```

## 配置体系

FlashDock 使用**双层配置**，业务与全局解耦：

| 文件 | 默认路径 | 作用 |
|------|----------|------|
| 业务配置 | `config.yaml`（可多个） | 项目、子项目、命令步骤 |
| 全局配置 | `~/.flashdock/global_config.yaml` | 窗口、机器、主题、环境变量 |
| 快捷键 | `~/.flashdock/shortcuts.json` | 可自定义系统快捷键 |

业务配置中的 `${KEY}` 由全局 `workPaths` 替换。全局配置仅在文件不存在时写入默认值，**不会覆盖**已有内容。

更细的操作说明见 **[USAGE.md](USAGE.md)**。

## 项目结构

```text
first-cmd/
├── app/           # 应用入口：菜单、事件、执行调度、Shell API
├── data/          # 配置 / 会话 / 快捷键 / 会话导入
├── define/        # 类型定义
├── machine/       # 本地命令、SSH、SFTP、PTY Shell
├── cmds/          # 远程特殊命令（upload、targz、chdir）
├── crypto/        # 敏感信息加密
├── frontend/      # Vue 3 舰桥 UI
├── main.go        # Wails 启动
└── config.yaml    # 业务配置示例
```

## 常用命令

```bash
wails dev
wails build
cd frontend && npm install && cd ..
go mod tidy
go test ./...
go fmt ./...
go vet ./...
```

## 运行模式

```bash
# 前台（默认）
./build/bin/FlashDock.app/Contents/MacOS/FlashDock -reg=desk

# 后台守护
./build/bin/FlashDock.app/Contents/MacOS/FlashDock -reg=back
```

后台日志：`/tmp/FlashDock.out`、`/tmp/FlashDock.err`。

## 典型航线

1. **系统设置 → 机器配置**：添加服务器并测试连接  
2. **环境变量**：配置 `WORKSPACE` 等路径  
3. 编辑 `config.yaml`：定义项目与子项目  
4. 首页进入 **任务** 执行流水线，或进入 **Shell** 交互运维  
5. **快捷键**：按习惯改绑，保存即生效  

```text
  机器入港 ──► 变量就绪 ──► YAML 编排 ──► 一键出航 / Shell 登舰
```

## 故障排除

| 现象 | 建议 |
|------|------|
| Wails 构建失败 | 更新 Wails CLI，确认 Go / Node 版本 |
| SSH 连接失败 | 设置中心测试连接；检查密钥权限与网络 |
| 全局配置被改写 | 编辑的应是 `~/.flashdock/global_config.yaml` |
| 配置不生效 | 顶部配置文件菜单刷新，或切换配置文件 |
| 快捷键不生效 | 系统设置 → 快捷键，保存后重试；输入框内默认不抢快捷键 |

## 许可证

MIT License — 详见 [LICENSE](LICENSE)。

---

<p align="center">
  <sub>FlashDock · 闪舵 — 把运维留在港里，把时间留给真正要开的船。</sub>
</p>

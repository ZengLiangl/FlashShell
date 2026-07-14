# FlashDock · 闪舵

**一次停靠，本地任务与远程 Shell 同港出海。**

FlashDock（闪舵）是基于 [Wails v2](https://wails.io) 的跨平台桌面工具：用 YAML 编排构建 / 部署流水线，同时提供多会话 SSH 终端与 SFTP，适合日常运维、联调与重复性发布。

📖 **详细使用说明 → [USAGE.md](USAGE.md)**

## 为什么叫闪舵

| 意象 | 含义 |
|------|------|
| **闪** | 配置即执行，一键跑通流水线，反馈实时到终端 |
| **舵** | 任务模式与 Shell 模式集中掌控，多机多会话并驾齐驱 |

## 功能亮点

- **任务模式**：项目 → 子项目 → 命令步骤，图形化一键执行
- **Shell 模式**：多 Tab SSH、交互式终端、SFTP 文件浏览
- **并行不互斥**：任务执行与 Shell 会话可同时进行，首页可来回切换
- **实时终端**：ANSI 颜色、搜索、虚拟滚动、粘底跟随
- **配置驱动**：YAML 业务配置 + 全局机器 / 环境变量 / 主题 / 快捷键
- **系统设置中心**：机器配置、环境变量、外观、快捷键、执行历史统一入口
- **平台快捷键**：Mac 显示 Command、Windows 显示 Ctrl，可自定义并保存到 JSON
- **安全连接**：SSH 密钥 / 密码、敏感信息加密落盘；支持导入 Xshell / FinalShell

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23+、Wails v2 |
| 前端 | Vue 3、Element Plus、Vite、xterm.js |
| 远程 | SSH、SFTP、PTY |

## 环境要求

- Go 1.23+
- Node.js 16+
- Wails CLI v2

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 快速开始

```bash
git clone <repository-url>
cd first-cmd

cd frontend && npm install && cd ..
go mod tidy

wails dev      # 开发（热重载）
wails build    # 发布构建
```

构建产物位于 `build/bin/`（macOS 为 `FlashDock.app`）。打 tag 发布时，CI 产物命名为 `FlashDock-<tag>-…`。

首次启动若当前目录没有 `config.yaml`，会自动创建示例业务配置。全局数据在 `~/.flashdock/`。

## 配置体系

FlashDock 使用**两层配置**：

| 文件 | 默认路径 | 作用 |
|------|----------|------|
| 业务配置 | `config.yaml`（可多个） | 项目、子项目、命令步骤 |
| 全局配置 | `~/.flashdock/global_config.yaml` | 窗口标题、配置历史、环境变量、机器清单、主题 |
| 快捷键 | `~/.flashdock/shortcuts.json` | 可自定义系统快捷键 |

业务配置中的 `${KEY}` 由全局 `workPaths` 替换。全局配置仅在文件不存在时写入默认值，不会覆盖已有内容。

## 项目结构

```
first-cmd/
├── app/              # 应用入口：菜单、事件、执行调度、Shell API
├── data/             # 配置 / 会话 / 快捷键 / 导入
├── define/           # 类型定义
├── machine/          # 本地命令、SSH、SFTP、PTY Shell
├── cmds/             # 远程特殊命令（upload、targz、chdir）
├── crypto/           # 敏感信息加密
├── frontend/         # Vue 3 前端
├── main.go           # Wails 启动入口
└── config.yaml       # 业务配置示例
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

## 典型工作流

1. 打开系统设置 → **机器配置**，添加服务器并测试连接  
2. 在 **环境变量** 中配置 `WORKSPACE` 等路径  
3. 编辑 `config.yaml` 定义项目与子项目  
4. 首页进入**任务模式**执行子项目，或进入 **Shell** 交互运维  
5. 在系统设置 → **快捷键** 中按习惯改绑按键  

## 故障排除

| 现象 | 建议 |
|------|------|
| Wails 构建失败 | 更新 Wails CLI，确认 Go / Node 版本 |
| SSH 连接失败 | 系统设置中测试连接；检查密钥权限与网络 |
| 全局配置被改写 | 编辑的应是 `~/.flashdock/global_config.yaml` |
| 配置不生效 | 顶部配置文件菜单中刷新，或切换配置文件 |
| 快捷键不生效 | 系统设置 → 快捷键，保存后重试；输入框内默认不抢快捷键 |

## 许可证

MIT License，详见 [LICENSE](LICENSE)。

# Quick Cmd

基于 [Wails v2](https://wails.io) 的跨平台桌面工具，用 YAML 配置驱动本地与远程（SSH）命令执行，适合日常构建、部署、重启服务等重复性操作。

📖 **详细使用说明见 [USAGE.md](USAGE.md)**

## 功能概览

- **图形化执行**：项目 → 子项目 → 命令，一键跑完整流程
- **实时终端**：ANSI 颜色渲染、错误/成功高亮、虚拟滚动、粘底跟随
- **事件推送输出**：后端通过 Wails 事件推送日志，无轮询，Windows 下更流畅
- **多配置文件**：菜单切换/刷新/打开，支持多套业务配置
- **远程执行**：SSH 命令、按需 SFTP 上传、Docker 重启等
- **全局管理**：机器配置（敏感信息加密）、环境变量 `${KEY}` 替换
- **原生菜单**：文件 / 设置 / 帮助，支持快捷键

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23+、Wails v2 |
| 前端 | Vue 3、Element Plus、Vite |
| 远程 | SSH、SFTP（仅 upload 步骤时建立） |

## 环境要求

- Go 1.23+
- Node.js 16+
- Wails CLI v2

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Windows 还需安装 [WebView2 运行时](https://developer.microsoft.com/microsoft-edge/webview2/)。

## 快速开始

```bash
# 克隆并进入项目
git clone <repository-url>
cd first-cmd

# 安装依赖
make install-deps

# 开发模式（热重载）
make dev

# 构建发布版
make build
```

构建产物位于 `build/bin/`。Windows 下一般为 `quick-cmd.exe`。

首次启动时，若当前目录没有 `config.yaml`，应用会自动创建示例业务配置。全局配置保存在用户目录（见下方）。

## 配置体系

Quick Cmd 使用**两层配置**：

| 文件 | 默认路径 | 作用 |
|------|----------|------|
| 业务配置 | `config.yaml`（可多个） | 项目、子项目、命令步骤 |
| 全局配置 | `~/.cmd-config/global_config.yaml` | 窗口标题、配置历史、环境变量、机器清单 |

> **注意**：全局配置只在文件**不存在**时自动创建默认内容；已有内容不会被启动时覆盖。业务配置中的 `${KEY}` 由全局配置的 `workPaths` 替换。

## 项目结构

```
first-cmd/
├── app/              # 应用入口：菜单、事件、执行调度
├── data/             # 配置加载/保存、全局配置管理
├── define/           # 类型定义
├── machine/          # 本地命令、SSH、SubProject 执行器
├── cmds/             # 远程特殊命令（upload、targz、chdir）
├── crypto/           # 机器敏感信息加密
├── frontend/         # Vue 3 前端
├── main.go           # Wails 启动入口
├── Makefile          # dev / build / test 等
├── dev.sh / build.sh # 开发/构建脚本
└── config.yaml       # 业务配置（示例）
```

## 常用命令

```bash
make dev            # 开发模式
make build          # 构建应用
make install-deps   # 安装前后端依赖
make test           # 运行 Go 测试
make fmt            # 格式化
make lint           # go vet
make clean          # 清理构建产物
make config         # 从 config.example.yaml 复制示例配置（若存在）
```

## 运行模式

```bash
# 前台（默认）
./build/bin/quick-cmd -reg=desk

# 后台守护（仅非 Windows）
./build/bin/quick-cmd -reg=back
```

后台模式日志写入 `/tmp/quick-cmd.out` 与 `/tmp/quick-cmd.err`。

## 典型工作流

1. 在「机器配置」中添加远程服务器并测试连接
2. 在「环境变量」中配置 `ACC-CLOUD`、`HOME` 等路径变量
3. 编辑 `config.yaml` 定义项目与子项目
4. 选择项目 → 点击子项目「执行」→ 在终端查看输出
5. 需要时用「停止」中断本地任务

## 故障排除

| 现象 | 建议 |
|------|------|
| Wails 构建失败 | 更新 Wails CLI，确认 Go/Node 版本 |
| Windows 全局卡顿 | 见 [USAGE.md - 性能说明](USAGE.md#性能说明) |
| SSH 连接失败 | 在「机器配置」中测试；检查密钥/密码 |
| 全局配置被改写 | 确认编辑的是 `~/.cmd-config/global_config.yaml`，不是项目目录下的文件 |
| 配置修改不生效 | 菜单「刷新配置列表」，或切换配置文件 |

## 许可证

MIT License，详见 [LICENSE](LICENSE)。

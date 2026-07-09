# Quick Cmd

基于 [Wails v2](https://wails.io) 的跨平台桌面工具，用 YAML 配置驱动本地与远程（SSH）命令执行，适合日常构建、部署、重启服务等重复性操作。

📖 **详细使用说明见 [USAGE.md](USAGE.md)**

## 功能概览

- **图形化执行**：项目 → 子项目 → 命令，一键跑完整流程
- **实时终端**：ANSI 颜色渲染、错误/成功高亮、虚拟滚动、粘底跟随
- **事件推送输出**：后端通过 Wails 事件推送日志，无轮询
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

## 快速开始

```bash
# 克隆并进入项目
git clone <repository-url>
cd first-cmd

# 安装依赖
cd frontend && npm install && cd ..
go mod tidy

# 开发模式（热重载）
wails dev

# 构建发布版
wails build
```

构建产物位于 `build/bin/`（macOS 为 `quick-cmd.app`）。

首次启动时，若当前目录没有 `config.yaml`，应用会自动创建示例业务配置。全局配置保存在 `~/.cmd-config/global_config.yaml`。

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
├── dev.sh / build.sh # 开发/构建脚本（可选）
└── config.yaml       # 业务配置（示例）
```

## 常用命令

```bash
wails dev                              # 开发模式
wails build                            # 构建应用
cd frontend && npm install && cd ..    # 安装前端依赖
go mod tidy                            # 整理 Go 依赖
go test ./...                          # 运行测试
go fmt ./...                           # 格式化 Go 代码
go vet ./...                           # 静态检查
```

## 运行模式

```bash
# 前台（默认）
./build/bin/quick-cmd.app/Contents/MacOS/quick-cmd -reg=desk

# 后台守护
./build/bin/quick-cmd.app/Contents/MacOS/quick-cmd -reg=back
```

后台模式日志写入 `/tmp/quick-cmd.out` 与 `/tmp/quick-cmd.err`。

## 典型工作流

1. 在「机器配置」中添加远程服务器并测试连接
2. 在「环境变量」中配置 `WORKSPACE`、`PROJECT_ROOT` 等路径变量
3. 编辑 `config.yaml` 定义项目与子项目
4. 选择项目 → 点击子项目「执行」→ 在终端查看输出
5. 需要时用「停止」中断本地任务

## 故障排除

| 现象 | 建议 |
|------|------|
| Wails 构建失败 | 更新 Wails CLI，确认 Go/Node 版本 |
| SSH 连接失败 | 在「机器配置」中测试；检查密钥/密码 |
| 全局配置被改写 | 确认编辑的是 `~/.cmd-config/global_config.yaml` |
| 配置修改不生效 | 菜单「刷新配置列表」，或切换配置文件 |

## 许可证

MIT License，详见 [LICENSE](LICENSE).

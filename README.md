# Quick Cmd

一个基于 Wails v2 的跨平台桌面应用，用于快速执行和管理本地/远程（SSH）命令，支持多项目分组、可视化执行、实时终端输出（ANSI 颜色）、环境变量管理、远程机器管理等。

## 功能特性

- 图形化执行：项目/子项目分组展示，一键执行整套命令
- 实时终端：完整 ANSI 转义序列渲染，自动高亮成功/错误
- 多配置支持：支持多个 `config.yaml`，菜单中可切换/刷新/打开
- 远程管理：SSH 连接测试、敏感信息加密缓存、SFTP 支持（见代码）
- 环境变量：全局“环境变量配置管理”对话框，变量在配置中可展开
- 机器管理：全局“机器配置”对话框，新增/编辑/删除/连接测试
- 窗口与菜单：原生应用菜单（文件/设置/帮助），含“关于”弹框

## 技术栈

- 后端：Go 1.20+、Wails v2（`wails.Run` 应用生命周期）
- 前端：Vue 3、Element Plus、Vite（构建输出内嵌到 Wails 资源）

## 目录结构（关键）

```
new_cmd/
├── app/                 # 应用主逻辑（菜单、事件、配置入口）
│   └── app.go
├── data/                # 配置与全局状态（加载/保存/切换）
├── define/              # 公共类型定义（状态、事件、模型）
├── machine/             # 本地/SSH 执行与传输
├── frontend/            # 前端应用（Vue 3 + Element Plus）
│   ├── src/App.vue      # 主界面（终端、列表、对话框、事件）
│   └── src/components/  # 机器/环境变量/关于 等弹框组件
├── build/               # 构建产物（macOS .app 等）
├── main.go              # 应用入口（菜单、窗口参数、资源绑定）
├── Makefile             # 常用任务（dev/build/test/lint 等）
├── build.sh, dev.sh     # 构建/开发脚本（由 Makefile 调用）
└── config.yaml          # 配置文件（可在应用中切换/打开）
```

## 全局配置（global_config.yaml）

- 默认路径：`~/.cmd-config/global_config.yaml`（若不存在将自动创建）
- 用途：
  - 记忆最后打开的业务配置文件（`lastOpenedFile`）与历史列表（`configFile`）
  - 维护工作路径变量映射 `workPaths`，在业务配置中可使用 `${KEY}` 引用
  - 维护全局机器清单 `machines`（敏感信息经加密后写入 `encrypted_data`）

字段说明：
- `appId`：应用标识
- `windowsName`：窗口标题
- `configFile`：历史业务配置文件数组
- `lastOpenedFile`：最后一次打开的业务配置文件
- `workPaths`：键值对的路径变量表，如 `HOME: ~`
- `machines`：机器数组，结构为 `name`、`key_file`、`encrypted_data`（敏感信息通过 UI 写入后加密存储）

默认示例：

```yaml
appId: com.runner
windowsName: "运行器"
configFile:
  - config.yaml
lastOpenedFile: config.yaml
workPaths:
  HOME: ~
machines:
  - name: 示例服务器
    key_file: ~/.ssh/id_rsa
    # encrypted_data: "..."  # UI 写入后生成
```

说明：业务配置在解析时会先用全局配置的 `workPaths` 替换 `${KEY}`，再展开 `~` 与环境变量（见 `data/config.go` 的 `processPathVariables` 与 `expandPath`）。

## 安装与依赖

1) 安装 Go 1.20+、Node.js 16+
2) 安装 Wails CLI（若本地未安装）：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

3) 安装依赖（前后端）：

```bash
make install-deps
```

> 提示：`make frontend-deps` 可单独安装前端依赖；`go mod tidy` 在 `install-deps` 中自动执行。

## 开发与构建

- 开发模式（热重载，通过脚本统一启动）：

```bash
make dev
```

- 构建发布版本（产物位于 `build/`）：

```bash
make build
```

- 其他常用任务：

```bash
make test   # 运行测试（Go）
make fmt    # 格式化（Go + 尝试前端）
make lint   # 静态检查（Go vet）
make clean  # 清理构建与前端产物
```

## 运行模式（前台/后台）

应用支持通过命令行切换运行模式（见 `main.go`）：

- 前台（默认）：`-reg=desk`
- 后台（守护）运行：`-reg=back`

当使用后台模式时，应用会自举为守护进程并将日志写入：
- 标准输出：`/tmp/quick-cmd.out`
- 标准错误：`/tmp/quick-cmd.err`

示例：

```bash
./build/bin/quick-cmd -reg=back
```

> 非 Windows 平台将设置 `Setsid` 实现后台化；进程内部使用 `QUICKCMD_DAEMONIZED=1` 防止递归。

## 应用菜单与对话框

- 文件
  - 新建窗口：启动新进程打开新的应用窗口

- 设置
  - 配置文件：动态列出所有配置；可切换、刷新、打开全局/当前配置
  - 机器配置：打开“机器配置”对话框（增删改查、连接测试、敏感信息）
  - 环境变量：打开“环境变量配置管理”对话框（增删改查、用法说明）

- 帮助
  - 关于：打开“关于 Quick Cmd”对话框，展示项目简介与版本信息

以上菜单均由 `app/app.go` 中的 `CreateApplicationMenu()` 构建，通过 Wails 事件与前端组件联动（例如 `open:machine-config`、`open:workpath-config`、`open:about`）。

## 配置文件与多配置切换

- 首次启动时若缺少 `config.yaml`，后端会创建默认配置并加载
- 菜单“设置/配置文件”支持：
  - 单选切换当前配置
  - 刷新配置列表
  - 打开全局配置文件/当前配置文件
- 切换配置会：
  - 停止正在运行的任务
  - 清空输出
  - 重建执行器并通知前端 `config:changed`（前端将自动刷新）

### 业务配置示例（与实现一致）

```yaml
projects:
  - name: 示例项目
    description: 这是一个示例项目
    workdir: "${HOME}/workspace/example"
    subprojects:
      - name: 构建
        description: 构建相关命令
        commands:
          - name: 编译
            description: 编译项目
            type: batch
            steps:
              - "${MVM} clean compile"
          - name: 测试
            description: 运行测试
            type: batch
            steps:
              - "${MVM} test"

machines:
  - name: 示例服务器
    key_file: ~/.ssh/id_rsa
    # encrypted_data: "..."  # 经 UI 写入
```

## 键盘快捷键（前端）

- Cmd/Ctrl+C：复制选中文本；若无选择则复制全部终端文本
- Cmd/Ctrl+K：清空终端输出
- Cmd/Ctrl+M：打开“机器配置”
- Cmd/Ctrl+E：打开“环境变量配置”
- Escape：关闭已打开对话框

## 窗口与外观

窗口标题从全局配置中读取（`windowsName`），若缺失则为“Quick Cmd”。应用窗口尺寸与限制见 `main.go`（宽 1200、高 768，设定最小值与背景色）。

## 典型工作流

1) 启动应用并加载配置
2) 在列表中选择一个项目
3) 在“子项目”区域选择要执行的单元并点击“执行”
4) 在右侧终端观察实时输出与进度
5) 如需中断，可点击状态栏“停止全部”或对具体子项目“停止”

## 故障排除

- Wails 构建失败：更新 Wails CLI；确保 Node/Go 版本满足要求
- SSH 连接失败：检查主机/端口/认证信息；可在“机器配置”中进行连接测试
- 前端无响应：查看控制台/刷新界面；确保后端正常运行
- 配置刷新无效：在菜单中使用“刷新配置列表”或切换配置

## 许可证

MIT License，详见 `LICENSE`。

# Quick Cmd

这是一个名为 **Quick Cmd** 的 Go 桌面应用程序，主要用于**快速执行和管理各种命令行任务**。它是一个基于 Wails GUI 框架的跨平台工具，可以帮助开发者快速执行本地和远程服务器的命令。

## 快速开始

### 环境要求

- Go 1.20+
- Node.js 16+
- Wails v2
- 支持 GUI 的操作系统 (Windows/macOS/Linux)

### 安装依赖

1. 安装 Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

2. 克隆项目并安装依赖:

```bash
git clone <repository-url>
cd quick-cmd
```

### 开发模式

运行开发模式（支持热重载）:

```bash
./dev.sh
```

### 构建应用

构建生产版本:

```bash
./build.sh
```

构建完成后，可执行文件位于 `build/bin/` 目录下。

### 快速使用

1. 创建配置文件:

```bash
make config
```

2. 编辑 `config.yaml` 文件，配置你的项目和机器信息

3. 启动应用:

```bash
# 开发模式
make dev

# 或者运行构建后的可执行文件
./build/bin/quick-cmd
```

📖 **详细使用说明请查看 [USAGE.md](USAGE.md)**


### 配置文件

创建 `config.yaml` 文件来配置你的项目和命令（下例为标准结构，机器敏感信息不以明文出现）：

```yaml
projects:
  - name: "我的项目"
    description: "项目描述"
    workdir: "${HOME}/workspace/myproject"
    subprojects:
      - name: "构建"
        description: "构建相关命令"
        commands:
          - name: "编译"
            description: "编译项目"
            type: "batch"
            steps:
              - "go build ."
          - name: "测试"
            description: "运行测试"
            type: "batch"
            steps:
              - "go test ./..."

machines:
  - name: "生产服务器"
    key_file: "~/.ssh/id_rsa"
    # encrypted_data: "..."  # 通过 UI 写入（host/port/user/password）
```

## 核心功能特性

### 1. **图形化界面管理**

- 使用 Wails 框架构建跨平台 GUI 应用
- 支持多项目分组管理
- 提供按钮式的命令执行界面
- 内置终端显示区域，实时查看命令执行结果
- **ANSI 转义序列支持**: 完整支持终端颜色、样式和 256 色/真彩色输出

### 2. **配置文件驱动**

- 使用 YAML 格式配置文件定义项目和命令
- 支持多配置文件切换
- 支持环境变量和路径变量的动态替换
- 配置包括项目、子项目、远程机器等信息

### 3. **多执行模式**

- **本地批处理模式 (batch)**: 在本地执行命令序列
- **远程执行模式 (remote)**: 通过 SSH 连接到远程服务器执行命令
- 支持自定义命令和特殊命令处理

### 4. **远程服务器管理**

- SSH 连接管理（支持密码和密钥认证）
- SFTP 文件传输功能
- 远程命令执行和交互
- 机器连接池管理

## 项目结构分析

### 核心模块

1. **`define/`** - 数据结构和接口定义

   - `Root`: 配置根结构，包含项目和机器列表
   - `Project/SubProject`: 项目层级结构
   - `Machine`: 远程机器配置
   - `Runner`: 命令执行器接口

2. **`data/`** - 数据处理和配置管理

   - YAML 配置文件解析
   - 路径变量替换
   - 配置缓存和持久化

3. **`ui/`** - 用户界面

   - 主窗口和布局管理
   - 终端组件
   - 环境变量设置界面
   - 机器管理界面

4. **`machine/`** - 远程机器管理

   - SSH 连接建立和维护
   - 命令执行管道
   - 特殊命令处理器

5. **`console/`** - 控制台和管道管理

   - 输入输出流处理
   - 多路复用管道
   - 实时数据流处理

6. **`cmds/`** - 命令工具集
   - 各种实用命令的实现
   - 文件操作、压缩、上传等工具

## 典型使用场景

根据配置文件示例，这个工具主要用于：

1. **项目构建和部署**

   - Gradle/Maven 项目构建
   - 应用打包和上传
   - Docker 容器管理

2. **远程服务器操作**

   - 服务重启
   - 文件上传下载
   - 远程命令批量执行

3. **开发环境管理**
   - 多项目快速切换
   - 环境变量管理
   - 工作目录管理

## 技术特点

- **跨平台**: 基于 Wails，支持 Windows、macOS、Linux
- **模块化设计**: 清晰的模块分离，易于扩展
- **配置驱动**: 通过 YAML 配置实现灵活的任务定义
- **实时交互**: 内置终端，支持实时命令执行和输出显示
- **安全连接**: 支持 SSH 密钥和密码认证
- **中文友好**: 界面和配置支持中文

这个工具特别适合需要频繁在不同项目间切换、执行重复性命令的开发者使用，可以大大提高开发效率。

## 使用指南

### 界面说明

启动应用后，你会看到三个主要区域：

1. **左上角 - 项目列表**: 显示配置文件中定义的所有项目
2. **左下角 - 命令列表**: 显示选中项目的可执行命令
3. **右侧 - 终端输出**: 显示命令执行的实时输出

### 基本操作

1. **选择项目**: 在项目列表中点击项目名称
2. **执行命令**: 在命令列表中点击"执行"按钮
3. **查看输出**: 在终端区域查看命令执行结果

### 命令类型

- **batch**: 本地批处理命令，在本地环境执行
- **remote**: 远程命令，通过 SSH 在远程服务器执行

### SSH 配置

支持两种认证方式：

```yaml
# 密钥认证（推荐）
machines:
  - name: "server1"
    host: "example.com"
    port: 22
    user: "deploy"
    keyfile: "~/.ssh/id_rsa"

# 密码认证
machines:
  - name: "server2"
    host: "example.com"
    port: 22
    user: "deploy"
    password: "your-password"
```

## 开发指南

### 项目结构

```
quick-cmd/
├── app/                 # 应用主逻辑
│   ├── app.go          # 主应用结构
│   └── app_test.go     # 应用测试
├── data/               # 数据处理
│   ├── config.go       # 配置管理
│   └── config_test.go  # 配置测试
├── define/             # 数据结构定义
│   └── types.go        # 类型定义
├── machine/            # 机器管理
│   ├── ssh.go          # SSH客户端
│   └── local.go        # 本地执行器
├── frontend/           # 前端界面
│   ├── src/            # Vue.js源码
│   └── dist/           # 构建输出
├── build/              # 构建输出目录
├── config.yaml         # 配置文件
├── wails.json          # Wails配置
├── go.mod              # Go模块
└── README.md           # 说明文档
```

### 可用命令

```bash
# 开发相关
make dev          # 启动开发模式
make build        # 构建应用
make test         # 运行测试
make fmt          # 格式化代码
make lint         # 代码检查

# 依赖管理
make install-deps # 安装所有依赖
make frontend-deps # 安装前端依赖

# 其他
make clean        # 清理构建文件
make config       # 创建示例配置
```

### 添加新功能

1. **添加新的命令类型**: 在 `define/types.go` 中定义新类型，在 `machine/` 目录下实现对应的执行器
2. **扩展配置选项**: 修改 `define/types.go` 中的结构体，更新 `data/config.go` 中的处理逻辑
3. **改进界面**: 修改 `frontend/src/App.vue` 文件，添加新的组件和功能

### 故障排除

**问题**: Wails 构建失败
**解决**: 确保已安装最新版本的 Wails CLI 和所有依赖

**问题**: SSH 连接失败
**解决**: 检查机器配置中的主机地址、端口、用户名和认证信息

**问题**: 前端界面无法加载
**解决**: 确保前端依赖已正确安装，运行 `make frontend-deps`

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 更新日志

### v1.0.0

- 初始版本发布
- 支持本地和远程命令执行
- 图形化界面
- 配置文件管理
- SSH 连接支持

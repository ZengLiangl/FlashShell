# Quick Cmd 使用指南

本指南结合当前代码实现，介绍如何安装、运行、配置与高效使用 Quick Cmd。

## 🚀 安装与启动

### 环境准备

- Go 1.20+
- Node.js 16+
- Wails v2（需安装 CLI）

### 安装依赖

```bash
make install-deps
```

### 开发模式（热重载）

```bash
make dev
```

### 构建生产版本

```bash
make build
# macOS 构建后可执行示例
./build/bin/quick-cmd.app/Contents/MacOS/quick-cmd
```

### 运行模式切换（main.go）

- 前台运行（默认）：`-reg=desk`
- 后台（守护）运行：`-reg=back`（输出写入 `/tmp/quick-cmd.out/.err`）

```bash
./build/bin/quick-cmd -reg=back
```

## 📁 配置文件

应用默认读取 `config.yaml`。如果不存在，后端会创建默认配置并加载。

### 全局配置（global_config.yaml）

- 默认路径：`~/.cmd-config/global_config.yaml`
- 用途：
  - 记录最近/历史业务配置（`lastOpenedFile` / `configFile`）
  - 提供 `workPaths` 变量表用于业务配置中的 `${KEY}` 替换
  - 维护全局 `machines`，敏感信息加密到 `encrypted_data`

示例：

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
    # encrypted_data: "..."  # 通过 UI 写入并加密
```

说明：业务配置解析时会先替换 `${KEY}`，再展开 `~` 与环境变量。

### 结构示例

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
            type: "batch"    # 或 remote（远程）
            steps:
              - "go build ."

machines:
  - name: "生产服务器"
    host: "your-server.com"
    port: 22
    user: "deploy"
    keyfile: "~/.ssh/id_rsa"   # 或使用 password
```

### 多配置切换

通过“设置 > 配置文件”菜单：
- 单选切换当前配置（自动停止运行任务、清空输出、重建 Runner）
- 刷新配置列表
- 打开全局配置/当前配置

切换后后端会发送 `config:changed`，前端自动刷新。

## 🖥️ 界面与交互（App.vue）

### 主界面结构

- 左侧：项目列表 + 子项目（可执行单元）
- 右侧：终端输出（ANSI 渲染）、执行进度、状态栏

### 顶部/状态栏操作

- 停止全部：终止当前执行的子项目
- 应用信息：显示版本（例如 `Quick Cmd v1.2.0`）

### 对话框

- 机器配置：增删改查、连接测试、敏感信息加密存储
- 环境变量配置管理：增删改查、使用说明
- 关于 Quick Cmd：项目简介/版本/技术栈

菜单或快捷键触发后端事件，前端通过 `EventsOn` 监听：
- `open:machine-config`
- `open:workpath-config`
- `open:about`

## ⛓️ 执行模型

### 概念

- Project：项目分组
- SubProject：可执行单元（一次执行一个完整步骤序列）
- Command：具体步骤（含 `type` 与 `steps`）

### 类型

- batch：本地执行
- remote：通过 SSH 远程执行

### 示例

```yaml
projects:
  - name: "Go项目"
    subprojects:
      - name: "构建"
        commands:
          - name: "编译"
            type: batch
            steps: ["go build"]
          - name: "测试"
            type: batch
            steps: ["go test"]
```

点击“构建”后，会依次执行其下所有命令并在终端显示输出，右侧显示执行状态与进度。

## 🔐 远程与敏感信息

### SSH 与敏感信息（当前实现）

- 机器明文字段：`name`、`key_file`、`encrypted_data`
- 主机/端口/用户名/密码由 UI 写入并加密到 `encrypted_data`
- 远程命令通过命令的 `machine` 字段引用机器名

### 敏感信息管理

- 在“机器配置”弹框中编辑信息
- 后端通过 `SetMachineSensitiveData` 写入敏感数据并加密缓存
- 运行前可通过 `TestMachineConnection` 测试连通性

## 🌍 环境变量与路径

- 在“环境变量配置管理”中增删改查
- 配置文件中可通过 `${VAR}` 或 `~` 等方式展开（见 `data/` 逻辑）

## ⌨️ 键盘快捷键

- Cmd/Ctrl+C：复制选中文本（若无选择则复制全部终端文本）
- Cmd/Ctrl+K：清空终端输出
- Cmd/Ctrl+M：打开机器配置
- Cmd/Ctrl+E：打开环境变量配置
- Escape：关闭已打开的对话框

## 🧰 常用 Make 任务

```bash
make dev            # 开发模式（调用 dev.sh）
make build          # 构建（调用 build.sh）
make install-deps   # 安装依赖（前端 + go mod tidy）
make test           # Go 测试
make fmt            # Go fmt + 尝试前端格式化
make lint           # go vet
make clean          # 清理构建与前端产物
```

## 🐞 故障排除

- 构建失败：更新 Wails CLI、确认 Node/Go 版本
- SSH 失败：检查主机/端口/认证；前端“机器配置”可测试连接
- 界面无响应：刷新或重启；查看浏览器控制台；确认后端正常
- 切换配置未生效：使用“刷新配置列表”或切换配置；前端收到 `config:changed` 会自动刷新

## 📌 小贴士

- 用 SubProject 聚合完整流程（如 构建 → 测试 → 打包）
- 用环境变量抽离路径/凭据，避免重复
- 推荐密钥认证代替密码，提升安全性

# Quick Cmd 使用指南

## 🚀 快速开始

### 1. 启动应用

#### 开发模式（推荐用于测试）

```bash
wails dev
```

这会启动开发服务器，支持热重载，方便调试。

#### 生产模式

```bash
# 构建应用
wails build

# 运行构建的应用
./build/bin/quick-cmd.app/Contents/MacOS/quick-cmd
```

### 2. 配置项目

应用启动后会自动加载 `config.yaml` 配置文件。如果文件不存在，会自动创建一个示例配置。

#### 配置文件结构

```yaml
# 结构说明:
# - Projects: 项目分组
# - SubProjects: 可执行的项目单元 (点击执行)
# - Commands: 执行步骤序列 (按顺序自动执行)

projects:
  - name: "项目名称"
    description: "项目描述"
    workdir: "~/workspace/project-path"
    subprojects:
      - name: "可执行项目名称"
        description: "可执行项目描述"
        commands:
          - name: "命令名称"
            description: "命令描述"
            type: "batch" # 或 "remote"
            steps:
              - "命令1"
              - "命令2"

machines:
  - name: "服务器名称"
    host: "server.example.com"
    port: 22
    user: "username"
    keyfile: "~/.ssh/id_rsa" # 或使用 password: "密码"
```

## 📋 界面说明

### 左侧面板

#### 项目列表区域

- 显示所有配置的项目名称和描述
- 点击项目卡片选择项目
- 选中的项目会高亮显示
- **刷新按钮**: 刷新整个页面重新加载配置
- **调试按钮**: 显示配置加载状态和调试信息

#### 可执行项目区域

- 显示选中项目下的所有 SubProjects（可执行项目单元）
- 每个 SubProject 显示名称、描述和包含的命令数量
- **执行按钮**: 点击后按顺序执行该 SubProject 下的所有 Commands
- **停止按钮**: 停止正在运行的 SubProject

### 右侧面板

#### 终端输出区域

- 实时显示命令执行结果
- **ANSI 转义序列支持**: 完整支持终端颜色、样式和格式
- 支持 256 色和真彩色输出
- 自动识别错误和成功消息并高亮显示
- **清空按钮**: 清除所有输出
- **刷新按钮**: 手动刷新输出

#### 状态栏

- 显示当前应用状态（就绪/执行中）
- 显示正在执行的 SubProject 和当前 Command
- 显示执行进度（已完成/总数）
- **停止执行按钮**: 停止正在运行的 SubProject

## 🔧 执行模型

### 正确的理解

1. **Projects** - 项目分组（如 "Go 项目", "前端项目"）
2. **SubProjects** - **可执行的项目单元**（如 "构建", "部署", "开发"）
3. **Commands** - **执行步骤序列**（如 "编译", "测试", "清理"）

### 执行流程

1. **选择项目**: 在左上角项目列表中点击项目
2. **查看 SubProjects**: 左下角显示该项目的可执行项目单元
3. **执行 SubProject**: 点击执行按钮，系统按顺序执行所有 Commands
4. **监控进度**: 右侧终端显示实时输出和执行进度
5. **完成或停止**: 所有 Commands 执行完成，或用户手动停止

### 示例执行

假设有以下配置：

```yaml
projects:
  - name: "Go项目"
    subprojects:
      - name: "构建"
        commands:
          - name: "编译"
            steps: ["go build"]
          - name: "测试"
            steps: ["go test"]
          - name: "清理"
            steps: ["go clean"]
```

当用户点击 "构建" SubProject 时：

1. 执行 "编译" Command: `go build`
2. 执行 "测试" Command: `go test`
3. 执行 "清理" Command: `go clean`
4. 显示 "构建完成"

## 🎯 命令类型

### batch（本地命令）

在本地环境执行命令序列，适用于：

- 编译构建
- 运行测试
- 文件操作
- 本地脚本执行

```yaml
- name: "构建项目"
  type: "batch"
  steps:
    - "go mod tidy"
    - "go build -o bin/app ."
```

### remote（远程命令）

通过 SSH 在远程服务器执行命令，适用于：

- 服务部署
- 远程操作
- 服务器管理

```yaml
- name: "部署到服务器"
  type: "remote"
  machine: "生产服务器"
  steps:
    - "systemctl stop myapp"
    - "cp /tmp/app /opt/myapp/"
    - "systemctl start myapp"
```

## 🛠️ 高级配置

### 环境变量

配置文件支持环境变量和路径展开：

- `~/` 会自动展开为用户主目录
- 支持 `$HOME`、`$USER` 等环境变量

### SSH 认证

支持两种认证方式：

#### 密钥认证（推荐）

```yaml
machines:
  - name: "服务器"
    host: "example.com"
    port: 22
    user: "deploy"
    keyfile: "~/.ssh/id_rsa"
```

#### 密码认证

```yaml
machines:
  - name: "服务器"
    host: "example.com"
    port: 22
    user: "deploy"
    password: "your-password"
```

### 工作目录与变量

可以为项目和命令分别设置工作目录：

```yaml
projects:
  - name: "项目"
    workdir: "${HOME}/workspace/project" # 项目默认工作目录
    subprojects:
      - name: "子项目"
        commands:
          - name: "特殊命令"
            workdir: "${HOME}/workspace/other" # 命令特定工作目录
            type: "batch"
            steps:
              - "ls -la"
```

## 🐛 故障排除

### 配置文件无法加载

1. 检查 `config.yaml` 文件格式是否正确
2. 确保文件编码为 UTF-8
3. 点击"调试"按钮查看详细错误信息

### SSH 连接失败

1. 检查服务器地址、端口、用户名
2. 确保 SSH 密钥文件路径正确
3. 测试手动 SSH 连接是否正常

### SubProject 执行失败

1. 检查 Commands 中的命令语法是否正确
2. 确保工作目录存在且有权限
3. 查看终端输出的错误信息

### 前端界面无响应

1. 刷新页面或重启应用
2. 检查浏览器控制台是否有错误
3. 确保后端服务正常运行

## 💡 使用技巧

1. **合理分组**: 使用 Projects 来组织相关的 SubProjects
2. **步骤设计**: 将复杂的操作分解为多个 Commands
3. **快速调试**: 使用"调试"按钮快速检查配置是否正确
4. **实时监控**: 终端输出会实时显示执行进度和结果
5. **安全管理**: 建议使用 SSH 密钥而不是密码进行远程连接

## 📝 配置示例

查看 `config.example.yaml` 文件获取完整的配置示例。

## 🔄 更新配置

修改 `config.yaml` 文件后，点击"刷新"按钮即可重新加载配置，无需重启应用。

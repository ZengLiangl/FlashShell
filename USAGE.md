# Quick Cmd 使用指南

本文档介绍 Quick Cmd 的安装、配置、界面操作与常见场景，内容与当前代码实现保持一致。

## 目录

- [安装与启动](#安装与启动)
- [配置体系](#配置体系)
- [界面与操作](#界面与操作)
- [执行模型](#执行模型)
- [命令类型与特殊命令](#命令类型与特殊命令)
- [远程机器与敏感信息](#远程机器与敏感信息)
- [环境变量与路径](#环境变量与路径)
- [键盘快捷键](#键盘快捷键)
- [性能说明](#性能说明)
- [故障排除](#故障排除)
- [配置示例](#配置示例)

---

## 安装与启动

### 环境准备

| 依赖 | 版本 |
|------|------|
| Go | 1.23+ |
| Node.js | 16+ |
| Wails CLI | v2 |

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
make install-deps
```

Windows 用户请确保已安装 WebView2 运行时。

### 开发模式

```bash
make dev
# 等价于 ./dev.sh → wails dev
```

支持前端热重载，适合调试配置与界面。

### 构建与运行

```bash
make build
# Windows: build/bin/quick-cmd.exe
# macOS:   build/bin/quick-cmd.app
# Linux:   build/bin/quick-cmd
```

### 运行模式

| 参数 | 说明 |
|------|------|
| `-reg=desk` | 前台运行（默认） |
| `-reg=back` | 后台守护（非 Windows） |

后台模式日志：`/tmp/quick-cmd.out`、`/tmp/quick-cmd.err`。

---

## 配置体系

Quick Cmd 将配置分为**全局配置**和**业务配置**两层。

### 全局配置 `global_config.yaml`

**路径（固定）**：

```
Windows: C:\Users\<用户名>\.cmd-config\global_config.yaml
macOS:   ~/.cmd-config/global_config.yaml
Linux:   ~/.cmd-config/global_config.yaml
```

> 全局配置保存在用户主目录，**不是**项目目录下的文件。在应用菜单中可通过「设置 → 配置文件 → 打开全局配置」直接打开。

**用途**：

- `windowsName`：窗口标题
- `configFile` / `lastOpenedFile`：业务配置文件历史与当前选中项
- `workPaths`：路径变量表，供业务配置 `${KEY}` 引用
- `machines`：远程机器清单（敏感信息加密存储）

**默认示例**（仅首次创建文件不存在时生成）：

```yaml
appId: com.runner
windowsName: "运行器"
configFile:
  - config.yaml
lastOpenedFile: config.yaml
workPaths:
  HOME: ~
machines: []
```

**保护策略**：

- 文件已存在且有内容时，启动**不会**覆盖为默认配置
- `lastOpenedFile` 仅在路径实际变化时才写回磁盘

### 业务配置 `config.yaml`

定义要执行的项目、子项目、命令步骤。可准备多个文件，通过菜单切换。

**层级结构**：

```
Project（项目）
 └── SubProject（子项目，可执行单元）
      └── Command（命令，含 type 与 steps）
           └── steps[]（具体 shell 步骤）
```

**最小示例**：

```yaml
projects:
  - name: 我的项目
    description: 本地 Go 项目
    workdir: "${HOME}/workspace/myapp"
    subprojects:
      - name: 构建
        description: 编译并测试
        commands:
          - name: 编译
            type: batch
            steps:
              - go build .
          - name: 测试
            type: batch
            steps:
              - go test ./...

      - name: 部署测试
        commands:
          - name: 上传并重启
            type: remote
            machine: test-server
            steps:
              - upload ./bin/app /opt/app/app
              - systemctl restart myapp
```

### 工作目录优先级

命令执行时，`workdir` 按以下优先级生效（高 → 低）：

1. `Command.workdir`
2. `SubProject.workdir`
3. `Project.workdir`

### 路径与变量展开

解析业务配置时，处理顺序为：

1. 用全局 `workPaths` 替换 `${KEY}`
2. 展开 `~/` 为用户主目录
3. 展开 `$VAR` 等操作系统环境变量

### 多配置切换

菜单 **设置 → 配置文件**：

| 操作 | 行为 |
|------|------|
| 单选切换 | 停止当前任务、清空终端、加载新配置 |
| 刷新配置列表 | 重新扫描并刷新菜单 |
| 打开全局配置 | 用系统默认编辑器打开 `global_config.yaml` |
| 打开当前配置 | 打开当前业务配置文件 |

切换后前端收到 `config:changed` 事件并自动刷新。

---

## 界面与操作

### 主界面布局

```
┌─────────────────────┬──────────────────────────────┐
│  项目列表（全屏）     │                              │
│  或                  │   终端输出（ANSI + 虚拟滚动）  │
│  子项目列表 + 执行按钮 │   进度条 / 状态               │
├─────────────────────┴──────────────────────────────┤
│  状态栏：运行状态 / 停止全部 / 版本信息              │
└────────────────────────────────────────────────────┘
```

### 基本流程

1. 启动应用，自动加载上次打开的业务配置
2. 在项目列表中点击项目，进入子项目视图
3. 点击子项目「执行」，按顺序运行其下所有 Command
4. 右侧终端实时显示输出；底部状态栏显示进度
5. 点击「停止」可中断**本地** batch 任务（远程 SSH 会话暂不支持强杀）

### 终端行为

- 新任务开始时**自动清空**终端
- 输出通过 Wails 事件推送（`output:line`），无定时轮询
- 虚拟滚动：大量日志时只渲染可见区域，降低卡顿
- **粘底跟随**：在底部附近时自动滚到最新行；向上翻看历史时不强制跳回

### 对话框

| 入口 | 功能 |
|------|------|
| 设置 → 机器配置（Ctrl+M） | 增删改机器、测试 SSH、管理密钥 |
| 设置 → 环境变量（Ctrl+E） | 管理 `workPaths` 变量 |
| 帮助 → 关于 | 版本与简介 |

### 应用菜单

- **文件 → 新建窗口**：启动新的应用进程
- **设置 → 配置文件**：切换/刷新/打开配置
- **设置 → 机器配置 / 环境变量**

---

## 执行模型

### 概念对照

| 概念 | 说明 | 用户操作 |
|------|------|----------|
| Project | 项目分组 | 点击进入 |
| SubProject | 一次完整执行单元 | 点击「执行」 |
| Command | 一组步骤（batch 或 remote） | 自动顺序执行 |
| Step | 单条 shell 命令 | 在配置中编写 |

### 执行过程

1. 点击「执行」→ 重读配置 → 清空终端
2. 按 SubProject 内 Command 顺序依次执行
3. 每个 Command 内 steps 逐步执行
4. 状态通过 `execution:status` 事件推送到前端
5. 全部完成或出错后停止，`IsRunning` 变为 false

### batch 与 remote 对比

| 类型 | 执行位置 | 适用场景 |
|------|----------|----------|
| `batch` | 本机 `cmd /C`（Windows）或 `bash -c` | Maven/Gradle/npm 构建、本地脚本 |
| `remote` | SSH 到指定 `machine` | 上传文件、重启服务、Docker 操作 |

---

## 命令类型与特殊命令

### batch（本地）

在本地 shell 中执行，工作目录由上述优先级决定。

```yaml
- name: 打包
  type: batch
  steps:
    - mvn clean package -DskipTests
    - echo 构建完成
```

Windows 下通过隐藏窗口的 `cmd /C` 执行；停止时会 `taskkill /T /F` 结束进程树。

### remote（远程）

通过 SSH 连接 `machine` 字段指定的机器执行步骤。

```yaml
- name: 重启服务
  type: remote
  machine: jz
  steps:
    - docker restart auth-service gateway
```

- 普通 shell 步骤：创建 SSH Session 执行
- 仅当步骤包含 `upload` 时才建立 SFTP 连接（纯命令如 `docker restart` 不建 SFTP，连接更快）

### 远程特殊命令

在 `remote` 类型的 steps 中可使用以下内置命令：

| 命令 | 格式 | 说明 |
|------|------|------|
| `upload` | `upload <本地路径> <远程路径>` | 上传文件或目录（目录先 zip 再传） |
| `targz` | `targz <源目录> <目标.tar.gz>` | 本地打包（特殊场景） |
| `chdir` | `chdir <远程目录>` | 切换远程工作目录 |

**upload 示例**：

```yaml
steps:
  - upload D:\build\app.jar /home/app/app.jar
  - upload D:\build\dist /usr/share/nginx/html/app
  - docker restart nginx
```

上传目录时会自动压缩为 zip、传到远程临时目录、解压后删除压缩包，并显示传输进度。

---

## 远程机器与敏感信息

### 配置方式

机器信息保存在**全局配置**的 `machines` 数组中，通过 UI「机器配置」管理。

磁盘上可见字段：

```yaml
machines:
  - name: jz
    key_file: C:\Users\ll\.ssh\id_rsa
    encrypted_data: "..."   # UI 写入后自动生成，勿手动编辑
```

主机、端口、用户名、密码通过 UI 填写，加密后写入 `encrypted_data`，**不以明文落盘**。

### 认证方式

- **密钥认证**：填写 `key_file` 路径（支持 `~/.ssh/id_rsa`）
- **密码认证**：在 UI 中填写密码（加密存储）
- 可同时配置，连接时两种都尝试

### 连接测试

在「机器配置」列表中点击「测试连接」，后端执行 `echo 'connection test'` 验证 SSH 可达。

---

## 环境变量与路径

在「环境变量配置管理」中维护键值对，例如：

| 键 | 值 |
|----|-----|
| `HOME` | `C:\Users\ll` |
| `ACC-CLOUD` | `D:\IdeaProjects\acc-cloud` |
| `MVM` | `mvn` |

业务配置中引用：

```yaml
workdir: "${ACC-CLOUD}"
steps:
  - mvn package -pl my-module -am
  - upload ${ACC-CLOUD}\target\app.jar /opt/app/app.jar
```

---

## 键盘快捷键

| 快捷键 | 功能 |
|--------|------|
| Ctrl/Cmd + C | 复制终端选中文本；无选中则复制全部 |
| Ctrl/Cmd + K | 清空终端 |
| Ctrl/Cmd + M | 打开机器配置 |
| Ctrl/Cmd + E | 打开环境变量配置 |
| Ctrl/Cmd + R | 刷新配置列表（菜单） |
| Escape | 关闭已打开对话框 |

---

## 性能说明

### 输出机制

- 后端通过 `output:line`、`execution:status` 事件推送，**已移除前端轮询**
- 本地输出通道满时非阻塞丢弃，避免卡死
- 执行新任务时自动清空终端，避免历史 DOM 堆积

### Windows 使用建议

| 场景 | 建议 |
|------|------|
| Maven/Gradle 构建 | 限制并行度（如 `-T 2`），日常可去掉 `clean` |
| 杀毒软件占用高 | 将项目目录、`.m2` 仓库加入 Defender 排除项 |
| 远程 docker restart | 本身不占本地 CPU；等待期间 UI 应保持流畅 |
| 终端历史过多 | 新任务会自动清空；也可手动 Ctrl+K |

---

## 故障排除

### 配置文件无法加载

- 检查 YAML 缩进与编码（UTF-8）
- 确认 `workdir` 引用的 `${KEY}` 在全局 `workPaths` 中已定义
- 菜单「刷新配置列表」后重试

### 全局配置丢失或被改写

- 确认编辑路径为 `~/.cmd-config/global_config.yaml`
- 应用不会在启动时覆盖已有内容的全局配置
- 仅 UI 主动修改（机器/环境变量/切换配置）时才会写盘

### SSH 连接失败

1. 机器配置中点击「测试连接」
2. 检查 host、port、user、密钥路径
3. 确认密钥权限（Linux/macOS 上通常为 600）
4. 手动 `ssh user@host` 验证网络

### 本地命令执行失败

- Windows 下命令通过 `cmd /C` 执行，复杂管道建议写成 `.bat` 或拆分步骤
- 检查 `workdir` 是否存在
- 查看终端 `[STDERR]` 行

### 远程 upload 失败

- 确认步骤以 `upload` 开头（此时才会建立 SFTP）
- 检查本地路径与远程目录权限
- 大目录上传会先 zip，留意磁盘空间

### 界面无响应

- 确认使用最新构建（含事件推送版本）
- 菜单刷新配置或重启应用
- Windows 任务管理器查看 `msedgewebview2` 占用

---

## 配置示例

### 本地构建 + 远程发布

```yaml
projects:
  - name: XYJ
    workdir: "${ACC-CLOUD}"
    subprojects:
      - name: 发布测试【merchant】
        commands:
          - name: 打包【merchant】
            type: batch
            steps:
              - mvn package -DskipTests -P dev -pl merchant-service -am
          - name: 发布测试【merchant】
            type: remote
            machine: jz
            steps:
              - upload ${ACC-CLOUD}\merchant-service\target\app.jar /opt/merchant/app.jar
              - docker restart merchant-service

      - name: 发布测试【auth&gateway】
        commands:
          - name: 重启【auth&gateway】
            type: remote
            machine: jz
            steps:
              - docker restart auth-service gateway
```

### 纯本地脚本

```yaml
projects:
  - name: 工具
    subprojects:
      - name: 启动测试环境
        commands:
          - name: 启用服务
            type: batch
            steps:
              - curl https://example.com/api/setServers?status=1
```

---

## Make 任务速查

```bash
make dev            # 开发模式
make build          # 构建
make install-deps   # 安装依赖
make test           # Go 测试
make fmt            # 格式化
make lint           # 静态检查
make clean          # 清理
make config         # 复制 config.example.yaml → config.yaml（若示例存在）
```

---

如有问题，请先查看终端输出与全局配置路径是否正确，再对照本文档排查。

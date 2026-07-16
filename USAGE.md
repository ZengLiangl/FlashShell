# FlashDock（闪舵）使用指南

本文档与当前代码行为对齐，覆盖安装、配置、任务模式、Shell 模式与系统设置。

## 目录

- [安装与启动](#安装与启动)
- [配置体系](#配置体系)
- [界面总览](#界面总览)
- [任务模式](#任务模式)
- [Shell 模式](#shell-模式)
- [系统设置中心](#系统设置中心)
- [执行模型](#执行模型)
- [命令类型与特殊命令](#命令类型与特殊命令)
- [远程机器与敏感信息](#远程机器与敏感信息)
- [环境变量与路径](#环境变量与路径)
- [键盘快捷键](#键盘快捷键)
- [故障排除](#故障排除)
- [配置示例](#配置示例)

---

## 安装与启动

### 环境准备

| 依赖 | 版本 |
|------|------|
| Go | 1.23+ |
| Node.js | 16+（推荐 20） |
| Wails CLI | v2 |

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
cd frontend && npm install && cd ..
go mod tidy
```

### 开发 / 构建

```bash
wails dev     # 热重载
wails build   # 产出在 build/bin/
```

### 运行模式

| 参数 | 说明 |
|------|------|
| `-reg=desk` | 前台（默认） |
| `-reg=back` | 后台守护 |

后台日志：`/tmp/FlashDock.out`、`/tmp/FlashDock.err`。

---

## 配置体系

FlashDock 将配置分为全局与业务两层，并单独存放快捷键。

### 全局配置 `~/.flashdock/global_config.yaml`

在顶部图标栏 **配置文件** 菜单中可「打开全局配置」。

用途包括：窗口标题、业务配置文件列表、`workPaths`、机器清单、主题与日志落盘等。

文件已存在时启动**不会**覆盖；仅通过 UI / 主动写盘时更新。

### 快捷键 `~/.flashdock/shortcuts.json`

系统设置 → **快捷键** 中修改并保存。Mac 展示 `Command+…`，Windows / Linux 展示 `Ctrl+…`。

### 业务配置 `config.yaml`

定义项目与可执行流水线，可多文件切换。

```
Project
 └── SubProject（点击「执行」）
      └── Command（batch / remote）
           └── steps[]
```

最小示例见文末「配置示例」。

### 工作目录优先级

1. `Command.workdir`  
2. `SubProject.workdir`  
3. `Project.workdir`  

解析时：先替换 `${KEY}` → 展开 `~/` → 展开系统环境变量。

---

## 界面总览

### 顶部图标栏

仅显示图标（悬停看提示）：

| 图标 | 功能 |
|------|------|
| 文件 | 新建窗口 |
| 配置文件 | 切换 / 刷新业务配置，打开全局或当前配置 |
| 系统设置 | 打开统一设置中心（左右分栏） |
| 帮助 | 关于 FlashDock |

### 首页

左右两大入口：

- **任务模式**：选择项目进入执行视图  
- **Shell 模式**：点机器直连，或「进入 Shell 终端」；有会话时可随时返回首页  

任务与 Shell **可并行**：执行流水线时仍可连 SSH；返回首页不会强制断开会话。

---

## 任务模式

1. 首页点击项目卡片  
2. 左侧选择子项目 → **执行**  
3. 右侧终端实时输出；状态栏可停止本地任务  

终端支持搜索（快捷键「查找」）、清空、粘底跟随。远程 SSH 步骤暂不支持从前端强杀。

---

## Shell 模式

- 多会话 Tab，可同时连接多台机器  
- 交互式 xterm 终端；右键：**复制 / 粘贴 / 查找 / 清空缓存**  
- 底部 SFTP：目录浏览、上传感知（任务 upload）、用户:组显示名解析  
- Tab 栏可 **返回首页**（会话保留）  
- 左侧监控面板可折叠  

`cd` 输入会乐观同步 SFTP 路径；若远端 shell 自带 OSC 7 cwd 上报也会自动对齐。

---

## 系统设置中心

点击齿轮图标打开，左侧导航：

| 分区 | 内容 |
|------|------|
| 机器配置 | 增删改、测连、导入 Xshell / FinalShell |
| 环境变量 | 管理 `${KEY}` 替换表 |
| 系统设置 | 全局 SSH 帐号、日志落盘、主题、Shell 字号 / 行高、会话 ID |
| 快捷键 | 录制 / 重置 / 保存到 `shortcuts.json` |
| 执行历史 | 查看与打开执行日志 |

机器列表按名称首字母排序：`a–z` → `0–9` → 其它。

---

## 执行模型

1. 点击「执行」→ 重读配置 → 清空任务终端  
2. 按 SubProject 内 Command 顺序执行  
3. Command 内 steps 逐步执行  
4. 状态经 Wails 事件推送到前端  

| type | 位置 | 场景 |
|------|------|------|
| `batch` | 本机 shell | 构建、测试 |
| `remote` | SSH 到 `machine` | 发布、重启、Docker |

仅步骤含 `upload` 时才建立 SFTP。

### 远程特殊命令

| 命令 | 格式 | 说明 |
|------|------|------|
| `upload` | `upload <本地> <远程>` | 文件或目录（目录先打包） |
| `targz` | `targz <源> <目标.tar.gz>` | 本地打包 |
| `chdir` | `chdir <远程目录>` | 切换远程工作目录 |

---

## 远程机器与敏感信息

机器保存在全局配置 `machines` 中，主机 / 端口 / 用户 / 密码经 UI 加密写入，不以明文落盘。

- 密钥：`key_file`（如 `~/.ssh/id_rsa`）  
- 密码：UI 填写，加密存储  
- 可配置全局 SSH 帐号，添加机器时一键填充  
- 「测试连接」验证可达性  

---

## 环境变量与路径

在系统设置 → 环境变量中维护，例如：

| 键 | 值 |
|----|-----|
| `WORKSPACE` | `~/workspace` |
| `PROJECT_ROOT` | `${WORKSPACE}/demo-api` |

业务配置：

```yaml
workdir: "${PROJECT_ROOT}"
steps:
  - upload ${PROJECT_ROOT}/target/app.jar /opt/app/app.jar
```

---

## 键盘快捷键

默认（修饰键按系统显示为 Command 或 Ctrl）：

| 功能 | 默认 |
|------|------|
| 新建窗口 | Mod+N |
| 机器配置 | Mod+M |
| 连接管理器 | Mod+E |
| 环境变量 | Mod+U |
| 系统设置 | Mod+, |
| 刷新配置列表 | Mod+R |
| 查找 | Mod+F |
| 复制 | Mod+C |
| 清空输出 | Mod+K |

均可在 **系统设置 → 快捷键** 中修改。焦点在输入框时默认不触发全局快捷键。

---

## 故障排除

### 配置无法加载

- 检查 YAML 缩进与 UTF-8  
- 确认 `${KEY}` 已在环境变量中定义  
- 配置文件菜单中「刷新配置列表」  

### SSH 失败

1. 机器配置中「测试连接」  
2. 核对 host / port / 用户 / 密钥权限（`chmod 600`）  
3. 本机 `ssh user@host` 验证  

### 机器列表空白

关闭系统设置后重新从首页打开；确认全局配置中确有 `machines` 数据。

### upload 失败

- 步骤需以 `upload` 开头  
- 检查本地路径与远端写权限  

---

## 配置示例

### 本地构建 + 远程发布

```yaml
projects:
  - name: sample-platform
    workdir: "${WORKSPACE}/sample-platform"
    subprojects:
      - name: 构建用户服务
        commands:
          - name: 打包
            type: batch
            steps:
              - mvn package -DskipTests -pl user-service -am
          - name: 发布到测试环境
            type: remote
            machine: staging-server
            steps:
              - upload ${WORKSPACE}/sample-platform/user-service/target/user-service.jar /opt/user-service/app.jar
              - docker restart user-service
```

### 纯本地脚本

```yaml
projects:
  - name: dev-tools
    subprojects:
      - name: 启用 mock
        commands:
          - name: 切换状态
            type: batch
            steps:
              - curl -s https://mock.example.com/api/v1/status?enabled=1
```

---

## 命令速查

```bash
wails dev
wails build
cd frontend && npm install && cd ..
go mod tidy
go test ./...
```

---

品牌：**FlashDock · 闪舵** — 任务与 Shell，一港调度。

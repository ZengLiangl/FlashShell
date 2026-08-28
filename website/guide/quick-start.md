# 快速开始

5 分钟跑通：安装 → 加机器 → 配变量 → 任务或 Shell 开工。

```text
① 添加机器 → ② 配置环境变量 → ③ 编辑 config.yaml → ④ 任务流水线 / Shell 登舰
```

## 前置准备

- 本机：Windows / macOS / Linux
- 至少一台可 SSH 登录的 Linux（或只用本机 PTY 也行）
- （可选）已有 Xshell / FinalShell 会话，可直接导入

## Step 1：安装 FlashShell

从 [下载页](/download) 或 [Releases](https://github.com/ZengLiangl/FlashShell/releases) 拿到安装包，安装后启动。

首次启动：

- 当前目录若无业务配置，生成示例 `config.yaml`
- 全局数据写入 `~/.flashshell/`

## Step 2：添加第一台机器

顶部 **系统设置 → 机器配置**：

1. 新增服务器（主机、端口、用户、密码或私钥）
2. 点「测试连接」
3. 首次连接确认 Host Key 指纹

> **Pro Tip：** 已有 Xshell / FinalShell？直接一键导入，跳过手工录入。

## Step 3：配置环境变量

在 **环境变量** 中设置常用路径，例如：

- `WORKSPACE`
- `PROJECT_ROOT`

业务 YAML 里用 `${WORKSPACE}` 引用，执行前自动替换。

## Step 4：选模式开工

| 模式 | 适合 | 入口 |
| --- | --- | --- |
| 任务模式 | 发布、构建、批量远程步骤 | 首页进入任务，点 SubProject「执行」 |
| Shell 模式 | 交互运维、分屏、广播、SFTP | 首页进入 Shell，开 Tab / 分屏 |

两边可以同时开着：左边流水线，右边终端。

## 接下来

<div class="cta-row">
  <a class="primary" href="/features/shell">Shell 工作台 →</a>
  <a href="/features/tasks">任务流水线 →</a>
  <a href="/features/security">安全与配置 →</a>
</div>

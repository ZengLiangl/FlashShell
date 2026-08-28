---
layout: home

hero:
  name: FlashShell
  text: 多会话 SSH / SFTP 桌面终端
  tagline: YAML 任务流水线 × 多会话 Shell × 本地 PTY。把构建、发布、联调、登舰运维收进同一个桌面港——任务与终端并行，互不抢舵。
  image:
    src: /logo.png
    alt: FlashShell
  actions:
    - theme: brand
      text: 立即开始
      link: /guide/quick-start
    - theme: alt
      text: 下载客户端
      link: /download
    - theme: alt
      text: 为什么用 FlashShell
      link: /why

features:
  - title: 真正的多会话 Shell
    details: SSH + 本机 PTY 多 Tab 并行；分屏最多 4 窗格；一键广播群发命令；xterm.js 搜索、粘底、右键菜单。
    link: /features/shell
    linkText: 了解详情
  - title: YAML 任务流水线
    details: Project → SubProject → Command → Step 四级编排；本地 batch 与远程 remote 自由混排；retry / on_fail；${ENV} 变量替换。
    link: /features/tasks
    linkText: 了解详情
  - title: 内置 SFTP
    details: Shell 底部文件面板，拖拽上传下载；任务侧 upload 步骤；无需再开第二个客户端。
    link: /features/sftp
    linkText: 了解详情
  - title: SSH 隧道
    details: 本地端口转发与动态 SOCKS；代理贯通 SSH / SFTP / 更新请求，适合内网与跳板场景。
    link: /features/tunnel
    linkText: 了解详情
  - title: 任务与终端并行
    details: 左侧跑流水线、右侧登舰排查；v-show 切换不断会话，发布失败立刻 SSH 进去看日志。
    link: /why
    linkText: 为什么与众不同
  - title: 安全与导入
    details: 密码 / 密钥加密落盘；Host Key 信任管理；一键导入 Xshell / FinalShell；HTTP / SOCKS 代理。
    link: /features/security
    linkText: 了解详情
---

## 一眼看懂 FlashShell

```text
                         ╔════════════════ FlashShell ════════════════╗
                         ║                                           ║
    config.yaml ────────►║   任务模式          Shell 模式            ║◄──── SSH / SFTP / PTY
    本地 & 远程混排       ║   一键流水线        多 Tab · 分屏 · 广播    ║      本机终端
                         ║   实时 ANSI 日志    SFTP · 监控 · 隧道      ║
                         ║                                           ║
                         ╚══════════════════ 同港出海 ═════════════════╝
```

统一窗口管理机器、终端、文件与发布流水线——不再在 Xshell、SFTP 客户端和脚本目录之间来回切。

## 三件事让 FlashShell 与众不同

1. **任务与 Shell 同港并行。** 左边 YAML 流水线在跑，右边 SSH 已经登着；失败不用另开窗口，直接在同一应用里排查。
2. **配置分层、变量可替换。** 业务 `config.yaml` 与 `~/.flashshell/` 全局配置分离；`${KEY}` 从环境变量表展开，团队可复用同一套流水线。
3. **桌面原生，跨平台。** Go + Wails v2 后端，Vue 3 + xterm.js 前端；Windows / macOS / Linux 同一套体验。

## 用 FlashShell 一天的真实样子

| 时段 | 你在做什么 | FlashShell 帮你做什么 |
| --- | --- | --- |
| 早上 | 把测试环境发布到三台机 | 任务模式跑 remote + upload，ANSI 日志实时回传 |
| 中午 | 某台机器发布失败 | 切到 Shell，对同一台机器开 Tab，直接看日志 / 改配置 |
| 下午 | 批量改 nginx 配置并 reload | 分屏 + 广播，一条命令同步到多窗格 |
| 晚上 | 拉日志、传包、开隧道联调 | SFTP 面板拖文件；本地转发连上内网服务 |

## 技术栈

| 层 | 选型 | 职责 |
| --- | --- | --- |
| 桌面壳 | Go 1.23+ · Wails v2 | 原生窗口、系统 API、并发执行 |
| 舰桥 UI | Vue 3 · Element Plus · xterm.js | 任务 / Shell 双模式界面 |
| 执行引擎 | SSH · SFTP · PTY | 本地命令、远程步骤、文件传输 |
| 数据层 | YAML · `~/.flashshell/` | 业务配置、机器、主题、快捷键 |

## 立即开始

<div class="cta-row">
  <a class="primary" href="/guide/quick-start">5 分钟快速上手 →</a>
  <a href="/download">下载安装包 →</a>
  <a href="/features/shell">浏览功能 →</a>
</div>

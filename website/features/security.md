# 安全与配置

FlashShell 把「业务流水线」和「本机凭据 / 机器清单」分层存放，敏感字段加密，Host Key 可信任管理。

## 配置分层

| 配置 | 路径 | 作用 |
| --- | --- | --- |
| 业务配置 | `config.yaml`（可多文件切换） | 项目与可执行流水线，适合进仓库 |
| 全局配置 | `~/.flashshell/global_config.yaml` | 窗口、机器、主题、代理、`workPaths` |
| 快捷键 | `~/.flashshell/shortcuts.json` | 系统级快捷键 |

全局配置**仅在文件不存在时**写默认值，不会静默覆盖你已有内容。

## 凭据与 Host Key

- SSH 密码 / 私钥 **加密落盘**，列表不以明文展示
- Host Key 信任库：`~/.flashshell/known_hosts.json`
- 首次连接确认指纹；变更时拒绝，降低中间人风险

## 代理

系统设置中可配置 HTTP / SOCKS 代理，贯通：

- SSH 连接
- SFTP
- 应用更新请求

## 导入现有会话

机器配置支持 **Xshell / FinalShell** 一键导入，减少迁移成本。

## 体验相关

- 亮 / 暗 / 跟随系统主题，终端配色可定制
- 快捷键可录制、可重置（macOS 展示 ⌘，Windows / Linux 展示 Ctrl）
- 多窗口、Shell 内存节省模式
- 应用内检查更新 / 下载安装包

## 相关页面

- [快速开始](/guide/quick-start)
- [常见问题](/faq)
- [下载](/download)

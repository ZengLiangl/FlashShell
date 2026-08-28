# 常见问题

## 安装与启动

### Windows 提示未知发布者？

安装包若尚未做 EV 代码签名，系统可能弹警告。可在受信任来源（官方 Releases）确认后继续安装。

### 全局配置会被覆盖吗？

不会。`~/.flashshell/global_config.yaml` **仅在文件不存在时**写入默认值；之后只通过 UI 保存或你主动改文件更新。

### 业务配置和全局配置有什么区别？

| 配置 | 路径 | 作用 |
| --- | --- | --- |
| 业务配置 | `config.yaml`（可多文件切换） | 项目 / 流水线 |
| 全局配置 | `~/.flashshell/global_config.yaml` | 机器、主题、代理、环境变量 |
| 快捷键 | `~/.flashshell/shortcuts.json` | 快捷键绑定 |

## 连接与安全

### SSH 连不上？

1. 系统设置 → 机器配置 →「测试连接」
2. 检查用户名 / 端口 / 密钥权限（私钥建议 `chmod 600`）
3. 若走代理，确认 HTTP / SOCKS 已启用并测试连通

### Host Key 怎么管理？

首次连接会确认指纹，信任记录保存在 `~/.flashshell/known_hosts.json`。指纹变更时会拒绝连接以防中间人。

### 密码会明文出现在界面吗？

敏感字段加密落盘，列表侧不以明文展示。

## 任务与 Shell

### 任务跑着的时候能开 Shell 吗？

可以。任务模式与 Shell 模式并行挂载，切换视图不会断掉已有会话。

### 远程步骤能中断吗？

可以。停止会关闭活跃 SSH session，并中止步骤执行循环。

### 变量 `${KEY}` 从哪来？

来自全局配置里的环境变量表（如 `workPaths`）。业务 YAML 里写 `${WORKSPACE}` 会在执行前展开。

## 导入与快捷键

### 能导入 Xshell / FinalShell 吗？

可以。在机器配置里一键导入现有会话，减少手工录入。

### 快捷键不生效？

到系统设置 → 快捷键保存后再试。输入框聚焦时默认不抢快捷键，避免和编辑冲突。

## 还有问题？

- 仓库手册：[USAGE.md](https://github.com/ZengLiangl/FlashShell/blob/main/USAGE.md)
- Issues：[GitHub Issues](https://github.com/ZengLiangl/FlashShell/issues)

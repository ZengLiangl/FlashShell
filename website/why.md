# 为什么用 FlashShell

传统桌面运维往往要同时开三样东西：终端客户端、SFTP 工具、一堆散落的 shell / bat 脚本。FlashShell 把它们收进**同一个桌面港**。

## 和传统做法比

| | 传统做法 | FlashShell |
| --- | --- | --- |
| 发布脚本 | 散落各处的 shell / bat | **YAML 编排**，图形化一键执行 |
| 远程运维 | 另开 Xshell / iTerm | **内置多会话 Shell**，Tab / 分屏 / 广播 |
| 文件传输 | 再开一个 SFTP 客户端 | **任务 upload** + Shell 侧 SFTP 面板 |
| 配置管理 | 环境变量靠记忆 | **全局变量表** + `${KEY}` 自动替换 |
| 任务与终端 | 二选一，来回切窗口 | **并行不互斥**，左跑任务右登舰 |

## 适合谁

- 需要频繁 SSH 上多台 Linux，又经常做发布 / 联调的开发与运维
- 想把「构建 → 上传 → 远程执行」固化成可重复流水线
- 已有 Xshell / FinalShell 会话，希望一键导入继续用
- 需要本机终端与远程终端混排在同一窗口

## 不适合谁

- 只需要纯云端 Web 终端、且从不在本机工作
- 需要企业级审计 / MCP / AI 审批队列（那是另一类产品定位）
- 只想用浏览器管理服务器、不要桌面安装包

## 设计原则

1. **会话优先**：多 Tab、分屏、广播是主路径，任务流水线是增强，不是取代终端。
2. **配置可版本化**：业务 YAML 可进仓库；凭据与机器清单落在本机全局目录并加密。
3. **切换不断线**：任务视图与 Shell 视图并行挂载，切回去会话还在。

## 下一步

<div class="cta-row">
  <a class="primary" href="/guide/quick-start">快速开始 →</a>
  <a href="/download">下载 →</a>
  <a href="https://github.com/ZengLiangl/FlashShell">GitHub →</a>
</div>

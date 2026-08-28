# SFTP 文件管理

不用再开第二个文件客户端：Shell 底部面板 + 任务 `upload` 步骤覆盖日常传文件。

## Shell 侧面板

- 浏览远程目录
- 上传 / 下载文件
- 与当前 SSH 会话绑定，上下文清晰

## 任务侧 upload

在远程 Command 步骤里用内置 `upload`，把构建产物推进目标机，再继续跑部署命令——整条链路写在同一份 YAML。

## 适用场景

| 场景 | 建议路径 |
| --- | --- |
| 临时改配置、拉日志 | Shell SFTP 面板 |
| 发布包固定路径上传 | 任务 `upload` 步骤 |
| 大批量目录同步 | 任务编排 + 多次 upload / 压缩包 |

## 相关页面

- [Shell 工作台](/features/shell)
- [任务流水线](/features/tasks)

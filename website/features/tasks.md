# 任务流水线

用 YAML 把「本地构建 → 上传 → 远程执行」编排成可重复流水线，图形化一键跑。

## 编排模型

```text
Project（项目）
 └── SubProject（子项目 · 点击「执行」）
      └── Command（命令组 · batch / remote）
           └── steps[]（步骤列表）
```

- **batch**：本机执行
- **remote**：SSH 到指定机器执行
- 同一流水线里本地与远程步骤可混排

## 执行能力

- 步骤级 **retry / on_fail** 失败策略
- `${ENV}` 全局变量替换（来自环境变量表）
- 实时进度与 ANSI 彩色日志回传
- 本地任务可中断；远程步骤也可停止活跃 session
- 内置远程特殊命令：`upload` / `targz` / `chdir`

## 工作目录解析优先级

1. `Command.workdir`
2. `SubProject.workdir`
3. `Project.workdir`

解析流程：`${KEY}` 替换 → 展开 `~/` → 展开系统环境变量。

## 和 Shell 的配合

发布失败时，切到 Shell 打开对应机器，无需另开客户端。任务是「固化流程」，Shell 是「临场处理」。

## 相关页面

- [快速开始](/guide/quick-start)
- [SFTP 文件管理](/features/sftp)
- [安全与配置](/features/security)

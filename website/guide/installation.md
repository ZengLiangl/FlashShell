# 安装与构建

## 下载安装（推荐）

1. 打开 [GitHub Releases](https://github.com/ZengLiangl/FlashShell/releases)
2. 下载对应平台安装包或压缩包
3. 安装 / 解压后启动 **FlashShell**

详见 [下载](/download)。

## 从源码构建

| 依赖 | 版本 |
| --- | --- |
| Go | 1.23+ |
| Node.js | 16+（推荐 20） |
| Wails CLI | v2 |

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
git clone https://github.com/ZengLiangl/FlashShell.git && cd FlashShell
cd frontend && npm install && cd ..
go mod tidy

wails dev      # 开发 · 热重载
wails build    # 产出在 build/bin/
```

## 运行参数

| 参数 | 说明 |
| --- | --- |
| `-reg=desk` | 前台运行（默认） |
| `-reg=back` | 后台守护进程 |

后台日志路径（类 Unix）：`/tmp/FlashShell.out`、`/tmp/FlashShell.err`。

## 全局数据目录

```text
~/.flashshell/
├── global_config.yaml    # 机器、主题、代理、环境变量…
├── shortcuts.json        # 快捷键绑定
├── shell_history.json    # Shell 连接历史
└── known_hosts.json      # SSH Host Key 信任库
```

## 常用命令

```bash
wails dev                          # 开发模式（热重载）
wails build                        # 发布构建
go test ./...                      # 运行测试
go fmt ./... && go vet ./...       # 格式化 & 静态检查
```

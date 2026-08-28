# 下载

从 GitHub Releases 获取对应平台安装包。

## 官方发布页

**[github.com/ZengLiangl/FlashShell/releases](https://github.com/ZengLiangl/FlashShell/releases)**

打 tag 发布时，CI 会构建多平台产物，命名类似：

```text
FlashShell-<tag>-windows-amd64.*
FlashShell-<tag>-darwin-amd64.* / darwin-arm64.*
FlashShell-<tag>-linux-amd64.*
```

## 平台支持

| 平台 | 状态 | 说明 |
| --- | --- | --- |
| Windows | ✅ | `FlashShell.exe` / NSIS 安装包 |
| macOS | ✅ | `FlashShell.app` |
| Linux | ✅ | 可执行文件 |

## 安装后第一次启动

1. 启动 **FlashShell**
2. 若当前目录没有业务配置，会生成示例 `config.yaml`
3. 全局数据写入 `~/.flashshell/`（机器清单、主题、快捷键、known_hosts 等）

## 从源码构建

需要 Go `1.23+`、Node.js `16+`（推荐 20）、Wails CLI v2：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
git clone https://github.com/ZengLiangl/FlashShell.git
cd FlashShell
cd frontend && npm install && cd ..
go mod tidy
wails build
```

产物位于 `build/bin/`。开发热重载用 `wails dev`。

详见 [安装与构建](/guide/installation)。

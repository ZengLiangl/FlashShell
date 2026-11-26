#!/bin/bash
# 开发模式（手动刷新）
#
# 与 dev.sh 的区别：不会因为改代码而自动重启应用窗口，
# 改完代码后按需执行 ./dev-reload.sh 手动刷新界面。
#
#   ./dev-noreload.sh          # 后端 + 前端都改，手动刷新
#   ./dev-noreload.sh -front   # 只改前端，后端不重编（最省事，前端热更新即可）

set -e

cd "$(dirname "$0")"

FRONT_ONLY=0
for arg in "$@"; do
    case "$arg" in
        -front|--front|--frontend-only) FRONT_ONLY=1 ;;
    esac
done

if ! command -v wails &> /dev/null; then
    echo "错误: Wails 未安装"
    echo "请运行: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    exit 1
fi

if [ ! -d "frontend/node_modules" ]; then
    echo "安装前端依赖..."
    (cd frontend && npm install)
fi

# -noreload      : 前端资源变化不自动 reload
# -nogorebuild   : 后端 .go 变化不自动重编（否则会关掉再弹开窗口）
# 前端改动仍由 Vite HMR 即时生效，不受这两个开关影响。
ARGS=(-noreload -nogorebuild)

if [ "$FRONT_ONLY" = "1" ]; then
    echo "只改前端：已禁用后端重编与资源 reload，前端走 Vite HMR"
else
    echo "已禁用自动重载：改完代码请执行 ./dev-reload.sh 手动刷新"
fi

echo "启动开发模式..."
exec wails dev "${ARGS[@]}"

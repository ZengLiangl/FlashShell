#!/bin/bash
# 手动刷新正在运行的 wails dev 应用
#
# 配合 ./dev-noreload.sh 使用：改完代码后执行本脚本，
# 应用窗口会原地刷新，不会关掉再弹开。
#
# 也可指定端口：./dev-reload.sh 34115

set -e

PORT="${1:-34115}"

url="http://localhost:${PORT}/wails/reload"

code=$(curl -s -o /dev/null -m 5 -w "%{http_code}" "$url" || true)

if [ "$code" = "204" ]; then
    echo "已刷新应用界面（${url}）"
else
    echo "刷新失败：${url} 返回 ${code:-无响应}"
    echo "确认 wails dev 正在运行，且 devserver 端口是 ${PORT}（见 wails.json 的 devserver）"
    exit 1
fi

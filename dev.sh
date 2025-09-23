#!/bin/bash

# Quick Cmd 开发脚本

echo "启动 Quick Cmd 开发模式..."

# 检查 Wails 是否安装
if ! command -v wails &> /dev/null; then
    echo "错误: Wails 未安装"
    echo "请运行: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    exit 1
fi

# 检查前端依赖
if [ ! -d "frontend/node_modules" ]; then
    echo "安装前端依赖..."
    cd frontend
    npm install
    cd ..
fi

# 启动开发模式
echo "启动开发服务器..."
wails dev
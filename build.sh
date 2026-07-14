#!/bin/bash

# FlashDock 构建脚本

echo "开始构建 FlashDock..."

# 检查 Wails 是否安装
if ! command -v wails &> /dev/null; then
    echo "错误: Wails 未安装"
    echo "请运行: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    exit 1
fi

# 检查 Node.js 是否安装
if ! command -v node &> /dev/null; then
    echo "错误: Node.js 未安装"
    echo "请安装 Node.js: https://nodejs.org/"
    exit 1
fi

# 进入前端目录安装依赖
echo "安装前端依赖..."
cd frontend
npm install
if [ $? -ne 0 ]; then
    echo "错误: 前端依赖安装失败"
    exit 1
fi
cd ..

# 构建应用
echo "构建应用..."
wails build

if [ $? -eq 0 ]; then
    echo "构建成功!"
    echo "可执行文件位置: ./build/bin/"
else
    echo "构建失败!"
    exit 1
fi
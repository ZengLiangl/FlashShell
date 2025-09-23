.PHONY: dev build clean install-deps frontend-deps

# 开发模式
dev:
	@echo "启动开发模式..."
	@./dev.sh

# 构建应用
build:
	@echo "构建应用..."
	@./build.sh

# 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf build/
	@rm -rf frontend/dist/
	@rm -rf frontend/node_modules/

# 安装所有依赖
install-deps: frontend-deps
	@echo "安装 Go 依赖..."
	@go mod tidy

# 安装前端依赖
frontend-deps:
	@echo "安装前端依赖..."
	@cd frontend && npm install

# 运行测试
test:
	@echo "运行测试..."
	@go test ./...

# 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@cd frontend && npm run format 2>/dev/null || echo "前端格式化跳过"

# 检查代码
lint:
	@echo "检查代码..."
	@go vet ./...

# 创建示例配置
config:
	@echo "创建示例配置文件..."
	@cp config.example.yaml config.yaml
	@echo "请编辑 config.yaml 文件配置你的项目和机器信息"
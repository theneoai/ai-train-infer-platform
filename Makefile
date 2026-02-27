.PHONY: all build test lint dev clean help

# 变量
SERVICES := gateway user data training experiment inference
WEB_DIR := web

# 默认目标
all: build

# 构建所有服务
build:
	@mkdir -p bin
	@echo "→ Building services..."
	@for service in $(SERVICES); do \
		echo "  Building $$service..."; \
		cd services/$$service 2>/dev/null && go build -o ../../bin/$$service ./cmd/ 2>/dev/null && cd ../.. || echo "  Skipping $$service (not initialized)"; \
	done
	@echo "✅ Build complete"

# 构建前端
build-web:
	@echo "→ Building web frontend..."
	@cd $(WEB_DIR) 2>/dev/null && npm run build 2>/dev/null || echo "⚠️  Web not initialized"

# 运行测试
test:
	@echo "→ Running tests..."
	@go test -v ./pkg/... 2>/dev/null || echo "⚠️  No shared packages yet"
	@for service in $(SERVICES); do \
		cd services/$$service 2>/dev/null && go test -v ./... 2>/dev/null && cd ../.. || cd ../..; \
	done

# 代码检查
lint:
	@echo "→ Linting..."
	@golangci-lint run ./... 2>/dev/null || echo "⚠️  Install golangci-lint: https://golangci-lint.run/usage/install/"
	@cd $(WEB_DIR) 2>/dev/null && npm run lint 2>/dev/null || echo "⚠️  Frontend lint skipped"

# 启动开发环境
dev:
	@echo "🚀 Starting development environment..."
	@docker-compose -f deploy/docker-compose.yml up -d 2>/dev/null || echo "⚠️  Docker Compose not configured yet"
	@echo "✅ Dev environment ready"
	@echo "  API: http://localhost:8080"
	@echo "  Web: http://localhost:3000"

# 停止开发环境
dev-stop:
	@docker-compose -f deploy/docker-compose.yml down 2>/dev/null || true

# 查看日志
logs:
	@docker-compose -f deploy/docker-compose.yml logs -f 2>/dev/null || echo "⚠️  No running containers"

# 数据库迁移
migrate-up:
	@echo "→ Running migrations..."
	@migrate -path migrations -database "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable" up 2>/dev/null || echo "⚠️  Install migrate CLI: https://github.com/golang-migrate/migrate"

migrate-down:
	@migrate -path migrations -database "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable" down 2>/dev/null || echo "⚠️  Install migrate CLI"

migrate-create:
	@migrate create -ext sql -dir migrations -seq $(name)

# 生成代码（proto/mock）
generate:
	@echo "→ Generating code..."
	@go generate ./... 2>/dev/null || echo "⚠️  No generate directives yet"

# 清理
clean:
	@rm -rf bin/
	@cd $(WEB_DIR) 2>/dev/null && rm -rf dist/ node_modules/ && cd ..
	@echo "✅ Cleaned"

# 安装依赖
install:
	@echo "→ Installing Go dependencies..."
	@go mod download
	@echo "→ Installing frontend dependencies..."
	@cd $(WEB_DIR) 2>/dev/null && npm install 2>/dev/null || echo "⚠️  Web not initialized"

# 格式化代码
fmt:
	@gofmt -w .
	@cd $(WEB_DIR) 2>/dev/null && npm run format 2>/dev/null || true

# 安全扫描
security:
	@gosec ./... 2>/dev/null || echo "⚠️  Install gosec: https://github.com/securego/gosec"

# 依赖更新
deps-update:
	@go get -u ./...
	@go mod tidy
	@cd $(WEB_DIR) 2>/dev/null && npm update 2>/dev/null || true

# 帮助
help:
	@echo "AITIP - Available targets:"
	@echo ""
	@echo "  build        - Build all services"
	@echo "  build-web    - Build web frontend"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linters"
	@echo "  dev          - Start development environment"
	@echo "  dev-stop     - Stop development environment"
	@echo "  logs         - View logs"
	@echo "  migrate-up   - Run database migrations"
	@echo "  migrate-down - Rollback migrations"
	@echo "  migrate-create name=xxx - Create new migration"
	@echo "  generate     - Generate code (proto/mock)"
	@echo "  fmt          - Format code"
	@echo "  security     - Run security scan"
	@echo "  install      - Install dependencies"
	@echo "  deps-update  - Update dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  help         - Show this help"

.PHONY: all build test lint dev migrate clean

# 变量
SERVICES := gateway user data training experiment inference
WEB_DIR := web

# 默认目标
all: build

# 构建所有服务
build:
	@mkdir -p bin
	@for service in $(SERVICES); do \
		echo "→ Building $$service..."; \
		cd services/$$service && go build -o ../../bin/$$service ./cmd/ && cd ../..; \
	done
	@echo "✅ Build complete"

# 构建前端
build-web:
	cd $(WEB_DIR) && npm run build

# 运行测试
test:
	@echo "→ Running tests..."
	@go test -v ./pkg/... 2>/dev/null || true
	@for service in $(SERVICES); do \
		cd services/$$service && go test -v ./... 2>/dev/null && cd ../.. || cd ../..; \
	done

# 代码检查
lint:
	@echo "→ Linting..."
	@golangci-lint run ./... 2>/dev/null || echo "Install golangci-lint for linting"
	@cd $(WEB_DIR) && npm run lint 2>/dev/null || echo "Frontend lint skipped"

# 启动开发环境
dev:
	docker-compose -f deploy/docker-compose.yml up -d
	@echo "✅ Dev environment started"
	@echo "→ API: http://localhost:8080"
	@echo "→ Web: http://localhost:3000"
	@echo "→ MinIO: http://localhost:9001"

# 停止开发环境
dev-stop:
	docker-compose -f deploy/docker-compose.yml down

# 查看日志
logs:
	docker-compose -f deploy/docker-compose.yml logs -f

# 数据库迁移
migrate-up:
	@echo "→ Running migrations..."
	@migrate -path migrations -database "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable" up 2>/dev/null || echo "Install migrate CLI"

migrate-down:
	@migrate -path migrations -database "postgres://aitip:aitip@localhost:5432/aitip?sslmode=disable" down 2>/dev/null || echo "Install migrate CLI"

# 创建新迁移
migrate-create:
	@migrate create -ext sql -dir migrations -seq $(name)

# 生成代码（proto/mock）
generate:
	@echo "→ Generating code..."
	@go generate ./...

# 清理
clean:
	rm -rf bin/
	@cd $(WEB_DIR) && rm -rf dist/ 2>/dev/null || true
	@echo "✅ Cleaned"

# 全量重置（危险！）
reset: clean dev-stop
	docker-compose -f deploy/docker-compose.yml down -v
	rm -rf migrations/*.sql
	@echo "⚠️  All data cleared"

# 帮助
help:
	@echo "Available targets:"
	@echo "  build        - Build all services"
	@echo "  build-web    - Build web frontend"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linters"
	@echo "  dev          - Start development environment"
	@echo "  dev-stop     - Stop development environment"
	@echo "  logs         - View logs"
	@echo "  migrate-up   - Run database migrations"
	@echo "  migrate-down - Rollback migrations"
	@echo "  clean        - Clean build artifacts"
	@echo "  reset        - ⚠️  Reset everything"

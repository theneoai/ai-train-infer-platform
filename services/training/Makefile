# Training Service Makefile

.PHONY: build run test clean docker-build docker-run

# 变量
SERVICE_NAME=training-service
PORT=8081
DOCKER_IMAGE=aitip/$(SERVICE_NAME):latest

# 构建
default: build

build:
	@echo "Building $(SERVICE_NAME)..."
	go build -o bin/$(SERVICE_NAME) cmd/main.go

run:
	@echo "Running $(SERVICE_NAME)..."
	go run cmd/main.go

dev:
	@echo "Running $(SERVICE_NAME) in dev mode..."
	LOG_LEVEL=debug LOG_FORMAT=console go run cmd/main.go

# 测试
test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Docker
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "Running Docker container..."
	docker run -p $(PORT):$(PORT) \
		-e DATABASE_URL=$${DATABASE_URL} \
		-e REDIS_URL=$${REDIS_URL} \
		-e LOG_LEVEL=info \
		$(DOCKER_IMAGE)

docker-compose-up:
	@echo "Starting with docker-compose..."
	docker-compose up -d

docker-compose-down:
	@echo "Stopping docker-compose..."
	docker-compose down

# 清理
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

# 依赖管理
deps:
	@echo "Downloading dependencies..."
	go mod download

deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# 帮助
help:
	@echo "Available targets:"
	@echo "  build              - Build the service"
	@echo "  run                - Run the service"
	@echo "  dev                - Run the service in dev mode"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-run         - Run Docker container"
	@echo "  docker-compose-up  - Start with docker-compose"
	@echo "  docker-compose-down- Stop docker-compose"
	@echo "  clean              - Clean build artifacts"
	@echo "  lint               - Run linter"
	@echo "  fmt                - Format code"
	@echo "  vet                - Run go vet"
	@echo "  deps               - Download dependencies"
	@echo "  deps-update        - Update dependencies"

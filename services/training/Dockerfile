# Training Service Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o training-service services/training/cmd/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 复制二进制文件
COPY --from=builder /app/training-service .

# 创建日志目录
RUN mkdir -p logs

# 暴露端口
EXPOSE 8081

# 运行
CMD ["./training-service"]

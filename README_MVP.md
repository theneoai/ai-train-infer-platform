# AI 训推仿真平台 - MVP 版本

## 快速开始

```bash
# 1. 启动开发环境
docker-compose up -d

# 2. 运行数据库迁移
make migrate-up

# 3. 启动服务
make dev

# 4. 访问前端
open http://localhost:3000
```

## MVP 功能范围

- [x] 用户注册/登录
- [x] 数据集上传
- [x] 单机训练任务
- [x] 训练日志实时查看
- [x] 模型部署（单实例）
- [x] Dashboard 监控

## 开发命令

```bash
make build       # 构建所有服务
make test        # 运行测试
make dev         # 启动开发环境
make migrate-up  # 数据库迁移
```

## 架构

```
Gateway (Go) → 路由分发 → Services (Go)
                      ↓
                 PostgreSQL + Redis + MinIO
```

## 技术栈

- **后端**: Go 1.21 + Gin + GORM
- **前端**: React 18 + Vite + Tailwind
- **数据库**: PostgreSQL 15 + Redis 7
- **存储**: MinIO (S3兼容)
- **部署**: Docker Compose

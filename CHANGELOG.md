# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - 2025-01 - MVP Release

### 🎉 MVP 版本发布

AITIP (AI Train-Infer-Platform) 首个 MVP 版本，提供基础的训推一体化能力。

### ✨ Features

#### 用户管理
- 用户注册/登录
- JWT Token 认证
- API Key 管理
- 用户资料管理

#### 数据管理
- 数据集上传/下载
- 文件格式自动检测
- MinIO 对象存储
- 数据集列表管理

#### 训练管理
- 训练任务提交
- Docker 本地执行
- GPU 资源分配
- 实时日志流（SSE）
- 训练状态追踪
- 基础指标收集（loss/accuracy）
- PyTorch/TensorFlow 模板支持

#### 实验追踪
- 实验创建与管理
- 实验-任务关联
- 指标存储与查询

#### 推理服务
- 模型部署
- Triton Inference Server 集成
- vLLM 大模型支持
- 服务状态管理
- 服务端点暴露

#### 前端界面
- 现代化 React UI
- Dashboard 监控
- 训练任务管理
- 数据集管理
- 推理服务管理
- 实时日志查看
- 响应式设计

### 🛠️ Technical Stack

- **Backend**: Go 1.21 + Gin + GORM
- **Frontend**: React 18 + Vite + Tailwind CSS
- **Database**: PostgreSQL 15 + Redis 7
- **Storage**: MinIO (S3-compatible)
- **Deployment**: Docker Compose

### 📦 Services

- gateway: API 网关
- user: 用户服务
- data: 数据服务
- training: 训练服务
- experiment: 实验服务
- inference: 推理服务
- web: React 前端

### 📝 API Endpoints

- POST /api/v1/auth/register
- POST /api/v1/auth/login
- GET /api/v1/auth/me
- POST /api/v1/api-keys

- GET /api/v1/datasets
- POST /api/v1/datasets
- GET /api/v1/datasets/:id
- GET /api/v1/datasets/:id/download

- GET /api/v1/training/jobs
- POST /api/v1/training/jobs
- GET /api/v1/training/jobs/:id
- GET /api/v1/training/jobs/:id/logs
- DELETE /api/v1/training/jobs/:id

- GET /api/v1/inference/services
- POST /api/v1/inference/services
- GET /api/v1/inference/services/:id
- POST /api/v1/inference/services/:id/start
- POST /api/v1/inference/services/:id/stop

### ⚠️ Known Issues

- GPU 调度为简化版，不支持多节点
- 推理服务无自动扩缩容
- 实验追踪无 MLflow 集成（可选配置）
- 前端无暗黑模式

### 🔜 Roadmap

#### v0.2.0
- 分布式训练支持（Ray）
- 训练评测自动化
- 集成测试框架
- 灰度发布

#### v0.3.0
- 仿真沙箱
- 智能调度
- Agent API
- 成本分析

#### v0.4.0
- AI Native 工作流
- 自然语言操作
- 智能推荐

### 🙏 Contributors

- Initial development by the AITIP team

### 📄 License

MIT License - see LICENSE file for details

# AI 训推仿真平台 - 架构设计文档

## 1. 设计原则

### 1.1 核心原则

| 原则 | 说明 | 实践 |
|------|------|------|
| **API First** | 一切从 API 设计开始 | OpenAPI 3.0 + gRPC |
| **Agent Native** | 原生支持 AI Agent | 结构化输出、工具调用 |
| **Cloud Native** | 云原生架构 | K8s、微服务、声明式 |
| **UX First** | 用户体验优先 | 实时反馈、零配置 |
| **Observability** | 可观测性内置 | 日志、指标、追踪 |

### 1.2 架构决策记录 (ADR)

#### ADR-001: 微服务 vs 单体
**决策**: 采用微服务架构
**原因**:
- 训练、推理、仿真三个领域独立演进
- 不同服务的资源需求差异大（GPU vs CPU）
- 团队可独立部署和扩展

#### ADR-002: Go vs Python 后端
**决策**: 核心服务用 Go，ML 相关用 Python
**原因**:
- Go: 高并发、低延迟、资源效率高
- Python: ML/AI 生态丰富
- gRPC 桥接两种语言

#### ADR-003: React vs Vue
**决策**: React 18 + TypeScript
**原因**:
- 类型安全
- 生态成熟
- 团队协作友好

---

## 2. 领域模型

### 2.1 核心实体

```
┌─────────────────────────────────────────────────────────────────┐
│                           User (用户)                             │
├─────────────────────────────────────────────────────────────────┤
│ id, email, name, role, org_id, created_at, updated_at          │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Project      │     │  Experiment   │     │  Resource     │
│  (项目)        │     │  (实验)        │     │  (资源配额)    │
└───────────────┘     └───────────────┘     └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Model        │     │  Run          │     │  GPU Pool     │
│  (模型版本)    │     │  (运行记录)    │     │  (GPU 资源池)  │
└───────────────┘     └───────────────┘     └───────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Training Job │     │  Inference    │     │  Simulation   │
│  (训练任务)    │     │  Service      │     │  Environment  │
│               │     │  (推理服务)    │     │  (仿真环境)    │
└───────────────┘     └───────────────┘     └───────────────┘
```

### 2.2 实体关系

```yaml
User:
  has_many: [Projects, Experiments, API Keys]
  belongs_to: Organization

Project:
  has_many: [Experiments, Models, Datasets]
  belongs_to: User

Experiment:
  has_many: [Runs, Artifacts]
  belongs_to: [User, Project]

Run:
  has_many: [Metrics, Logs, Artifacts]
  belongs_to: Experiment
  polymorphic: [TrainingJob, InferenceService, Simulation]

Model:
  has_many: [Versions, Deployments]
  belongs_to: Project

TrainingJob:
  has_one: Run
  has_many: [Checkpoints, Metrics]
  
InferenceService:
  has_one: Run
  has_many: [Endpoints, Metrics]
  
Simulation:
  has_one: Run
  has_many: [Scenarios, Results]
```

---

## 3. 服务架构

### 3.1 服务划分

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│                    (Kong / Nginx Ingress)                        │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  User Service │     │  Resource     │     │  Scheduler    │
│  (用户服务)    │     │  Service      │     │  Service      │
│               │     │  (资源管理)    │     │  (调度器)      │
│ - 认证授权     │     │               │     │               │
│ - 组织管理     │     │ - GPU 池管理   │     │ - 任务调度     │
│ - API Key     │     │ - 配额管理     │     │ - 队列管理     │
└───────────────┘     │ - 成本核算     │     │ - 抢占策略     │
                      └───────────────┘     └───────────────┘
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Training     │     │  Inference    │     │  Simulation   │
│  Service      │     │  Service      │     │  Service      │
│  (训练服务)    │     │  (推理服务)    │     │  (仿真服务)    │
│               │     │               │     │               │
│ - 训练任务提交 │     │ - 模型部署     │     │ - 环境创建     │
│ - 分布式训练   │     │ - 自动扩缩容   │     │ - 场景模拟     │
│ - 超参调优     │     │ - A/B 测试     │     │ - 对抗测试     │
│ - 模型检查点   │     │ - 金丝雀发布   │     │ - 安全评估     │
└───────────────┘     └───────────────┘     └───────────────┘
                              │
                              ▼
                      ┌───────────────┐
                      │  Experiment   │
                      │  Service      │
                      │  (实验追踪)    │
                      │               │
                      │ - 实验管理     │
                      │ - 指标收集     │
                      │ - 工件存储     │
                      │ - 可视化       │
                      └───────────────┘
```

### 3.2 服务间通信

| 场景 | 协议 | 说明 |
|------|------|------|
| 同步调用 | gRPC + HTTP/2 | 服务间直接调用 |
| 异步事件 | NATS / RabbitMQ | 训练完成、部署通知 |
| 实时推送 | WebSocket / SSE | 日志流、指标更新 |
| 数据流 | S3 / MinIO | 模型、数据集、工件 |

### 3.3 数据流图

```
用户提交训练任务
       │
       ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Training   │────▶│  Scheduler   │────▶│  K8s API     │
│   Service    │     │   Service    │     │              │
└──────────────┘     └──────────────┘     └──────────────┘
       │                                        │
       ▼                                        ▼
┌──────────────┐                      ┌──────────────┐
│  Experiment  │◀─────────────────────│ Training Pod │
│   Service    │    指标/日志/工件      │  (Ray/PyTorch)│
└──────────────┘                      └──────────────┘
       │
       ▼
┌──────────────┐
│  Web UI      │
│  (实时更新)   │
└──────────────┘
```

---

## 4. 数据库设计

### 4.1 PostgreSQL Schema

```sql
-- 用户与组织
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    org_id UUID REFERENCES organizations(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    plan VARCHAR(50) DEFAULT 'free',
    quota JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- 项目
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id UUID REFERENCES users(id),
    org_id UUID REFERENCES organizations(id),
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- 实验
CREATE TABLE experiments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    project_id UUID REFERENCES projects(id),
    user_id UUID REFERENCES users(id),
    config JSONB DEFAULT '{}',
    tags TEXT[],
    status VARCHAR(50) DEFAULT 'running',
    created_at TIMESTAMP DEFAULT NOW()
);

-- 运行记录 (Training/Inference/Simulation)
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID REFERENCES experiments(id),
    run_type VARCHAR(50) NOT NULL, -- 'training', 'inference', 'simulation'
    status VARCHAR(50) DEFAULT 'pending',
    config JSONB DEFAULT '{}',
    metrics_summary JSONB DEFAULT '{}',
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 指标时序数据 (可使用 TimescaleDB 扩展)
CREATE TABLE metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID REFERENCES runs(id),
    key VARCHAR(255) NOT NULL,
    value FLOAT NOT NULL,
    step INTEGER,
    timestamp TIMESTAMP DEFAULT NOW()
);

-- 模型注册
CREATE TABLE models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    project_id UUID REFERENCES projects(id),
    latest_version VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE model_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID REFERENCES models(id),
    version VARCHAR(50) NOT NULL,
    run_id UUID REFERENCES runs(id),
    storage_path VARCHAR(500),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(model_id, version)
);
```

### 4.2 Redis 数据结构

```
# 任务队列
train:queue:pending    [List]    待处理的训练任务
train:queue:running    [Set]     运行中的训练任务

# 资源状态
gpu:pool:status        [Hash]    GPU 池状态 {node_id: status}
resource:quota:{org}   [Hash]    组织配额使用情况

# 实时数据
run:{id}:logs          [Stream]  运行日志流
run:{id}:metrics       [Stream]  实时指标流
websocket:connections  [Set]     WebSocket 连接

# 缓存
cache:user:{id}        [String]  用户信息缓存
cache:project:{id}     [String]  项目信息缓存
```

---

## 5. API 设计

### 5.1 REST API 规范

```yaml
# 训练任务 API
/train/jobs:
  GET:  列出训练任务
  POST: 创建训练任务
  
/train/jobs/{id}:
  GET:    获取任务详情
  DELETE: 停止/删除任务
  
/train/jobs/{id}/logs:
  GET: 获取日志 (SSE 流式)
  
/train/jobs/{id}/metrics:
  GET: 获取指标数据

# 推理服务 API
/inference/services:
  GET:  列出自建推理服务
  POST: 部署推理服务
  
/inference/services/{id}:
  GET:    获取服务详情
  PATCH:  更新配置
  DELETE: 下线服务

# 仿真环境 API
/simulation/environments:
  GET:  列出仿真环境
  POST: 创建仿真环境
  
/simulation/environments/{id}:
  GET:    获取环境详情
  POST:   启动仿真
  DELETE: 销毁环境

# 实验追踪 API
/experiments:
  GET:  列出实验
  POST: 创建实验
  
/experiments/{id}/runs:
  GET:  列出运行
  POST: 开始运行
  
/runs/{id}/metrics:
  POST: 记录指标
  GET:  查询指标
```

### 5.2 AI Agent 专用 API

```yaml
# Agent 工具调用
/agent/tools:
  GET: 列出可用工具

/agent/execute:
  POST: 执行工具调用
  
# 结构化输出示例
POST /agent/execute
{
  "tool": "train.submit",
  "params": {
    "model": "meta-llama/Llama-2-7b",
    "dataset": "s3://bucket/data",
    "hyperparameters": {
      "learning_rate": 2e-5,
      "batch_size": 32,
      "epochs": 3
    },
    "resources": {
      "gpu": "4xA100",
      "memory": "128Gi"
    }
  },
  "callback": "https://agent.callback/url"
}

Response:
{
  "job_id": "train-abc123",
  "status": "queued",
  "estimated_time": "2h 30m",
  "webhook_url": "/webhooks/train-abc123",
  "monitoring": {
    "dashboard_url": "/train/abc123/dashboard",
    "metrics_endpoint": "/train/abc123/metrics"
  }
}
```

---

## 6. 安全设计

### 6.1 认证与授权

```
用户认证
   │
   ├──▶ OAuth 2.0 / OIDC (Google, GitHub)
   ├──▶ SAML (企业 SSO)
   └──▶ API Key (程序化访问)
   
权限控制 (RBAC)
   │
   ├──▶ Role: admin, manager, user, viewer
   ├──▶ Resource: project, experiment, model
   └──▶ Action: create, read, update, delete
```

### 6.2 多租户隔离

```
Organization A                Organization B
   │                              │
   ├── Namespace: org-a           ├── Namespace: org-b
   ├── Network Policy             ├── Network Policy
   ├── Resource Quota             ├── Resource Quota
   └── Storage: s3://bucket/a     └── Storage: s3://bucket/b
```

---

## 7. 部署架构

### 7.1 Kubernetes 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Ingress Controller                        │
│                    (Nginx / Traefik / Istio)                     │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Web (React)  │     │  API Gateway  │     │  gRPC Services│
│               │     │               │     │               │
│  ┌─────────┐  │     │  ┌─────────┐  │     │  ┌─────────┐  │
│  │   Pod   │  │     │  │   Pod   │  │     │  │   Pod   │  │
│  └─────────┘  │     │  └─────────┘  │     │  └─────────┘  │
│  ┌─────────┐  │     │  ┌─────────┐  │     │  ┌─────────┐  │
│  │   Pod   │  │     │  │   Pod   │  │     │  │   Pod   │  │
│  └─────────┘  │     │  └─────────┘  │     │  └─────────┘  │
└───────────────┘     └───────────────┘     └───────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│                      GPU Workload (Ray)                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ Head Node   │  │ Worker Node │  │ Worker Node (GPU)       │  │
│  │ (CPU)       │  │ (CPU)       │  │ (NVIDIA A100/H100)      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Data Storage                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │ PostgreSQL  │  │    Redis    │  │ MinIO / S3              │  │
│  │ (Primary)   │  │  (Sentinel) │  │ (Object Storage)        │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 环境划分

| 环境 | 用途 | 配置 |
|------|------|------|
| local | 本地开发 | Docker Compose |
| dev | 开发测试 | K8s dev namespace |
| staging | 预发布 | K8s staging namespace |
| prod | 生产 | K8s prod namespace + 多可用区 |

---

## 8. 监控与可观测性

### 8.1 监控栈

```
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│   Prometheus  │────▶│    Grafana    │────▶│   Alerts      │
│  (指标收集)    │     │  (可视化)      │     │  (告警)        │
└───────────────┘     └───────────────┘     └───────────────┘
        │
        ├──▶ Metrics: 训练吞吐量、GPU 利用率、API 延迟
        ├──▶ Logs: Fluentd → Elasticsearch → Kibana
        └──▶ Traces: Jaeger (分布式追踪)
```

### 8.2 关键指标

| 类别 | 指标 | 说明 |
|------|------|------|
| 业务 | train_job_completed_total | 训练任务完成数 |
| 业务 | inference_request_duration | 推理请求延迟 |
| 系统 | gpu_utilization_percent | GPU 利用率 |
| 系统 | api_request_rate | API 请求速率 |
| 资源 | storage_usage_bytes | 存储使用量 |

---

## 9. 演进路线图

### Phase 1: MVP (0.1.0)
- [ ] 基础训练任务管理
- [ ] 实验追踪 (类似 MLflow)
- [ ] 简单 Web UI

### Phase 2: 核心功能 (0.2.0)
- [ ] 推理服务部署
- [ ] 资源调度器
- [ ] 多租户支持

### Phase 3: 高级功能 (0.3.0)
- [ ] 仿真沙箱
- [ ] 分布式训练优化
- [ ] 自动化 MLOps

### Phase 4: AI Native (0.4.0)
- [ ] Agent 工具集成
- [ ] 自然语言操作
- [ ] 智能推荐

---

## 10. 参考

- [Run:ai Documentation](https://docs.run.ai/)
- [Weights & Biases](https://docs.wandb.ai/)
- [ClearML Documentation](https://clear.ml/docs/)
- [Ray Documentation](https://docs.ray.io/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/)

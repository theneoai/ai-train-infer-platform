# AI 训推仿真平台 - 技术方案设计文档

## 1. 技术架构总览

### 1.1 架构原则

| 原则 | 说明 | 实践 |
|------|------|------|
| **API First** | 所有功能先设计 API | OpenAPI 3.0 + gRPC |
| **Cloud Native** | 云原生架构 | K8s、微服务、声明式 |
| **Event-Driven** | 异步事件驱动 | 任务状态变更、日志流 |
| **Polyglot** | 多语言支持 | Go(服务) + Python(ML) + TS(前端) |
| **Observability** | 可观测性内置 | Metrics + Logs + Traces |

### 1.2 系统架构图

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              接入层 (Access Layer)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌───────────┐  │
│   │   Web UI     │   │   CLI Tool   │   │   SDK        │   │  Agent    │  │
│   │  (React 18)  │   │  (Go/Rust)   │   │  (Python/Go) │   │   API     │  │
│   └──────┬───────┘   └──────┬───────┘   └──────┬───────┘   └─────┬─────┘  │
│          │                  │                  │                 │        │
│          └──────────────────┴──────────────────┘                 │        │
│                             │                                     │        │
│                             ▼                                     ▼        │
│                    ┌─────────────────┐              ┌─────────────────┐   │
│                    │   API Gateway   │              │  Agent Gateway  │   │
│                    │   (Kong/Nginx)  │              │  (WebSocket)    │   │
│                    └────────┬────────┘              └─────────────────┘   │
│                             │                                              │
└─────────────────────────────┼──────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            网关层 (Gateway Layer)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                        API Gateway Service                           │  │
│   │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │  │
│   │  │  Auth/OIDC   │  │ Rate Limit   │  │  Routing     │              │  │
│   │  │  (Keycloak)  │  │  (Redis)     │  │  (Path)      │              │  │
│   │  └──────────────┘  └──────────────┘  └──────────────┘              │  │
│   │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │  │
│   │  │  Load Balancer│  │  SSL/TLS     │  │  Request ID  │              │  │
│   │  │  (Nginx)     │  │  (Cert)      │  │  (Trace)     │              │  │
│   │  └──────────────┘  └──────────────┘  └──────────────┘              │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          业务服务层 (Service Layer)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐│
│  │   User      │  │    Data     │  │  Training   │  │    Evaluation       ││
│  │  Service    │  │   Service   │  │  Service    │  │    Service          ││
│  │             │  │             │  │             │  │                     ││
│  │ • 用户管理  │  │ • 数据集    │  │ • 任务管理  │  │ • 自动评测          ││
│  │ • 权限控制  │  │ • 版本控制  │  │ • 分布式    │  │ • 基准测试          ││
│  │ • 组织管理  │  │ • 血缘追踪  │  │ • 超参调优  │  │ • 报告生成          ││
│  │ • API Key   │  │ • 标注管理  │  │ • 检查点    │  │ • 性能分析          ││
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘│
│         │                │                │                    │           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐│
│  │ Experiment  │  │   Test      │  │  Inference  │  │     Release         ││
│  │  Service    │  │   Service   │  │  Service    │  │    Service          ││
│  │             │  │             │  │             │  │                     ││
│  │ • 实验管理  │  │ • 测试计划  │  │ • 模型部署  │  │ • 发布流水线        ││
│  │ • 指标收集  │  │ • 执行引擎  │  │ • 服务发现  │  │ • 灰度策略          ││
│  │ • 工件存储  │  │ • 结果分析  │  │ • 扩缩容    │  │ • 回滚机制          ││
│  │ • 可视化    │  │ • 报告生成  │  │ • A/B 测试  │  │ • 审批流程          ││
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘│
│         │                │                │                    │           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────────────────┐│
│  │ Simulation  │  │  Resource   │  │           Scheduler                 ││
│  │  Service    │  │   Service   │  │           Service                   ││
│  │             │  │             │  │                                     ││
│  │ • 环境管理  │  │ • GPU 池    │  │ • 任务队列                          ││
│  │ • 场景模拟  │  │ • 配额管理  │  │ • 调度算法                          ││
│  │ • 安全评估  │  │ • 成本核算  │  │ • 资源分配                          ││
│  │ • 沙箱隔离  │  │ • 使用报表  │  │ • 抢占策略                          ││
│  └─────────────┘  └─────────────┘  └─────────────────────────────────────┘│
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            集成层 (Integration Layer)                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│   │ Message Queue│  │   Event Bus  │  │   Cache      │  │   Search     │   │
│   │  (NATS/Rabbit│  │   (NATS)     │  │   (Redis)    │  │  (Elasticsearch)│
│   └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      Workflow Engine (Temporal/Cadence)              │   │
│   │  • 训练工作流编排    • 发布流水线    • 测试工作流    • 审批流程       │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            数据层 (Data Layer)                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│   │  PostgreSQL  │  │  TimescaleDB │  │    Redis     │  │   MinIO/S3   │   │
│   │  (Metadata)  │  │  (Metrics)   │  │   (Cache)    │  │   (Objects)  │   │
│   │              │  │              │  │              │  │              │   │
│   │ • Users      │  │ • Time-series│  │ • Session    │  │ • Datasets   │   │
│   │ • Projects   │  │   metrics    │  │ • Queue      │  │ • Models     │   │
│   │ • Jobs       │  │ • Logs       │  │ • Cache      │  │ • Artifacts  │   │
│   │ • Experiments│  │              │  │ • Stream     │  │ • Checkpoints│   │
│   └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      MLflow (Experiment Tracking)                    │   │
│   │  • 实验参数记录    • 指标追踪    • 模型注册    • 工件管理            │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           资源层 (Resource Layer)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                      Kubernetes Cluster                              │   │
│   │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │   │
│   │  │ Training │  │ Inference│  │  Test    │  │   Simulation         │ │   │
│   │  │  Pods    │  │  Pods    │  │  Pods    │  │   Pods               │ │   │
│   │  │(Ray/DDP) │  │(Triton)  │  │(pytest)  │  │   (Container/VM)     │ │   │
│   │  └──────────┘  └──────────┘  └──────────┘  └──────────────────────┘ │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                         GPU Pool                                     │   │
│   │  • NVIDIA A100/H100    • NVIDIA RTX    • AMD MI      • TPU          │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. 服务拆分与职责

### 2.1 微服务划分

| 服务 | 语言 | 职责 | 关键依赖 |
|------|------|------|----------|
| **gateway** | Go | API 网关、认证、路由 | Kong/Nginx |
| **user** | Go | 用户、组织、权限 | PostgreSQL |
| **data** | Go | 数据集、版本、血缘 | PostgreSQL + MinIO |
| **training** | Go + Python | 训练任务、分布式 | Ray + K8s |
| **evaluation** | Python | 评测任务、基准测试 | GPU + Dataset |
| **experiment** | Go | 实验追踪、指标 | MLflow + TimescaleDB |
| **test** | Go + Python | 测试执行、报告 | Test Runner |
| **inference** | Go | 模型部署、服务 | Triton + K8s |
| **release** | Go | 发布管理、流水线 | ArgoCD + K8s |
| **simulation** | Go + Python | 仿真环境、沙箱 | Container/VM |
| **resource** | Go | GPU 池、配额 | K8s API |
| **scheduler** | Go | 任务调度、队列 | Redis + K8s |
| **agent** | Go | Agent API、工具 | All Services |

### 2.2 服务通信

```
同步调用 (Sync)
─────────────────────────────────────────
Web UI/CLI ──REST/gRPC──▶ Gateway ──gRPC──▶ Services
                          │
                          └─▶ Auth (OIDC)

异步事件 (Async)
─────────────────────────────────────────
Services ──Event──▶ NATS/Event Bus ──▶ Subscribers

数据流 (Data Flow)
─────────────────────────────────────────
Training Pod ──Metrics──▶ Prometheus ──▶ Grafana
Training Pod ──Logs─────▶ Loki ───────▶ Grafana
Training Pod ──Artifacts─▶ MinIO/S3

实时推送 (Real-time)
─────────────────────────────────────────
Server ──WebSocket/SSE──▶ Web UI
       ──gRPC Stream────▶ Agent
```

---

## 3. 数据库设计

### 3.1 数据库选型

| 数据类型 | 数据库 | 选型理由 |
|----------|--------|----------|
| 元数据 | PostgreSQL | ACID、关系型、成熟稳定 |
| 时序指标 | TimescaleDB | 基于 PG，时序优化 |
| 缓存 | Redis | 高性能、Pub/Sub |
| 对象存储 | MinIO/S3 | 标准接口、高可用 |
| 搜索 | Elasticsearch | 全文搜索、日志分析 |
| 实验追踪 | MLflow | 社区标准、生态丰富 |

### 3.2 核心表结构

```sql
-- ============================================
-- 用户与组织
-- ============================================
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    plan VARCHAR(50) DEFAULT 'free', -- free/pro/enterprise
    settings JSONB DEFAULT '{}',
    quota JSONB DEFAULT '{}', -- GPU hours, storage, etc.
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(50) DEFAULT 'member', -- owner/admin/member/viewer
    settings JSONB DEFAULT '{}',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255),
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    permissions JSONB DEFAULT '[]', -- ["training:read", "inference:write"]
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 项目管理
-- ============================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, slug)
);

-- ============================================
-- 数据管理
-- ============================================
CREATE TABLE datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) DEFAULT 'v1',
    description TEXT,
    storage_path VARCHAR(500) NOT NULL, -- s3://bucket/path
    format VARCHAR(50), -- csv, parquet, tfrecord, jsonl, etc.
    schema JSONB, -- {"columns": [...], "types": [...]}
    size_bytes BIGINT,
    num_rows BIGINT,
    checksum VARCHAR(64),
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    lineage JSONB DEFAULT '{}', -- upstream datasets, transformations
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE dataset_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    storage_path VARCHAR(500) NOT NULL,
    changes TEXT, -- description of changes
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(dataset_id, version)
);

CREATE TABLE data_annotations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    task_type VARCHAR(50), -- classification, detection, segmentation, etc.
    status VARCHAR(50) DEFAULT 'pending', -- pending/in_progress/completed
    assignee_id UUID REFERENCES users(id),
    annotations JSONB, -- annotation data
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- ============================================
-- 实验与训练
-- ============================================
CREATE TABLE experiments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    hypothesis TEXT,
    config JSONB DEFAULT '{}', -- hyperparameters, etc.
    tags TEXT[],
    status VARCHAR(50) DEFAULT 'running',
    mlflow_experiment_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE training_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID REFERENCES experiments(id),
    project_id UUID REFERENCES projects(id),
    owner_id UUID REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    
    -- 代码配置
    code_repo VARCHAR(500),
    code_commit VARCHAR(100),
    code_path VARCHAR(500),
    
    -- 数据配置
    dataset_id UUID REFERENCES datasets(id),
    data_config JSONB DEFAULT '{}',
    
    -- 训练配置
    framework VARCHAR(50), -- pytorch, tensorflow, etc.
    distributed_mode VARCHAR(50), -- single, ddp, deepspeed, horovod
    config JSONB DEFAULT '{}', -- hyperparameters, training args
    
    -- 资源配置
    gpu_type VARCHAR(50),
    gpu_count INTEGER DEFAULT 1,
    cpu_cores INTEGER,
    memory_gb INTEGER,
    
    -- 状态
    status VARCHAR(50) DEFAULT 'pending', -- pending/queued/running/completed/failed/cancelled
    status_message TEXT,
    
    -- 执行信息
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- 结果
    checkpoint_path VARCHAR(500),
    model_id UUID, -- reference to models table
    
    -- 成本
    cost_estimate DECIMAL(10,2),
    cost_actual DECIMAL(10,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 模型管理
-- ============================================
CREATE TABLE models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    framework VARCHAR(50),
    model_format VARCHAR(50), -- pytorch, onnx, tensorrt, etc.
    latest_version VARCHAR(50),
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE model_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID REFERENCES models(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    training_job_id UUID REFERENCES training_jobs(id),
    
    -- 存储
    storage_path VARCHAR(500) NOT NULL,
    config_path VARCHAR(500),
    
    -- 元数据
    metrics JSONB DEFAULT '{}', -- accuracy, loss, etc.
    parameters BIGINT, -- model size
    model_size_bytes BIGINT,
    
    -- 签名与验证
    signature VARCHAR(500),
    verified BOOLEAN DEFAULT FALSE,
    
    -- MLflow 集成
    mlflow_run_id VARCHAR(100),
    mlflow_model_uri VARCHAR(500),
    
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(model_id, version)
);

-- ============================================
-- 推理服务
-- ============================================
CREATE TABLE inference_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    owner_id UUID REFERENCES users(id),
    model_id UUID REFERENCES models(id),
    model_version VARCHAR(50),
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 部署配置
    runtime VARCHAR(50), -- triton, vllm, torchserve, etc.
    runtime_config JSONB DEFAULT '{}',
    
    -- 资源配置
    gpu_type VARCHAR(50),
    gpu_count INTEGER DEFAULT 1,
    replicas INTEGER DEFAULT 1,
    min_replicas INTEGER DEFAULT 1,
    max_replicas INTEGER DEFAULT 10,
    
    -- 端点
    endpoint_url VARCHAR(500),
    
    -- 状态
    status VARCHAR(50) DEFAULT 'pending', -- pending/deploying/running/stopped/error
    health_status VARCHAR(50),
    
    -- A/B 测试
    is_canary BOOLEAN DEFAULT FALSE,
    traffic_percentage INTEGER DEFAULT 100,
    parent_service_id UUID REFERENCES inference_services(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 测试管理
-- ============================================
CREATE TABLE test_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    owner_id UUID REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    test_type VARCHAR(50), -- unit, integration, e2e, regression
    config JSONB DEFAULT '{}',
    schedule VARCHAR(100), -- cron expression or null
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE test_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_plan_id UUID REFERENCES test_plans(id),
    trigger_type VARCHAR(50), -- manual, scheduled, webhook
    triggered_by UUID REFERENCES users(id),
    
    -- 执行状态
    status VARCHAR(50) DEFAULT 'pending', -- pending/running/completed/failed
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- 结果统计
    total_cases INTEGER DEFAULT 0,
    passed_cases INTEGER DEFAULT 0,
    failed_cases INTEGER DEFAULT 0,
    skipped_cases INTEGER DEFAULT 0,
    
    -- 报告
    report_url VARCHAR(500),
    logs_url VARCHAR(500),
    coverage_percent DECIMAL(5,2),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 发布管理
-- ============================================
CREATE TABLE releases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    model_id UUID REFERENCES models(id),
    model_version VARCHAR(50),
    
    name VARCHAR(255) NOT NULL,
    release_type VARCHAR(50), -- model, firmware, service, config
    
    -- 版本信息
    version VARCHAR(50) NOT NULL,
    changelog TEXT,
    
    -- 发布策略
    strategy VARCHAR(50) DEFAULT 'canary', -- canary, blue-green, rolling
    canary_percentage INTEGER DEFAULT 5,
    canary_stages INTEGER[] DEFAULT ARRAY[5, 20, 50, 100],
    
    -- 审批
    approval_status VARCHAR(50) DEFAULT 'pending', -- pending/approved/rejected
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    
    -- 状态
    status VARCHAR(50) DEFAULT 'draft', -- draft/pending_approval/canary/production/rolled_back
    current_stage INTEGER DEFAULT 0,
    
    -- 回滚
    rollback_available BOOLEAN DEFAULT TRUE,
    rolled_back_at TIMESTAMPTZ,
    rolled_back_by UUID REFERENCES users(id),
    rollback_reason TEXT,
    
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 仿真沙箱
-- ============================================
CREATE TABLE simulation_environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    owner_id UUID REFERENCES users(id),
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 模板与配置
    template VARCHAR(100), -- predefined template
    scenario_config JSONB DEFAULT '{}',
    
    -- 资源配置
    isolation_type VARCHAR(50) DEFAULT 'container', -- container, vm
    resources JSONB DEFAULT '{}',
    
    -- 模型绑定
    model_id UUID REFERENCES models(id),
    model_version VARCHAR(50),
    
    -- 状态
    status VARCHAR(50) DEFAULT 'creating', -- creating/ready/running/completed/error
    endpoint_url VARCHAR(500),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE simulation_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    env_id UUID REFERENCES simulation_environments(id),
    
    -- 运行配置
    scenario_params JSONB DEFAULT '{}',
    
    -- 执行状态
    status VARCHAR(50) DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    duration_seconds INTEGER,
    
    -- 结果
    results JSONB DEFAULT '{}',
    report_url VARCHAR(500),
    safety_score DECIMAL(5,2), -- 0-100
    passed BOOLEAN,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 资源与调度
-- ============================================
CREATE TABLE gpu_nodes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    node_type VARCHAR(50), -- a100, h100, rtx4090, etc.
    
    -- 硬件信息
    gpu_count INTEGER DEFAULT 1,
    gpu_memory_gb INTEGER,
    cpu_cores INTEGER,
    memory_gb INTEGER,
    
    -- 状态
    status VARCHAR(50) DEFAULT 'available', -- available/busy/maintenance/offline
    labels JSONB DEFAULT '{}', -- {"region": "cn-north", "zone": "a"}
    
    -- 使用统计
    total_jobs INTEGER DEFAULT 0,
    total_hours DECIMAL(10,2) DEFAULT 0,
    
    last_heartbeat_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE resource_quotas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    
    -- 配额限制
    max_gpu_hours_month INTEGER,
    max_storage_gb INTEGER,
    max_concurrent_jobs INTEGER,
    max_models INTEGER,
    
    -- 使用统计
    used_gpu_hours_month DECIMAL(10,2) DEFAULT 0,
    used_storage_gb INTEGER DEFAULT 0,
    
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================
-- 索引优化
-- ============================================
CREATE INDEX idx_users_org ON users(org_id);
CREATE INDEX idx_users_email ON users(email);

CREATE INDEX idx_projects_org ON projects(org_id);
CREATE INDEX idx_datasets_project ON datasets(project_id);
CREATE INDEX idx_experiments_project ON experiments(project_id);
CREATE INDEX idx_training_jobs_experiment ON training_jobs(experiment_id);
CREATE INDEX idx_training_jobs_status ON training_jobs(status);
CREATE INDEX idx_models_project ON models(project_id);
CREATE INDEX idx_model_versions_model ON model_versions(model_id);
CREATE INDEX idx_inference_services_project ON inference_services(project_id);
CREATE INDEX idx_inference_services_status ON inference_services(status);
CREATE INDEX idx_test_executions_plan ON test_executions(test_plan_id);
CREATE INDEX idx_releases_project ON releases(project_id);
CREATE INDEX idx_simulation_env_project ON simulation_environments(project_id);
```

---

## 4. API 设计

### 4.1 REST API 规范

```yaml
# 认证相关
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me

# 组织管理
GET    /api/v1/orgs
POST   /api/v1/orgs
GET    /api/v1/orgs/{org_id}
PUT    /api/v1/orgs/{org_id}
DELETE /api/v1/orgs/{org_id}
GET    /api/v1/orgs/{org_id}/members
POST   /api/v1/orgs/{org_id}/members
DELETE /api/v1/orgs/{org_id}/members/{user_id}

# 项目管理
GET    /api/v1/projects
POST   /api/v1/projects
GET    /api/v1/projects/{project_id}
PUT    /api/v1/projects/{project_id}
DELETE /api/v1/projects/{project_id}

# 数据管理
GET    /api/v1/projects/{project_id}/datasets
POST   /api/v1/projects/{project_id}/datasets
GET    /api/v1/datasets/{dataset_id}
PUT    /api/v1/datasets/{dataset_id}
DELETE /api/v1/datasets/{dataset_id}
POST   /api/v1/datasets/{dataset_id}/versions
GET    /api/v1/datasets/{dataset_id}/versions
GET    /api/v1/datasets/{dataset_id}/lineage
GET    /api/v1/datasets/{dataset_id}/annotations
POST   /api/v1/datasets/{dataset_id}/annotations

# 实验与训练
GET    /api/v1/projects/{project_id}/experiments
POST   /api/v1/projects/{project_id}/experiments
GET    /api/v1/experiments/{experiment_id}
PUT    /api/v1/experiments/{experiment_id}
DELETE /api/v1/experiments/{experiment_id}

GET    /api/v1/experiments/{experiment_id}/runs
POST   /api/v1/experiments/{experiment_id}/runs
GET    /api/v1/runs/{run_id}
DELETE /api/v1/runs/{run_id}

# 训练任务
GET    /api/v1/training/jobs
POST   /api/v1/training/jobs
GET    /api/v1/training/jobs/{job_id}
DELETE /api/v1/training/jobs/{job_id}
GET    /api/v1/training/jobs/{job_id}/logs        # SSE streaming
GET    /api/v1/training/jobs/{job_id}/metrics
POST   /api/v1/training/jobs/{job_id}/stop

# 训练评测
GET    /api/v1/evaluation/benchmarks              # 标准评测集列表
POST   /api/v1/evaluation/jobs
GET    /api/v1/evaluation/jobs/{eval_id}
GET    /api/v1/evaluation/jobs/{eval_id}/results
GET    /api/v1/evaluation/jobs/{eval_id}/report   # PDF report

# 推理服务
GET    /api/v1/inference/services
POST   /api/v1/inference/services
GET    /api/v1/inference/services/{service_id}
PUT    /api/v1/inference/services/{service_id}
DELETE /api/v1/inference/services/{service_id}
POST   /api/v1/inference/services/{service_id}/start
POST   /api/v1/inference/services/{service_id}/stop
POST   /api/v1/inference/services/{service_id}/scale
GET    /api/v1/inference/services/{service_id}/metrics

# 集成测试
GET    /api/v1/test/plans
POST   /api/v1/test/plans
GET    /api/v1/test/plans/{plan_id}
PUT    /api/v1/test/plans/{plan_id}
DELETE /api/v1/test/plans/{plan_id}

POST   /api/v1/test/plans/{plan_id}/execute
GET    /api/v1/test/executions
GET    /api/v1/test/executions/{execution_id}
GET    /api/v1/test/executions/{execution_id}/results
GET    /api/v1/test/executions/{execution_id}/report

# 发布管理
GET    /api/v1/releases
POST   /api/v1/releases
GET    /api/v1/releases/{release_id}
PUT    /api/v1/releases/{release_id}
DELETE /api/v1/releases/{release_id}
POST   /api/v1/releases/{release_id}/approve
POST   /api/v1/releases/{release_id}/deploy
POST   /api/v1/releases/{release_id}/promote      # 推进到下一阶段
POST   /api/v1/releases/{release_id}/rollback

# 仿真沙箱
GET    /api/v1/simulation/environments
POST   /api/v1/simulation/environments
GET    /api/v1/simulation/environments/{env_id}
DELETE /api/v1/simulation/environments/{env_id}
POST   /api/v1/simulation/environments/{env_id}/start
POST   /api/v1/simulation/environments/{env_id}/stop
GET    /api/v1/simulation/environments/{env_id}/runs
POST   /api/v1/simulation/environments/{env_id}/runs
GET    /api/v1/simulation/runs/{run_id}
GET    /api/v1/simulation/runs/{run_id}/report

# 资源管理
GET    /api/v1/resources/gpu
GET    /api/v1/resources/quota
GET    /api/v1/resources/usage
GET    /api/v1/resources/costs

# Agent API
GET    /api/v1/agent/tools
POST   /api/v1/agent/execute
GET    /api/v1/agent/executions/{execution_id}
POST   /api/v1/agent/natural-language            # 自然语言接口

# WebSocket 端点
WS     /ws/training/{job_id}/logs
WS     /ws/inference/{service_id}/metrics
WS     /ws/agent/stream
```

### 4.2 Agent API 设计

```yaml
# Agent 工具定义
GET /api/v1/agent/tools

Response:
{
  "tools": [
    {
      "name": "training.submit",
      "description": "Submit a training job",
      "parameters": {
        "type": "object",
        "properties": {
          "name": {"type": "string"},
          "dataset_id": {"type": "string"},
          "config": {"type": "object"},
          "resources": {
            "type": "object",
            "properties": {
              "gpu_type": {"type": "string", "enum": ["a100", "h100", "rtx4090"]},
              "gpu_count": {"type": "integer", "default": 1}
            }
          }
        },
        "required": ["name", "dataset_id"]
      }
    },
    {
      "name": "inference.deploy",
      "description": "Deploy a model as inference service",
      "parameters": {...}
    },
    {
      "name": "evaluation.run",
      "description": "Run evaluation on a model",
      "parameters": {...}
    },
    {
      "name": "test.execute",
      "description": "Execute integration tests",
      "parameters": {...}
    },
    {
      "name": "release.create",
      "description": "Create a new release",
      "parameters": {...}
    }
  ]
}

# Agent 工具执行
POST /api/v1/agent/execute

Request:
{
  "tool": "training.submit",
  "params": {
    "name": "llama-finetune-v2",
    "dataset_id": "ds-abc123",
    "config": {
      "learning_rate": 2e-5,
      "batch_size": 32,
      "epochs": 3
    },
    "resources": {
      "gpu_type": "a100",
      "gpu_count": 4
    }
  },
  "callback_url": "https://agent.callback.url/webhook",
  "async": true
}

Response:
{
  "execution_id": "exec-xyz789",
  "status": "queued",
  "estimated_time": "2h 30m",
  "job_id": "train-abc456",
  "webhook_url": "/webhooks/exec-xyz789",
  "monitoring": {
    "dashboard_url": "/training/train-abc456",
    "logs_url": "/api/v1/training/jobs/train-abc456/logs",
    "metrics_url": "/api/v1/training/jobs/train-abc456/metrics"
  }
}

# 自然语言接口
POST /api/v1/agent/natural-language

Request:
{
  "message": "Train a Llama-2 model on my customer service dataset using 4 A100 GPUs",
  "context": {
    "project_id": "proj-123",
    "user_id": "user-456"
  }
}

Response:
{
  "intent": "training.submit",
  "params": {
    "name": "llama-2-customer-service",
    "dataset_id": "ds-customer-001",
    "framework": "pytorch",
    "config": {...},
    "resources": {"gpu_type": "a100", "gpu_count": 4}
  },
  "confirmation_required": false,
  "execution_id": "exec-789"
}
```

---

## 5. 部署架构

### 5.1 Kubernetes 架构

```yaml
# 命名空间结构
namespaces:
  - aitip-system        # 核心系统服务
  - aitip-dev           # 开发环境
  - aitip-staging       # 预发环境
  - aitip-prod          # 生产环境
  - aitip-gpu           # GPU 工作负载

# 核心组件部署
components:
  # 基础设施
  - postgresql-ha       # PostgreSQL 高可用
  - redis-cluster       # Redis 集群
  - minio               # 对象存储
  - elasticsearch       # 搜索
  - mlflow              # 实验追踪
  
  # 消息与事件
  - nats                # 消息队列
  - temporal            # 工作流引擎
  
  # 可观测性
  - prometheus          # 指标收集
  - grafana             # 可视化
  - loki                # 日志聚合
  - jaeger              # 分布式追踪
  
  # 服务网格 (可选)
  - istio               # 流量管理
```

### 5.2 Helm Chart 结构

```
deploy/helm/aitip/
├── Chart.yaml
├── values.yaml
├── values-dev.yaml
├── values-staging.yaml
├── values-prod.yaml
└── templates/
    ├── _helpers.tpl
    ├── configmap.yaml
    ├── secret.yaml
    │
    ├── gateway/
    │   ├── deployment.yaml
    │   ├── service.yaml
    │   ├── ingress.yaml
    │   └── hpa.yaml
    │
    ├── services/          # 业务服务
    │   ├── user.yaml
    │   ├── data.yaml
    │   ├── training.yaml
    │   ├── evaluation.yaml
    │   ├── inference.yaml
    │   ├── test.yaml
    │   ├── release.yaml
    │   ├── simulation.yaml
    │   ├── resource.yaml
    │   └── scheduler.yaml
    │
    ├── workers/           # 后台任务
    │   ├── training-worker.yaml
    │   └── evaluation-worker.yaml
    │
    ├── web/               # 前端
    │   ├── deployment.yaml
    │   └── service.yaml
    │
    └── jobs/              # 初始化任务
        └── db-migrate.yaml
```

---

## 6. 安全设计

### 6.1 认证与授权

```yaml
# 认证方式
authentication:
  - OIDC (Keycloak)          # Web UI 登录
  - API Key                  # 程序化访问
  - JWT Token                # 短期令牌
  - mTLS (服务间)             # 内部通信

# 授权模型 (RBAC + ABAC)
authorization:
  roles:
    - owner:      # 组织所有者，全部权限
    - admin:      # 管理员，除删除组织外全部权限
    - member:     # 成员，标准开发权限
    - viewer:     # 只读权限
  
  permissions:
    - training:read, training:write, training:delete
    - inference:read, inference:write, inference:delete
    - data:read, data:write
    - release:read, release:approve
    
  resource_attributes:
    - project_id
    - org_id
    - created_by
```

### 6.2 数据安全

- **传输加密**: TLS 1.3
- **存储加密**: 数据库透明加密 (TDE)，对象存储服务端加密
- **密钥管理**: HashiCorp Vault / AWS KMS
- **数据脱敏**: 敏感字段自动脱敏
- **审计日志**: 所有操作记录，保留 1 年

### 6.3 沙箱安全

```yaml
# 仿真沙箱隔离
isolation_levels:
  container:
    - gVisor / Kata Containers
    - 无特权容器
    - 资源限制 (CPU/内存/网络)
    - 只读根文件系统
    
  vm:
    - Firecracker microVM
    - 完全内核隔离
    - 用于高危场景
```

---

## 7. 技术选型对比

### 7.1 已确认选型

| 领域 | 技术 | 备选 | 决策理由 |
|------|------|------|----------|
| 后端语言 | Go | Python/Rust | 高并发、云原生生态 |
| 前端框架 | React 18 | Vue/Svelte | 生态成熟、类型安全 |
| API 网关 | Kong | Nginx/Envoy | 插件丰富、易配置 |
| 数据库 | PostgreSQL | MySQL | 功能丰富、扩展多 |
| 时序数据 | TimescaleDB | InfluxDB | 基于 PG、兼容好 |
| 缓存 | Redis | Memcached | 功能丰富、持久化 |
| 消息队列 | NATS | RabbitMQ/Kafka | 轻量、性能高 |
| 工作流 | Temporal | Airflow | 可靠、可观测 |
| 对象存储 | MinIO | Ceph | S3 兼容、易部署 |
| 实验追踪 | MLflow | W&B/ClearML | 开源、标准 |
| 容器编排 | Kubernetes | Docker Swarm | 行业标准 |
| 服务网格 | Istio | Linkerd | 功能完整 |

### 7.2 待决策事项

| 问题 | 选项 | 建议 |
|------|------|------|
| Python 服务框架 | FastAPI / Flask | FastAPI（现代、自动文档） |
| 训练框架集成 | Ray / Kueue | Ray（生态完善） |
| 推理服务 | Triton / vLLM | 都支持，按模型选择 |
| 前端状态管理 | Zustand / Redux | Zustand（轻量） |
| UI 组件库 | shadcn / Ant Design | shadcn（定制性强） |

---

## 8. 技术债务与风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 微服务复杂度 | 高 | 先单体后拆分，明确边界 |
| 数据库性能瓶颈 | 中 | 读写分离、分库分表预案 |
| GPU 调度复杂 | 高 | 先用 K8s device plugin，再自研 |
| 多租户隔离 | 高 | 网络策略 + 命名空间隔离 |
| Agent 安全风险 | 高 | 严格权限 + 审批流程 + 审计 |

---

## 9. 开发阶段规划

### Phase 1: 基础架构 (Week 1-2)
- [ ] 数据库设计与迁移
- [ ] 共享库开发 (pkg/)
- [ ] Docker Compose 环境
- [ ] CI/CD 流水线

### Phase 2: 核心服务 (Week 3-6)
- [ ] Gateway + User Service
- [ ] Data Service
- [ ] Training Service (基础版)
- [ ] Experiment Service

### Phase 3: 功能完善 (Week 7-10)
- [ ] Evaluation Service
- [ ] Inference Service
- [ ] Test Service
- [ ] Release Service

### Phase 4: 高级功能 (Week 11-14)
- [ ] Simulation Service
- [ ] Resource + Scheduler
- [ ] Agent Service
- [ ] 前端完整功能

### Phase 5: 优化与发布 (Week 15-16)
- [ ] 性能优化
- [ ] 安全加固
- [ ] 文档完善
- [ ] 开源发布

---

## 10. 待评审确认

1. **微服务拆分** - 12 个服务是否过多？是否需要合并？
2. **数据库选型** - TimescaleDB 是否必须？先用 PG 扩展是否可行？
3. **工作流引擎** - Temporal 是否过重？先用简单实现？
4. **Python 服务** - 是否统一用 Go？Python 仅做 ML 脚本？
5. **前端技术栈** - shadcn/ui 确定？还是其他选择？

---

**文档版本**: v1.0
**技术负责人**: 待指定
**创建日期**: 2025-01
**状态**: 技术方案设计完成，待评审确认

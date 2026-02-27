# Architecture Design

> AITIP 系统架构设计文档

## 总体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client Layer                             │
├─────────────────────────────────────────────────────────────────┤
│  Web UI (React) │ CLI (Go) │ Python SDK │ AI Agent              │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                      Gateway Layer                               │
│         Authentication │ Rate Limit │ Routing │ Logging          │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                     Service Layer                                │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┬─────────┐  │
│  │  User   │  Data   │ Training│Inference│Experiment│  Agent  │  │
│  │ Service │ Service │ Service │ Service │ Service │ Service │  │
│  └─────────┴─────────┴─────────┴─────────┴─────────┴─────────┘  │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┐             │
│  │   Sim   │Scheduler│ Resource│ Release │   Test  │             │
│  │ Service │ Service │ Service │ Service │ Service │             │
│  └─────────┴─────────┴─────────┴─────────┴─────────┘             │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                     Infrastructure Layer                       │
│  PostgreSQL │ Redis │ MinIO │ RabbitMQ │ Kubernetes │ Prometheus│
└─────────────────────────────────────────────────────────────────┘
```

## 核心服务

### 1. User Service
- 用户注册/登录
- JWT Token 认证
- API Key 管理
- 用户权限管理

### 2. Data Service
- 数据集上传/下载
- 文件格式验证
- MinIO 对象存储
- 数据集版本管理

### 3. Training Service
- 训练任务调度
- Docker 容器执行
- 分布式训练支持
- 实时日志流 (SSE)

### 4. Inference Service
- 模型部署管理
- Triton/vLLM 集成
- 服务扩缩容
- A/B 测试支持

### 5. Experiment Service
- 实验追踪
- 指标收集
- 实验对比
- MLflow 集成

### 6. Agent Service ⭐
- 自然语言接口
- 工具调用 (Function Calling)
- 异步任务回调
- 智能推荐

### 7. Simulation Service ⭐
- 安全测试沙箱
- 对抗样本生成
- 合规性检查
- 评估报告生成

## 数据流

### 训练流程
```
用户提交训练任务
    ↓
Gateway 认证
    ↓
Training Service 接收任务
    ↓
Scheduler 分配资源
    ↓
启动 Docker 容器
    ↓
实时日志 → WebSocket → 前端
    ↓
训练完成 → 保存模型 → 更新实验状态
```

### 推理部署流程
```
用户提交模型部署请求
    ↓
Inference Service 验证模型
    ↓
生成 Triton 配置
    ↓
Kubernetes 部署服务
    ↓
健康检查通过 → 暴露端点
    ↓
Gateway 路由流量
```

## 技术选型

| 组件 | 技术 | 理由 |
|------|------|------|
| 后端 | Go 1.21+ | 高性能、并发友好、云原生 |
| 前端 | React 18 | 生态丰富、响应式 |
| 数据库 | PostgreSQL 15 | ACID、JSON支持、扩展性好 |
| 缓存 | Redis 7 | 高性能、发布订阅 |
| 存储 | MinIO | S3兼容、高性能 |
| 消息队列 | RabbitMQ | 可靠、灵活 |
| 容器编排 | Kubernetes | 标准、可扩展 |
| 监控 | Prometheus + Grafana | 云原生标准 |
| 日志 | ELK / Loki | 可观测性 |

## 扩展性设计

### 水平扩展
- 所有服务无状态化
- 支持多实例部署
- 基于 Kubernetes HPA 自动扩缩容

### 垂直扩展
- 支持 GPU 节点调度
- 大模型专用推理节点
- 训练/推理资源池分离

## 安全设计

- JWT Token 认证
- API Key 权限控制
- 敏感数据加密存储
- 网络隔离 (Service Mesh)
- 审计日志

## 多租户

- 命名空间隔离
- 资源配额管理
- 数据隔离
- 计费统计

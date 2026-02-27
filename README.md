# 🤖 AITIP - AI Train-Infer-Sim Platform

> 现代化的 AI 训推仿真 DevOps 平台 - 让 AI 开发像使用云原生服务一样简单

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org)
[![React](https://img.shields.io/badge/react-18-61DAFB.svg)](https://reactjs.org)
[![Kubernetes](https://img.shields.io/badge/k8s-1.27+-326CE5.svg)](https://kubernetes.io)

[English](./README_EN.md) | 中文

---

## 📋 目录

- [项目简介](#-项目简介)
- [功能特性](#-功能特性)
- [快速开始](#-快速开始)
- [系统架构](#-系统架构)
- [项目结构](#-项目结构)
- [开发命令](#-开发命令)
- [贡献指南](./CONTRIBUTING.md)
- [许可证](./LICENSE)

---

## 🎯 项目简介

**AITIP (AI Train-Infer-Sim Platform)** 是面向 AI 开发者和团队的**一站式训推仿真 DevOps 平台**。

### 核心价值主张

> "让 AI 开发像使用云原生服务一样简单，从训练到部署，一条命令完成。"

### 目标用户

| 用户角色 | 典型场景 |
|---------|---------|
| **AI 研究员** | 快速提交训练任务，追踪实验指标 |
| **MLOps 工程师** | 自动化模型部署，监控服务健康 |
| **AI 产品经理** | 安全沙箱测试，效果评估报告 |
| **AI Agent** | 通过 API 自主完成训推全流程 |

---

## ✨ 功能特性

### 🚀 训练管理 (Training)
- **一键训练** - 支持 PyTorch、TensorFlow 等多种框架
- **分布式训练** - 自动配置 DDP、DeepSpeed、Horovod
- **超参调优** - 集成 Optuna、Ray Tune 自动搜索
- **实时日志** - SSE 流式日志，实时监控训练过程
- **检查点管理** - 自动保存和恢复模型检查点

### 🎮 仿真沙箱 (Simulation) ⭐ 差异化功能
- **安全测试** - LLM 提示注入、越狱测试
- **场景模板** - 预置多种仿真场景模板
- **对抗测试** - 自动生成对抗样本评估模型鲁棒性
- **合规报告** - 自动生成安全评估报告

### 🎛️ 推理服务 (Inference)
- **一键部署** - 支持 Triton、vLLM、TorchServe
- **自动扩缩容** - 基于负载自动水平扩展
- **A/B 测试** - 流量分割，对比模型效果
- **多版本管理** - 金丝雀发布，一键回滚

### 📊 实验追踪 (Experiment)
- **指标可视化** - 实时损失曲线、准确率图表
- **工件管理** - 模型、数据集、配置版本控制
- **实验对比** - 并排对比多个实验结果
- **MLflow 集成** - 兼容开源 MLflow 生态

### 🤖 AI Agent 接口 ⭐ 核心差异化
- **自然语言操作** - "使用 4 张 A100 训练 Llama-2 模型"
- **工具调用** - 结构化 API，方便 Agent 集成
- **异步回调** - 任务完成自动通知
- **智能推荐** - 自动推荐资源配置和超参数

---

## 🚀 快速开始

### 环境要求

| 组件 | 版本要求 |
|------|---------|
| Docker | 20.10+ |
| Docker Compose | 2.0+ |
| Go | 1.21+ |
| Node.js | 18+ |
| Kubernetes | 1.27+ (生产环境) |

### 一键部署

```bash
# 1. 克隆仓库
git clone https://github.com/theneoai/ai-train-infer-platform.git
cd ai-train-infer-platform

# 2. 启动开发环境
make dev

# 3. 访问平台
# Web UI: http://localhost:3000
# API: http://localhost:8080
```

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        用户层 (User Layer)                   │
├─────────────────────────────────────────────────────────────┤
│  Web UI (React)    CLI Tool    Python SDK    AI Agent       │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                      API 网关 (Gateway)                      │
│              认证 · 限流 · 路由 · 日志                        │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                    微服务层 (Services)                       │
├──────────┬──────────┬──────────┬──────────┬─────────────────┤
│  User    │  Data    │ Training │ Inference│  Experiment     │
│  用户服务 │  数据服务 │ 训练服务 │ 推理服务 │  实验追踪       │
├──────────┼──────────┼──────────┼──────────┼─────────────────┤
│  Agent   │  Sim     │ Scheduler│ Resource │  Release        │
│  Agent接口│  仿真服务 │ 调度服务 │ 资源管理 │  模型发布       │
└──────────┴──────────┴──────────┴──────────┴─────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│                      基础设施层 (Infra)                      │
├──────────┬──────────┬──────────┬──────────┬─────────────────┤
│PostgreSQL│  Redis   │  MinIO   │Kubernetes│   Docker        │
│  数据库   │  缓存    │  对象存储 │ 容器编排  │   容器运行时     │
└──────────┴──────────┴──────────┴──────────┴─────────────────┘
```

---

## 📁 项目结构

```
ai-train-infer-platform/
├── README.md                 # 项目介绍
├── LICENSE                   # 许可证
├── Makefile                  # 构建脚本
├── go.mod                    # Go 依赖
├── docker-compose.yml        # 开发环境编排
├── .github/                  # GitHub Actions
│   └── workflows/
│       ├── ci.yml            # CI 流水线
│       └── cd.yml            # CD 流水线
├── docs/                     # 文档
│   ├── ARCHITECTURE.md       # 架构设计
│   ├── API.md                # API 文档
│   └── DEPLOYMENT.md         # 部署指南
├── services/                 # 微服务目录
│   ├── gateway/              # API 网关
│   ├── user/                 # 用户服务
│   ├── data/                 # 数据服务
│   ├── training/             # 训练服务
│   ├── inference/            # 推理服务
│   ├── experiment/           # 实验服务
│   ├── agent/                # AI Agent 接口
│   └── simulation/           # 仿真沙箱
├── web/                      # 前端应用
│   ├── src/
│   ├── public/
│   └── package.json
├── pkg/                      # 共享库
│   ├── models/               # 数据模型
│   ├── utils/                # 工具函数
│   └── clients/              # 客户端封装
├── deploy/                   # 部署配置
│   ├── docker/               # Dockerfile
│   ├── k8s/                  # Kubernetes 配置
│   └── terraform/            # 基础设施即代码
├── migrations/               # 数据库迁移
└── tests/                    # 测试
    ├── integration/          # 集成测试
    └── e2e/                  # 端到端测试
```

---

## 🛠️ 开发命令

```bash
# 构建所有服务
make build

# 构建前端
make build-web

# 运行测试
make test

# 启动开发环境
make dev

# 查看日志
make logs

# 停止环境
make dev-stop

# 数据库迁移
make migrate-up

# 代码检查
make lint

# 清理构建产物
make clean
```

---

## 🗓️ 路线图

### Phase 1: MVP (当前)
- [x] 项目初始化
- [ ] 用户管理 + 认证
- [ ] 数据集管理
- [ ] 基础训练功能
- [ ] 实验追踪

### Phase 2: Core
- [ ] 分布式训练
- [ ] 推理服务部署
- [ ] 仿真沙箱
- [ ] AI Agent 接口

### Phase 3: Advanced
- [ ] 自动扩缩容
- [ ] A/B 测试
- [ ] 模型版本管理
- [ ] 多租户支持

---

## 🤝 贡献指南

请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解如何参与项目开发。

---

## 📄 许可证

[MIT License](./LICENSE)

---

## 🙏 致谢

感谢所有贡献者的支持！

---

> **注意**: 本项目处于早期开发阶段，API 可能会发生变化。

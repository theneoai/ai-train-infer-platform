# AI 训推仿真 DevOps 平台 - 开发团队组建

## 项目结构概览

```
ai-train-infer-platform/
├── .github/workflows/        # CI/CD
├── api/                      # API 定义
├── web/                      # React 前端
├── services/                 # Go 后端服务
│   ├── gateway/              # API 网关 (已完成基础)
│   ├── training/             # 训练服务 (待开发)
│   ├── inference/            # 推理服务 (待开发)
│   ├── simulation/           # 仿真服务 (待开发)
│   ├── experiment/           # 实验追踪 (待开发)
│   └── scheduler/            # 资源调度 (待开发)
├── deploy/                   # 部署配置
├── docs/                     # 文档 (已完成架构)
└── tests/                    # 测试
```

## 开发任务分配

### 1. 前端开发 (web/)
- [ ] Dashboard 仪表板
- [ ] Training Job 管理界面
- [ ] Inference Service 管理界面
- [ ] Simulation 环境管理界面
- [ ] Experiment 追踪界面
- [ ] Agent Console 交互界面
- [ ] 组件库 (Button, Card, Modal, Form 等)
- [ ] 图表可视化 (Recharts)
- [ ] WebSocket 实时日志

### 2. 后端服务开发 (services/)
- [ ] Gateway: 完善路由、认证、限流
- [ ] Training: 训练任务管理、Ray 集成
- [ ] Inference: 模型部署、Triton/vLLM 集成
- [ ] Simulation: 仿真环境、沙箱管理
- [ ] Experiment: 实验追踪、指标收集
- [ ] Scheduler: 资源调度、GPU 池管理

### 3. 基础设施
- [ ] Kubernetes manifests
- [ ] Helm charts
- [ ] 数据库迁移脚本
- [ ] 监控告警配置

## Gitflow 工作流

1. `main` - 生产分支
2. `develop` - 开发分支
3. `feature/*` - 功能分支
4. `release/*` - 发布分支
5. `hotfix/*` - 热修复分支

## 提交规范 (Conventional Commits)

```
feat(scope): description
fix(scope): description
docs(scope): description
refactor(scope): description
test(scope): description
chore(scope): description
```

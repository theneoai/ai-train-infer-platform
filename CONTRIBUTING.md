# Contributing to AI Train-Infer-Sim Platform

感谢你的贡献！本文档将帮助你了解如何参与项目开发。

## 🎯 开发哲学

1. **API First** - 所有功能先设计 API，再实现 UI
2. **Agent Native** - 考虑 AI Agent 的使用场景
3. **Cloud Native** - 云原生架构，Kubernetes 优先
4. **UX Matters** - 用户体验至上

## 🔄 开发流程

### 1. 创建功能分支

```bash
# 从 develop 分支创建
git checkout develop
git pull origin develop
git checkout -b feature/your-feature-name
```

### 2. 开发规范

#### 提交信息 (Conventional Commits)
```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

类型说明：
- `feat` - 新功能
- `fix` - 修复
- `docs` - 文档
- `style` - 代码格式
- `refactor` - 重构
- `test` - 测试
- `chore` - 构建/工具

示例：
```
feat(training): 添加分布式训练支持

- 集成 Ray Train
- 支持 PyTorch DDP
- 自动容错机制

Closes #123
```

#### 代码规范

**Go**
- 使用 `gofmt` 格式化
- 遵循 Effective Go
- 单元测试覆盖率 > 80%

**TypeScript/React**
- 使用 ESLint + Prettier
- 函数式组件 + Hooks
- 类型安全优先

### 3. 创建 Pull Request

```bash
# 推送分支
git push -u origin feature/your-feature-name

# 创建 PR (使用 gh CLI)
gh pr create --base develop --title "feat: xxx" --body "## Changes..."
```

PR 模板：
- 描述变更内容
- 关联的 Issue
- 测试方式
- 截图 (UI 变更)

### 4. Code Review

- 至少 1 人审批
- CI 检查通过
- 解决所有评论

### 5. 合并

使用 **Squash Merge** 合并到 develop

---

## 🏗️ 项目结构

```
feature/
├── api/                    # API 变更
├── web/                    # 前端变更
├── services/               # 后端服务变更
├── pkg/                    # 共享库变更
├── deploy/                 # 部署配置变更
└── docs/                   # 文档变更
```

---

## 🧪 测试

### 单元测试
```bash
# Go
go test ./...

# TypeScript
npm test
```

### 集成测试
```bash
make test-integration
```

### E2E 测试
```bash
make test-e2e
```

---

## 📝 文档

- API 变更需更新 `api/openapi.yaml`
- 架构变更需更新 `docs/architecture/`
- 用户功能需更新 `docs/user-guide/`

---

## 🐛 提交 Issue

- 使用 Issue 模板
- 提供复现步骤
- 附上日志和截图

---

## 💬 沟通渠道

- GitHub Issues - 功能请求和 Bug
- GitHub Discussions - 一般讨论
- Discord - 实时交流

---

## 🏆 贡献者

感谢所有贡献者！

[贡献者列表]

---

## 📜 行为准则

遵循 [Code of Conduct](./CODE_OF_CONDUCT.md)。

# Contributing to AITIP

感谢您对 AITIP 项目的关注！本文档将帮助您了解如何参与项目开发。

## 🎯 开发流程

我们采用 [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/) 工作流：

```
main (生产分支)
  ↑
develop (开发分支)
  ↑
feature/* (特性分支)
```

### 标准开发流程

1. **Fork 仓库**（外部贡献者）或创建特性分支（内部开发者）
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **开发并提交**
   ```bash
   git add .
   git commit -m "feat: 添加新功能描述"
   ```

3. **推送到远程**
   ```bash
   git push -u origin feature/your-feature-name
   ```

4. **创建 Pull Request**
   - 目标分支: `develop`
   - 填写 PR 模板
   - 关联相关 Issue

5. **Code Review**
   - 至少需要 1 个 approving review
   - 所有 CI 检查通过
   - 解决所有评论

6. **合并**
   - 使用 Squash and Merge
   - 删除特性分支

---

## 📝 提交规范

我们遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

[optional body]

[optional footer(s)]
```

### 类型说明

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `style` | 代码格式（不影响功能） |
| `refactor` | 重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建/工具/依赖更新 |

### 示例

```bash
feat(training): 添加分布式训练支持

- 支持 PyTorch DDP
- 支持 DeepSpeed
- 自动资源分配

Closes #123
```

---

## 🏗️ 项目结构

```
services/
├── gateway/          # API 网关
├── user/             # 用户服务
├── data/             # 数据服务
├── training/         # 训练服务
├── inference/        # 推理服务
├── experiment/       # 实验服务
├── agent/            # AI Agent 接口
└── simulation/       # 仿真沙箱
```

每个服务目录结构：
```
services/xxx/
├── cmd/              # 入口程序
├── internal/         # 内部代码
│   ├── handlers/     # HTTP 处理器
│   ├── services/     # 业务逻辑
│   ├── repositories/ # 数据访问
│   └── models/       # 数据模型
├── pkg/              # 公开 API
├── api/              # API 定义 (proto/openapi)
├── configs/          # 配置文件
├── Dockerfile
└── README.md
```

---

## ✅ 开发检查清单

提交 PR 前请确认：

- [ ] 代码符合项目编码规范
- [ ] 所有测试通过 (`make test`)
- [ ] 新增功能有对应测试
- [ ] 文档已更新
- [ ] Commit message 符合规范
- [ ] 本地开发环境验证通过

---

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定服务测试
cd services/training && go test ./...

# 运行前端测试
cd web && npm test
```

### 测试覆盖率

目标覆盖率：
- 单元测试: >= 70%
- 集成测试: >= 60%

---

## 📚 文档

- [架构设计](./docs/ARCHITECTURE.md)
- [API 文档](./docs/API.md)
- [部署指南](./docs/DEPLOYMENT.md)
- [开发环境搭建](./docs/DEVELOPMENT.md)

---

## 💬 社区

- 讨论区: [GitHub Discussions](https://github.com/theneoai/ai-train-infer-platform/discussions)
- Issue: [GitHub Issues](https://github.com/theneoai/ai-train-infer-platform/issues)

---

## 🙏 感谢

感谢所有贡献者的付出！

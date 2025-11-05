# 代码质量检查清单

快速参考清单，用于日常开发和代码审查。

---

## 🚀 提交前检查（每次提交）

### 基础检查
- [ ] 代码已格式化：`make fmt`
- [ ] Lint 检查通过：`make lint`
- [ ] 所有测试通过：`make test`
- [ ] 无编译警告：`go build ./...`
- [ ] （推荐）已启用预提交钩子：`cp scripts/pre-commit.sample .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit`

### 代码质量
- [ ] 无未使用的导入
- [ ] 无未使用的变量
- [ ] 错误已正确处理（无裸 `_` 忽略）
- [ ] 添加了必要的注释
- [ ] 公共 API 有文档注释

### 测试
- [ ] 新功能有单元测试
- [ ] 测试覆盖关键路径
- [ ] 测试用例有意义的名称
- [ ] 边界条件已测试

---

## 📝 代码审查检查（PR Review）

### 架构和设计
- [ ] 符合项目架构模式
- [ ] 模块职责单一
- [ ] 接口设计合理
- [ ] 无循环依赖

### 代码质量
- [ ] 命名清晰有意义
- [ ] 函数长度合理（< 50 行）
- [ ] 复杂度可接受（圈复杂度 < 10）
- [ ] 无重复代码

### 错误处理
- [ ] 所有错误都被处理
- [ ] 错误信息有上下文
- [ ] 使用 `fmt.Errorf` 包装错误
- [ ] 关键错误有日志记录

### 并发安全
- [ ] 共享数据有锁保护
- [ ] 无数据竞争
- [ ] Goroutine 有生命周期管理
- [ ] Context 正确传递

### 资源管理
- [ ] 文件/连接使用 defer 关闭
- [ ] 无资源泄漏
- [ ] 内存分配合理
- [ ] 使用对象池（如适用）

### 安全性
- [ ] 无 SQL 注入风险
- [ ] 输入已验证
- [ ] 敏感数据已脱敏
- [ ] 无硬编码密钥

### 测试
- [ ] 测试覆盖率 ≥ 60%
- [ ] 测试用例独立
- [ ] 无测试数据泄漏
- [ ] Mock 使用合理

### 文档
- [ ] README 已更新（如需要）
- [ ] API 文档已更新
- [ ] 配置文档已更新
- [ ] 变更日志已更新

---

## 🔍 周度质量检查（每周五）

### 代码健康度
```bash
# 1. 测试覆盖率
make test-coverage
# 目标：≥ 60%

# 2. 代码格式
make fmt-check
# 目标：100% 一致

# 3. Lint 检查
make lint
# 目标：0 警告

# 4. 类型检查（前端）
cd web && npm run typecheck
# 目标：0 错误
```

### 技术债务
- [ ] 检查 TODO/FIXME 数量
- [ ] 评估代码重复率
- [ ] 识别性能瓶颈
- [ ] 更新技术债务清单

### 依赖管理
- [ ] 检查依赖更新：`go list -u -m all`
- [ ] 检查安全漏洞：`go list -json -m all | nancy sleuth`
- [ ] 前端依赖：`npm audit`

---

## 📊 月度质量审查（每月末）

### 指标回顾
- [ ] 测试覆盖率趋势
- [ ] Bug 数量和修复时间
- [ ] 代码审查周期
- [ ] 技术债务变化

### 文档审查
- [ ] 文档准确性
- [ ] 文档完整性
- [ ] 过时文档清理

### 性能审查
- [ ] 响应时间分析
- [ ] 内存使用分析
- [ ] 并发性能测试

---

## 🎯 发布前检查（Release）

### 功能验证
- [ ] 所有功能正常工作
- [ ] 集成测试通过
- [ ] 端到端测试通过
- [ ] 性能测试通过

### 文档
- [ ] 发布说明已准备
- [ ] API 文档已更新
- [ ] 迁移指南已准备（如需要）
- [ ] 已知问题已记录

### 安全
- [ ] 安全扫描通过
- [ ] 依赖漏洞已修复
- [ ] 敏感信息已移除

### 部署
- [ ] 部署脚本已测试
- [ ] 回滚计划已准备
- [ ] 监控已配置
- [ ] 告警已配置

---

## 🛠️ 常用命令

### 开发
```bash
# 格式化代码
make fmt

# 运行测试
make test

# 生成覆盖率报告
make test-coverage

# Lint 检查
make lint

# 构建
make build
```

### 质量检查
```bash
# 快速检查
./scripts/quality_check.sh quick

# 完整检查
./scripts/quality_check.sh all

# 只检查格式
./scripts/quality_check.sh format

# 只检查 lint
./scripts/quality_check.sh lint
```

### 前端
```bash
# 进入前端目录
cd web

# 安装依赖
npm install

# 运行测试
npm test

# 类型检查
npm run typecheck

# Lint
npm run lint

# 构建
npm run build
```

---

## 📚 参考资源

### 内部文档
- [代码质量指南](docs/code-quality.md)
- [测试指南](docs/testing.md)
- [架构文档](docs/architecture.md)
- [技术债务报告](docs/reports/2025-11-04-technical-debt-analysis.md)

### 外部资源
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)

---

## 🎓 最佳实践提醒

### Go
1. **错误处理**: 总是检查错误，使用 `fmt.Errorf` 包装
2. **并发**: 使用 Context 控制生命周期
3. **资源**: 使用 defer 确保资源释放
4. **测试**: 使用表驱动测试

### TypeScript
1. **类型**: 避免使用 `any`，优先使用 `unknown`
2. **空值**: 使用可选链 `?.` 和空值合并 `??`
3. **不可变**: 优先使用 `const`
4. **测试**: 每个组件都应有测试

### 通用
1. **命名**: 清晰胜于简洁
2. **注释**: 解释"为什么"而非"是什么"
3. **简单**: 优先选择简单的解决方案
4. **一致**: 遵循项目现有模式

---

**最后更新**: 2025-11-04  
**维护者**: gcli2api-go 团队

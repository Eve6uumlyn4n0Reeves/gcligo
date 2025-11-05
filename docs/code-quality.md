# 代码质量指南

本文档介绍 gcli2api-go 项目的代码质量标准、工具和最佳实践。

## 📋 目录

- [代码质量标准](#代码质量标准)
- [工具配置](#工具配置)
- [代码格式化](#代码格式化)
- [Lint 检查](#lint-检查)
- [类型检查](#类型检查)
- [提交前检查](#提交前检查)
- [CI 质量门禁](#ci-质量门禁)
- [最佳实践](#最佳实践)

---

## 代码质量标准

### 质量指标

| 指标 | 目标 | 当前 | 工具 |
|------|------|------|------|
| Go 测试覆盖率 | ≥ 60% | 13.9% | go test |
| 前端测试覆盖率 | ≥ 60% | 5.09% | vitest |
| TypeScript 类型覆盖率 | ≥ 85% | ~60% | type-coverage |
| Go lint 通过率 | 100% | - | golangci-lint |
| 前端 lint 通过率 | 100% | - | ESLint |
| 代码格式一致性 | 100% | - | gofmt, prettier |

### 质量门禁

所有代码提交必须通过以下检查：

1. ✅ 代码格式检查
2. ✅ Lint 检查
3. ✅ 类型检查（TypeScript）
4. ✅ 单元测试
5. ✅ 覆盖率阈值检查

---

## 工具配置

### EditorConfig

项目使用 `.editorconfig` 统一不同编辑器的代码风格：

```ini
# 所有文件
[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true
trim_trailing_whitespace = true

# Go 文件
[*.go]
indent_style = tab
indent_size = 4

# TypeScript/JavaScript
[*.{ts,js}]
indent_style = space
indent_size = 2
```

**支持的编辑器**：
- VS Code（需要安装 EditorConfig 插件）
- IntelliJ IDEA / GoLand（内置支持）
- Vim（需要安装插件）
- Sublime Text（需要安装插件）

### Prettier

前端代码使用 Prettier 进行格式化：

```json
{
  "semi": true,
  "singleQuote": true,
  "printWidth": 100,
  "tabWidth": 2
}
```

### ESLint

前端代码使用 ESLint 进行 lint 检查，配置文件：`eslint.config.js`

支持：
- JavaScript (ES2021)
- TypeScript
- 自动修复

### golangci-lint

Go 代码使用 golangci-lint 进行 lint 检查，配置文件：`.golangci.yml`

启用的 linters：
- errcheck - 检查未处理的错误
- gosimple - 简化代码建议
- govet - Go 官方 vet 工具
- staticcheck - 静态分析
- gosec - 安全检查
- gocyclo - 复杂度检查
- dupl - 重复代码检查
- 等等...

---

## 代码格式化

### Go 代码格式化

```bash
# 格式化所有 Go 代码
make fmt

# 检查格式（不修改）
make fmt-check

# 使用 gofmt 直接格式化
gofmt -w .

# 格式化特定文件
gofmt -w internal/handler/gemini.go
```

**规则**：
- 使用 tab 缩进
- 每个文件末尾有换行符
- 移除尾随空格
- 遵循 Go 官方格式规范

### 前端代码格式化

```bash
# 格式化所有前端代码
make web-fmt

# 检查格式（不修改）
make web-fmt-check

# 自动修复格式问题
make web-fmt-fix

# 使用 prettier 直接格式化
prettier --write "web/**/*.{js,ts,json,css,html}"
```

**规则**：
- 使用空格缩进（2 个空格）
- 使用单引号
- 每行最多 100 个字符
- 使用分号
- 尾随逗号（多行）

### 格式化所有代码

```bash
# 格式化 Go + 前端代码
make fmt-fix
```

---

## Lint 检查

### Go Lint

```bash
# 基础 lint（go vet）
make lint

# 完整 lint（golangci-lint）
golangci-lint run

# 自动修复部分问题
make lint-fix
golangci-lint run --fix

# 只检查新代码
golangci-lint run --new-from-rev=HEAD~1
```

**常见问题修复**：

1. **未使用的变量**：
   ```go
   // 错误
   func example() {
       unused := 123
   }
   
   // 正确
   func example() {
       _ = 123  // 明确忽略
   }
   ```

2. **未检查的错误**：
   ```go
   // 错误
   file.Close()
   
   // 正确
   defer file.Close()
   // 或
   if err := file.Close(); err != nil {
       log.Printf("failed to close: %v", err)
   }
   ```

3. **循环变量引用**：
   ```go
   // 错误
   for _, item := range items {
       go func() {
           process(item)  // 可能引用错误的 item
       }()
   }
   
   // 正确
   for _, item := range items {
       item := item  // 创建副本
       go func() {
           process(item)
       }()
   }
   ```

### 前端 Lint

```bash
# 运行 ESLint
make web-lint
npm run lint

# 自动修复
make web-lint-fix
npm run lint:fix

# 检查特定文件
npx eslint web/src/auth.ts
```

**常见问题修复**：

1. **未使用的变量**：
   ```typescript
   // 错误
   const unused = 123;
   
   // 正确 - 移除或使用下划线前缀
   const _unused = 123;
   ```

2. **使用 any 类型**：
   ```typescript
   // 警告
   function process(data: any) {}
   
   // 推荐
   function process(data: unknown) {}
   // 或定义具体类型
   function process(data: UserData) {}
   ```

3. **缺少分号**：
   ```typescript
   // 错误
   const x = 1
   
   // 正确
   const x = 1;
   ```

### 运行所有 Lint

```bash
# Go + 前端 lint
make lint-all
```

---

## 类型检查

### TypeScript 类型检查

```bash
# 运行类型检查
make typecheck
cd web && npm run typecheck

# 查看详细错误
cd web && npx tsc --noEmit --pretty

# 检查类型覆盖率
cd web && npm run type:coverage
```

**类型覆盖率目标**：≥ 85%

**常见类型错误修复**：

1. **隐式 any**：
   ```typescript
   // 错误
   function process(data) {
       return data.value;
   }
   
   // 正确
   function process(data: { value: string }) {
       return data.value;
   }
   ```

2. **可能为 undefined**：
   ```typescript
   // 错误
   window.credsManager.load();
   
   // 正确
   window.credsManager?.load();
   // 或
   if (window.credsManager) {
       window.credsManager.load();
   }
   ```

3. **类型断言**：
   ```typescript
   // 不推荐
   const element = document.getElementById('id') as HTMLInputElement;
   
   // 推荐
   const element = document.getElementById('id');
   if (element instanceof HTMLInputElement) {
       element.value = 'test';
   }
   ```

---

## 提交前检查

### 手动检查

```bash
# 快速检查（格式 + lint + 类型）
./scripts/quality_check.sh quick

# 完整检查（包括测试）
./scripts/quality_check.sh all

# 只检查格式
./scripts/quality_check.sh format

# 只检查 lint
./scripts/quality_check.sh lint

# 只检查类型
./scripts/quality_check.sh types

# 只运行测试
./scripts/quality_check.sh test
```

### 自动检查（Git Hook）

安装 pre-commit hook：

```bash
# 复制示例文件
cp scripts/pre-commit.sample .git/hooks/pre-commit

# 设置可执行权限
chmod +x .git/hooks/pre-commit
```

Hook 会在每次 `git commit` 前自动运行：
- Go 代码格式化
- Go lint 检查
- TypeScript 类型检查
- 前端 lint 检查

**跳过 hook**（不推荐）：
```bash
git commit --no-verify -m "message"
```

---

## CI 质量门禁

### GitHub Actions 工作流

项目的 CI 流程包含以下质量检查：

#### 后端检查
1. Go mod tidy
2. Go 代码格式检查
3. Go lint（go vet + golangci-lint）
4. Go 测试（带覆盖率阈值）
5. 构建检查

#### 前端检查
1. 依赖安装
2. TypeScript 类型检查
3. 前端 lint
4. 前端测试（带覆盖率阈值）
5. Bundle 大小检查

#### 集成检查
1. 类型生成
2. Web 同步检查
3. 类型覆盖率检查

### 本地模拟 CI

```bash
# 运行完整 CI 流程
make ci

# 运行快速 CI（跳过耗时检查）
make ci-fast

# 运行质量检查
make quality-check
```

---

## 最佳实践

### 代码风格

1. **保持一致性**：
   - 遵循项目的代码风格
   - 使用自动格式化工具
   - 不要手动调整格式

2. **命名规范**：
   - Go：驼峰命名（CamelCase）
   - TypeScript：驼峰命名（camelCase）
   - 常量：大写下划线（UPPER_SNAKE_CASE）
   - 私有成员：下划线前缀（_private）

3. **注释规范**：
   - 公共 API 必须有文档注释
   - 复杂逻辑添加解释注释
   - 使用 TODO/FIXME 标记待办事项

### 提交规范

1. **提交前检查**：
   ```bash
   # 运行快速检查
   ./scripts/quality_check.sh quick
   
   # 或使用 pre-commit hook
   ```

2. **提交信息格式**：
   ```
   <type>(<scope>): <subject>
   
   <body>
   
   <footer>
   ```
   
   类型：
   - feat: 新功能
   - fix: 修复 bug
   - docs: 文档更新
   - style: 代码格式（不影响功能）
   - refactor: 重构
   - test: 测试相关
   - chore: 构建/工具相关

3. **小步提交**：
   - 每次提交只做一件事
   - 保持提交历史清晰
   - 便于代码审查和回滚

### 代码审查

1. **自我审查**：
   - 提交前自己先审查一遍
   - 运行所有质量检查
   - 确保测试通过

2. **审查清单**：
   - [ ] 代码格式正确
   - [ ] Lint 检查通过
   - [ ] 类型检查通过
   - [ ] 测试覆盖充分
   - [ ] 文档已更新
   - [ ] 无安全问题
   - [ ] 性能可接受

---

## 故障排查

### 常见问题

#### 1. golangci-lint 未安装

```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# 验证安装
golangci-lint --version
```

#### 2. Prettier 未安装

```bash
# 全局安装
npm install -g prettier

# 或使用 npx
npx prettier --version
```

#### 3. 格式检查失败

```bash
# 自动修复所有格式问题
make fmt-fix

# 或分别修复
make fmt        # Go
make web-fmt    # 前端
```

#### 4. Lint 检查失败

```bash
# 查看详细错误
golangci-lint run --verbose

# 自动修复部分问题
make lint-fix
make web-lint-fix
```

---

## 相关资源

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- [EditorConfig](https://editorconfig.org/)
- [Prettier](https://prettier.io/)
- [ESLint](https://eslint.org/)
- [golangci-lint](https://golangci-lint.run/)

---

**最后更新**: 2025-11-01  
**维护者**: gcli2api-go 团队


# 测试指南

本文档介绍 gcli2api-go 项目的测试策略、工具和最佳实践。

## 📋 目录

- [测试策略](#测试策略)
- [快速开始](#快速开始)
- [后端测试](#后端测试)
- [前端测试](#前端测试)
- [覆盖率要求](#覆盖率要求)
- [CI/CD 流程](#cicd-流程)
- [最佳实践](#最佳实践)

---

## 测试策略

### 测试金字塔

```
        /\
       /  \      E2E 测试 (少量)
      /----\
     /      \    集成测试 (适量)
    /--------\
   /          \  单元测试 (大量)
  /____________\
```

### 覆盖率目标

| 类型 | 当前 | 阶段目标 | 最终目标 |
|------|------|----------|----------|
| Go 后端 | 13.9% | 50% | 60%+ |
| 前端 | 5.09% | 40% | 60%+ |
| 整体 | ~10% | 45% | 60%+ |

---

## 快速开始

### 运行所有测试

```bash
# 后端测试
make test

# 前端测试
make web-test

# 所有测试（带覆盖率）
make test-with-threshold
make web-test-with-threshold
```

### 查看覆盖率报告

```bash
# 后端覆盖率
make go-coverage

# 前端覆盖率（生成 HTML 报告）
make web-test-coverage
open web/coverage/index.html
```

---

## 后端测试

### 测试框架

- **框架**: Go 标准库 `testing`
- **断言**: `testify/assert`
- **Mock**: `testify/mock`
- **数据库**: PostgreSQL (测试容器)

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/storage/...

# 运行特定测试
go test -run TestCredentialManager ./internal/storage/...

# 带覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 使用 Makefile

```bash
# 基础测试
make test

# 带覆盖率阈值检查（50%）
make test-with-threshold

# 生成覆盖率报告
make go-coverage
```

### 测试结构

```
internal/
├── storage/
│   ├── file_test.go          # 文件存储测试
│   ├── redis_test.go         # Redis 存储测试
│   ├── postgres_test.go      # PostgreSQL 存储测试
│   └── mongodb_test.go       # MongoDB 存储测试
├── handler/
│   ├── gemini_test.go        # Gemini 处理器测试
│   └── openai_test.go        # OpenAI 处理器测试
└── middleware/
    └── auth_test.go          # 认证中间件测试
```

### 编写测试示例

```go
package storage

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestFileStorage_SaveCredential(t *testing.T) {
    // Arrange
    storage := NewFileStorage("./testdata")
    cred := &Credential{
        Email: "test@example.com",
        Token: "test-token",
    }
    
    // Act
    err := storage.SaveCredential(cred)
    
    // Assert
    assert.NoError(t, err)
    assert.FileExists(t, "./testdata/test@example.com.json")
    
    // Cleanup
    defer os.Remove("./testdata/test@example.com.json")
}
```

---

## 前端测试

### 测试框架

- **框架**: Vitest
- **环境**: jsdom
- **覆盖率**: v8
- **断言**: Vitest 内置

### 运行测试

```bash
# 进入 web 目录
cd web

# 运行所有测试
npm test

# 监听模式
npm run test:watch

# 带覆盖率
npm run test:coverage

# 查看覆盖率报告
open coverage/index.html
```

### 使用 Makefile

```bash
# 基础测试
make web-test

# 带覆盖率阈值检查（40%）
make web-test-with-threshold

# 生成覆盖率报告
make web-test-coverage
```

### 测试结构

```
web/
├── tests/
│   ├── unit/
│   │   ├── auth.test.ts      # 认证模块测试
│   │   ├── api.test.ts       # API 客户端测试
│   │   └── ui.test.ts        # UI 工具测试
│   ├── integration/
│   │   └── admin.test.ts     # 管理界面集成测试
│   └── e2e/
│       └── workflow.test.ts  # 端到端测试
└── vitest.config.ts
```

### 编写测试示例

```typescript
import { describe, it, expect, beforeEach } from 'vitest';
import { AuthManager } from '../src/auth';

describe('AuthManager', () => {
  let auth: AuthManager;

  beforeEach(() => {
    auth = new AuthManager();
  });

  it('should login successfully with valid credentials', async () => {
    // Arrange
    const username = 'admin';
    const password = 'password';

    // Act
    const result = await auth.login(username, password);

    // Assert
    expect(result.success).toBe(true);
    expect(result.token).toBeDefined();
  });

  it('should fail login with invalid credentials', async () => {
    // Arrange
    const username = 'admin';
    const password = 'wrong';

    // Act & Assert
    await expect(auth.login(username, password)).rejects.toThrow();
  });
});
```

---

## 覆盖率要求

### 阈值配置

#### 后端 (Go)

当前阈值：**50%**

```bash
# 在 Makefile 中配置
GO_THRESHOLD=50 make test-with-threshold
```

#### 前端 (TypeScript)

当前阈值：**40%**

```javascript
// vitest.config.ts
export default defineConfig({
  test: {
    coverage: {
      thresholds: {
        lines: 40,
        statements: 40,
        functions: 30,
        branches: 25,
      },
    },
  },
});
```

### 覆盖率检查脚本

```bash
# 运行统一的覆盖率检查
./scripts/check_coverage.sh

# 自定义阈值
GO_THRESHOLD=60 WEB_THRESHOLD=50 ./scripts/check_coverage.sh
```

### 排除规则

#### 后端排除

- 生成的代码 (`*.pb.go`)
- 测试文件 (`*_test.go`)
- Main 函数 (`cmd/`)

#### 前端排除

- `node_modules/`
- `dist/`
- `coverage/`
- 类型定义文件 (`*.d.ts`)
- 配置文件 (`*.config.*`)
- Mock 数据 (`mockData/`)
- 测试文件 (`tests/`)

---

## CI/CD 流程

### GitHub Actions 工作流

项目使用 GitHub Actions 进行持续集成，包含三个并行任务：

#### 1. 后端检查 (`backend`)

```yaml
- Go mod tidy
- 数据库迁移
- Go lint (vet)
- Go 测试（带覆盖率阈值）
- 构建
```

#### 2. 前端检查 (`frontend`)

```yaml
- 安装依赖
- TypeScript 类型检查
- ESLint 检查
- 前端测试（带覆盖率阈值）
- 上传覆盖率报告到 Codecov
```

#### 3. 集成检查 (`integration`)

```yaml
- 类型生成
- Web 同步检查
- Bundle 大小检查
- 类型覆盖率检查
```

### 本地 CI 模拟

```bash
# 运行完整 CI 流程
make ci

# 运行快速 CI（跳过耗时检查）
make ci-fast
```

### CI 失败处理

1. **后端测试失败**:
   ```bash
   # 查看失败的测试
   go test ./... -v
   
   # 运行特定测试
   go test -run TestFailingTest ./path/to/package
   ```

2. **前端测试失败**:
   ```bash
   cd web
   npm test -- --reporter=verbose
   ```

3. **覆盖率不达标**:
   ```bash
   # 查看覆盖率报告
   make go-coverage
   make web-test-coverage
   
   # 识别未覆盖的代码
   go tool cover -html=coverage.out
   open web/coverage/index.html
   ```

---

## 最佳实践

### 测试命名

```go
// Go: TestFunctionName_Scenario_ExpectedBehavior
func TestUserService_CreateUser_WithValidData_ReturnsUser(t *testing.T) {}
```

```typescript
// TypeScript: describe + it
describe('UserService', () => {
  it('should create user with valid data', () => {});
});
```

### AAA 模式

所有测试应遵循 **Arrange-Act-Assert** 模式：

```go
func TestExample(t *testing.T) {
    // Arrange - 准备测试数据和环境
    user := &User{Name: "John"}
    
    // Act - 执行被测试的操作
    result := service.CreateUser(user)
    
    // Assert - 验证结果
    assert.NoError(t, result.Error)
    assert.Equal(t, "John", result.User.Name)
}
```

### 测试隔离

- 每个测试应该独立运行
- 使用 `beforeEach` / `afterEach` 清理状态
- 避免测试之间的依赖

### Mock 使用

```go
// 使用 testify/mock
type MockStorage struct {
    mock.Mock
}

func (m *MockStorage) Save(data interface{}) error {
    args := m.Called(data)
    return args.Error(0)
}

func TestWithMock(t *testing.T) {
    mockStorage := new(MockStorage)
    mockStorage.On("Save", mock.Anything).Return(nil)
    
    // 使用 mock
    service := NewService(mockStorage)
    err := service.DoSomething()
    
    assert.NoError(t, err)
    mockStorage.AssertExpectations(t)
}
```

### 表驱动测试

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"negative numbers", -1, -2, -3},
        {"mixed", 1, -1, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

---

## 故障排查

### 常见问题

#### 1. 测试超时

```bash
# 增加超时时间
go test ./... -timeout 30s
```

#### 2. 数据库连接失败

```bash
# 确保 PostgreSQL 正在运行
docker-compose up -d postgres

# 检查连接
psql $POSTGRES_DSN
```

#### 3. 前端测试失败

```bash
# 清理缓存
cd web
rm -rf node_modules coverage
npm install
npm test
```

#### 4. 覆盖率计算错误

```bash
# 清理旧的覆盖率文件
rm -f coverage.out
rm -rf web/coverage

# 重新运行
make test-with-threshold
make web-test-with-threshold
```

---

## 相关资源

- [Go Testing 文档](https://golang.org/pkg/testing/)
- [Testify 文档](https://github.com/stretchr/testify)
- [Vitest 文档](https://vitest.dev/)
- [测试最佳实践](https://github.com/goldbergyoni/javascript-testing-best-practices)

---

**最后更新**: 2025-11-01  
**维护者**: gcli2api-go 团队


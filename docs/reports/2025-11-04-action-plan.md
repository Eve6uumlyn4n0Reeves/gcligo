# GCLI2API-Go 技术债务改进行动计划

**生成日期**: 2025-11-04  
**关联报告**: [2025-11-04-technical-debt-analysis.md](./2025-11-04-technical-debt-analysis.md)

---

## 🎯 总体目标

在接下来的 2 个月内，将项目质量从当前的 **7.2/10** 提升至 **8.5/10**。

### 关键指标目标

| 指标 | 当前 | 目标 | 截止日期 |
|------|------|------|----------|
| Go 测试覆盖率 | 13.9% | 60% | 2025-12-31 |
| 前端测试覆盖率 | 5.09% | 60% | 2025-12-31 |
| TypeScript 类型覆盖率 | ~60% | 85% | 2025-12-15 |
| 代码格式一致性 | ~94% | 100% | 2025-11-08 |
| 代码重复率 | ~15% | <5% | 2025-12-15 |

---

## 📅 第一周行动清单（2025-11-04 至 2025-11-10）

### Day 1-2: 代码格式修复 ✅

**优先级**: P0  
**预计工时**: 2 小时

```bash
# 1. 修复所有格式问题
cd gcli2api-go
make fmt

# 2. 验证格式
make fmt-check

# 3. 提交修复
git add .
git commit -m "chore: fix code formatting issues"

# 4. 启用 pre-commit hook
cp scripts/pre-commit.sample .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

**验收标准**:
- [ ] `make fmt-check` 无输出
- [ ] `gofmt -l .` 无输出
- [ ] pre-commit hook 已启用

---

### Day 3-5: 核心模块测试补充（第一批）

**优先级**: P0  
**预计工时**: 12 小时

#### 任务 1: OpenAI Handler 测试

**文件**: `internal/handlers/openai/openai_chat_test.go`

```go
package openai

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "gcli2api-go/internal/config"
    "gcli2api-go/internal/credential"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestChatCompletions_ValidRequest(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    // Setup
    cfg := &config.Config{
        Server: config.ServerConfig{OpenAIPort: "8317"},
    }
    credMgr := credential.NewManager(credential.Options{})
    
    handler := NewHandler(cfg, credMgr, nil, nil, nil)
    
    // Create request
    reqBody := map[string]interface{}{
        "model": "gemini-2.5-pro",
        "messages": []map[string]interface{}{
            {"role": "user", "content": "Hello"},
        },
    }
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    c, _ := gin.CreateTestContext(w)
    c.Request = req
    
    // Execute
    handler.ChatCompletions(c)
    
    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
}

func TestChatCompletions_InvalidJSON(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    cfg := &config.Config{}
    handler := NewHandler(cfg, nil, nil, nil, nil)
    
    req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte("invalid")))
    w := httptest.NewRecorder()
    
    c, _ := gin.CreateTestContext(w)
    c.Request = req
    
    handler.ChatCompletions(c)
    
    assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestChatCompletions_MissingModel(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    cfg := &config.Config{}
    handler := NewHandler(cfg, nil, nil, nil, nil)
    
    reqBody := map[string]interface{}{
        "messages": []map[string]interface{}{
            {"role": "user", "content": "Hello"},
        },
    }
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    c, _ := gin.CreateTestContext(w)
    c.Request = req
    
    handler.ChatCompletions(c)
    
    // Should use default model
    assert.NotEqual(t, http.StatusBadRequest, w.Code)
}
```

**验收标准**:
- [ ] 至少 10 个测试用例
- [ ] 覆盖正常流程、错误处理、边界条件
- [ ] 模块覆盖率 > 50%

---

#### 任务 2: 路由策略测试

**文件**: `internal/upstream/strategy/strategy_pick_test.go`

```go
package strategy

import (
    "context"
    "net/http"
    "testing"
    "time"

    "gcli2api-go/internal/config"
    "gcli2api-go/internal/credential"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestStrategy_Pick_NoCredentials(t *testing.T) {
    cfg := &config.Config{}
    credMgr := credential.NewManager(credential.Options{})
    
    s := NewStrategy(cfg, credMgr, nil)
    
    cred := s.Pick(context.Background(), http.Header{})
    assert.Nil(t, cred)
}

func TestStrategy_Pick_SingleCredential(t *testing.T) {
    cfg := &config.Config{}
    credMgr := credential.NewManager(credential.Options{})
    
    // Add a credential
    testCred := &credential.Credential{
        ID:          "test-1",
        AccessToken: "token-1",
        ProjectID:   "project-1",
    }
    // ... setup credential manager
    
    s := NewStrategy(cfg, credMgr, nil)
    
    cred := s.Pick(context.Background(), http.Header{})
    require.NotNil(t, cred)
    assert.Equal(t, "test-1", cred.ID)
}

func TestStrategy_Pick_StickyRouting(t *testing.T) {
    cfg := &config.Config{
        Routing: config.RoutingConfig{
            StickyTTLSeconds: 300,
        },
    }
    credMgr := credential.NewManager(credential.Options{})
    
    // Setup multiple credentials
    // ...
    
    s := NewStrategy(cfg, credMgr, nil)
    
    // First request with sticky key
    hdr := http.Header{}
    hdr.Set("X-Session-ID", "session-123")
    
    cred1 := s.Pick(context.Background(), hdr)
    require.NotNil(t, cred1)
    
    // Second request with same sticky key should get same credential
    cred2 := s.Pick(context.Background(), hdr)
    require.NotNil(t, cred2)
    assert.Equal(t, cred1.ID, cred2.ID)
}

func TestStrategy_Pick_Cooldown(t *testing.T) {
    cfg := &config.Config{
        Routing: config.RoutingConfig{
            CooldownBaseMS: 1000,
            CooldownMaxMS:  5000,
        },
    }
    credMgr := credential.NewManager(credential.Options{})
    
    // Setup credentials
    // ...
    
    s := NewStrategy(cfg, credMgr, nil)
    
    // Pick a credential
    cred1 := s.Pick(context.Background(), http.Header{})
    require.NotNil(t, cred1)
    
    // Mark it as cooled down
    s.RecordFailure(cred1.ID, 429)
    
    // Next pick should skip the cooled down credential
    cred2 := s.Pick(context.Background(), http.Header{})
    if cred2 != nil {
        assert.NotEqual(t, cred1.ID, cred2.ID)
    }
}

func TestStrategy_Pick_WeightedSelection(t *testing.T) {
    cfg := &config.Config{}
    credMgr := credential.NewManager(credential.Options{})
    
    // Setup multiple credentials with different scores
    // ...
    
    s := NewStrategy(cfg, credMgr, nil)
    
    // Pick multiple times and verify distribution
    picks := make(map[string]int)
    for i := 0; i < 100; i++ {
        cred := s.Pick(context.Background(), http.Header{})
        if cred != nil {
            picks[cred.ID]++
        }
    }
    
    // Verify that picks are distributed (not all same credential)
    assert.Greater(t, len(picks), 1)
}
```

**验收标准**:
- [ ] 至少 8 个测试用例
- [ ] 覆盖粘性路由、冷却、权重选择
- [ ] 模块覆盖率 > 60%

---

### Day 6-7: 存储后端测试

**优先级**: P0  
**预计工时**: 8 小时

#### 任务 3: 文件后端测试

**文件**: `internal/storage/file_backend_comprehensive_test.go`

```go
package storage

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFileBackend_CRUD(t *testing.T) {
    // Create temp directory
    tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)
    
    backend := NewFileBackend(tmpDir)
    ctx := context.Background()
    
    // Initialize
    err = backend.Initialize(ctx)
    require.NoError(t, err)
    defer backend.Close()
    
    // Test SetCredential
    cred := map[string]interface{}{
        "id":           "test-1",
        "access_token": "token-1",
        "project_id":   "project-1",
    }
    err = backend.SetCredential(ctx, "test-1", cred)
    require.NoError(t, err)
    
    // Test GetCredential
    retrieved, err := backend.GetCredential(ctx, "test-1")
    require.NoError(t, err)
    assert.Equal(t, "token-1", retrieved["access_token"])
    
    // Test ListCredentials
    ids, err := backend.ListCredentials(ctx)
    require.NoError(t, err)
    assert.Contains(t, ids, "test-1")
    
    // Test DeleteCredential
    err = backend.DeleteCredential(ctx, "test-1")
    require.NoError(t, err)
    
    _, err = backend.GetCredential(ctx, "test-1")
    assert.Error(t, err)
}

func TestFileBackend_Config(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)
    
    backend := NewFileBackend(tmpDir)
    ctx := context.Background()
    
    err = backend.Initialize(ctx)
    require.NoError(t, err)
    defer backend.Close()
    
    // Test SetConfig
    err = backend.SetConfig(ctx, "test-key", "test-value")
    require.NoError(t, err)
    
    // Test GetConfig
    value, err := backend.GetConfig(ctx, "test-key")
    require.NoError(t, err)
    assert.Equal(t, "test-value", value)
    
    // Test ListConfigs
    configs, err := backend.ListConfigs(ctx)
    require.NoError(t, err)
    assert.Equal(t, "test-value", configs["test-key"])
    
    // Test DeleteConfig
    err = backend.DeleteConfig(ctx, "test-key")
    require.NoError(t, err)
    
    _, err = backend.GetConfig(ctx, "test-key")
    assert.Error(t, err)
}

func TestFileBackend_Usage(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)
    
    backend := NewFileBackend(tmpDir)
    ctx := context.Background()
    
    err = backend.Initialize(ctx)
    require.NoError(t, err)
    defer backend.Close()
    
    // Test IncrementUsage
    err = backend.IncrementUsage(ctx, "user-1", "requests", 1)
    require.NoError(t, err)
    
    err = backend.IncrementUsage(ctx, "user-1", "requests", 5)
    require.NoError(t, err)
    
    // Test GetUsage
    usage, err := backend.GetUsage(ctx, "user-1")
    require.NoError(t, err)
    assert.Equal(t, int64(6), usage["requests"])
    
    // Test ResetUsage
    err = backend.ResetUsage(ctx, "user-1")
    require.NoError(t, err)
    
    _, err = backend.GetUsage(ctx, "user-1")
    assert.Error(t, err)
}

func TestFileBackend_Persistence(t *testing.T) {
    tmpDir, err := os.MkdirTemp("", "file-backend-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)
    
    // First instance
    backend1 := NewFileBackend(tmpDir)
    ctx := context.Background()
    
    err = backend1.Initialize(ctx)
    require.NoError(t, err)
    
    cred := map[string]interface{}{
        "id":           "test-1",
        "access_token": "token-1",
    }
    err = backend1.SetCredential(ctx, "test-1", cred)
    require.NoError(t, err)
    
    err = backend1.Close()
    require.NoError(t, err)
    
    // Second instance (should load persisted data)
    backend2 := NewFileBackend(tmpDir)
    err = backend2.Initialize(ctx)
    require.NoError(t, err)
    defer backend2.Close()
    
    retrieved, err := backend2.GetCredential(ctx, "test-1")
    require.NoError(t, err)
    assert.Equal(t, "token-1", retrieved["access_token"])
}
```

**验收标准**:
- [ ] 至少 15 个测试用例
- [ ] 覆盖 CRUD、持久化、并发
- [ ] 模块覆盖率 > 70%

---

## 📅 第二周行动清单（2025-11-11 至 2025-11-17）

### 存储后端重构

**优先级**: P1  
**预计工时**: 16 小时

#### 步骤 1: 创建通用适配器

**文件**: `internal/storage/common/backend_adapter.go`

```go
package common

import (
    "context"
    "encoding/json"
)

// BackendAdapter 提供存储后端的通用适配逻辑
type BackendAdapter struct {
    codec *CredentialCodec
}

func NewBackendAdapter() *BackendAdapter {
    return &BackendAdapter{
        codec: NewCredentialCodec(),
    }
}

// AdaptGetCredential 适配 GetCredential 操作
func (a *BackendAdapter) AdaptGetCredential(
    ctx context.Context,
    id string,
    getter func(context.Context, string) ([]byte, error),
) (map[string]interface{}, error) {
    data, err := getter(ctx, id)
    if err != nil {
        return nil, err
    }
    return a.codec.UnmarshalMap(data)
}

// AdaptSetCredential 适配 SetCredential 操作
func (a *BackendAdapter) AdaptSetCredential(
    ctx context.Context,
    id string,
    data map[string]interface{},
    setter func(context.Context, string, []byte) error,
) error {
    payload, err := a.codec.MarshalMap(data)
    if err != nil {
        return err
    }
    return setter(ctx, id, payload)
}

// AdaptGetConfig 适配 GetConfig 操作
func (a *BackendAdapter) AdaptGetConfig(
    ctx context.Context,
    key string,
    getter func(context.Context, string) ([]byte, error),
) (interface{}, error) {
    data, err := getter(ctx, key)
    if err != nil {
        return nil, err
    }
    var value interface{}
    if err := json.Unmarshal(data, &value); err != nil {
        return nil, err
    }
    return value, nil
}

// AdaptSetConfig 适配 SetConfig 操作
func (a *BackendAdapter) AdaptSetConfig(
    ctx context.Context,
    key string,
    value interface{},
    setter func(context.Context, string, []byte) error,
) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return setter(ctx, key, data)
}
```

#### 步骤 2: 重构 MongoDB 后端

修改 `internal/storage/mongodb_backend.go`:

```go
type MongoDBBackend struct {
    storage *mongodb.MongoDBStorage
    adapter *common.BackendAdapter  // 新增
}

func NewMongoDBBackend(uri, dbName string) (*MongoDBBackend, error) {
    storage, err := mongodb.NewMongoDBStorage(uri, dbName)
    if err != nil {
        return nil, err
    }
    
    return &MongoDBBackend{
        storage: storage,
        adapter: common.NewBackendAdapter(),  // 新增
    }, nil
}

func (m *MongoDBBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
    return m.adapter.AdaptGetCredential(ctx, id, m.storage.GetCredential)
}

func (m *MongoDBBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
    return m.adapter.AdaptSetCredential(ctx, id, data, m.storage.SetCredential)
}
```

**验收标准**:
- [ ] 减少 200+ 行重复代码
- [ ] 所有后端使用统一适配器
- [ ] 测试全部通过

---

### 前端测试补充

**优先级**: P1  
**预计工时**: 12 小时

#### 任务: 核心组件测试

**文件**: `web/tests/auth.test.ts`

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { AuthManager } from '../src/auth';

describe('AuthManager', () => {
  let auth: AuthManager;

  beforeEach(() => {
    auth = new AuthManager();
    localStorage.clear();
  });

  describe('isAuthenticated', () => {
    it('should return false when no token', () => {
      expect(auth.isAuthenticated()).toBe(false);
    });

    it('should return true when valid token exists', () => {
      localStorage.setItem('auth_token', 'test-token');
      expect(auth.isAuthenticated()).toBe(true);
    });
  });

  describe('login', () => {
    it('should store token on successful login', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ token: 'new-token' }),
      });
      global.fetch = mockFetch;

      await auth.login('test-key');

      expect(localStorage.getItem('auth_token')).toBe('new-token');
      expect(auth.isAuthenticated()).toBe(true);
    });

    it('should throw error on failed login', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
      });
      global.fetch = mockFetch;

      await expect(auth.login('invalid-key')).rejects.toThrow();
    });
  });

  describe('logout', () => {
    it('should clear token', () => {
      localStorage.setItem('auth_token', 'test-token');
      auth.logout();
      expect(localStorage.getItem('auth_token')).toBeNull();
      expect(auth.isAuthenticated()).toBe(false);
    });
  });
});
```

**验收标准**:
- [ ] 至少 20 个前端测试用例
- [ ] 覆盖率 > 40%
- [ ] 所有测试通过

---

## 📅 第三-四周行动清单（2025-11-18 至 2025-12-01）

### 集成测试补充

**优先级**: P1  
**预计工时**: 20 小时

- [ ] 端到端测试（OpenAI API 流程）
- [ ] 存储后端集成测试（使用 testcontainers）
- [ ] 路由策略集成测试
- [ ] 凭证管理集成测试

### 性能优化

**优先级**: P2  
**预计工时**: 12 小时

- [ ] 热路径性能分析
- [ ] 内存分配优化
- [ ] 缓存策略优化
- [ ] 性能基准测试

---

## 📊 进度跟踪

### 每周检查点

**每周五下午**进行进度检查：

```bash
# 1. 运行测试并生成覆盖率报告
make test-coverage

# 2. 检查代码质量
make lint
make fmt-check

# 3. 更新进度表
# 在本文档中更新完成状态
```

### 月度回顾

**每月最后一天**进行月度回顾：

1. 回顾本月完成的任务
2. 分析未完成任务的原因
3. 调整下月计划
4. 更新技术债务报告

---

## 🎓 学习资源

### Go 测试最佳实践
- [Go Testing By Example](https://go.dev/doc/tutorial/add-a-test)
- [Table Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Testify Documentation](https://github.com/stretchr/testify)

### 前端测试
- [Vitest Documentation](https://vitest.dev/)
- [Testing Library](https://testing-library.com/)
- [TypeScript Testing](https://www.typescriptlang.org/docs/handbook/testing.html)

### 性能优化
- [Go Performance Tips](https://github.com/dgryski/go-perfbook)
- [pprof Tutorial](https://go.dev/blog/pprof)

---

## 📞 支持和协作

### 遇到问题时

1. **查看文档**: `docs/` 目录
2. **运行诊断**: `make health-check`
3. **查看日志**: `tail -f logs/server.log`
4. **寻求帮助**: 创建 GitHub Issue

### 代码审查

所有改进都应该：
- [ ] 通过所有测试
- [ ] 通过 lint 检查
- [ ] 更新相关文档
- [ ] 添加测试用例

---

**维护者**: gcli2api-go 团队  
**最后更新**: 2025-11-04


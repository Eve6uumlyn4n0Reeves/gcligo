# Storage 模块文档

## 模块定位与职责

Storage 模块是 gcli2api-go 的**持久化存储核心**，提供统一的存储抽象层，支持多种后端实现，包括：

- **统一接口**：Backend 接口定义凭证、配置、用量统计、缓存的 CRUD 操作
- **多后端支持**：File、Redis、MongoDB、PostgreSQL 四种存储后端
- **批量操作**：高性能批量读写、批量删除
- **事务支持**：PostgreSQL 支持 ACID 事务（可选）
- **数据迁移**：导出/导入功能，支持跨后端迁移
- **可观测性**：Instrumented Backend 包装器，自动记录指标和追踪
- **连接池管理**：Redis/MongoDB/PostgreSQL 连接池统计与监控
- **幂等性保证**：ConfigBatchApplier 支持两阶段配置变更与重放保护

## 目录结构与文件职责

```
internal/storage/
├── interface.go                          # Backend 接口定义、错误类型、统计结构
├── file_backend.go                       # File 后端实现（内存 + 磁盘持久化）
├── file_backend_io.go                    # File 后端 I/O 操作（加载/保存）
├── redis_backend.go                      # Redis 后端实现
├── redis_backend_batch.go                # Redis 批量操作
├── redis_backend_usage.go                # Redis 用量统计
├── redis_backend_config_cache.go         # Redis 配置与缓存操作
├── redis_backend_plan.go                 # Redis 配置批量应用（幂等性）
├── mongodb_backend.go                    # MongoDB 后端实现
├── mongodb_backend_batch.go              # MongoDB 批量操作
├── mongodb_backend_usage.go              # MongoDB 用量统计
├── mongodb_backend_config.go             # MongoDB 配置操作
├── mongodb_backend_plan.go               # MongoDB 配置批量应用
├── postgres_backend.go                   # PostgreSQL 后端实现
├── postgres_backend_batch.go             # PostgreSQL 批量操作
├── postgres_backend_usage.go             # PostgreSQL 用量统计
├── postgres_backend_config.go            # PostgreSQL 配置操作
├── postgres_backend_tx.go                # PostgreSQL 事务实现
├── instrumented_backend.go               # 可观测性包装器（指标 + 追踪）
├── backend_helpers.go                    # 通用辅助函数（导出/导入/统计）
├── unsupported_ops.go                    # 不支持操作的默认实现
├── labels.go                             # 标签管理（用于分类存储）
├── git_backend.go                        # Git 后端（实验性，用于版本控制）
├── common/
│   ├── backend_adapter.go                # 后端适配器（序列化/反序列化）
│   ├── batch_processor.go                # 批量处理器（并发控制）
│   ├── credential_codec.go               # 凭证编解码器
│   ├── serializer.go                     # JSON 序列化器
│   ├── error_mapper.go                   # 错误映射器
│   ├── validator.go                      # 数据验证器
│   ├── pools.go                          # 对象池（减少 GC 压力）
│   ├── copy.go                           # 深拷贝/浅拷贝工具
│   └── unsupported_ops.go                # 通用不支持操作实现
├── mongodb/
│   └── mongodb_storage.go                # MongoDB 底层存储实现
├── postgres/
│   └── postgres_storage.go               # PostgreSQL 底层存储实现
└── migration/
    └── migrator.go                       # 数据迁移工具
```

## 核心设计与数据流

### 1. 存储架构层次

```
Application Layer (Credential Manager / Config Manager)
    ↓
Instrumented Backend (Metrics + Tracing)
    ↓
Backend Interface (Unified API)
    ↓
Backend Implementation (File / Redis / MongoDB / PostgreSQL)
    ↓
Low-Level Storage (Disk / Network / Database)
```

### 2. Backend 接口设计

Backend 接口定义了 5 类操作：

1. **生命周期管理**：`Initialize()`、`Close()`、`Health()`
2. **凭证操作**：`GetCredential()`、`SetCredential()`、`DeleteCredential()`、`ListCredentials()`
3. **配置操作**：`GetConfig()`、`SetConfig()`、`DeleteConfig()`、`ListConfigs()`
4. **用量统计**：`IncrementUsage()`、`GetUsage()`、`ResetUsage()`、`ListUsage()`
5. **缓存操作**：`GetCache()`、`SetCache()`、`DeleteCache()`（可选）
6. **批量操作**：`BatchGetCredentials()`、`BatchSetCredentials()`、`BatchDeleteCredentials()`
7. **事务支持**：`BeginTransaction()`（可选）
8. **数据迁移**：`ExportData()`、`ImportData()`
9. **监控统计**：`GetStorageStats()`

### 3. 后端特性对比

| 特性 | File | Redis | MongoDB | PostgreSQL |
|------|------|-------|---------|------------|
| 凭证存储 | ✅ | ✅ | ✅ | ✅ |
| 配置存储 | ✅ | ✅ | ✅ | ✅ |
| 用量统计 | ✅ | ✅ | ✅ | ✅ |
| 缓存支持 | ❌ | ✅ | ❌ | ❌ |
| 批量操作 | ✅ | ✅ | ✅ | ✅ |
| 事务支持 | ❌ | ❌ | ❌ | ✅ |
| 连接池 | N/A | ✅ | ✅ | ✅ |
| 持久化 | 磁盘 | 内存+AOF | 磁盘 | 磁盘 |
| 性能 | 中 | 高 | 中 | 中 |
| 适用场景 | 开发/测试 | 生产（高并发） | 生产（文档型） | 生产（关系型） |

### 4. Instrumented Backend 包装

所有后端都可以通过 `WithInstrumentation()` 包装，自动记录：

- **Prometheus 指标**：操作延迟、错误率、连接池状态
- **OpenTelemetry 追踪**：每个操作的 Span、错误记录
- **连接池监控**：活跃连接数、空闲连接数、命中率

### 5. 批量操作优化

批量操作使用 `BatchProcessor` 实现并发控制：

- **并发限制**：默认 10 个并发 goroutine，可配置
- **错误隔离**：单个操作失败不影响其他操作
- **结果汇总**：返回成功/失败统计和详细错误列表

## 关键类型与接口

### Backend 接口

```go
type Backend interface {
    // 生命周期
    Initialize(ctx context.Context) error
    Close() error
    Health(ctx context.Context) error
    
    // 凭证操作
    GetCredential(ctx context.Context, id string) (map[string]interface{}, error)
    SetCredential(ctx context.Context, id string, data map[string]interface{}) error
    DeleteCredential(ctx context.Context, id string) error
    ListCredentials(ctx context.Context) ([]string, error)
    
    // 配置操作
    GetConfig(ctx context.Context, key string) (interface{}, error)
    SetConfig(ctx context.Context, key string, value interface{}) error
    DeleteConfig(ctx context.Context, key string) error
    ListConfigs(ctx context.Context) (map[string]interface{}, error)
    
    // 用量统计
    IncrementUsage(ctx context.Context, key string, field string, delta int64) error
    GetUsage(ctx context.Context, key string) (map[string]interface{}, error)
    ResetUsage(ctx context.Context, key string) error
    ListUsage(ctx context.Context) (map[string]map[string]interface{}, error)
    
    // 缓存操作（可选）
    GetCache(ctx context.Context, key string) ([]byte, error)
    SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error
    DeleteCache(ctx context.Context, key string) error
    
    // 批量操作
    BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error)
    BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error
    BatchDeleteCredentials(ctx context.Context, ids []string) error
    
    // 事务支持（可选）
    BeginTransaction(ctx context.Context) (Transaction, error)
    
    // 数据迁移
    ExportData(ctx context.Context) (map[string]interface{}, error)
    ImportData(ctx context.Context, data map[string]interface{}) error
    
    // 监控统计
    GetStorageStats(ctx context.Context) (StorageStats, error)
}
```

### Transaction 接口

```go
type Transaction interface {
    GetCredential(ctx context.Context, id string) (map[string]interface{}, error)
    SetCredential(ctx context.Context, id string, data map[string]interface{}) error
    DeleteCredential(ctx context.Context, id string) error
    
    GetConfig(ctx context.Context, key string) (interface{}, error)
    SetConfig(ctx context.Context, key string, value interface{}) error
    DeleteConfig(ctx context.Context, key string) error
    
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

### StorageStats 结构

```go
type StorageStats struct {
    Backend          string                 `json:"backend"`
    Healthy          bool                   `json:"healthy"`
    ConnectionCount  int                    `json:"connection_count,omitempty"`
    CredentialCount  int                    `json:"credential_count"`
    ConfigCount      int                    `json:"config_count"`
    UsageRecordCount int                    `json:"usage_record_count"`
    TotalSize        int64                  `json:"total_size_bytes,omitempty"`
    LastBackup       *time.Time             `json:"last_backup,omitempty"`
    Performance      *PerformanceStats      `json:"performance,omitempty"`
    Details          map[string]interface{} `json:"details,omitempty"`
}
```

### ConfigBatchApplier 接口

```go
type ConfigBatchApplier interface {
    ApplyConfigBatch(ctx context.Context, mutations []ConfigMutation, opts BatchApplyOptions) error
}

type ConfigMutation struct {
    Key    string
    Value  interface{}
    Delete bool
}

type BatchApplyOptions struct {
    IdempotencyKey string        // 幂等性键（防重放）
    TTL            time.Duration // 元数据保留时间
    Stage          string        // 阶段标签（apply/rollback）
}
```

## 重要配置项

### File Backend

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `baseDir` | string | - | 数据存储根目录 |

### Redis Backend

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `addr` | string | - | Redis 地址（host:port） |
| `password` | string | "" | Redis 密码 |
| `db` | int | 0 | Redis 数据库编号 |
| `prefix` | string | "gcli2api:" | 键前缀 |
| `dialTimeout` | duration | 5s | 连接超时 |
| `readTimeout` | duration | 3s | 读取超时 |
| `writeTimeout` | duration | 3s | 写入超时 |
| `poolSize` | int | 10 | 连接池大小 |
| `minIdleConns` | int | 2 | 最小空闲连接数 |

### MongoDB Backend

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `uri` | string | - | MongoDB 连接 URI |
| `dbName` | string | - | 数据库名称 |

### PostgreSQL Backend

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `dsn` | string | - | PostgreSQL DSN（连接字符串） |

## 与其他模块的依赖关系

### 依赖的模块

- **oauth**：凭证结构定义（`oauth.Credentials`）
- **monitoring**：指标记录（`EnhancedMetrics`）、连接池统计（`StoragePoolStats`）
- **tracing**：OpenTelemetry 追踪（`StartSpan`）

### 被依赖的模块

- **credential**：使用 Backend 存储凭证状态、健康评分、封禁记录
- **config**：使用 Backend 存储动态配置（可选）
- **stats/usage**：使用 Backend 存储用量统计

## 可执行示例

### 示例 1：初始化 File Backend

```go
package main

import (
    "context"
    "fmt"
    "gcli2api-go/internal/storage"
)

func main() {
    // 创建 File Backend
    backend := storage.NewFileBackend("/var/lib/gcli2api/data")

    // 初始化（创建目录、加载数据）
    ctx := context.Background()
    if err := backend.Initialize(ctx); err != nil {
        panic(err)
    }
    defer backend.Close()

    // 健康检查
    if err := backend.Health(ctx); err != nil {
        fmt.Printf("Backend unhealthy: %v\n", err)
    } else {
        fmt.Println("Backend healthy")
    }
}
```

### 示例 2：凭证 CRUD 操作

```go
package main

import (
    "context"
    "fmt"
    "gcli2api-go/internal/storage"
)

func main() {
    backend := storage.NewFileBackend("/tmp/gcli2api")
    backend.Initialize(context.Background())
    defer backend.Close()

    ctx := context.Background()

    // 存储凭证
    credData := map[string]interface{}{
        "access_token":  "ya29.xxx",
        "refresh_token": "1//xxx",
        "expiry":        "2024-12-31T23:59:59Z",
        "project_id":    "my-project-123",
    }
    if err := backend.SetCredential(ctx, "cred-001", credData); err != nil {
        panic(err)
    }
    fmt.Println("Credential saved")

    // 读取凭证
    retrieved, err := backend.GetCredential(ctx, "cred-001")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Retrieved: %+v\n", retrieved)

    // 列出所有凭证
    ids, err := backend.ListCredentials(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Credential IDs: %v\n", ids)

    // 删除凭证
    if err := backend.DeleteCredential(ctx, "cred-001"); err != nil {
        panic(err)
    }
    fmt.Println("Credential deleted")
}
```

### 示例 3：批量操作

```go
package main

import (
    "context"
    "fmt"
    "gcli2api-go/internal/storage"
)

func main() {
    backend := storage.NewFileBackend("/tmp/gcli2api")
    backend.Initialize(context.Background())
    defer backend.Close()

    ctx := context.Background()

    // 批量写入凭证
    batchData := map[string]map[string]interface{}{
        "cred-001": {"access_token": "token1", "project_id": "proj1"},
        "cred-002": {"access_token": "token2", "project_id": "proj2"},
        "cred-003": {"access_token": "token3", "project_id": "proj3"},
    }
    if err := backend.BatchSetCredentials(ctx, batchData); err != nil {
        panic(err)
    }
    fmt.Println("Batch set completed")

    // 批量读取凭证
    ids := []string{"cred-001", "cred-002", "cred-003"}
    results, err := backend.BatchGetCredentials(ctx, ids)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Batch get results: %d credentials\n", len(results))

    // 批量删除凭证
    if err := backend.BatchDeleteCredentials(ctx, ids); err != nil {
        panic(err)
    }
    fmt.Println("Batch delete completed")
}
```

### 示例 4：Redis Backend 使用

```go
package main

import (
    "context"
    "fmt"
    "gcli2api-go/internal/storage"
    "time"
)

func main() {
    // 创建 Redis Backend
    backend, err := storage.NewRedisBackend(
        "localhost:6379", // addr
        "",               // password
        0,                // db
        "gcli2api:",      // prefix
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()
    if err := backend.Initialize(ctx); err != nil {
        panic(err)
    }
    defer backend.Close()

    // 使用缓存功能（Redis 独有）
    cacheKey := "model:gemini-2.5-pro:capabilities"
    cacheValue := []byte(`{"supports_thinking":true,"max_tokens":8192}`)

    // 设置缓存（TTL 5 分钟）
    if err := backend.SetCache(ctx, cacheKey, cacheValue, 5*time.Minute); err != nil {
        panic(err)
    }
    fmt.Println("Cache set")

    // 读取缓存
    retrieved, err := backend.GetCache(ctx, cacheKey)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Cache retrieved: %s\n", string(retrieved))

    // 删除缓存
    if err := backend.DeleteCache(ctx, cacheKey); err != nil {
        panic(err)
    }
    fmt.Println("Cache deleted")
}
```

### 示例 5：PostgreSQL 事务

```go
package main

import (
    "context"
    "fmt"
    "gcli2api-go/internal/storage"
)

func main() {
    backend, err := storage.NewPostgresBackend(
        "postgres://user:pass@localhost:5432/gcli2api?sslmode=disable",
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()
    if err := backend.Initialize(ctx); err != nil {
        panic(err)
    }
    defer backend.Close()

    // 开始事务
    tx, err := backend.BeginTransaction(ctx)
    if err != nil {
        panic(err)
    }

    // 在事务中操作
    credData := map[string]interface{}{
        "access_token": "ya29.xxx",
        "project_id":   "my-project",
    }
    if err := tx.SetCredential(ctx, "cred-tx-001", credData); err != nil {
        tx.Rollback(ctx)
        panic(err)
    }

    configData := map[string]interface{}{
        "max_retries": 3,
        "timeout":     30,
    }
    if err := tx.SetConfig(ctx, "retry_config", configData); err != nil {
        tx.Rollback(ctx)
        panic(err)
    }

    // 提交事务
    if err := tx.Commit(ctx); err != nil {
        panic(err)
    }
    fmt.Println("Transaction committed")
}
```

### 示例 6：Instrumented Backend

```go
package main

import (
    "context"
    "gcli2api-go/internal/monitoring"
    "gcli2api-go/internal/storage"
)

func main() {
    // 创建原始 Backend
    rawBackend := storage.NewFileBackend("/tmp/gcli2api")

    // 创建指标收集器
    metrics := monitoring.NewEnhancedMetrics()

    // 包装为 Instrumented Backend
    backend := storage.WithInstrumentation(rawBackend, metrics, "file")

    ctx := context.Background()
    backend.Initialize(ctx)
    defer backend.Close()

    // 所有操作自动记录指标和追踪
    backend.SetCredential(ctx, "cred-001", map[string]interface{}{
        "access_token": "token",
    })

    // 查看统计信息
    stats, _ := backend.GetStorageStats(ctx)
    fmt.Printf("Backend: %s, Credentials: %d\n", stats.Backend, stats.CredentialCount)
}
```

### 示例 7：数据导出与导入

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "gcli2api-go/internal/storage"
    "os"
)

func main() {
    // 从 File Backend 导出数据
    fileBackend := storage.NewFileBackend("/var/lib/gcli2api/data")
    fileBackend.Initialize(context.Background())

    ctx := context.Background()
    exportData, err := fileBackend.ExportData(ctx)
    if err != nil {
        panic(err)
    }

    // 保存到 JSON 文件
    jsonData, _ := json.MarshalIndent(exportData, "", "  ")
    os.WriteFile("/tmp/backup.json", jsonData, 0644)
    fmt.Println("Data exported to /tmp/backup.json")

    fileBackend.Close()

    // 导入到 Redis Backend
    redisBackend, _ := storage.NewRedisBackend("localhost:6379", "", 0, "gcli2api:")
    redisBackend.Initialize(ctx)

    var importData map[string]interface{}
    jsonData, _ = os.ReadFile("/tmp/backup.json")
    json.Unmarshal(jsonData, &importData)

    if err := redisBackend.ImportData(ctx, importData); err != nil {
        panic(err)
    }
    fmt.Println("Data imported to Redis")

    redisBackend.Close()
}
```

## 架构示意图

```mermaid
graph TB
    subgraph "Application Layer"
        CM[Credential Manager]
        CFG[Config Manager]
        STATS[Usage Stats]
    end

    subgraph "Instrumentation Layer"
        IB[Instrumented Backend]
        METRICS[Prometheus Metrics]
        TRACE[OpenTelemetry Traces]
    end

    subgraph "Backend Interface"
        IFACE[Backend Interface]
    end

    subgraph "Backend Implementations"
        FILE[File Backend]
        REDIS[Redis Backend]
        MONGO[MongoDB Backend]
        PG[PostgreSQL Backend]
    end

    subgraph "Common Utilities"
        ADAPTER[Backend Adapter]
        BATCH[Batch Processor]
        CODEC[Credential Codec]
        SERIAL[Serializer]
    end

    subgraph "Low-Level Storage"
        DISK[Disk Files]
        REDISDB[(Redis)]
        MONGODB[(MongoDB)]
        POSTGRES[(PostgreSQL)]
    end

    CM --> IB
    CFG --> IB
    STATS --> IB

    IB --> METRICS
    IB --> TRACE
    IB --> IFACE

    IFACE --> FILE
    IFACE --> REDIS
    IFACE --> MONGO
    IFACE --> PG

    FILE --> ADAPTER
    REDIS --> ADAPTER
    MONGO --> ADAPTER
    PG --> ADAPTER

    FILE --> BATCH
    REDIS --> BATCH
    MONGO --> BATCH
    PG --> BATCH

    ADAPTER --> CODEC
    ADAPTER --> SERIAL

    FILE --> DISK
    REDIS --> REDISDB
    MONGO --> MONGODB
    PG --> POSTGRES

    style IB fill:#4CAF50
    style IFACE fill:#2196F3
    style ADAPTER fill:#FF9800
    style BATCH fill:#9C27B0
```

## 已知限制

1. **File Backend 并发性能**
   - 使用全局锁保护内存数据，高并发场景性能受限
   - 解决方案：生产环境使用 Redis/MongoDB/PostgreSQL

2. **Redis 持久化风险**
   - 默认配置可能丢失数据（仅内存）
   - 解决方案：启用 AOF 或 RDB 持久化

3. **MongoDB 事务支持**
   - 当前实现未使用 MongoDB 事务特性
   - 解决方案：扩展 `BeginTransaction()` 支持 MongoDB 会话

4. **PostgreSQL 连接池**
   - 连接池大小由 DSN 参数控制，无法动态调整
   - 解决方案：使用连接池配置参数（`pool_max_conns`）

5. **批量操作原子性**
   - 批量操作非原子，部分成功/部分失败
   - 解决方案：使用 PostgreSQL 事务或实现补偿逻辑

6. **缓存 TTL 精度**
   - Redis 缓存 TTL 精度为秒级
   - 解决方案：如需毫秒级 TTL，使用 `PSETEX` 命令

7. **导出/导入性能**
   - 大数据量导出/导入可能耗时较长
   - 解决方案：使用流式导出或分批处理

## 最佳实践

1. **选择合适的后端**：开发用 File，生产用 Redis/MongoDB/PostgreSQL
2. **启用 Instrumentation**：使用 `WithInstrumentation()` 包装后端，便于监控
3. **使用批量操作**：批量读写性能远高于单条操作
4. **合理设置连接池**：根据并发量调整连接池大小
5. **定期备份数据**：使用 `ExportData()` 定期导出数据
6. **监控连接池状态**：通过 `PoolStats()` 监控连接池健康度
7. **处理 ErrNotFound**：区分 `ErrNotFound` 和其他错误，避免误报
8. **使用事务保证一致性**：关键操作使用 PostgreSQL 事务
9. **配置合理的超时**：避免长时间阻塞操作
10. **测试数据迁移**：跨后端迁移前先在测试环境验证



# Monitoring 模块文档

## 模块定位与职责

Monitoring 模块是 gcli2api-go 的**可观测性核心**，负责指标收集、慢查询日志和分布式追踪：

- **Prometheus 指标**：定义 40+ 个 Prometheus 指标（Counter、Gauge、Histogram）
- **增强指标**：EnhancedMetrics 提供内存聚合统计（P50/P95/P99、错误分类）
- **指标收集器**：MetricsCollector 提供时间窗口统计和端点级指标
- **慢查询日志**：SlowQueryLogger 记录超过阈值的操作（默认 100ms）
- **分布式追踪**：OpenTelemetry 集成（OTLP gRPC 导出）
- **全局访问**：DefaultMetrics() 提供进程级指标访问
- **多维度标签**：支持 server、provider、model、endpoint、status_class 等标签

## 目录结构与文件职责

```
internal/monitoring/
├── metrics.go                          # Prometheus 指标定义（40+ 指标）
├── detailed_metrics.go                 # EnhancedMetrics 增强指标（内存聚合）
├── global.go                           # 全局指标访问（DefaultMetrics）
├── metrics_collector.go                # MetricsCollector 指标收集器（时间窗口）
├── slow_query.go                       # SlowQueryLogger 慢查询日志
└── tracing/
    └── tracing.go                      # OpenTelemetry 追踪集成
```

## 核心设计与数据流

### 1. 指标体系架构

```
HTTP Request
    ↓
Middleware (mw.Metrics())
    ↓
HTTPRequestsTotal.Inc()
HTTPRequestDuration.Observe()
    ↓
Handler 处理
    ↓
Upstream Request
    ↓
UpstreamRequestsTotal.Inc()
UpstreamRequestDuration.Observe()
    ↓
EnhancedMetrics.RecordUpstreamRequest()
    ↓
Prometheus /metrics 端点
```

### 2. 指标分类

**HTTP 请求指标**（3 个）：
- `gcli2api_http_requests_total`：HTTP 请求总数（server、method、path、status_class）
- `gcli2api_http_request_duration_seconds`：HTTP 请求延迟（Histogram）
- `gcli2api_http_inflight`：当前并发请求数（Gauge）

**凭证指标**（3 个）：
- `gcli2api_credential_rotations_total`：凭证轮换次数
- `gcli2api_credential_errors_total`：凭证错误次数（credential、error_code）
- `gcli2api_credential_refreshes_total`：Token 刷新次数（credential、status）

**上游 API 指标**（6 个）：
- `gcli2api_upstream_requests_total`：上游请求总数（provider、status_class）
- `gcli2api_upstream_request_duration_seconds`：上游请求延迟（provider）
- `gcli2api_upstream_request_duration_server_seconds`：按服务器分组的延迟（provider、server）
- `gcli2api_upstream_errors_total`：上游错误次数（provider、reason）
- `gcli2api_upstream_retry_attempts_total`：重试次数（provider、outcome）
- `gcli2api_upstream_model_requests_total`：按模型分组的请求（provider、model、status_class）

**流式传输指标**（6 个）：
- `gcli2api_sse_lines_total`：SSE 行数（server、path）
- `gcli2api_sse_disconnects_total`：SSE 断连次数（server、path、reason）
- `gcli2api_tool_calls_total`：工具调用次数（server、path）
- `gcli2api_anti_truncation_attempts_total`：抗截断尝试次数（server、path）
- `gcli2api_model_fallbacks_total`：模型回退次数（server、path、from_model、to_model）
- `gcli2api_thinking_removed_total`：Thinking 配置移除次数（server、path、model）

**管理端指标**（4 个）：
- `gcli2api_management_access_total`：管理端访问决策（route、result、source）
- `gcli2api_ratelimit_keys`：限流器数量（Gauge）
- `gcli2api_ratelimit_sweeps_total`：限流器清理次数
- `gcli2api_assembly_operations_total`：装配台操作次数（action、status、actor）

**系统指标**（3 个）：
- `gcli2api_active_credentials`：活跃凭证数（Gauge）
- `gcli2api_disabled_credentials`：禁用凭证数（Gauge）
- `gcli2api_tokens_used_total`：Token 使用量（model、type）

**自动探活指标**（6 个）：
- `gcli2api_auto_probe_runs_total`：探活运行次数（source、status、model）
- `gcli2api_auto_probe_duration_seconds`：探活延迟（source、model）
- `gcli2api_auto_probe_success_ratio`：探活成功率（source、model）
- `gcli2api_auto_probe_target_credentials`：探活凭证数（source、model）
- `gcli2api_auto_probe_last_success_unix`：最后成功时间戳（source、model）

**上游发现指标**（6 个）：
- `gcli2api_upstream_discovery_cache_hits_total`：缓存命中次数
- `gcli2api_upstream_discovery_fetch_total`：刷新尝试次数（result）
- `gcli2api_upstream_discovery_fetch_duration_seconds`：刷新延迟
- `gcli2api_upstream_discovery_known_bases`：已知基础模型数（Gauge）
- `gcli2api_upstream_discovery_cache_expires_unix`：缓存过期时间戳（Gauge）
- `gcli2api_upstream_discovery_last_success_unix`：最后成功时间戳（Gauge）

**路由策略指标**（5 个）：
- `gcli2api_routing_sticky_hits_total`：粘性路由命中次数（source）
- `gcli2api_routing_cooldown_events_total`：冷却事件次数（status）
- `gcli2api_routing_sticky_size`：粘性路由条目数（Gauge）
- `gcli2api_routing_cooldown_size`：冷却条目数（Gauge）
- `gcli2api_routing_cooldown_remaining_seconds`：冷却剩余时间分布（Histogram）

### 3. EnhancedMetrics 架构

```
EnhancedMetrics（内存聚合）
    ↓
RecordUpstreamRequest()
    ↓
存储到 map[provider][]durations
    ↓
GetSnapshot()
    ↓
计算 P50/P95/P99、平均值、错误分类
    ↓
返回 JSON 快照
```

### 4. 慢查询日志流程

```
操作开始
    ↓
SlowQueryLogger.Track()
    ↓
执行操作（fn()）
    ↓
计算耗时
    ↓
耗时 >= 阈值（100ms）？
    ↓ 是
记录到 queries 切片
    ↓
超过 maxSize（1000）？
    ↓ 是
移除最旧记录
```

### 5. OpenTelemetry 追踪

```
环境变量 OTEL_EXPORTER_OTLP_ENDPOINT
    ↓
tracing.Init(ctx)
    ↓
创建 OTLP gRPC Exporter
    ↓
创建 TracerProvider
    ↓
otel.SetTracerProvider()
    ↓
tracing.StartSpan(ctx, component, spanName)
    ↓
执行操作
    ↓
span.End()
    ↓
导出到 OTLP Collector
```

## 关键类型与接口

### EnhancedMetrics 结构

```go
type EnhancedMetrics struct {
    mu sync.RWMutex

    // 上游请求指标
    upstreamRequests    map[string]int64            // provider -> count
    upstreamDurations   map[string][]float64        // provider -> durations
    upstreamErrors      map[string]map[string]int64 // provider -> error_type -> count
    upstreamRetries     map[string]int64            // provider -> retry_count
    upstreamStatusCodes map[string]map[int]int64    // provider -> status_code -> count

    // 端点指标
    endpointRequests  map[string]int64     // endpoint -> count
    endpointDurations map[string][]float64 // endpoint -> durations
    endpointErrors    map[string]int64     // endpoint -> error_count

    // 流式指标
    streamingRequests    int64
    streamingChunks      int64
    streamingDisconnects map[string]int64 // reason -> count

    // 凭证指标
    credentialRotations   int64
    credentialFailures    map[string]int64   // cred_id -> failure_count
    credentialHealthScore map[string]float64 // cred_id -> score

    // 缓存指标
    cacheHits   int64
    cacheMisses int64

    // Token 使用
    totalTokens      int64
    promptTokens     int64
    completionTokens int64

    // 事务指标
    transactionAttempts map[string]int64 // backend -> attempts
    transactionSuccess  map[string]int64 // backend -> commits
    transactionFailures map[string]int64 // backend -> rollbacks

    // 存储指标
    storageOps       map[string]map[string]*storageOpAggregate
    storageSlowOps   map[string]map[string]int64
    storagePoolStats map[string]StoragePoolStats

    // 装配台指标
    planOps map[planOpKey]*PlanOpStats
}
```

### MetricsCollector 结构

```go
type MetricsCollector struct {
    mu sync.RWMutex

    // 请求计数
    totalRequests   atomic.Int64
    successRequests atomic.Int64
    failedRequests  atomic.Int64

    // 响应时间
    totalDuration atomic.Int64 // 纳秒
    minDuration   atomic.Int64 // 纳秒
    maxDuration   atomic.Int64 // 纳秒

    // 按端点统计
    endpointStats map[string]*EndpointStats

    // 按状态码统计
    statusCodeStats map[int]*atomic.Int64

    // 时间窗口统计
    windowStats *WindowStats

    // 启动时间
    startTime time.Time
}
```

### SlowQueryLogger 结构

```go
type SlowQueryLogger struct {
    mu        sync.RWMutex
    threshold time.Duration // 慢查询阈值（默认 100ms）
    enabled   bool
    queries   []SlowQuery   // 慢查询记录
    maxSize   int           // 最大记录数（默认 1000）
}

type SlowQuery struct {
    Timestamp time.Time     `json:"timestamp"`
    Operation string        `json:"operation"`
    Duration  time.Duration `json:"duration"`
    Details   string        `json:"details"`
    Stack     string        `json:"stack,omitempty"`
}
```

## 重要配置项

### OpenTelemetry 环境变量

| 环境变量 | 类型 | 默认值 | 说明 |
|---------|------|--------|------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | - | OTLP gRPC 端点（如 `localhost:4317`） |
| `OTEL_EXPORTER_OTLP_INSECURE` | bool | `true` | 是否使用不安全连接 |

### 慢查询配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `threshold` | duration | `100ms` | 慢查询阈值 |
| `maxSize` | int | `1000` | 最大记录数 |

### Histogram Buckets

| 指标 | Buckets（秒） |
|------|--------------|
| `http_request_duration_seconds` | `0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10` |
| `upstream_request_duration_seconds` | `0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10` |
| `routing_cooldown_remaining_seconds` | `0, 1, 2, 5, 10, 20, 30, 60, 120, 300, 600` |

## 与其他模块的依赖关系

### 依赖的模块

- **version**：服务版本信息（OpenTelemetry Resource）
- **prometheus/client_golang**：Prometheus 客户端库
- **go.opentelemetry.io/otel**：OpenTelemetry SDK

### 被依赖的模块

- **middleware**：中间件调用 Prometheus 指标
- **upstream**：上游调用记录指标
- **credential**：凭证管理记录指标
- **storage**：存储操作记录指标
- **server**：服务器暴露 `/metrics` 端点
- **handlers**：处理器记录端点指标

## 可执行示例

### 示例 1：记录上游请求指标

```go
package main

import (
    "gcli2api-go/internal/monitoring"
    "time"
)

func main() {
    metrics := monitoring.NewEnhancedMetrics()

    // 记录成功请求
    metrics.RecordUpstreamRequest("gemini", 250*time.Millisecond, 200, nil)

    // 记录失败请求
    metrics.RecordUpstreamRequest("gemini", 500*time.Millisecond, 500, fmt.Errorf("server error"))

    // 获取快照
    snapshot := metrics.GetSnapshot()
    fmt.Printf("Upstream metrics: %+v\n", snapshot["upstream"])
}
```

### 示例 2：使用 Prometheus 指标

```go
package main

import (
    "gcli2api-go/internal/monitoring"
    "time"
)

func handleRequest() {
    start := time.Now()

    // 记录请求
    monitoring.HTTPRequestsTotal.WithLabelValues("openai", "POST", "/v1/chat/completions", "2xx").Inc()

    // 处理请求...
    time.Sleep(100 * time.Millisecond)

    // 记录延迟
    duration := time.Since(start).Seconds()
    monitoring.HTTPRequestDuration.WithLabelValues("openai", "POST", "/v1/chat/completions", "2xx").Observe(duration)
}
```

### 示例 3：记录慢查询

```go
package main

import (
    "context"
    "gcli2api-go/internal/monitoring"
    "time"
)

func main() {
    logger := monitoring.NewSlowQueryLogger(100*time.Millisecond, 1000)

    // 跟踪操作
    err := logger.Track(context.Background(), "database_query", func() error {
        time.Sleep(150 * time.Millisecond) // 模拟慢查询
        return nil
    })

    // 获取慢查询记录
    queries := logger.GetRecentQueries(10)
    for _, q := range queries {
        fmt.Printf("[%s] %s took %v\n", q.Timestamp, q.Operation, q.Duration)
    }

    // 获取统计信息
    stats := logger.GetStats()
    fmt.Printf("Slow queries: %d, avg: %v, max: %v\n",
        stats.Count, stats.AvgDuration, stats.MaxDuration)
}
```

### 示例 4：启用 OpenTelemetry 追踪

```bash
# 设置环境变量
export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
export OTEL_EXPORTER_OTLP_INSECURE="true"

# 启动服务
./gcli2api-go
```

```go
package main

import (
    "context"
    "gcli2api-go/internal/monitoring/tracing"
)

func main() {
    ctx := context.Background()

    // 初始化追踪
    shutdown, err := tracing.Init(ctx)
    if err != nil {
        panic(err)
    }
    defer shutdown(ctx)

    // 创建 Span
    ctx, span := tracing.StartSpan(ctx, "handler", "process_request")
    defer span.End()

    // 执行操作...
    processRequest(ctx)
}

func processRequest(ctx context.Context) {
    ctx, span := tracing.StartSpan(ctx, "upstream", "call_gemini")
    defer span.End()

    // 调用上游...
}
```

### 示例 5：使用全局指标

```go
package main

import (
    "gcli2api-go/internal/monitoring"
)

func main() {
    // 创建并注册全局指标
    metrics := monitoring.NewEnhancedMetrics()
    monitoring.SetDefaultMetrics(metrics)

    // 在其他地方访问全局指标
    globalMetrics := monitoring.DefaultMetrics()
    if globalMetrics != nil {
        globalMetrics.RecordCacheHit()
    }
}
```

### 示例 6：记录存储操作指标

```go
package main

import (
    "gcli2api-go/internal/monitoring"
    "time"
)

func main() {
    metrics := monitoring.NewEnhancedMetrics()

    // 记录存储操作
    metrics.RecordStorageOperation("postgres", "GetCredential", 50*time.Millisecond, nil)
    metrics.RecordStorageOperation("postgres", "SaveCredential", 300*time.Millisecond, nil) // 慢操作

    // 获取存储指标
    ops, slow, pools := metrics.StorageMetrics()
    fmt.Printf("Storage operations: %+v\n", ops)
    fmt.Printf("Slow operations: %+v\n", slow)
}
```

### 示例 7：记录事务指标

```go
package main

import (
    "gcli2api-go/internal/monitoring"
    "time"
)

func main() {
    metrics := monitoring.NewEnhancedMetrics()

    // 记录事务
    metrics.RecordTransactionAttempt("postgres")

    // 执行事务...
    time.Sleep(100 * time.Millisecond)

    // 提交成功
    metrics.RecordTransactionCommit("postgres")

    // 或回滚失败
    // metrics.RecordTransactionFailure("postgres")
}
```

### 示例 8：查询 Prometheus 指标

```bash
# 查询 HTTP 请求总数
curl http://localhost:8317/metrics | grep gcli2api_http_requests_total

# 查询上游请求延迟 P95
curl http://localhost:8317/metrics | grep gcli2api_upstream_request_duration_seconds

# 查询活跃凭证数
curl http://localhost:8317/metrics | grep gcli2api_active_credentials

# 查询限流器数量
curl http://localhost:8317/metrics | grep gcli2api_ratelimit_keys
```

### 示例 9：使用 MetricsCollector

```go
package main

import (
    "gcli2api-go/internal/monitoring"
    "time"
)

func main() {
    collector := monitoring.NewMetricsCollector(time.Minute, 60)

    // 记录请求
    collector.RecordRequest("/v1/chat/completions", 200, 150*time.Millisecond, true)
    collector.RecordRequest("/v1/models", 200, 50*time.Millisecond, true)
    collector.RecordRequest("/v1/chat/completions", 500, 200*time.Millisecond, false)

    // 获取统计信息
    stats := collector.GetStats()
    fmt.Printf("Total requests: %d\n", stats.TotalRequests)
    fmt.Printf("Success rate: %.2f%%\n", stats.SuccessRate)
    fmt.Printf("Avg duration: %v\n", stats.AvgDuration)
    fmt.Printf("Endpoint stats: %+v\n", stats.EndpointStats)
}
```

### 示例 10：记录装配台操作

```go
package main

import (
    "gcli2api-go/internal/monitoring"
    "time"
)

func main() {
    metrics := monitoring.NewEnhancedMetrics()

    // 记录装配台计划应用
    metrics.RecordPlanApply("postgres", "validate", "success", 50*time.Millisecond)
    metrics.RecordPlanApply("postgres", "apply", "success", 200*time.Millisecond)

    // 获取装配台指标
    planMetrics := metrics.PlanMetrics()
    fmt.Printf("Plan metrics: %+v\n", planMetrics)
}
```

## 架构示意图

```mermaid
graph TB
    subgraph "Prometheus Metrics"
        HTTP[HTTP Metrics]
        UPSTREAM[Upstream Metrics]
        CRED[Credential Metrics]
        SSE[Streaming Metrics]
        MGMT[Management Metrics]
        SYS[System Metrics]
    end

    subgraph "EnhancedMetrics"
        MEMORY[Memory Aggregation]
        SNAPSHOT[Snapshot API]
        PERCENTILE[P50/P95/P99]
    end

    subgraph "MetricsCollector"
        WINDOW[Time Window]
        ENDPOINT[Endpoint Stats]
        STATUSCODE[Status Code Stats]
    end

    subgraph "SlowQueryLogger"
        THRESHOLD[Threshold Check]
        QUERIES[Query Records]
        STATS[Statistics]
    end

    subgraph "OpenTelemetry"
        TRACER[TracerProvider]
        EXPORTER[OTLP Exporter]
        SPAN[Span Creation]
    end

    subgraph "Data Sources"
        MIDDLEWARE[Middleware]
        HANDLERS[Handlers]
        UPSTREAMCALL[Upstream Calls]
        STORAGE[Storage Ops]
        ASSEMBLY[Assembly Ops]
    end

    subgraph "Consumers"
        PROMENDPOINT[/metrics Endpoint]
        MGMTAPI[Management API]
        OTLPCOLLECTOR[OTLP Collector]
    end

    MIDDLEWARE --> HTTP
    MIDDLEWARE --> SSE
    HANDLERS --> ENDPOINT
    UPSTREAMCALL --> UPSTREAM
    UPSTREAMCALL --> MEMORY
    STORAGE --> MEMORY
    ASSEMBLY --> MEMORY

    HTTP --> PROMENDPOINT
    UPSTREAM --> PROMENDPOINT
    CRED --> PROMENDPOINT
    SSE --> PROMENDPOINT
    MGMT --> PROMENDPOINT
    SYS --> PROMENDPOINT

    MEMORY --> SNAPSHOT
    SNAPSHOT --> MGMTAPI

    STORAGE --> THRESHOLD
    THRESHOLD --> QUERIES
    QUERIES --> STATS
    STATS --> MGMTAPI

    HANDLERS --> SPAN
    UPSTREAMCALL --> SPAN
    SPAN --> EXPORTER
    EXPORTER --> OTLPCOLLECTOR

    style HTTP fill:#4CAF50
    style UPSTREAM fill:#2196F3
    style MEMORY fill:#FF9800
    style THRESHOLD fill:#9C27B0
    style TRACER fill:#F44336
```

## 已知限制

1. **内存占用**
   - EnhancedMetrics 在内存中存储所有延迟数据（限制 1000 条）
   - 解决方案：定期清理或使用外部时序数据库

2. **标签基数**
   - 某些指标标签基数较高（如 `path`、`model`），可能导致内存膨胀
   - 解决方案：使用标签白名单或聚合高基数标签

3. **慢查询记录限制**
   - 慢查询记录最多 1000 条，超过后移除最旧记录
   - 解决方案：导出到外部日志系统

4. **无持久化**
   - 所有指标和慢查询记录仅存储在内存中，重启后丢失
   - 解决方案：使用 Prometheus 持久化或外部存储

5. **OpenTelemetry 单端点**
   - 仅支持单一 OTLP 端点，无法同时导出到多个 Collector
   - 解决方案：使用 OTLP Collector 的多 Exporter 功能

6. **时间窗口固定**
   - MetricsCollector 时间窗口大小固定（1 分钟），无法动态调整
   - 解决方案：支持配置文件或环境变量

7. **无采样**
   - OpenTelemetry 追踪无采样策略，所有请求都创建 Span
   - 解决方案：配置 TracerProvider 采样器

8. **指标无过期**
   - Prometheus 指标一旦创建，标签组合永久存在
   - 解决方案：定期重启服务或使用 Prometheus 的 `metric_relabel_configs`

## 最佳实践

1. **使用标签白名单**：限制高基数标签（如 `path`）的取值范围
2. **定期导出慢查询**：将慢查询记录导出到外部日志系统
3. **配置 Prometheus 抓取间隔**：建议 15-30 秒，避免过于频繁
4. **启用 OpenTelemetry 采样**：生产环境使用概率采样（如 10%）
5. **监控指标基数**：通过 Prometheus 查询 `count({__name__=~".+"})`
6. **使用 EnhancedMetrics 快照**：定期导出快照到外部存储
7. **设置合理的慢查询阈值**：根据业务场景调整（如 P95 延迟）
8. **分离 Prometheus 端点**：生产环境将 `/metrics` 绑定到内网
9. **使用 Grafana 可视化**：导入预设仪表板监控关键指标
10. **告警配置**：配置 Prometheus AlertManager 监控异常指标

## Prometheus 指标速查表

| 指标名称 | 类型 | 标签 | 说明 |
|---------|------|------|------|
| `gcli2api_http_requests_total` | Counter | server, method, path, status_class | HTTP 请求总数 |
| `gcli2api_http_request_duration_seconds` | Histogram | server, method, path, status_class | HTTP 请求延迟 |
| `gcli2api_http_inflight` | Gauge | - | 当前并发请求数 |
| `gcli2api_credential_rotations_total` | Counter | credential | 凭证轮换次数 |
| `gcli2api_credential_errors_total` | Counter | credential, error_code | 凭证错误次数 |
| `gcli2api_credential_refreshes_total` | Counter | credential, status | Token 刷新次数 |
| `gcli2api_upstream_requests_total` | Counter | provider, status_class | 上游请求总数 |
| `gcli2api_upstream_request_duration_seconds` | Histogram | provider | 上游请求延迟 |
| `gcli2api_upstream_errors_total` | Counter | provider, reason | 上游错误次数 |
| `gcli2api_upstream_retry_attempts_total` | Counter | provider, outcome | 重试次数 |
| `gcli2api_sse_lines_total` | Counter | server, path | SSE 行数 |
| `gcli2api_sse_disconnects_total` | Counter | server, path, reason | SSE 断连次数 |
| `gcli2api_management_access_total` | Counter | route, result, source | 管理端访问决策 |
| `gcli2api_active_credentials` | Gauge | - | 活跃凭证数 |
| `gcli2api_disabled_credentials` | Gauge | - | 禁用凭证数 |
| `gcli2api_tokens_used_total` | Counter | model, type | Token 使用量 |
| `gcli2api_auto_probe_runs_total` | Counter | source, status, model | 探活运行次数 |
| `gcli2api_routing_sticky_hits_total` | Counter | source | 粘性路由命中次数 |
| `gcli2api_routing_cooldown_size` | Gauge | - | 冷却条目数 |

## PromQL 查询示例

```promql
# HTTP 请求 QPS（每秒请求数）
rate(gcli2api_http_requests_total[5m])

# HTTP 请求 P95 延迟
histogram_quantile(0.95, rate(gcli2api_http_request_duration_seconds_bucket[5m]))

# 上游请求成功率
sum(rate(gcli2api_upstream_requests_total{status_class="2xx"}[5m]))
/
sum(rate(gcli2api_upstream_requests_total[5m]))

# 凭证错误率（按错误码）
sum by (error_code) (rate(gcli2api_credential_errors_total[5m]))

# SSE 断连率
rate(gcli2api_sse_disconnects_total[5m])

# 活跃凭证数趋势
gcli2api_active_credentials

# 探活成功率
gcli2api_auto_probe_success_ratio

# 粘性路由命中率
sum(rate(gcli2api_routing_sticky_hits_total[5m]))
/
sum(rate(gcli2api_http_requests_total[5m]))
```


## Phase 1-3 改进功能

### 回退透明化指标（Phase 2）

**新增指标**：
- `fallbackEvents`：模型回退事件统计（内存聚合）
  - 维度：`from_model:to_model:reason`
  - 字段：
    - `Count`：总回退次数
    - `SuccessCount`：成功回退次数
    - `FailureCount`：失败回退次数
    - `TotalDurationMS`：总耗时（毫秒）
    - `AvgDurationMS`：平均耗时（毫秒）

**API 访问**：
```go
// 记录回退事件
metrics.RecordFallback(fromModel, toModel, reason, success, durationMS)

// 获取回退统计
stats := metrics.GetFallbackStats()
// 返回 map[string]*FallbackStats
// 键格式："{from_model}:{to_model}:{reason}"
// 为避免高基数无限增长，GetFallbackStats() 仅返回 Top‑N（按 Count 降序，当前 N=200）。
```

**使用场景**：
- 监控模型可用性
- 分析回退模式
- 优化回退策略
- 端到端时延分解

### 缓存失效指标（Phase 2）

**新增指标**：
- `cacheInvalidations`：缓存失效事件统计（内存聚合）
  - 维度：`reason`
  - 字段：失效次数

**API 访问**：
```go
// 记录缓存失效
metrics.RecordCacheInvalidation(credID, reason)

// 获取失效统计
stats := metrics.GetCacheInvalidationStats()
// 返回 map[string]int64
// 键：失效原因，值：失效次数
```

**失效原因示例**：
- `credential_refresh`：凭证刷新
- `credential_rotation`：凭证轮换
- `manual_invalidation`：手动失效
- `credential_disabled`：凭证禁用
- `credential_error`：凭证错误

**使用场景**：
- 监控缓存一致性
- 分析失效模式
- 优化缓存策略
- 时序审计

### 熔断与冷却指标（Phase 3）

**新增指标**：
- `cooldownByModel`：冷却状态统计（内存聚合）
  - 维度：`credential_id:model:project`
  - 字段：
    - `ActiveCooldowns`：当前活跃冷却数
    - `TotalCooldowns`：总冷却次数
    - `LastCooldownAt`：最后冷却时间
    - `CooldownReason`：冷却原因

**API 访问**：
```go
// 记录冷却事件
metrics.RecordCooldown(credentialID, model, project, reason, active)

// 获取冷却统计
stats := metrics.GetCooldownStats()
// 返回 map[string]*CooldownStats
// 键格式："{credential_id}:{model}:{project}"
// 高基数治理：仅返回最近活跃 Top‑N（按 LastCooldownAt 降序，当前 N=200）。
```

**冷却原因示例**：
- `rate_limit`：速率限制
- `quota_exceeded`：配额超限
- `error_threshold`：错误阈值
- `manual_cooldown`：手动冷却

**使用场景**：
- 监控熔断状态
- 分析冷却模式
- 优化路由策略
- 多维度观测（凭证-模型-项目）

### 指标导出

所有新增指标通过 `EnhancedMetrics` 提供内存聚合，可通过以下方式访问：

1. **程序内访问**：
```go
metrics := monitoring.DefaultMetrics()
fallbackStats := metrics.GetFallbackStats()
invalidationStats := metrics.GetCacheInvalidationStats()
cooldownStats := metrics.GetCooldownStats()
```

2. **管理端点**（待实现）：
```
GET /api/management/metrics/fallback
GET /api/management/metrics/cache-invalidation
GET /api/management/metrics/cooldown
```

3. **Prometheus 导出**（待实现）：
```
gcli2api_fallback_total{from_model="...",to_model="...",reason="..."}
gcli2api_cache_invalidation_total{reason="..."}
gcli2api_cooldown_active{credential_id="...",model="...",project="..."}
gcli2api_cooldown_total{credential_id="...",model="...",project="..."}
```

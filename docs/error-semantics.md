# 错误语义与兼容矩阵

本文档定义 gcli2api-go 的统一错误处理策略，包括对外 API 的错误码映射、响应格式以及内部错误处理最佳实践。

## 架构决策

**唯一上游**：gcli2api-go 仅对接 **Gemini Code Assist** 作为上游提供者（详见 [ADR-0001](adr/ADR-0001-geminicli-only.md)）。所有错误语义围绕 Gemini 与 OpenAI 兼容性设计，不支持其他上游（如 Claude/Anthropic）。

**错误格式支持**：
- **OpenAI 兼容格式**（默认）：用于 `/v1/*` 端点
- **Gemini 原生格式**：用于 `/v1beta/*` 和 `/v1internal/*` 端点

## 1. 标准 HTTP → 业务语义映射

| HTTP Status | 业务 Code            | Type                | 默认消息                        | 是否重试 | 备注 |
|-------------|----------------------|---------------------|---------------------------------|----------|------|
| 400         | `invalid_request_error` | `invalid_request_error` | Invalid request                 | 否       | 参数校验失败、上游 400 透传 |
| 401         | `invalid_api_key`     | `authentication_error` | Invalid authentication          | 否       | API Key 缺失/错误 |
| 403         | `permission_denied`   | `permission_error`   | Permission denied               | 否       | 凭证无权限；标记为 Critical |
| 404         | `not_found`           | `invalid_request_error` | Resource not found            | 否       | 资源不存在 |
| 408         | `timeout`             | `timeout_error`      | Request timeout                 | 是       | 客户端取消由网络映射触发 |
| 429         | `rate_limit_exceeded` | `rate_limit_error`   | Rate limit exceeded             | 是       | 读取 `Retry-After`（默认为 60s） |
| 500         | `server_error`        | `server_error`       | Internal server error           | 是       | 上游 5xx 统一映射 |
| 502         | `bad_gateway`         | `server_error`       | Bad gateway                     | 是       | 包含连接错误等网络异常 |
| 503         | `service_unavailable` | `server_error`       | Service temporarily unavailable | 是       | 维护或限流 |
| 504         | `timeout`             | `timeout_error`      | Request timeout                 | 是       | 包括 deadline exceeded |
| 5xx 其他    | `unknown_error`       | `server_error`       | HTTP {status} error             | 是       | 捕获全部未定义 5xx |

**重试判断**：HTTP 状态为 429/5xx/408，或业务 Code 属于 `timeout` / `connection_error` / `network_error` / `dns_error`。

## 2. 网络错误映射

来自 SDK/Transport 的错误统一转换为如下业务语义：

| 匹配条件包含                      | HTTP Status | Code               | Type          | 示例消息                                      | 是否重试 |
|-----------------------------------|-------------|--------------------|---------------|-----------------------------------------------|----------|
| `timeout` / `deadline exceeded`   | 504         | `timeout`          | `timeout_error` | Request timeout: ...                          | 是 |
| `connection refused`             | 502         | `connection_error` | `server_error` | Connection refused: ...                       | 是 |
| `EOF` / `connection reset`        | 502         | `connection_error` | `server_error` | Connection error: ...                         | 是 |
| `no such host` / `name resolution`| 502         | `dns_error`        | `server_error` | DNS resolution error: ...                     | 是 |
| `certificate` / `tls`             | 502         | `tls_error`        | `server_error` | TLS/Certificate error: ...                    | 是 |
| `context canceled`                | 408         | `request_canceled` | `timeout_error` | Request was canceled: ...                     | 否 |
| 其他网络错误                      | 502         | `network_error`    | `server_error` | Network error: ...                            | 是 |

## 3. 错误响应格式

### 3.1 OpenAI 兼容格式（默认）

用于 `/v1/chat/completions`、`/v1/models` 等端点：

```json
{
  "error": {
    "message": "Rate limit exceeded",
    "type": "rate_limit_error",
    "code": "rate_limit_exceeded",
    "details": {
      "retry_after": 60
    }
  }
}
```

### 3.2 Gemini 原生格式

用于 `/v1beta/*` 和 `/v1internal/*` 端点：

```json
{
  "error": {
    "code": 429,
    "message": "Rate limit exceeded",
    "status": "RESOURCE_EXHAUSTED"
  }
}
```

## 4. 内部错误类型层次

### 4.1 领域错误（Domain Errors）

定义在 `internal/errors/types.go`：

```go
type APIError struct {
    HTTPStatus int                    // HTTP 状态码
    Code       string                 // 错误代码
    Message    string                 // 错误消息
    Type       string                 // 错误类型
    Details    map[string]interface{} // 额外详情
}
```

### 4.2 存储错误（Storage Errors）

定义在 `internal/storage/common/error_mapper.go`：

- `ErrNotFound` - 资源未找到
- `ErrAlreadyExists` - 资源已存在
- `ErrInvalidData` - 数据无效
- `ErrNotSupported` - 操作不支持

### 4.3 网络错误（Network Errors）

定义在 `internal/errors/network_mapping.go`，处理上游请求失败。

### 4.4 验证错误（Validation Errors）

定义在 `internal/storage/common/validator.go`，处理输入验证失败。

## 5. 错误处理最佳实践

### 5.1 错误应该被包装（Wrap）而不是丢弃

✅ **正确示例**：
```go
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
```

❌ **错误示例**：
```go
if err != nil {
    return errors.New("operation failed")
}
```

### 5.2 使用领域错误类型

✅ **正确示例**：
```go
if credential == nil {
    return errors.NewAPIError(
        http.StatusNotFound,
        "credential_not_found",
        "Credential not found",
        "invalid_request_error",
        nil,
    )
}
```

### 5.3 网络错误应使用映射器

✅ **正确示例**：
```go
resp, err := client.Do(req)
if err != nil {
    return errors.MapNetworkError(err)
}
```

### 5.4 存储错误应使用统一映射

✅ **正确示例**：
```go
data, err := storage.GetConfig(ctx, key)
if err != nil {
    if errors.Is(err, storage.ErrNotFound) {
        return nil, errors.NewAPIError(404, "not_found", "Config not found", "invalid_request_error", nil)
    }
    return nil, fmt.Errorf("storage error: %w", err)
}
```

### 5.5 Handler 层统一响应助手

- 通过 `internal/handlers/common.AbortWithError` / `AbortWithUpstreamError` / `AbortWithAPIError` 统一生成响应，自动选择 OpenAI 或 Gemini 错误格式。
- 旧的 `JSONOpenAIError` / `JSONGeminiError` 已移除，迁移时请替换为上述助手并使用 `internal/errors` 中的标准化结构体。

## 6. 重试策略

### 6.1 自动重试条件

- HTTP 429（速率限制）
- HTTP 5xx（服务器错误）
- 网络错误（连接失败、超时、DNS 错误等）

### 6.2 重试配置

```yaml
retry_enabled: true
retry_max: 3
retry_interval_sec: 1
retry_max_interval_sec: 10
retry_on_5xx: true
retry_on_network_error: true
```

### 6.3 指数退避

重试间隔使用指数退避算法：
- 第 1 次重试：1 秒
- 第 2 次重试：2 秒
- 第 3 次重试：4 秒
- 最大间隔：10 秒（可配置）

## 7. 错误观测

### 7.1 Prometheus 指标

- `gcli2api_upstream_errors_total{provider,reason}` - 上游错误计数
- `gcli2api_upstream_requests_total{provider,status_class}` - 按状态分类的请求计数
- `gcli2api_http_requests_total{server,method,path,status_class}` - HTTP 请求计数

### 7.2 日志字段

结构化日志包含以下错误相关字段：
- `error_code` - 业务错误码
- `error_type` - 错误类型
- `http_status` - HTTP 状态码
- `upstream_status` - 上游返回的状态码
- `retry_attempt` - 重试次数
- `is_retryable` - 是否可重试

## 8. 兼容性矩阵

| 场景 | OpenAI 格式 | Gemini 格式 | 说明 |
|------|------------|------------|------|
| `/v1/chat/completions` | ✅ | ❌ | 仅 OpenAI 格式 |
| `/v1/models` | ✅ | ❌ | 仅 OpenAI 格式 |
| `/v1beta/models` | ❌ | ✅ | 仅 Gemini 格式 |
| `/v1internal:streamGenerateContent` | ❌ | ✅ | 仅 Gemini 格式 |
| 管理 API (`/routes/api/management/*`) | ✅ | ❌ | 自定义 JSON 格式 |

## 9. 常见错误场景

### 9.1 凭证相关

| 场景 | HTTP | Code | 处理建议 |
|------|------|------|---------|
| API Key 无效 | 401 | `invalid_api_key` | 检查 `Authorization` 头 |
| 凭证无权限 | 403 | `permission_denied` | 检查 Gemini 凭证权限 |
| 凭证已禁用 | 403 | `credential_disabled` | 管理端恢复凭证 |

### 9.2 速率限制

| 场景 | HTTP | Code | 处理建议 |
|------|------|------|---------|
| 上游速率限制 | 429 | `rate_limit_exceeded` | 等待 `Retry-After` 秒后重试 |
| 本地速率限制 | 429 | `rate_limit_exceeded` | 调整 `rate_limit_rps` 配置 |

### 9.3 模型相关

| 场景 | HTTP | Code | 处理建议 |
|------|------|------|---------|
| 模型不存在 | 404 | `model_not_found` | 检查 `/v1/models` 列表 |
| 模型未启用 | 400 | `model_disabled` | 管理端启用模型 |

## 10. 参考资料

- [OpenAI API 错误码](https://platform.openai.com/docs/guides/error-codes)
- [Gemini API 错误处理](https://ai.google.dev/gemini-api/docs/error-handling)
- [ADR-0001: Gemini CLI 作为唯一上游](adr/ADR-0001-geminicli-only.md)
- [内部错误处理实现](../internal/errors/)

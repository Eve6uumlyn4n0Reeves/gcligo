# 模型变体系统

## 概述

GCLI2API-Go 提供了强大的模型变体系统，允许通过模型名称的前缀和后缀来控制请求行为。这个系统自动为每个基础模型生成所有可能的变体组合，并在 `/v1/models` 端点中暴露。

## 变体类型

### 前缀变体

前缀变体用于控制流式响应的行为：

| 前缀 | 说明 | 示例 |
|------|------|------|
| `假流式/` | 启用假流式模式，将非流式响应转换为流式输出 | `假流式/gemini-2.5-pro` |
| `流式抗截断/` | 启用流式抗截断模式，自动检测并继续被截断的响应 | `流式抗截断/gemini-2.5-flash` |

### 后缀变体

后缀变体用于控制模型的思考模式和搜索功能：

| 后缀 | 说明 | 示例 |
|------|------|------|
| `-maxthinking` | 最大思考模式，启用深度推理 | `gemini-2.5-pro-maxthinking` |
| `-nothinking` | 无思考模式，禁用推理过程 | `gemini-2.5-flash-nothinking` |
| `-lowthinking` | 低思考模式 | `gemini-2.5-pro-lowthinking` |
| `-medthinking` | 中等思考模式 | `gemini-2.5-pro-medthinking` |
| `-autothinking` | 自动思考模式（默认） | `gemini-2.5-pro-autothinking` |
| `-search` | 启用搜索功能 | `gemini-2.5-flash-search` |

### 组合变体

前缀和后缀可以自由组合使用：

```
假流式/gemini-2.5-pro-maxthinking
流式抗截断/gemini-2.5-flash-nothinking
假流式/gemini-2.5-pro-search
流式抗截断/gemini-2.5-flash-maxthinking-search
```

## 配置

### 启用/禁用变体系统

在 `config.yaml` 中配置：

```yaml
# 设置为 true 禁用模型变体，只暴露基础模型
disable_model_variants: false
```

### 禁用特定变体

可以通过 `disabled_models` 配置禁用特定的基础模型或变体：

```yaml
disabled_models:
  - gemini-2.5-pro-maxthinking
  - 假流式/gemini-2.5-flash
```

## 使用示例

### OpenAI SDK

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8317/v1",
    api_key="your-management-key"
)

# 使用假流式变体
response = client.chat.completions.create(
    model="假流式/gemini-2.5-pro",
    messages=[{"role": "user", "content": "Hello!"}],
    stream=True
)

for chunk in response:
    print(chunk.choices[0].delta.content, end="")

# 使用思考模式变体
response = client.chat.completions.create(
    model="gemini-2.5-flash-maxthinking",
    messages=[{"role": "user", "content": "Solve this complex problem..."}]
)

print(response.choices[0].message.content)

# 使用组合变体
response = client.chat.completions.create(
    model="流式抗截断/gemini-2.5-pro-search",
    messages=[{"role": "user", "content": "Research this topic..."}],
    stream=True
)
```

### cURL

```bash
# 列出所有可用模型（包括变体）
curl http://localhost:8317/v1/models \
  -H "Authorization: Bearer your-management-key"

# 使用假流式变体
curl http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-management-key" \
  -d '{
    "model": "假流式/gemini-2.5-pro",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'

# 使用组合变体
curl http://localhost:8317/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-management-key" \
  -d '{
    "model": "流式抗截断/gemini-2.5-flash-maxthinking",
    "messages": [{"role": "user", "content": "Complex task..."}],
    "stream": true
  }'
```

## 变体行为详解

### 假流式模式 (`假流式/`)

当使用假流式前缀时：
1. 系统会将非流式 API 响应转换为流式输出
2. 响应会被分块发送，模拟真实的流式行为
3. 可以通过 `fake_streaming_chunk_size` 和 `fake_streaming_delay_ms` 配置分块大小和延迟

配置示例：
```yaml
fake_streaming_enabled: true
fake_streaming_chunk_size: 20
fake_streaming_delay_ms: 50
```

### 流式抗截断模式 (`流式抗截断/`)

当使用流式抗截断前缀时：
1. 系统会自动检测响应是否被截断
2. 如果检测到截断，会自动发起续写请求
3. 最多重试次数由 `anti_truncation_max` 配置控制

配置示例：
```yaml
anti_truncation_enabled: true
anti_truncation_max: 3
```

### 思考模式后缀

思考模式后缀会影响模型的推理行为：
- `-maxthinking`: 启用深度推理，适合复杂问题
- `-nothinking`: 禁用推理过程，适合简单问答
- `-lowthinking`/`-medthinking`: 中间级别的推理
- `-autothinking`: 让模型自动决定推理深度

### 搜索后缀 (`-search`)

启用搜索功能，允许模型访问外部信息源。

## 变体生成逻辑

系统会为每个基础模型生成以下变体：

1. 基础模型（无前缀无后缀）
2. 每个前缀 × 基础模型
3. 基础模型 × 每个后缀
4. 每个前缀 × 基础模型 × 每个后缀
5. 思考后缀 + 搜索后缀的组合

例如，对于 `gemini-2.5-pro`，会生成：
- `gemini-2.5-pro`
- `假流式/gemini-2.5-pro`
- `流式抗截断/gemini-2.5-pro`
- `gemini-2.5-pro-maxthinking`
- `gemini-2.5-pro-nothinking`
- `gemini-2.5-pro-search`
- `假流式/gemini-2.5-pro-maxthinking`
- `流式抗截断/gemini-2.5-pro-nothinking`
- `gemini-2.5-pro-maxthinking-search`
- ... 等等

## API 响应示例

### GET /v1/models

```json
{
  "object": "list",
  "data": [
    {
      "id": "gemini-2.5-pro",
      "object": "model",
      "owned_by": "gcli2api-go",
      "created": 1730419200,
      "modalities": ["text"],
      "description": "Gemini model with feature variants",
      "context_length": 1048576,
      "capabilities": {
        "completion": true,
        "chat": true,
        "images": false
      }
    },
    {
      "id": "假流式/gemini-2.5-pro",
      "object": "model",
      "owned_by": "gcli2api-go",
      "created": 1730419200,
      "modalities": ["text"],
      "description": "Gemini model with feature variants",
      "context_length": 1048576,
      "capabilities": {
        "completion": true,
        "chat": true,
        "images": false
      }
    },
    ...
  ]
}
```

## 最佳实践

1. **选择合适的变体**：根据实际需求选择变体，避免不必要的开销
2. **测试变体行为**：在生产环境使用前，先在测试环境验证变体行为
3. **监控性能**：某些变体（如抗截断）可能会增加请求延迟，需要监控
4. **合理配置**：根据实际情况调整假流式和抗截断的配置参数
5. **禁用不需要的变体**：通过 `disabled_models` 禁用不需要的变体，减少模型列表大小

## 故障排查

### 变体不显示在模型列表中

1. 检查 `disable_model_variants` 配置是否为 `false`
2. 检查变体是否在 `disabled_models` 列表中
3. 检查日志中是否有相关错误信息

### 变体行为不符合预期

1. 检查相关功能的配置（如 `fake_streaming_enabled`、`anti_truncation_enabled`）
2. 查看日志中的请求处理详情
3. 使用 `/health` 端点检查系统状态

### 性能问题

1. 如果模型列表过大，考虑禁用不需要的变体
2. 如果抗截断导致延迟过高，调整 `anti_truncation_max` 参数
3. 监控 Prometheus 指标，识别性能瓶颈

## 相关文档

- [配置说明](configuration.md)
- [管理控制台](management-console.md)
- [监控指标](monitoring.md)
- [故障排查](troubleshooting.md)


# API 参考（OpenAI 兼容 + 管理接口）

本文档概述主要对外端点、认证方式与常用示例。

## 认证
- OpenAI 端点：`Authorization: Bearer <api_key>`（支持单 Key 或多 Key 配置）
- 管理端点：`Authorization: Bearer <management_key>` 或 Cookie `mgmt_session`

---

## OpenAI 兼容端点

### GET /v1/models
返回可用模型列表（含可选变体）。

示例：
```bash
curl -H "Authorization: Bearer $OPENAI_KEY" http://localhost:8317/v1/models
```

### GET /v1/models/:id
返回指定模型信息。

### POST /v1/chat/completions
- 请求体：OpenAI Chat Completions 兼容
- 支持流式：`{"stream": true}` → SSE（`text/event-stream`）

示例：
```bash
curl -s -H "Authorization: Bearer $OPENAI_KEY" -H "Content-Type: application/json" \
  -d '{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"hello"}],"stream":true}' \
  http://localhost:8317/v1/chat/completions
```

### POST /v1/completions
- OpenAI 文本补全兼容（非对话）
- 支持 `stream: true` 返回分块

### POST /v1/responses
- OpenAI Responses 规范兼容
- 根据 `stream` 与配置决定走 fake/stream/final 实现

### POST /v1/images/generations
- 调用 Gemini 图像模型（如 `gemini-2.5-flash-image`）生成图片
- 返回 `data[].b64_json`（可选包含 `mime_type`）

---

## 管理 API（前缀：/routes/api/management）

### 健康与系统
- GET /system：系统信息
- GET /health：健康检查
- GET /metrics：Prometheus 文本格式（包含 HTTP、上游、SSE、存储等所有指标）
- GET /usage：用量/请求统计快照
- GET /capabilities：系统能力与开关

### 凭证
- GET /credentials：列出凭证文件名
- GET /credentials/:id：读取单个凭证内容（脱敏）
- POST /credentials：上传单个凭证（JSON）
- POST /credentials/reload：重载凭证
- POST /credentials/recover-all：批量恢复封禁
- POST /credentials/:id/recover：恢复单个凭证
- POST /credentials/:id/disable|enable：禁用/启用
- POST /credentials/validate|validate-batch|validate-zip：形状/令牌校验
- POST /credentials/probe：测活；GET /credentials/probe/history：历史

### 配置与特性
- GET /config：读取配置
- PUT /config：更新配置
- POST /config/reload：重载配置
- GET /features：特性开关列表
- PUT /features/:feature：更新某开关

### 模型注册与模板
- GET/PUT/POST/DELETE /models/registry[/:id]
- GET/PUT /models/:channel/template
- GET/POST /models/:channel/registry[/(import|export|seed-defaults|bulk-enable|bulk-disable)]
- GET /models/:channel/groups；POST/PUT/DELETE /models/:channel/groups/:id
- GET/POST/PUT/DELETE /models/groups 与 /models/capabilities 系列

### 会话
- POST /login：颁发会话 Cookie/Token
- POST /logout：注销（清理 Cookie 与内存会话）



# 监控与可观测性

`gcli2api-go` 内置 Prometheus 指标、健康检查、流式统计以及结构化日志，方便在生产环境中观测服务状态。本文件总结主要能力与采集要点。

## 1. 指标端点

- 默认暴露在 `http://<host>:<openai_port>/metrics`。
- 若设置了 `base_path`，指标也会自动带上该前缀。
- 输出为 Prometheus 文本格式，可直接被 `prometheus-server` 抓取。若需标准处理程序，可通过反向代理包装 `promhttp.Handler()`。

### 1.1 核心指标

| 类别 | 指标 | 说明 |
| --- | --- | --- |
| HTTP 请求 | `gcli2api_http_requests_total{method,endpoint,status_class}` | 各 API 路径的请求总数。 |
| HTTP 延迟 | `gcli2api_http_request_duration_seconds_bucket` | 直方图，用于计算 P95/P99 延迟。 |
| 上游请求 | `gcli2api_upstream_requests_total{provider,status_class}` | 对 Gemini 上游的调用次数与状态。 |
| 上游延迟 | `gcli2api_upstream_request_duration_seconds_bucket` | 上游调用的延迟分布。 |
| 上游错误 | `gcli2api_upstream_errors_total{provider,reason}` | 按原因划分的错误次数。 |
| 上游模型请求 | `gcli2api_upstream_model_requests_total{provider,model,status_class}` | 细化到模型粒度的请求计数。 |
| 流式统计 | `gcli2api_stream_chunks_sent_total{model}`、`gcli2api_stream_duration_seconds_bucket` | SSE 推送次数与耗时。 |
| Token 用量 | `gcli2api_tokens_used_total{model,type}` | 按 Prompt/Completion 分类的 Token 统计。 |
| 凭证探活 | `gcli2api_auto_probe_runs_total{source,status,model}`<br>`gcli2api_auto_probe_duration_seconds{source,model}`<br>`gcli2api_auto_probe_success_ratio{source,model}`<br>`gcli2api_auto_probe_target_credentials{source,model}`<br>`gcli2api_auto_probe_last_success_unix{source,model}` | 自动/手动探活的次数、耗时、成功率与覆盖范围。 |
| 上游模型发现 | `gcli2api_upstream_discovery_fetch_total{result}`<br>`gcli2api_upstream_discovery_fetch_duration_seconds`<br>`gcli2api_upstream_discovery_known_bases`<br>`gcli2api_upstream_discovery_last_success_unix` | 每日同步 Gemini CLI 模型目录的结果。 |
| 路由状态 | `gcli2api_routing_sticky_hits_total{server}`<br>`gcli2api_routing_cooldown_events_total{status}`<br>`gcli2api_routing_sticky_size` | 粘性映射与冷却队列规模。 |
| 凭证数量 | `gcli2api_active_credentials`、`gcli2api_disabled_credentials` | 当前可用/禁用凭证数量。 |

> 推荐告警示例可参见 [`prometheus-alerts.md`](prometheus-alerts.md)。

### 1.2 管理控制台指标

管理端调用 `GET /routes/api/management/metrics` 会返回 Prometheus 文本格式的所有指标（包含 HTTP、上游、SSE、存储、计划执行等），可直接被 Prometheus 抓取或通过管理台查看。所有指标已统一到 `prometheus/client_golang`，不再维护独立的 JSON 快照端点。

## 2. 健康检查

| 路径 | 说明 |
| --- | --- |
| `/healthz` | 基础健康检查，包含存储、凭证管理器、上游连通性自检。 |
| `/debug/pprof/*` | 若启用 `enable_pprof`，可访问 Go 运行时剖析数据。 |

healthz 适合用作容器编排的 liveness/readiness 探针。

## 3. 日志

- 默认输出结构化 JSON 日志，包含请求 ID、延迟、上游状态等字段。
- 管理端“日志”页通过 WebSocket 订阅实时日志；也支持按请求 ID 搜索。
- 可通过 `request_log_exclude_paths` 忽略静态资源等不重要请求。

## 4. 管理台指标面板

在 `/admin` 中，“指标”页会汇总：

- 请求吞吐与成功率。
- 凭证健康评分、封禁/恢复历史。
- 模型曝光与流式统计。
- 路由粘性、冷却队列状态。

这些数据基于前述 Prometheus 指标与运行时快照计算，适合运维人员进行手动巡检。

## 5. 用量统计与聚合

- 管理端 `GET /routes/api/management/usage` 现在返回分层结构：`api_keys` 为各调用方明细，`aggregates.total` 汇总整个实例的请求/Token 用量，`aggregates.models` 以基础模型（自动折叠变体，例如 `gemini-2.5-pro`）维度统计。
- 用量重置周期由 `usage_reset_interval_hours` 决定；默认每天根据 `usage_reset_timezone` + `usage_reset_hour_local`（默认 UTC+7 的 00:00）刷新一次。
- 如果只需聚合数据，可直接读取 `aggregates` 节点；若需要禁用聚合，可关闭 `UsageStats` 功能或在管理端过滤。

## 6. 集成建议

- **抓取策略**：对 `/metrics` 设置 15～30 秒抓取间隔，确保 `gcli2api_auto_probe_*` 与 `upstream_discovery_*` 等低频指标也能被采集。
- **仪表盘**：可使用 Grafana 将 HTTP 延迟、上游错误、凭证成功率等关键指标绘制成图表。
- **告警阈值示例**：
  - `gcli2api_auto_probe_success_ratio < 0.6` 持续 15 分钟：凭证整体异常。
  - `rate(gcli2api_upstream_discovery_fetch_total{result="error"}[5m]) > 0`：模型目录同步失败。
  - `histogram_quantile(0.95, sum(rate(gcli2api_http_request_duration_seconds_bucket[5m])) by (le)) > 2`：P95 大幅升高。
  - 详见 [`prometheus-alerts.md`](prometheus-alerts.md)。

---

更多部署层面的建议请参考 [`deployment.md`](deployment.md)，管理端操作细节见 [`management-console.md`](management-console.md)。

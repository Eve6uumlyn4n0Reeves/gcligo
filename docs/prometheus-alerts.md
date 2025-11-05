# Prometheus 告警示例（节选）

```yaml
groups:
- name: gcli2api-go
  rules:
  - alert: UpstreamDiscoveryErrors
    expr: rate(gcli2api_upstream_discovery_fetch_total{result="error"}[5m]) > 0
    for: 10m
    labels: { severity: warning }
    annotations:
      summary: "上游模型发现持续失败"
      description: "近 10 分钟内存在持续的上游模型发现失败。"

  - alert: CredentialProbeLowSuccess
    expr: gcli2api_auto_probe_success_ratio < 0.6
    for: 15m
    labels: { severity: warning }
    annotations:
      summary: "凭证测活成功率过低 (<60%)"
      description: "请检查凭证有效性或上游可用性。"

  - alert: HTTPHighLatencyP95
    expr: histogram_quantile(0.95, sum(rate(gcli2api_http_request_duration_seconds_bucket[5m])) by (le)) > 2
    for: 10m
    labels: { severity: warning }
    annotations:
      summary: "HTTP P95 延迟持续 > 2s"
      description: "请检查上游响应与存储后端。"
```


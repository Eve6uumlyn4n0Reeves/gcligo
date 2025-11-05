# ADR-0004: 日志系统结构化与动态等级

## 背景
需要统一日志格式、支持动态等级调整与采样，便于排查与审计。

## 决策
- 使用结构化日志接口（已存在 logging 包，可扩展）
- 动态等级：通过管理 API 暴露 `GET/PUT /features/log-level`
- 采样：对高频路由与上游请求启用采样开关

## 设计要点
- 字段规范：`request_id`、`route`、`actor`、`upstream`、`duration_ms`、`status`
- 敏感字段屏蔽：凭证/令牌/个人信息
- 与指标关联：日志中包含关键信息以便与 Prometheus 标签对齐

## 样例
```json
{"ts":"2025-11-01T12:00:00Z","level":"info","route":"/v1/chat/completions","status":200,"duration_ms":123,"request_id":"...","actor":"api_key"}
```

## 状态
- 本 ADR 提供落地指南；后续迭代实现动态等级与采样逻辑（含特性开关）。


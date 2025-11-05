# Deployment Guide

This document summarises the minimum configuration required to run **gcli2api-go** in different environments. The focus is on management console availability, authentication safety, and reverse‑proxy compatibility. Architectural scope说明请参考 [`ADR-0001-geminicli-only.md`](ADR-0001-geminicli-only.md)。

## 1. Configuration Overview

| Setting | Location | Default | Notes |
| --- | --- | --- | --- |
| `management_key` / `management_key_hash` | `config.yaml` / env | _unset_ | Provide either a plaintext management key or its bcrypt hash. The key is required for every management API call. |
| `base_path` | `config.yaml` / env `BASE_PATH` | `""` | Set when serving the admin UI behind a sub-path (for example `/gcli`). |
| `proxy_url` | `config.yaml` / env `PROXY_URL` | `""` | Optional upstream proxy for outbound Gemini requests. |
| `redis_*`, `mongodb_*`, `postgres_dsn` | Backing store settings | _n/a_ | Configure only when switching away from the default file-based storage. |

Feel free to override any YAML option by exporting the corresponding environment variable before starting the process.

## 2. Authentication

- 管理端登录：`POST /routes/api/management/login`，Body: `{ "key": "<management_key>" }`。
  - 成功后服务端签发 `mgmt_session` Cookie（HttpOnly、SameSite=Lax，反代 https 下标记 Secure）。
  - 管理前端依赖此 Cookie 访问所有管理 API；浏览器端不再存储明文密钥。
- 脚本/工具：管理 API 仍支持 `Authorization: Bearer <management_key>` 方式调用（不依赖 Cookie）。
- 为更安全的配置，推荐同时设置 `management_key` 与 `management_key_hash`。

## 3. Deployment Checklist

1. **Reverse proxies / sub-paths**
   - Set `base_path` to match the mounted path (e.g. `/gcli`).
   - Ensure static assets (`/admin.js`, `/admin.css`, `/dist/**`) are reachable under the same prefix.
2. **TLS terminators / load balancers**
   - Forward the original client IP via `X-Forwarded-For` or `X-Real-IP` when the service runs behind another proxy; this keeps request logging accurate.
3. **Credential hygiene**
   - Prefer `management_key_hash` for long-term storage and inject the plaintext key via environment variables or secret managers.
   - Rotate the management key periodically and restart the service after updating the configuration.
4. **Optional backing stores**
   - When enabling Redis/Mongo/Postgres, verify connectivity and credentials before toggling the respective backend to avoid boot failures.

## 4. Troubleshooting Quick Reference

| Symptom | Root Cause | Resolution |
| --- | --- | --- |
| 登录页循环 | 反代未正确传递 `X-Forwarded-Proto=https` 导致 Cookie 未标记 Secure | 修正反代设置，或用 http 直连测试；确认登录响应里 `mgmt_session` 已下发。 |
| 访问 `/admin` 返回 404/500 | 反向代理未转发静态资源或 `base_path` 未对齐 | 校验代理配置，并确认 `base_path` 与实际挂载路径一致。 |
| WebSocket 日志无法连接 | 代理不支持 WS 或未携带管理密钥 | 使用 `wss://`，并确认前端能够读取管理密钥（或在查询参数中带上 `?key=<management_key>`）。 |
| 管理 API 返回 401 | 浏览器无 `mgmt_session` 或脚本缺少鉴权 | 浏览器重新登录；脚本使用 `Authorization: Bearer <management_key>`。 |

## 5. Recommended Automation

- 将 `MANAGEMENT_KEY` / `MANAGEMENT_KEY_HASH` 写入部署清单或 CI/CD 密钥管理中，避免在仓库中保存明文。
- 在交付流程中运行一次 UI 烟测（例如 `make smoke-ui`）以确保 `/admin` 可以加载并完成登录。
- 关注日志输出和 Prometheus 指标，确认管理接口仅被期望的操作人员访问。

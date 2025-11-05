# 配置参考

`gcli2api-go` 的配置来源支持：静态文件 `config.yaml`、环境变量覆盖以及运行时管理 API 更新。本文档列出了常用配置项及推荐做法，便于在开发、测试与生产环境中保持一致性。

- 默认配置加载顺序：内置默认值 → `config.yaml` → 环境变量 → 管理端运行时更新（存储在所选后端）。
- 若提供 `management_key_hash`，明文密钥仅需在服务启动时通过环境变量或密钥管理器注入。
- 详细的存储后端参数及迁移步骤见 [`storage.md`](storage.md)。

## ⚠️ 配置结构演进与弃用计划

**当前状态（v2.x）**：配置系统采用**领域结构（Domain Structures）**组织，将相关配置分组到 11 个子结构体中（`Server`、`Upstream`、`Security`、`Execution`、`Storage`、`Retry`、`RateLimit`、`APICompat`、`ResponseShaping`、`OAuth`、`AutoBan`、`AutoProbe`、`Routing`）。

**向后兼容层**：为保证平滑迁移，顶层字段（如 `OpenAIPort`、`ManagementKey`、`RetryEnabled` 等）仍然保留，并通过 `SyncFromDomains()` 和 `SyncToDomains()` 实现双向同步。

**推荐做法**：
- ✅ **新代码**：仅读写领域结构（如 `cfg.Server.OpenAIPort`、`cfg.Retry.Enabled`）
- ⚠️ **遗留代码**：可继续使用顶层字段，但应逐步迁移到领域结构
- 🔄 **配置文件**：YAML/JSON 配置文件仍使用扁平键名（如 `openai_port`），加载时会自动填充到领域结构

**弃用时间表**：
- **v2.x（当前）**：顶层字段保留，双向同步正常工作
- **v3.0（计划）**：顶层字段标记为 `@deprecated`，编译时发出警告
- **v4.0（未来）**：移除顶层字段，仅保留领域结构

> 运行时提醒：从 v2.5 起，`SyncToDomains()` 会检测任何仍在使用的顶级字段并输出一次性日志：
> `legacy config field OpenAIPort is still in use; migrate to Server.OpenAIPort`，便于追踪尚未迁移的模块。

**迁移示例**：
```go
// ❌ 旧写法（将在 v4.0 移除）
cfg.OpenAIPort = "8080"
cfg.RetryEnabled = true

// ✅ 新写法（推荐）
cfg.Server.OpenAIPort = "8080"
cfg.Retry.Enabled = true
```

详见 `internal/config/sync_test.go` 中的双向同步一致性测试。

## 1. 核心端口与路由

| 键 | 描述 | 默认值 |
| --- | --- | --- |
| `openai_port` | OpenAI 兼容端口（必填） | `8317` |
| `gemini_port` | Gemini 原生端口（可选） | `""`（禁用） |
| `base_path` | 反向代理或子路径部署时的统一前缀（影响 API 与静态资源） | `""` |
| `listen_addr` | 监听地址，空值代表 `0.0.0.0` | `""` |
| `proxy_url` | 访问 Gemini 上游时使用的 HTTP/HTTPS 代理 | `""` |

> 若部署在反向代理背后，请确保健康检查、指标与管理端静态资源均可通过 `base_path` 访问。

## 2. 管理端与认证

| 键 | 用途 | 说明 |
| --- | --- | --- |
| `management_key` | 管理 API 明文密钥 | 开发环境下可直接配置。生产环境建议配合 `management_key_hash` 使用。 |
| `management_key_hash` | 管理 API 密钥的 bcrypt 哈希 | 配合 `management_key` 可支持明文登录与校验。 |
| `session_secret` / `SESSION_SECRET` | 为管理控制台签发会话 Token | 设置后浏览器以签名 Token 代替内存会话，更易横向扩展。 |
| `allowed_admin_networks` | 管理端访问白名单 | CIDR 列表；为空视作仅限本地。 |

- 若未设置 `session_secret`，系统会自动使用 `management_key_hash`（如不存在则退回 `management_key`）派生签名密钥，保证多实例部署下 Cookie 可互认。
- 登出操作会撤销当前签名 Token 并清理 Cookie；Token 默认有效期 2 小时，可在 `SessionLogin` 请求体中覆盖。

所有管理 API 都接受：

- Cookie：`mgmt_session=<signed-token>`
- 或 HTTP 头：`Authorization: Bearer <management_key>`

## 3. 凭证与轮询策略

| 键 | 描述 |
| --- | --- |
| `auth_dir` | 默认凭证目录。文件后端会在此目录按文件名存储 OAuth JSON。 |
| `credential_refresh_ahead_seconds` | 凭证过期前提前刷新时间（默认 600 秒）。 |
| `auto_ban_enabled` / `auto_ban_thresholds` | 自动封禁策略，基于连续错误与速率限制组合判定。 |
| `auto_recovery_enabled` | 启用后将定期尝试恢复被封禁凭证。 |
| `max_concurrent_per_credential` | 单凭证并发上限，超出后会寻找备选凭证。 |

增强恢复接口位于 `/routes/api/management/credentials/*`，详见 [`management-console.md`](management-console.md)。

## 4. 功能开关

| 区域 | 常用键 | 说明 |
| --- | --- | --- |
| 重试 | `retry_enabled`, `retry_max`, `retry_interval_sec`, `retry_max_interval_sec` | 控制上游失败后的指数退避重试。 |
| 抗截断 | `anti_truncation_enabled`, `anti_truncation_max` | 自动补发被截断的响应。 |
| 假流式 | `fake_streaming_enabled`, `fake_streaming_chunk_size`, `fake_streaming_delay_ms` | 在上游不支持流式时提供伪流式体验。 |
| 速率限制 | `rate_limit_enabled`, `rate_limit_rps`, `rate_limit_burst` | 基于调用方 API Key 的速率控制。 |
| 自动探活 | `auto_probe_enabled`, `auto_probe_model`, `auto_probe_hour_utc`, `auto_probe_timeout_sec` | 定时测活并记录探活指标。 |
| 模型默认集 | `preferred_base_models`, `disabled_models` | 控制 `/v1/models` 暴露与默认标记。 |
| 文本清洗 | `sanitizer_enabled`, `sanitizer_patterns` | 对上游返回文本应用自定义正则替换。默认关闭，开启后建议在管理端做好审计。 |

> 用量统计相关字段：`usage_reset_interval_hours` 控制周期，`usage_reset_timezone` 与 `usage_reset_hour_local` 决定每日重置的参考时区（默认 UTC+7 的 00:00）。

## 5. 存储后端

| 键 | 描述 | 说明 |
| --- | --- | --- |
| `storage_backend` | `auto` / `file` / `redis` / `mongodb` / `postgres` | `auto` 会按 Redis → Postgres → Mongo → File 顺序尝试。 |
| `storage_base_dir` | `file` 后端的数据目录 | 默认为 `~/.gcli2api/storage`。 |
| `redis_*` | Redis 连接参数 | 常用：`redis_addr`, `redis_password`, `redis_db`, `redis_prefix`。 |
| `mongodb_uri`, `mongodb_database` | MongoDB 连接信息 | 与官方驱动兼容的 URI。 |
| `postgres_dsn` | PostgreSQL DSN | 例如 `postgresql://user:pass@host:5432/db`。 |

不同后端能力与迁移流程请参阅 [`storage.md`](storage.md)。

## 6. 日志与可观测

| 键 | 描述 |
| --- | --- |
| `log_level` | 支持 `debug` / `info` / `warn` / `error`。 |
| `request_log_exclude_paths` | 需要忽略访问日志的路径列表。 |
| `metrics_namespace` | 自定义 Prometheus 指标命名空间。 |
| `enable_pprof` | 若启用，可通过 `/debug/pprof` 访问运行时分析。 |

更多指标详见 [`monitoring.md`](monitoring.md)。

## 7. 环境变量覆盖

所有 YAML 键均可通过大写、下划线格式的环境变量覆盖，例如：

| YAML 键 | 环境变量 | 示例 |
| --- | --- | --- |
| `openai_port` | `OPENAI_PORT` | `OPENAI_PORT=9000` |
| `gemini_port` | `GEMINI_PORT` | 禁用：`GEMINI_PORT=""` |
| `management_key` | `MANAGEMENT_KEY` | `MANAGEMENT_KEY="$(pass show mgmt)"` |
| `redis_addr` | `REDIS_ADDR` | `REDIS_ADDR=redis.internal:6379` |
| `auto_probe_hour_utc` | `AUTO_PROBE_HOUR_UTC` | `AUTO_PROBE_HOUR_UTC=3` |

在容器或 systemd 中，只需在启动前导出对应变量即可。

## 8. 运行时配置更新

管理端“配置”页面或 `PUT /routes/api/management/config` API 可以在不重启服务的情况下修改大多数开关。更新后会：

1. 写入所选存储后端（例如 Redis、Postgres 或文件）。
2. 通知运行中的组件动态加载新配置。
3. 触发相关缓存失效与指标刷新。

对于高风险项（如 `auto_probe_*`），更新前会进行增量校验并提示潜在影响。若需要恢复默认配置，可删除对应条目或使用 `DELETE /routes/api/management/config/:key`。

---

如需了解部署层面的额外设置（反向代理、TLS、密钥轮换等），请继续阅读 [`deployment.md`](deployment.md) 与 [`management-console.md`](management-console.md)。

# STORAGE 后端说明与迁移指南

本项目提供多种存储后端：`file`、`redis`、`mongodb`、`postgres`。本文档总结各后端的能力差异、适用场景与迁移步骤。

## 能力矩阵（概览）

- 凭证（CRUD/批量）：File ✅ / Redis ✅ / Mongo ✅ / Postgres ✅
- 配置（Get/Set/List/Delete）：File ✅ / Redis ✅ / Mongo ✅ / Postgres ✅
- 用量统计（Increment/Get/Reset/List）：File ✅ / Redis ✅ / Mongo ✅ / Postgres ✅
- Cache（Get/Set/Delete）：全部返回 ErrNotSupported（当前版本未启用）
- 事务（Begin/Commit/Rollback）：全部返回 ErrNotSupported（当前版本未启用）

说明：各实现位于 `internal/storage/`，接口定义见 `internal/storage/interface.go`。

## 选择建议

- 单机/轻量：`file`（默认），便于调试与低成本运行。
- 中小并发/横向扩容：`redis`，便于集中化配置与用量统计。
- 需要集中持久化且偏好文档存储：`mongodb`。
- 需要关系型、强一致统计：`postgres`。

## 配置示例（含 auto）

参考 `config.example.yaml`：

```yaml
storage_backend: auto        # 新增：auto 将按优先级自动选择
                             # 明确选择也可：file/redis/mongo/postgres
storage_base_dir: "~/.gcli2api/storage"  # file 后端根目录

# Redis
redis_addr: "localhost:6379"
redis_password: ""
redis_db: 0
redis_prefix: "gcli2api:"

# MongoDB
mongodb_uri: "mongodb://localhost:27017"
mongodb_database: "gcli2api"

# PostgreSQL
postgres_dsn: "postgresql://user:password@localhost:5432/gcli2api"
```

当 `storage_backend: auto` 时，选择顺序为：

1) 显式配置或环境存在的 Redis（`redis_addr` 或 `REDIS_ADDR`）
2) 显式配置或环境存在的 Postgres（`postgres_dsn` 或 `POSTGRES_DSN`）
3) 显式配置或环境存在的 MongoDB（`mongodb_uri` 或 `MONGODB_URI`）
4) 回退到 File（`storage_base_dir` 或基于 `auth_dir` 的默认目录）

初始化失败会自动尝试下一项，并在日志中给出 `storage auto ... failed` 的告警。

## 迁移流程（通用）

1. 在源后端导出：
   - 通过管理 API 或代码调用 `ExportData(ctx)` 导出 `credentials/configs/usage`。
2. 切换配置并启动目标后端：
   - 修改 `storage_backend` 与对应连接信息，启动服务确保健康检查通过。
3. 在目标后端导入：
   - 调用 `ImportData(ctx, exported)` 将数据写入目标后端。

注意：
- Redis/Mongo/Postgres 中的数字字段类型在读回时可能以字符串或 float64 表示；代码已做兼容解析。
- 事务与缓存接口当前统一返回 `ErrNotSupported`，不影响核心功能（凭证、配置、用量）。

## 性能与运维

- Redis：建议设置持久化策略与监控连接池参数；键空间：`prefix + cred:*`、`prefix + usage:*`、`prefix + config`。
- Postgres：初始化时自动建表与索引，连接池默认 `MaxOpenConns=25`，可按负载调整。
- MongoDB：默认集合 `credentials/usage_stats/configs`，包含 `id/key` 索引。

## 常见问题

- 切换后端导致统计清零：如未执行导入步骤，目标后端会从空状态开始；请先导出再导入。
- 权限不足：确保数据库用户具备相应的读写与建表权限。
- 健康检查失败：检查连接字符串、网络连通、认证配置。

# GCLI2API-Go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)

将 Gemini CLI（Code Assist）转化为既兼容 OpenAI 协议、又保留 Gemini 原生端点的双路由网关。服务内置凭证调度、模型注册中心、路由装配台与 Prometheus 监控，适合构建企业级内部 API。

## 项目定位与范围

- ✅ 上游仅对接 **Gemini Code Assist**，基于 OAuth 凭证轮询；不计划集成 Claude/Anthropic。
- ✅ 对外提供 `/v1/chat/completions`、`/v1/responses`、`/v1/images` 等 OpenAI 兼容端点，并可选暴露 Gemini 原生 REST。
- ✅ 管理端以单一管理密钥控制，可通过浏览器或 API 进行运维操作。
- 相关架构决策记录见 [`docs/ADR-0001-geminicli-only.md`](docs/ADR-0001-geminicli-only.md)。

## 核心特性速览

- **双端点路由**：OpenAI 兼容 & Gemini 原生共享一套凭证池与路由策略。
- **凭证生命周期管理**：自动刷新、封禁、恢复与测活；支持批量 ZIP 导入。
- **模型注册中心与装配台**：动态维护 `/v1/models`，管理路由计划、粘性与冷却。
- **模型变体系统**：支持前缀（假流式/、流式抗截断/）和后缀（-maxthinking、-nothinking、-search）组合，自动生成所有变体模型。
- **丰富可观测性**：自带 Prometheus 指标、增强快照、健康检查与实时日志。
- **多后端存储**：支持 `file`、`redis`、`mongodb`、`postgres`，并提供 `auto` 自动选择。
- **精细化开关**：重试、速率限制、抗截断、假流式、自动探活等均可运行时调整。

更详细的部署、运维与排障说明已拆分至 `docs/` 目录。

## 快速开始概览

1. 克隆仓库并执行 `scripts/check_build.sh` 或 `go build ./cmd/server`。
2. 复制 `config.example.yaml` 为 `config.yaml`，至少设置 `management_key` 与 `openai_port`。
3. 启动服务后，访问 `http://localhost:8317/admin` 导入 Gemini OAuth 凭证。
4. 使用 OpenAI SDK 或 HTTP 客户端调用 `http://localhost:8317/v1/chat/completions` 验证。

详细步骤与示例代码参见 [`docs/quickstart.md`](docs/quickstart.md)。

## 构建与运行

> 前置要求：Go 1.21+、Node.js 18+（管理端前端编译），可选的 Redis/MongoDB/PostgreSQL 服务。

1. 安装 Go 依赖（首次或变更依赖后）：
   ```bash
   go mod download
   ```
2. 编译服务端：
   ```bash
   go build ./cmd/server
   ```
3. 启动二进制（默认读取 `config.yaml`）：
   ```bash
   ./server --config config.yaml
   ```
4. （可选）构建管理端静态资源：
   ```bash
   npm install --prefix web
   npm run build --prefix web
   ```
5. 验证：
   - `go test ./...` 运行全部单元/集成测试；
   - `scripts/check_build.sh` 触发与 CI 等价的检查。

如果当前环境无法自动下载更高版本 Go 工具链，可临时设置 `GOTOOLCHAIN=local` 再执行上述命令，以强制使用本地安装的 Go。

## 文档导航

- `docs/README.md`：完整文档索引。
- `docs/quickstart.md`：零到一安装、配置与验证。
- `docs/configuration.md`：`config.yaml` 与环境变量说明。
- `docs/management-console.md`：控制台各功能操作指南。
- `docs/monitoring.md`：指标、日志与健康检查。
- `docs/deployment.md`：生产环境部署与访问控制建议。
- 其他专题文档（存储、告警、迁移、错误码等）同样位于 `docs/`。

## 代码结构

> **注意**: `web/dist/` 目录包含从 TypeScript 编译的 JavaScript 文件。
> 开发时请只修改 `web/src/` 中的 TypeScript 源码，编译产物会自动生成到 `web/dist/`。

```
cmd/            # 可执行入口（server、migrate、storageutil）
internal/
  config/       # 配置加载与运行时更新
  credential/   # 凭证管理器、自动封禁/恢复
  handlers/     # OpenAI、Gemini、管理 API
  models/       # 模型注册、变体生成、能力管理
  server/       # 路由构建、装配台、管理资源
  storage/      # 多后端实现（file/redis/mongo/postgres）
  upstream/     # Gemini 客户端、模型发现
  monitoring/   # Prometheus 指标与增强快照
web/
  src/          # TypeScript 源代码
  dist/         # 编译后的 JavaScript（由 tsc 生成）
  admin.html    # 管理控制台主页面
  login.html    # 登录页面
scripts/        # 构建、检查、示例脚本
docs/           # 项目文档集合
```

## 管理操作审计

- 通过管理 API 触发装配计划、路由持久化或清理冷却队列时，支持在请求头中携带 `X-Change-Reason` 描述变更原因。
- 所有关键操作都会写入 `gcli2api_assembly_operations_total{action,status,actor}` 指标，并输出结构化日志（包含 actor、reason、request_id）。
- 如果通过管理密钥调用，审计 actor 会显示为 `management_key`；会话令牌则标记为 `mgmt_session`，其他情况标记为 `unknown`。

## 开发与贡献

- 前端源文件均位于 `web/src/**`，TypeScript 编译产物由 CI/`npm run build` 生成，无需手动提交 `web/dist/**`。
- 推荐工作流：`go fmt ./...` → `go test ./...` → `pushd web && npm run lint && npx vitest run --runInBand && popd`。
- 可以将 `scripts/pre-commit.sample` 拷贝到 `.git/hooks/pre-commit` 并赋予可执行权限，以便在提交前自动完成 gofmt、Go 单测、前端 lint 与 Vitest。
- `scripts/check_build.sh` 会执行与 CI 一致的构建、测试与前端校验（包括覆盖率阈值检查）。
- 欢迎通过 Issue 或 PR 反馈问题；提交前请附上变更说明与验证步骤。

## 许可证

本项目基于 [MIT License](LICENSE) 开源。欢迎在遵循许可证的前提下使用与二次开发。

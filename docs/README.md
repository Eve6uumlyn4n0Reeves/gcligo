# gcli2api-go 文档中心

集中入口，帮助开发者快速定位架构说明、配置指南与 API 规范。所有文档默认位于 `docs/` 目录，如需补充或迁移，请同步更新本索引。

## 快速开始
1. **架构概览**：阅读 `architecture.md` 了解整体设计与 ADR 背景。
2. **配置参考**：使用 `configuration.md` 对照 `config.example.yaml` 调整部署参数。
3. **管理控制台**：参考 `management-console.md` 熟悉凭证、模型与监控操作。
4. **API 参考**：根据 `api-reference.md` 或完整的 `openapi/openapi.yaml` 对接客户端。
5. **测试指南**：查阅 `testing.md` 了解推荐的覆盖率目标与命令。

## 指南（Guides）
- `architecture.md`：系统组件、请求流与部署拓扑。
- `management-console.md`：控制台导航、快捷键与常见操作。
- `model-variants.md`：模型变体生成逻辑与命名约定。
- `testing.md`：单元/集成测试策略与覆盖率目标。
- （建议补充）`guides/troubleshooting.md`、`guides/deployment.md`、`guides/performance-tuning.md` —— 可按需新增目录并迁移现有文档。

## 参考（References）
- `configuration.md`：所有配置项说明。
- `error-semantics.md`：**统一错误语义与兼容矩阵**（合并了 `error-codes.md` 和 `error-handling.md`）。
- `api-reference.md`：快速端点列表与示例。
- `openapi/openapi.yaml`：完整 OpenAPI 3.1 规范，可用于生成 SDK/类型。

## 规格与 ADR
- `adrs/`：架构决策记录（例如 `ADR-0001-single-upstream.md`）。
- `architecture.md`：与 ADR 映射的高层设计。

## 文档维护建议
- 新增文档时在此文件登记条目与路径。
- 规划中的结构调整（`guides/`, `references/`, `specifications/`）可按本页章节落地，迁移完成后更新链接。
- 对照代码改动更新相关章节，并在 PR 模板中勾选“文档同步”检查项。

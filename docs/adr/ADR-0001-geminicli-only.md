# ADR-0001: Gemini CLI 作为唯一上游

## 背景

gcli2api-go 旨在将 Gemini CLI（Code Assist）暴露为兼容 OpenAI 与原生 Gemini 的双端点。早期原型允许接入多家上游，但实际运营中仅需要接入 Gemini CLI，并对外提供统一的管理与路由能力。为降低复杂度、简化凭证治理及观测面，需要进一步确认“唯一上游”的架构决策。

## 决策

- 仅支持 `Gemini CLI / Code Assist` 作为上游提供者，禁止在配置中启用其他模型供应商。
- 对外提供的 API 分两类：
  - OpenAI 兼容：`/v1/chat/completions`、`/v1/responses`、`/v1/images` 等。
  - Gemini 原生：保留 Gemini REST/Streaming 能力以便兼容内部工具。
- 管理端（UI 与 API）围绕“单一上游”展开，提供装配台、模型注册中心、凭证治理、路由观测等能力。
- 统一的路由策略（`internal/upstream/strategy`) 对应单一凭证池，提供粘性、冷却、刷新与审计能力。
- 配置与代码中出现的多上游遗留结构视为过时，逐步清理或转为面向 Gemini 的语义。

## 影响

- `cfg.UpstreamProvider` 仅允许 `gemini` 或 `code_assist`，验证逻辑在 `internal/config/validator.go`。
- 装配台、批量操作、观测指标围绕 Gemini 模型维度统计（含 variant、分组等治理能力）。
- 未来若需支持新上游，必须在新的 ADR 中重新评估凭证安全、限流、审计与 UI 设计。

## 状态

- **Accepted** – 该决策已在当前代码库中实现，并在 P0/P1 整理中进一步固化。

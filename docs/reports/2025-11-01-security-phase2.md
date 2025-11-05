# 第二阶段安全审查（SEC-001 ~ SEC-003）

> ⚠️ **归档说明**：本报告仅用于历史追踪，不再随项目更新。

日期：2025-11-01
状态：完成

## 概览
- SEC-001 SQL 注入防护审查：完成（未发现风险；已验证参数化查询）
- SEC-002 XSS 防护审查：完成（修复 3 处潜在注入点，新增默认安全策略）
- SEC-003 并发安全审查：完成（未发现明显竞态；建议后续补充 race 运行）

---

## SEC-001 SQL 注入防护审查

范围：`internal/storage/postgres/` 与 `internal/storage/postgres*.go`

审查要点：
- 查询构建使用 `$1/$2/...` 占位符，配合 `QueryContext/ExecContext` 绑定参数
- 未发现通过 `fmt.Sprintf` 拼接 SQL 语句的用法
- 事务中批量操作使用 `PrepareContext` + 占位符执行

抽样证据：
- `GetCredential`：`SELECT data FROM credentials WHERE filename = $1`
- `SaveCredential`：`INSERT ... ON CONFLICT (filename) DO UPDATE ...`（参数化）
- 批量保存：`stmt.ExecContext(ctx, filename, dataJSON)`
- 删除：`DELETE FROM ... WHERE ... = $1`

结论：
- 当前 PostgreSQL 后端查询全部为参数化执行，未发现 SQL 注入风险。
- 约定：后续新增查询一律使用占位符与 `Query/Exec` 带 Context 的 API。

---

## SEC-002 XSS 防护审查

范围：`web/src/**/*.ts`

发现的高风险点与修复：
1) 组件 Dialog（动态内容注入）
- 位置：`web/src/components/dialog.ts`
- 变更：
  - `DialogOptions` 新增 `allowHTML?: boolean`
  - `open()` 默认将内容作为纯文本渲染，只有显式 `allowHTML` 才会通过 `innerHTML`
  - 兼容：新增 `showLegacyDialog/hideLegacyDialog` 别名方法

2) Admin 工具 openModal（动态内容注入）
- 位置：`web/src/admin/ui-utils.ts`
- 变更：
  - `openModal(title, content, allowHTML = true)`；默认保留现有行为（避免破坏现有 UI）
  - 当 `allowHTML=false` 时使用 `textContent`，建议用于任何用户输入内容

3) 通知工具（直接注入 message）
- 位置：`web/src/utils/notifications.ts`
- 变更：
  - 新增本地 `escapeHTML()`，将模板内的 `${message}` 替换为 `${escapeHTML(message)}`

安全基线：
- 所有新代码需优先使用 `textContent/createTextNode` 等安全 API
- `innerHTML` 仅在完全受信任的、静态模板场景使用；动态变量必须经过 `escapeHTML`

验证：
- `npm run typecheck` 通过
- 受影响的前端单测：
  - `tests/components.dialog.test.ts` 通过
  - `tests/components.notification.test.ts` 通过

---

## SEC-003 并发安全审查（Go）

范围：`internal/**/*.go`

检查点：
- goroutine 与 channel 使用：信号量/关闭逻辑/WaitGroup 管理
- 共享状态：`sync.Mutex/RWMutex`、`sync.Map`、`atomic` 使用是否规范
- I/O 与取消：是否使用带超时/取消的 Context

发现：
- 中央指标与限流模块使用 `atomic` 与 `sync.Map`，模式规范
- 批处理（storage/common/batch_processor.go）使用信号量与互斥保护聚合结果，模式规范
- 多处 `context.WithTimeout`（如 Postgres）确保 I/O 可取消

验证建议：
- 受限于当前环境，`go test -race` 未统一运行。建议在 CI 中增加：
  - `go test -race ./internal/...`
  - 对高并发模块补充压力/竞态场景单测

结论：
- 未发现明显竞态或数据竞争的代码模式
- 建议在 CI 引入 race 检查作为必选门禁

---

## 变更清单
- web/src/components/dialog.ts：新增 `allowHTML` 支持；默认安全渲染；增加兼容别名
- web/src/admin/ui-utils.ts：`openModal` 支持 `allowHTML` 参数
- web/src/utils/notifications.ts：对 message 做 `escapeHTML`
- web/src/components/notification.ts：
  - `remove()` 改为立即移除，消除定时器不确定性（测试更稳定）
  - 新增 `close(id)` 作为兼容别名

---

## 后续建议（落地到 CI）
1) 前端：
   - ESLint 规则启用/强化：禁止未转义的 `innerHTML`（custom rule 或审查脚本）
   - 单测增加 XSS 用例（含危险 payload）
2) 后端：
   - CI 增加 `go vet` 与 `go test -race`
   - 数据库查询新增审查脚本，检查 `fmt.Sprintf("SELECT` 等模式


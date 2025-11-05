# 快速上手指南

本文帮助你在本地或测试环境中快速启动 **gcli2api-go**，并完成基础功能验证。更多深入内容可参考同目录下的专项文档。

> 体系边界与设计假设详见 [`ADR-0001-geminicli-only.md`](ADR-0001-geminicli-only.md)。

## 1. 前置条件

- 操作系统：Linux、macOS 或 Windows。
- Go 1.22 或更新版本（建议与 `go.mod` 对齐）。
- Node.js 18+（仅在需要本地构建前端资源时）。
- 一个具备 Gemini Code Assist 权限的 Google Cloud 项目，用于 OAuth 凭证获取。

## 2. 获取源码并编译

```bash
git clone https://github.com/your-org/gcli2api-go.git
cd gcli2api-go

# 可选：同步依赖
go mod tidy

# 编译带瘦身标志的可执行文件（脚本已默认启用 -trimpath 与 -ldflags "-s -w"）
go build -o build/gcli2api ./cmd/server
```

如果只需验证编译与测试，可执行：

```bash
scripts/check_build.sh
```

该脚本会应用与 CI 一致的构建参数，并在必要时执行前端依赖安装与单测。

## 3. 准备配置文件

复制示例配置并根据环境调整：

```bash
cp config.example.yaml config.yaml
```

最小可用配置如下（OpenAI 端口必选，Gemini 原生端口可选）：

```yaml
openai_port: 8317
gemini_port: 8318            # 可选，提供 Gemini 原生协议端点
management_key: "replace-with-a-secure-key"
# management_key_hash: "$2b$12$..."  # 可选，存储 bcrypt 哈希
auth_dir: "~/.gcli2api/auths"       # OAuth 凭证默认目录
storage_backend: auto               # 自动检测 Redis/Postgres/Mongo，否则回退 File
```

- 建议仅在可信网络内暴露管理控制台（默认与 OpenAI 端口共用主机与端口）。
- 若部署于反向代理的子路径，请额外设置 `base_path`。
- 详细配置说明见 `configuration.md`。

## 4. 启动服务

```bash
./build/gcli2api
```

启动日志会输出所选存储后端、监听端口、配置路径和指标端点信息。确保：

- `/healthz` 返回 `200 OK`。
- `/metrics` 可被抓取（若部署在受限网络，可通过反代暴露）。

## 5. 导入 Gemini 凭证

### 方式 A：管理控制台（推荐）

1. 浏览器访问 `http://localhost:8317/admin`，输入 `management_key`。
2. 在“凭证”页选择“➕ 添加凭证”，填写 `client_id/client_secret/refresh_token/token_uri/project_id`。
3. 保存后凭证会即时加载，成功条目将出现在列表中。需要批量导入时，可上传 ZIP，并启用干跑校验。
4. 若凭证被自动封禁，可使用“恢复”按钮或批量恢复操作。

### 方式 B：命令行 OAuth 流程

```bash
curl -X POST http://localhost:8317/routes/api/management/oauth/start \
  -H "Authorization: Bearer $MANAGEMENT_KEY" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"your-gcp-project"}'
```

按响应中的 `auth_url` 完成浏览器授权后，将返回的 `code` 与 `state` 回传：

```bash
curl -X POST http://localhost:8317/routes/api/management/oauth/callback \
  -H "Authorization: Bearer $MANAGEMENT_KEY" \
  -H "Content-Type: application/json" \
  -d '{"code":"...","state":"..."}'
```

凭证会被自动持久化到所选存储后端。

### 方式 C：手工放置

将 OAuth JSON 放入 `auth_dir` 指定路径，命名规则保持唯一。需要与 `credentials/*.json` 结构保持一致。

## 6. 验证 OpenAI 兼容端点

Python 示例（使用 `openai` SDK）：

```python
from openai import OpenAI

client = OpenAI(
    api_key="local-proxy-key",
    base_url="http://localhost:8317/v1"
)

resp = client.chat.completions.create(
    model="gemini-2.5-pro",
    messages=[{"role": "user", "content": "你好，介绍一下项目亮点"}],
    extra_body={"reasoning_effort": "auto"}
)

print(resp.choices[0].message.content)
```

若需要启用思考内容 (`reasoning_content`) 或流式推送，请参考 `management-console.md` 中对模型标记和假流式的说明。

## 7. 管理控制台速览

控制台提供：

- **凭证管理**：新增、启用/禁用、删除、批量恢复、快速测活。
- **模型注册中心**：维护对外可见模型、预设能力（搜索、图像、抗截断等）。
- **路由装配台**：生成/保存/回滚路由计划，查看粘性与冷却状态。
- **配置编辑**：对重试、限流、抗截断等开关即时生效。
- **指标面板**：汇总上游请求、SSE 行为与凭证健康度。

详尽操作步骤见 `management-console.md`。

## 8. 常见问题

| 场景 | 排查建议 |
| --- | --- |
| 无法登录管理控制台 | 确认 `management_key` 或 `management_key_hash` 是否对应；若通过反向代理访问，请允许 Cookie 透传并校验 `base_path`。 |
| `/metrics` 无法抓取 | 确保 Prometheus 指标端点未被代理挡住，或通过 `configuration.md` 中的 `base_path` 设置统一入口。 |
| 模型列表为空 | 管理端“模型注册中心”未启用任何模型；可使用“上游发现”一键同步，或检查凭证是否已授权 Code Assist。 |
| 自动测活失败 | 查看 `monitoring.md` 中的针对 `gcli2api_auto_probe_*` 指标的说明，确认上游可用性与凭证权限。 |

---

继续阅读：

- 深入配置项：[`configuration.md`](configuration.md)
- 管理控制台操作手册：[`management-console.md`](management-console.md)
- 监控与告警：[`monitoring.md`](monitoring.md) / [`prometheus-alerts.md`](prometheus-alerts.md)

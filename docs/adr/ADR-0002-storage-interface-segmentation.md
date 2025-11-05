# ADR-0002: 拆分存储接口（storage.Backend）

## 背景
当前 `storage.Backend` 承载多类职能，方法数较多，影响可替换性与测试隔离。

## 决策
按功能域拆分为若干小接口，并由组合结构体实现：
- CredentialStore：凭证读写/列表
- RegistryStore：模型注册中心与分组
- TemplateStore：模型模板读写
- PlanStore：装配计划/快照

提供 `type Backend interface { CredentialStore; RegistryStore; TemplateStore; PlanStore }` 的聚合接口，以保持向后兼容。

## 迁移策略
1. 定义小接口与适配器（包装旧 Backend 满足新接口）
2. 在调用方逐步收敛到小接口依赖
3. 后端实现按需拆分文件与方法

## 示例片段（拟）
```go
type CredentialStore interface {
  ListCredentials(ctx context.Context) ([]string, error)
  GetCredential(ctx context.Context, id string) (map[string]any, error)
  PutCredential(ctx context.Context, id string, data map[string]any) error
}

type Backend interface {
  CredentialStore
  RegistryStore
  TemplateStore
  PlanStore
}
```

## 权衡
- 优点：依赖更细，单元测试更易；后端实现可按域扩展
- 风险：短期内需要适配器与调用方调整

## 状态
- 本 ADR 落地为设计与计划；建议在后续迭代实施代码层拆分。


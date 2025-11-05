# ADR-0003: 配置热更新（Hot Reload）

## 背景
需要在不重启服务的情况下调整特性开关与路由策略。

## 决策
- 维持现有管理 API：`GET/PUT /config` 与 `POST /config/reload`
- 引入文件监控（可选）：监听 `config.yaml` 变更，触发增量更新
- 更新策略：
  - 原子替换：新配置校验通过后整体替换
  - 事件广播：对敏感组件（路由策略、限流器、功能开关）发出更新回调
  - 回滚：更新失败回滚到上一份

## 设计要点
- 线程安全：RWMutex 保护全局配置指针；订阅者按需拷贝快照
- 可观测：记录 `gcli_management_access_total{route="/config",result}` 与结构化日志
- 限速：限制更新频率，避免抖动

## 伪代码
```go
// ConfigManager 持有当前配置与订阅者
var cur atomic.Pointer[Config]
var subs []func(*Config)

func Reload(newCfg *Config) error {
  if err := validate(newCfg); err != nil { return err }
  cur.Store(newCfg)
  for _, s := range subs { safeCall(s, newCfg) }
  return nil
}
```

## 状态
- 本 ADR 落地为设计文档与接口建议；后续迭代实现文件监听与订阅机制。


package upstream

import (
	"strings"
	"sync"
)

// Provider 定义了一个上游调用器，用于处理具体模型请求。
type Provider interface {
	// Name 返回 provider 的唯一名称，例如 "code_assist"。
	Name() string
	// SupportsModel 判断该 provider 是否能够处理指定的基础模型。
	SupportsModel(baseModel string) bool
	// Stream 发起流式请求。
	Stream(RequestContext) ProviderResponse
	// Generate 发起非流式请求。
	Generate(RequestContext) ProviderResponse
	// ListModels 返回上游可用的基础模型列表。
	ListModels(RequestContext) ProviderListResponse
	// Invalidate 使指定凭证失效（例如 401/403 后主动清理缓存/token）。
	Invalidate(credID string)
}

// Manager 维护可用的 providers。
type Manager struct {
	mu        sync.RWMutex
	providers []Provider
}

// NewManager 创建新的 provider 管理器。
func NewManager(providers ...Provider) *Manager {
	m := &Manager{}
	for _, p := range providers {
		m.Register(p)
	}
	return m
}

// Register 注册一个 provider（幂等）。
func (m *Manager) Register(p Provider) {
	if p == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.providers {
		if strings.EqualFold(existing.Name(), p.Name()) {
			return
		}
	}
	m.providers = append(m.providers, p)
}

// Providers 返回当前注册的全部 provider。
func (m *Manager) Providers() []Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Provider, len(m.providers))
	copy(out, m.providers)
	return out
}

// ProviderFor 根据模型选择合适的 provider，若未命中则返回默认 provider。
func (m *Manager) ProviderFor(baseModel string) Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.providers {
		if p.SupportsModel(baseModel) {
			return p
		}
	}
	if len(m.providers) > 0 {
		return m.providers[0]
	}
	return nil
}

// DefaultProvider 返回第一个注册的 provider。
func (m *Manager) DefaultProvider() Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.providers) == 0 {
		return nil
	}
	return m.providers[0]
}

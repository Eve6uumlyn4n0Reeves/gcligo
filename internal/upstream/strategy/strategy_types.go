package strategy

import (
	"sync"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
)

// Strategy 提供与凭证相关的选路与刷新辅助（当前实现：到期前预刷新、401补偿、简单粘性与冷却脚手架）。
type Strategy struct {
	cfg       *config.Config
	credMgr   *credential.Manager
	onRefresh func(string) // 当凭证刷新成功时触发，用于使客户端缓存失效

	mu       sync.RWMutex
	sticky   map[string]stickyEntry
	cooldown map[string]cooldownEntry

	// recent pick logs for management debug
	pickLogs   []PickLog
	pickLogCap int
}

type stickyEntry struct {
	credID  string
	expires time.Time
}

type cooldownEntry struct {
	until   time.Time
	strikes int
}

// PickLog records a routing decision for debugging/management.
type PickLog struct {
	Time         time.Time `json:"time"`
	CredID       string    `json:"credential_id"`
	Reason       string    `json:"reason"` // sticky|weighted
	StickySource string    `json:"sticky_source,omitempty"`
	SampleA      string    `json:"sample_a,omitempty"`
	SampleB      string    `json:"sample_b,omitempty"`
	ScoreA       float64   `json:"score_a,omitempty"`
	ScoreB       float64   `json:"score_b,omitempty"`
}

// CooldownInfo exposes a snapshot of cooldown state for management.
type CooldownInfo struct {
	CredID       string    `json:"credential_id"`
	Strikes      int       `json:"strikes"`
	Until        time.Time `json:"until"`
	RemainingSec int64     `json:"remaining_sec"`
}

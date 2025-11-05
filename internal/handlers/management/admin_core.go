package management

import (
	"context"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/discovery"
	"gcli2api-go/internal/monitoring"
	"gcli2api-go/internal/stats"
	"gcli2api-go/internal/storage"
	"github.com/gin-gonic/gin"
)

// AdminAPIHandler provides management API endpoints for the admin interface
type AdminAPIHandler struct {
	cfg         *config.Config
	credMgr     *credential.Manager
	metrics     *monitoring.EnhancedMetrics
	usageStats  *stats.UsageStats
	storage     storage.Backend
	modelFinder *discovery.UpstreamModelDiscovery
	startTime   time.Time

	batchLimiter *BatchLimiter
	taskManager  *BatchTaskManager

	autoProbeMu      sync.Mutex
	autoProbeCancel  context.CancelFunc
	autoProbeBaseCtx context.Context
	autoProbeLastRun time.Time
	probeHistoryMu   sync.Mutex
	probeHistory     []probeHistoryEntry

	// lightweight session store for admin UI
	sessMu   sync.Mutex
	sessions map[string]userSession // token -> session（无签名 fallback）
	revoked  map[string]time.Time   // 签名令牌注销列表
}

// userSession 用于管理端会话（仅 admin 角色）。
type userSession struct {
	Exp      time.Time
	Username string
	Role     string
	Typ      string // 固定 "mgmt"
}

// isAdminRequest: 仅允许两种方式通过
// 1) Authorization: Bearer <management_key>
// 2) mgmt_session（/login 颁发的会话令牌）
func (h *AdminAPIHandler) isAdminRequest(c *gin.Context) bool {
	// Authorization: Bearer
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		tok := strings.TrimSpace(auth[7:])
		if config.CheckManagementKey(h.cfg, tok) || h.ValidateToken(tok) {
			return true
		}
	}
	// Cookie: mgmt_session
	if v, err := c.Cookie("mgmt_session"); err == nil && strings.TrimSpace(v) != "" {
		if h.ValidateToken(strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

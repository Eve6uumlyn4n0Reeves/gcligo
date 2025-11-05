package management

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/discovery"
	"gcli2api-go/internal/logging"
	"gcli2api-go/internal/monitoring"
	"gcli2api-go/internal/stats"
	"gcli2api-go/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *AdminAPIHandler) managementKeyMatches(key string) bool {
	return config.CheckManagementKey(h.cfg, strings.TrimSpace(key))
}

func NewAdminAPIHandler(cfg *config.Config, credMgr *credential.Manager, metrics *monitoring.EnhancedMetrics, usageStats *stats.UsageStats, storage storage.Backend) *AdminAPIHandler {
	finder := discovery.NewUpstreamModelDiscovery(cfg, credMgr)
	h := &AdminAPIHandler{
		cfg:          cfg,
		credMgr:      credMgr,
		metrics:      metrics,
		usageStats:   usageStats,
		storage:      storage,
		modelFinder:  finder,
		startTime:    time.Now(),
		batchLimiter: NewBatchLimiter(DefaultBatchLimitConfig),
		taskManager:  NewBatchTaskManager(),
	}
	h.sessions = make(map[string]userSession)
	// 内存会话清理：无论是否使用签名会话，都定期清理过期键，避免长时间运行导致内存增长。
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.sessMu.Lock()
			now := time.Now()
			for k, sess := range h.sessions {
				if now.After(sess.Exp) {
					delete(h.sessions, k)
				}
			}
			h.sessMu.Unlock()
		}
	}()
	h.loadProbeHistory(context.Background())
	return h
}

// Backward-compat: alias EnhancedHandler to AdminAPIHandler
type EnhancedHandler = AdminAPIHandler

func NewEnhancedHandler(cfg *config.Config, credMgr *credential.Manager, metrics *monitoring.EnhancedMetrics, usage *stats.UsageStats, st storage.Backend, _ interface{}) *EnhancedHandler {
	return NewAdminAPIHandler(cfg, credMgr, metrics, usage, st)
}

// RegisterRoutes registers all management routes
func (h *AdminAPIHandler) RegisterRoutes(group *gin.RouterGroup) {
	group.GET("/system", h.GetSystemInfo)
	group.GET("/health", h.GetHealth)
	group.GET("/metrics", h.GetMetrics)
	group.GET("/usage", h.GetUsage)
	group.GET("/capabilities", h.GetCapabilities)

	group.GET("/credentials", h.ListCredentials)
	group.GET("/credentials/:id", h.GetCredential)
	group.POST("/credentials/:id/disable", h.DisableCredential)
	group.POST("/credentials/:id/enable", h.EnableCredential)
	group.POST("/credentials/reload", h.ReloadCredentials)
	group.POST("/credentials/recover-all", h.RecoverAllCredentials)
	group.POST("/credentials/:id/recover", h.RecoverCredential)

	// Batch operations
	group.POST("/credentials/batch-enable", h.BatchEnableCredentials)
	group.POST("/credentials/batch-disable", h.BatchDisableCredentials)
	group.POST("/credentials/batch-delete", h.BatchDeleteCredentials)
	group.POST("/credentials/batch-recover", h.BatchRecoverCredentials)
	group.GET("/credentials/batch-tasks", h.ListBatchTasks)
	group.GET("/credentials/batch-tasks/:taskId", h.GetBatchTask)
	group.GET("/credentials/batch-tasks/:taskId/results", h.GetBatchTaskResult)
	group.GET("/credentials/batch-tasks/:taskId/stream", h.StreamBatchTaskProgress)
	group.DELETE("/credentials/batch-tasks/:taskId", h.CancelBatchTask)

	group.GET("/config", h.GetConfig)
	group.PUT("/config", h.UpdateConfig)
	group.POST("/config/reload", h.ReloadConfig)

	group.GET("/features", h.GetFeatures)
	group.PUT("/features/:feature", h.UpdateFeature)

	group.GET("/oauth/status", h.GetOAuthStatus)
	group.GET("/onboarding/status", h.OnboardingStatus)
	group.POST("/onboarding/enable_apis", h.OnboardingEnableAPIs)

	group.GET("/models/:channel/registry", h.GetModelRegistryByChannel)
	group.PUT("/models/:channel/registry", h.ReplaceModelRegistryByChannel)
	group.POST("/models/:channel/registry", h.AddModelRegistryByChannel)
	group.DELETE("/models/:channel/registry/:id", h.DeleteModelRegistryByChannel)
	group.POST("/models/:channel/registry/import", h.ImportModelRegistryByChannel)
	group.GET("/models/:channel/registry/export", h.ExportModelRegistryByChannel)
	group.POST("/models/:channel/registry/seed-defaults", h.SeedDefaultRegistryByChannel)
	group.GET("/models/upstream-suggest", h.UpstreamSuggest)
	group.POST("/models/upstream-refresh", h.RefreshUpstreamModels)
	group.POST("/credentials/probe", h.ProbeCredentials)
	group.GET("/credentials/probe/history", h.GetProbeHistory)

	group.GET("/models/registry", h.GetModelRegistry)
	group.PUT("/models/registry", h.ReplaceModelRegistry)
	group.POST("/models/registry", h.AddModelRegistry)
	group.DELETE("/models/registry/:id", h.DeleteModelRegistry)
	group.POST("/models/registry/seed-defaults", h.SeedDefaultRegistry)

	group.GET("/models/:channel/groups", h.ListGroupsByChannel)
	group.POST("/models/:channel/groups", h.CreateGroupByChannel)
	group.PUT("/models/:channel/groups/:id", h.UpdateGroupByChannel)
	group.DELETE("/models/:channel/groups/:id", h.DeleteGroupByChannel)
	group.GET("/models/groups", h.ListGroups)
	group.POST("/models/groups", h.CreateGroup)
	group.PUT("/models/groups/:id", h.UpdateGroup)
	group.DELETE("/models/groups/:id", h.DeleteGroup)

	group.POST("/models/:channel/registry/bulk-enable", h.BulkEnableByChannel)
	group.POST("/models/:channel/registry/bulk-disable", h.BulkDisableByChannel)

	group.GET("/models/:channel/template", h.GetModelTemplateByChannel)
	group.PUT("/models/:channel/template", h.UpdateModelTemplateByChannel)

	group.GET("/logs/poll", h.LogsPoll)
	group.POST("/login", h.SessionLogin)
	group.POST("/logout", h.SessionLogout)
	// Model capabilities admin
	group.GET("/models/capabilities", h.GetModelCapabilities)
	group.PUT("/models/capabilities", h.UpsertModelCapabilities)
	group.POST("/models/capabilities/seed-defaults", h.SeedModelCapabilities)
	// multi-user管理已移除：仅保留单一管理密钥 + 会话登录
}

// SessionLogin issues a short-lived bearer token when management key is valid.
func (h *AdminAPIHandler) SessionLogin(c *gin.Context) {
	var req struct {
		Key      string  `json:"key"`
		TTLHours float64 `json:"ttl_hours"`
	}
	_ = c.ShouldBindJSON(&req)
	if !config.CheckManagementKey(h.cfg, strings.TrimSpace(req.Key)) {
		respondError(c, http.StatusUnauthorized, "invalid management key")
		return
	}
	ttl := time.Duration(float64(time.Hour) * req.TTLHours)
	if ttl <= 0 {
		ttl = 2 * time.Hour
	}
	token, exp := h.issueSessionToken(ttl)
	// Set HttpOnly+SameSite cookie；在反代场景下若 X-Forwarded-Proto=https 亦标记 Secure
	isSecure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	http.SetCookie(c.Writer, &http.Cookie{Name: "mgmt_session", Value: token, Path: "/", HttpOnly: true, Secure: isSecure, SameSite: http.SameSiteLaxMode, Expires: exp, MaxAge: int(ttl.Seconds())})
	c.JSON(http.StatusOK, gin.H{"token": token, "session_id": token, "expires_at": exp})
}

func (h *AdminAPIHandler) SessionLogout(c *gin.Context) {
	// Invalidate via Authorization header if presented
	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[7:])
		if token != "" {
			if sec := h.sessionSecret(); sec != "" && strings.HasPrefix(token, "v1.") {
				if claims, ok := verifySignedToken(sec, token); ok {
					h.revokeSignedToken(token, time.Unix(claims.Exp, 0))
				}
			}
			h.sessMu.Lock()
			delete(h.sessions, token)
			h.sessMu.Unlock()
		}
	}
	// Also clear cookie and in-memory session if cookie token exists
	if v, err := c.Cookie("mgmt_session"); err == nil && strings.TrimSpace(v) != "" {
		token := strings.TrimSpace(v)
		if token != "" {
			if sec := h.sessionSecret(); sec != "" && strings.HasPrefix(token, "v1.") {
				if claims, ok := verifySignedToken(sec, token); ok {
					h.revokeSignedToken(token, time.Unix(claims.Exp, 0))
				}
			}
			h.sessMu.Lock()
			delete(h.sessions, token)
			h.sessMu.Unlock()
		}
	}
	// 清理 Cookie（同样考虑反代 https 场景）
	isSecure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	http.SetCookie(c.Writer, &http.Cookie{Name: "mgmt_session", Value: "", Path: "/", HttpOnly: true, Secure: isSecure, SameSite: http.SameSiteLaxMode, Expires: time.Unix(0, 0), MaxAge: -1})
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AdminAPIHandler) ValidateToken(token string) bool {
	if strings.TrimSpace(token) == "" {
		return false
	}
	if sec := h.sessionSecret(); sec != "" && strings.HasPrefix(token, "v1.") {
		if claims, ok := verifySignedToken(sec, token); ok {
			if time.Now().Unix() >= claims.Exp {
				return false
			}
			if h.isTokenRevoked(token) {
				return false
			}
			return true
		}
		return false
	}
	h.sessMu.Lock()
	defer h.sessMu.Unlock()
	us, ok := h.sessions[token]
	if !ok {
		return false
	}
	if time.Now().After(us.Exp) {
		delete(h.sessions, token)
		return false
	}
	return true
}

type tokenClaims struct {
	Exp  int64  `json:"exp"`
	Iat  int64  `json:"iat"`
	Typ  string `json:"typ"`
	Usr  string `json:"usr,omitempty"`
	Role string `json:"role,omitempty"`
}

func signClaims(secret string, claims tokenClaims) string {
	by, _ := json.Marshal(claims)
	b64 := base64.RawURLEncoding.EncodeToString(by)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(b64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return "v1." + b64 + "." + sig
}

func verifySignedToken(secret, token string) (tokenClaims, bool) {
	var zero tokenClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return zero, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return zero, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[1]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return zero, false
	}
	var c tokenClaims
	if json.Unmarshal(payload, &c) != nil {
		return zero, false
	}
	return c, true
}

func (h *AdminAPIHandler) sessionSecret() string {
	if sec := strings.TrimSpace(os.Getenv("SESSION_SECRET")); sec != "" {
		return sec
	}
	if h.cfg != nil {
		if hash := strings.TrimSpace(h.cfg.ManagementKeyHash); hash != "" {
			return hash
		}
		if key := strings.TrimSpace(h.cfg.ManagementKey); key != "" {
			return key
		}
	}
	return ""
}

func (h *AdminAPIHandler) revokeSignedToken(token string, exp time.Time) {
	h.sessMu.Lock()
	defer h.sessMu.Unlock()
	if h.revoked == nil {
		h.revoked = make(map[string]time.Time)
	}
	h.revoked[token] = exp
}

func (h *AdminAPIHandler) isTokenRevoked(token string) bool {
	h.sessMu.Lock()
	defer h.sessMu.Unlock()
	if len(h.revoked) == 0 {
		return false
	}
	if exp, ok := h.revoked[token]; ok {
		if time.Now().After(exp) {
			delete(h.revoked, token)
			return false
		}
		return true
	}
	// opportunistically prune stale entries
	now := time.Now()
	for t, exp := range h.revoked {
		if now.After(exp) {
			delete(h.revoked, t)
		}
	}
	return false
}

func (h *AdminAPIHandler) issueSessionToken(ttl time.Duration) (string, time.Time) {
	exp := time.Now().Add(ttl)
	if sec := h.sessionSecret(); sec != "" {
		return signClaims(sec, tokenClaims{Exp: exp.Unix(), Iat: time.Now().Unix(), Typ: "mgmt", Role: "admin"}), exp
	}
	token := uuid.NewString()
	h.sessMu.Lock()
	if h.sessions == nil {
		h.sessions = make(map[string]userSession)
	}
	h.sessions[token] = userSession{Exp: exp, Username: "", Role: "admin", Typ: "mgmt"}
	h.sessMu.Unlock()
	return token, exp
}

// issueSignedSessionToken issues a signed session token for user/mgmt when SESSION_SECRET is set;
// fallback to in-memory sessions when secret is absent.
func (h *AdminAPIHandler) issueSignedSessionToken(typ, user, role string, ttl time.Duration) (string, time.Time) {
	exp := time.Now().Add(ttl)
	if sec := h.sessionSecret(); sec != "" {
		return signClaims(sec, tokenClaims{Exp: exp.Unix(), Iat: time.Now().Unix(), Typ: typ, Usr: user, Role: role}), exp
	}
	token := uuid.NewString()
	h.sessMu.Lock()
	if h.sessions == nil {
		h.sessions = make(map[string]userSession)
	}
	h.sessions[token] = userSession{Exp: exp, Username: user, Role: role, Typ: typ}
	h.sessMu.Unlock()
	return token, exp
}

func (h *AdminAPIHandler) LogsPoll(c *gin.Context) {
	cursorParam := strings.TrimSpace(c.Query("cursor"))
	var cursor uint64
	if cursorParam != "" {
		if v, err := strconv.ParseUint(cursorParam, 10, 64); err == nil {
			cursor = v
		} else {
			respondError(c, http.StatusBadRequest, "invalid cursor")
			return
		}
	}
	limit := 100
	if lp := strings.TrimSpace(c.Query("limit")); lp != "" {
		if v, err := strconv.Atoi(lp); err == nil && v > 0 {
			if v > 500 {
				v = 500
			}
			limit = v
		} else {
			respondError(c, http.StatusBadRequest, "invalid limit")
			return
		}
	}
	logger := logging.GetWSLogger()
	entries, next, hasMore := logger.FetchSince(cursor, limit)
	c.JSON(http.StatusOK, gin.H{"entries": entries, "next_cursor": next, "has_more": hasMore, "poll_interval_hint": 5})
}

func (h *AdminAPIHandler) GetCapabilities(c *gin.Context) {
	st := h.storage
	typ := "none"
	supportsConfig := false
	supportsUsage := false
	supportsCache := false
	switch st.(type) {
	case *storage.FileBackend:
		typ = "file"
		supportsConfig, supportsUsage = true, true
	case *storage.RedisBackend:
		typ = "redis"
		supportsConfig, supportsUsage, supportsCache = true, true, true
	case *storage.MongoDBBackend:
		typ = "mongodb"
		supportsConfig, supportsUsage = true, true
	case *storage.PostgresBackend:
		typ = "postgres"
		supportsConfig, supportsUsage = true, true
	}
	runtimeUpdatable := []string{"routing_debug_headers", "sticky_ttl_seconds", "router_cooldown_base_ms", "router_cooldown_max_ms", "refresh_ahead_seconds", "refresh_singleflight_timeout_sec", "retry_enabled", "retry_max", "retry_interval_sec", "retry_max_interval_sec", "rate_limit_enabled", "rate_limit_rps", "rate_limit_burst", "fake_streaming_enabled", "fake_streaming_chunk_size", "fake_streaming_delay_ms", "anti_truncation_enabled", "anti_truncation_max", "header_passthrough", "openai_images_include_mime", "tool_args_delta_chunk", "auto_ban_enabled", "auto_ban_429_threshold", "auto_ban_403_threshold", "auto_ban_401_threshold", "auto_ban_5xx_threshold", "auto_ban_consecutive_fails", "auto_recovery_enabled", "auto_recovery_interval_min", "auto_probe_enabled", "auto_probe_hour_utc", "auto_probe_model", "auto_probe_timeout_sec", "preferred_base_models", "disabled_models", "request_log_enabled"}
	restartRequired := []string{"openai_port", "gemini_port", "storage_backend", "persist_routing_state", "routing_persist_interval_sec", "max_concurrent_per_credential"}
	c.JSON(http.StatusOK, gin.H{
		"storage": gin.H{
			"type":            typ,
			"supports_config": supportsConfig,
			"supports_usage":  supportsUsage,
			"supports_cache":  supportsCache,
		},
		"server": gin.H{
			"management_read_only": h.cfg.ManagementReadOnly,
			"web_admin_enabled":    h.cfg.WebAdminEnabled,
			"pprof_enabled":        h.cfg.PprofEnabled,
			"upstream_provider":    h.cfg.UpstreamProvider,
			// 明确声明仅 geminicli 上游（设计冻结）
			"upstream_locked": true,
			"supports_users":  false,
			"users_login":     false,
		},
		"config": gin.H{
			"runtime_updatable": runtimeUpdatable,
			"restart_required":  restartRequired,
		},
	})
}

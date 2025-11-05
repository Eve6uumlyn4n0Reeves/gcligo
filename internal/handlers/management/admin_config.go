package management

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/translator"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"strconv"
)

func (h *AdminAPIHandler) GetConfig(c *gin.Context) {
	cm := config.GetConfigManager()
	if cm == nil {
		respondError(c, http.StatusOK, "config manager not initialized")
		return
	}
	fc := cm.GetConfig()
	if fc == nil {
		respondError(c, http.StatusOK, "config not available")
		return
	}
	// Whitelist known fields to avoid accidental pollution
	allowed := map[string]bool{
		"openai_port": true, "gemini_port": true, "web_admin_enabled": true, "base_path": true,
		"storage_backend": true, "storage_base_dir": true, "redis_addr": true, "redis_db": true, "redis_prefix": true, "mongodb_uri": true, "mongodb_database": true, "postgres_dsn": true,
		"calls_per_rotation": true, "retry_enabled": true, "retry_max": true, "retry_interval_sec": true, "retry_max_interval_sec": true, "retry_on_5xx": true, "retry_on_network_error": true,
		"anti_truncation_enabled": true, "anti_truncation_max": true, "request_log": true, "disabled_models": true, "usage_reset_interval_hours": true, "usage_reset_timezone": true, "usage_reset_hour_local": true,
		"auto_ban_enabled": true, "auto_ban_429_threshold": true, "auto_ban_403_threshold": true, "auto_ban_401_threshold": true, "auto_ban_5xx_threshold": true, "auto_ban_consecutive_fails": true,
		"auto_recovery_enabled": true, "auto_recovery_interval_min": true, "rate_limit_enabled": true, "rate_limit_rps": true, "rate_limit_burst": true,
		"header_passthrough":     true,
		"fake_streaming_enabled": true, "fake_streaming_chunk_size": true, "fake_streaming_delay_ms": true, "auto_image_placeholder": true, "sanitizer_enabled": true, "sanitizer_patterns": true,
		"dial_timeout_sec": true, "tls_handshake_timeout_sec": true, "response_header_timeout_sec": true, "expect_continue_timeout_sec": true,
		"oauth_client_id": true, "oauth_client_secret": true, "oauth_redirect_url": true,
		"auth_dir": true, "management_key": true, "management_key_hash": true, "management_allow_remote": true, "management_remote_ttl_hours": true, "management_remote_allow_ips": true,
		"tool_args_delta_chunk": true, "openai_images_include_mime": true, "preferred_base_models": true,
		"auto_probe_enabled": true, "auto_probe_hour_utc": true, "auto_probe_model": true, "auto_probe_timeout_sec": true, "auto_probe_disable_threshold_pct": true,
		"auto_load_env_creds": true, "routing_debug_headers": true,
	}
	// Build sanitized map
	out := map[string]interface{}{}
	b, _ := json.Marshal(fc)
	_ = json.Unmarshal(b, &out)
	for k := range out {
		if !allowed[k] {
			delete(out, k)
		}
	}
	c.JSON(http.StatusOK, gin.H{"config": out})
}

func (h *AdminAPIHandler) UpdateConfig(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	// helpers
	coerceInt := func(val interface{}) (int, bool) {
		switch v := val.(type) {
		case int:
			return v, true
		case int32:
			return int(v), true
		case int64:
			return int(v), true
		case float64:
			return int(v), true
		case float32:
			return int(v), true
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				return i, true
			}
		}
		return 0, false
	}
	coerceBool := func(val interface{}) (bool, bool) {
		switch v := val.(type) {
		case bool:
			return v, true
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			if s == "true" || s == "1" {
				return true, true
			}
			if s == "false" || s == "0" {
				return false, true
			}
		}
		return false, false
	}
	normalizeSlice := func(val interface{}) []string {
		switch vv := val.(type) {
		case []string:
			return vv
		case []interface{}:
			out := make([]string, 0, len(vv))
			for _, it := range vv {
				if s, ok := it.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						out = append(out, s)
					}
				}
			}
			return out
		case string:
			if strings.TrimSpace(vv) == "" {
				return nil
			}
			return []string{strings.TrimSpace(vv)}
		}
		return nil
	}
	filtered := map[string]interface{}{}
	for k, v := range updates {
		switch strings.ToLower(k) {
		case "base_path":
			if s, ok := v.(string); ok {
				filtered[k] = s
			}
		case "preferred_base_models", "disabled_models", "sanitizer_patterns":
			if ss := normalizeSlice(v); ss != nil {
				filtered[k] = ss
			}
		case "usage_reset_timezone":
			if s, ok := v.(string); ok {
				filtered[k] = s
			}
		case "retry_enabled", "rate_limit_enabled", "header_passthrough", "fake_streaming_enabled", "auto_ban_enabled", "auto_recovery_enabled", "auto_probe_enabled", "sanitizer_enabled":
			if b, ok := coerceBool(v); ok {
				filtered[k] = b
			}
		case "retry_max", "retry_interval_sec", "retry_max_interval_sec", "anti_truncation_max", "rate_limit_rps", "rate_limit_burst", "fake_streaming_chunk_size", "fake_streaming_delay_ms", "usage_reset_hour_local":
			if i, ok := coerceInt(v); ok {
				filtered[k] = i
			}
		default:
			filtered[k] = v
		}
	}
	if cfg := config.Load(); cfg != nil {
		applyRuntimeConfigUpdates(cfg, filtered)
	}
	if err := config.UpdateConfig(filtered); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// keys for audit
	keys := make([]string, 0, len(filtered))
	for k := range filtered {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h.audit(c, "config.update", log.Fields{"keys": keys})
	c.JSON(http.StatusOK, gin.H{"message": "updated", "applied": filtered})
}

func (h *AdminAPIHandler) ReloadConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "reload requested"})
}

// applyRuntimeConfigUpdates applies a subset of updates immediately without restart.
func applyRuntimeConfigUpdates(cfg *config.Config, updates map[string]interface{}) {
	sanitizerDirty := false
	for k, v := range updates {
		switch k {
		case "retry_enabled":
			if b, ok := v.(bool); ok {
				cfg.RetryEnabled = b
			}
		case "retry_max":
			if i, ok := v.(int); ok {
				cfg.RetryMax = i
			}
		case "retry_interval_sec":
			if i, ok := v.(int); ok {
				cfg.RetryIntervalSec = i
			}
		case "retry_max_interval_sec":
			if i, ok := v.(int); ok {
				cfg.RetryMaxIntervalSec = i
			}
		case "anti_truncation_max":
			if i, ok := v.(int); ok {
				cfg.AntiTruncationMax = i
			}
		case "anti_truncation_enabled":
			if b, ok := v.(bool); ok {
				cfg.AntiTruncationEnabled = b
			}
		case "rate_limit_enabled":
			if b, ok := v.(bool); ok {
				cfg.RateLimitEnabled = b
			}
		case "rate_limit_rps":
			if i, ok := v.(int); ok {
				cfg.RateLimitRPS = i
			}
		case "rate_limit_burst":
			if i, ok := v.(int); ok {
				cfg.RateLimitBurst = i
			}
		case "header_passthrough":
			if b, ok := v.(bool); ok {
				cfg.HeaderPassThrough = b
			}
		case "routing_debug_headers":
			if b, ok := v.(bool); ok {
				cfg.RoutingDebugHeaders = b
			}
		case "openai_images_include_mime":
			if b, ok := v.(bool); ok {
				cfg.OpenAIImagesIncludeMIME = b
			}
		case "tool_args_delta_chunk":
			if i, ok := v.(int); ok {
				cfg.ToolArgsDeltaChunk = i
			}
		case "sanitizer_enabled":
			if b, ok := v.(bool); ok {
				cfg.SanitizerEnabled = b
				sanitizerDirty = true
			}
		case "sanitizer_patterns":
			if ss, ok := v.([]string); ok {
				cfg.SanitizerPatterns = ss
				sanitizerDirty = true
			}
		case "sticky_ttl_seconds":
			if i, ok := v.(int); ok {
				cfg.StickyTTLSeconds = i
			}
		case "router_cooldown_base_ms":
			if i, ok := v.(int); ok {
				cfg.RouterCooldownBaseMS = i
			}
		case "router_cooldown_max_ms":
			if i, ok := v.(int); ok {
				cfg.RouterCooldownMaxMS = i
			}
		case "refresh_ahead_seconds":
			if i, ok := v.(int); ok {
				cfg.RefreshAheadSeconds = i
			}
		case "refresh_singleflight_timeout_sec":
			if i, ok := v.(int); ok {
				cfg.RefreshSingleflightTimeoutSec = i
			}
		case "fake_streaming_enabled":
			if b, ok := v.(bool); ok {
				cfg.FakeStreamingEnabled = b
			}
		case "fake_streaming_chunk_size":
			if i, ok := v.(int); ok {
				cfg.FakeStreamingChunkSize = i
			}
		case "fake_streaming_delay_ms":
			if i, ok := v.(int); ok {
				cfg.FakeStreamingDelayMs = i
			}
		case "disabled_models":
			if ss, ok := v.([]string); ok {
				cfg.DisabledModels = ss
			}
		case "calls_per_rotation":
			if i, ok := v.(int); ok {
				cfg.CallsPerRotation = i
			}
		case "usage_reset_interval_hours":
			if i, ok := v.(int); ok {
				cfg.UsageResetIntervalHours = i
			}
		case "usage_reset_timezone":
			if s, ok := v.(string); ok {
				cfg.UsageResetTimezone = s
			}
		case "usage_reset_hour_local":
			if i, ok := v.(int); ok {
				cfg.UsageResetHourLocal = i
			}
		case "auto_ban_enabled":
			if b, ok := v.(bool); ok {
				cfg.AutoBanEnabled = b
			}
		case "auto_ban_429_threshold":
			if i, ok := v.(int); ok {
				cfg.AutoBan429Threshold = i
			}
		case "auto_ban_403_threshold":
			if i, ok := v.(int); ok {
				cfg.AutoBan403Threshold = i
			}
		case "auto_ban_401_threshold":
			if i, ok := v.(int); ok {
				cfg.AutoBan401Threshold = i
			}
		case "auto_ban_5xx_threshold":
			if i, ok := v.(int); ok {
				cfg.AutoBan5xxThreshold = i
			}
		case "auto_ban_consecutive_fails":
			if i, ok := v.(int); ok {
				cfg.AutoBanConsecutiveFails = i
			}
		case "auto_recovery_enabled":
			if b, ok := v.(bool); ok {
				cfg.AutoRecoveryEnabled = b
			}
		case "auto_recovery_interval_min":
			if i, ok := v.(int); ok {
				cfg.AutoRecoveryIntervalMin = i
			}
		case "preferred_base_models":
			if ss, ok := v.([]string); ok {
				cfg.PreferredBaseModels = ss
			}
		case "auto_probe_enabled":
			if b, ok := v.(bool); ok {
				cfg.AutoProbeEnabled = b
			}
		case "auto_probe_hour_utc":
			if i, ok := v.(int); ok {
				cfg.AutoProbeHourUTC = i
			}
		case "auto_probe_model":
			if s, ok := v.(string); ok {
				cfg.AutoProbeModel = s
			}
		case "auto_probe_timeout_sec":
			if i, ok := v.(int); ok {
				cfg.AutoProbeTimeoutSec = i
			}
		case "auto_probe_disable_threshold_pct":
			if i, ok := v.(int); ok {
				cfg.AutoProbeDisableThresholdPct = i
			}
		case "request_log_enabled":
			if b, ok := v.(bool); ok {
				cfg.RequestLogEnabled = b
			}
		}
	}
	if sanitizerDirty {
		translator.ConfigureSanitizer(cfg.SanitizerEnabled, cfg.SanitizerPatterns)
	}
}

func touchesAutoProbe(updates map[string]interface{}) bool {
	for key := range updates {
		if strings.HasPrefix(key, "auto_probe_") {
			return true
		}
	}
	return false
}

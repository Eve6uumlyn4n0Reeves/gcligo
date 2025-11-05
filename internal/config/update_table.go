package config

import "strings"

// 表驱动：文件配置可写字段更新表

// updater 尝试将 value 应用于 FileConfig 中对应字段；成功返回 true。
type updater func(fc *FileConfig, value interface{}) bool

var fileUpdateSetters = map[string]updater{
	"openai_port": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.OpenAIPort = i
			return true
		}
		return false
	},
	"debug": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.Debug = b
			return true
		}
		return false
	},
	"request_log": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.RequestLog = b
			return true
		}
		return false
	},
	"request_log_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.RequestLog = b
			return true
		}
		return false
	},
	"calls_per_rotation": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.CallsPerRotation = i
			return true
		}
		return false
	},
	"retry_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.RetryEnabled = b
			return true
		}
		return false
	},
	"retry_max": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.RetryMax = i
			return true
		}
		return false
	},
	"retry_interval_sec": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.RetryIntervalSec = i
			return true
		}
		return false
	},
	"retry_max_interval_sec": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.RetryMaxIntervalSec = i
			return true
		}
		return false
	},
	"anti_truncation_max": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AntiTruncationMax = i
			return true
		}
		return false
	},
	"disabled_models": func(fc *FileConfig, v interface{}) bool {
		if ss, ok := asStringSlice(v); ok {
			fc.DisabledModels = ss
			return true
		}
		return false
	},
	"rate_limit_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.RateLimitEnabled = b
			return true
		}
		return false
	},
	"rate_limit_rps": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.RateLimitRPS = i
			return true
		}
		return false
	},
	"rate_limit_burst": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.RateLimitBurst = i
			return true
		}
		return false
	},
	"header_passthrough": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.HeaderPassThrough = b
			return true
		}
		return false
	},
	"fake_streaming_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.FakeStreamingEnabled = b
			return true
		}
		return false
	},
	"fake_streaming_chunk_size": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.FakeStreamingChunkSize = i
			return true
		}
		return false
	},
	"fake_streaming_delay_ms": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.FakeStreamingDelayMs = i
			return true
		}
		return false
	},
	"usage_reset_interval_hours": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.UsageResetIntervalHours = i
			return true
		}
		return false
	},
	"usage_reset_timezone": func(fc *FileConfig, v interface{}) bool {
		if s, ok := v.(string); ok {
			fc.UsageResetTimezone = s
			return true
		}
		return false
	},
	"usage_reset_hour_local": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.UsageResetHourLocal = i
			return true
		}
		return false
	},
	"auto_ban_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.AutoBanEnabled = b
			return true
		}
		return false
	},
	"auto_ban_429_threshold": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoBan429Threshold = i
			return true
		}
		return false
	},
	"auto_ban_403_threshold": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoBan403Threshold = i
			return true
		}
		return false
	},
	"auto_ban_401_threshold": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoBan401Threshold = i
			return true
		}
		return false
	},
	"auto_ban_5xx_threshold": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoBan5xxThreshold = i
			return true
		}
		return false
	},
	"auto_ban_consecutive_fails": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoBanConsecutiveFails = i
			return true
		}
		return false
	},
	"auto_recovery_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.AutoRecoveryEnabled = b
			return true
		}
		return false
	},
	"auto_recovery_interval_min": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoRecoveryIntervalMin = i
			return true
		}
		return false
	},
	"management_key_hash": func(fc *FileConfig, v interface{}) bool {
		if s, ok := v.(string); ok {
			fc.ManagementKeyHash = s
			return true
		}
		return false
	},
	"openai_images_include_mime": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.OpenAIImagesIncludeMime = b
			return true
		}
		return false
	},
	"tool_args_delta_chunk": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.ToolArgsDeltaChunk = i
			return true
		}
		return false
	},
	"sanitizer_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.SanitizerEnabled = b
			return true
		}
		return false
	},
	"sanitizer_patterns": func(fc *FileConfig, v interface{}) bool {
		switch vv := v.(type) {
		case []string:
			fc.SanitizerPatterns = vv
			return true
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
			fc.SanitizerPatterns = out
			return true
		case string:
			s := strings.TrimSpace(vv)
			if s == "" {
				fc.SanitizerPatterns = nil
			} else {
				fc.SanitizerPatterns = []string{s}
			}
			return true
		}
		return false
	},
	"preferred_base_models": func(fc *FileConfig, v interface{}) bool {
		if ss, ok := asStringSlice(v); ok {
			fc.PreferredBaseModels = ss
			return true
		}
		return false
	},
	"auto_probe_enabled": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.AutoProbeEnabled = b
			return true
		}
		return false
	},
	"auto_probe_hour_utc": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoProbeHourUTC = i
			return true
		}
		return false
	},
	"auto_probe_model": func(fc *FileConfig, v interface{}) bool {
		if s, ok := v.(string); ok {
			fc.AutoProbeModel = s
			return true
		}
		return false
	},
	"auto_probe_timeout_sec": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.AutoProbeTimeoutSec = i
			return true
		}
		return false
	},
	// Routing state persistence
	"persist_routing_state": func(fc *FileConfig, v interface{}) bool {
		if b, ok := v.(bool); ok {
			fc.PersistRoutingState = b
			return true
		}
		return false
	},
	"routing_persist_interval_sec": func(fc *FileConfig, v interface{}) bool {
		if i, ok := v.(int); ok {
			fc.RoutingPersistIntervalSec = i
			return true
		}
		return false
	},
}

func asStringSlice(v interface{}) ([]string, bool) {
	if ss, ok := v.([]string); ok {
		return ss, true
	}
	if vs, ok := v.([]interface{}); ok {
		out := make([]string, 0, len(vs))
		for _, it := range vs {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out, true
	}
	return nil, false
}

// applyFileConfigUpdate 按表驱动应用单个键的更新。
func applyFileConfigUpdate(fc *FileConfig, key string, value interface{}) bool {
	if fc == nil {
		return false
	}
	if fn, ok := fileUpdateSetters[key]; ok && fn != nil {
		return fn(fc, value)
	}
	return false
}

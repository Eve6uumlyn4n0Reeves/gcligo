package config

// RegexReplacement represents a regex pattern replacement rule
type RegexReplacement struct {
	Name        string `yaml:"name" json:"name"`               // Rule name for identification
	Pattern     string `yaml:"pattern" json:"pattern"`         // Regex pattern to match
	Replacement string `yaml:"replacement" json:"replacement"` // Replacement text
	Enabled     bool   `yaml:"enabled" json:"enabled"`         // Whether this rule is enabled
}

// FileConfig represents the configuration loaded from file
type FileConfig struct {
	// Server settings
	Port       int    `yaml:"port" json:"port"`
	OpenAIPort int    `yaml:"openai_port" json:"openai_port"`
	GeminiPort int    `yaml:"gemini_port" json:"gemini_port"`
	Debug      bool   `yaml:"debug" json:"debug"`
	LogFile    string `yaml:"log_file" json:"log_file"`
	RunProfile string `yaml:"run_profile" json:"run_profile"`

	// Auth settings
	AuthDir                  string   `yaml:"auth_dir" json:"auth_dir"`
	APIKeys                  []string `yaml:"api_keys" json:"api_keys"`
	OpenAIKey                string   `yaml:"openai_key" json:"openai_key"`
	GeminiKey                string   `yaml:"gemini_key" json:"gemini_key"`
	ManagementKey            string   `yaml:"management_key" json:"management_key"`
	ManagementKeyHash        string   `yaml:"management_key_hash" json:"management_key_hash"`
	ManagementAllowRemote    bool     `yaml:"management_allow_remote" json:"management_allow_remote"`
	ManagementRemoteTTlHours int      `yaml:"management_remote_ttl_hours" json:"management_remote_ttl_hours"`
	ManagementRemoteAllowIPs []string `yaml:"management_remote_allow_ips" json:"management_remote_allow_ips"`
	WebAdminEnabled          bool     `yaml:"web_admin_enabled" json:"web_admin_enabled"`
	BasePath                 string   `yaml:"base_path" json:"base_path"`
	StorageBackend           string   `yaml:"storage_backend" json:"storage_backend"`
	StorageBaseDir           string   `yaml:"storage_base_dir" json:"storage_base_dir"`
	RedisAddr                string   `yaml:"redis_addr" json:"redis_addr"`
	RedisPassword            string   `yaml:"redis_password" json:"redis_password"`
	RedisDB                  int      `yaml:"redis_db" json:"redis_db"`
	RedisPrefix              string   `yaml:"redis_prefix" json:"redis_prefix"`
	MongoDBURI               string   `yaml:"mongodb_uri" json:"mongodb_uri"`
	MongoDatabase            string   `yaml:"mongodb_database" json:"mongodb_database"`
	PostgresDSN              string   `yaml:"postgres_dsn" json:"postgres_dsn"`
	GitRemoteURL             string   `yaml:"git_remote_url" json:"git_remote_url"`
	GitBranch                string   `yaml:"git_branch" json:"git_branch"`
	GitUsername              string   `yaml:"git_username" json:"git_username"`
	GitPassword              string   `yaml:"git_password" json:"git_password"`
	GitAuthorName            string   `yaml:"git_author_name" json:"git_author_name"`
	GitAuthorEmail           string   `yaml:"git_author_email" json:"git_author_email"`

	// Upstream settings
	CodeAssistEndpoint string `yaml:"code_assist_endpoint" json:"code_assist_endpoint"`
	GoogleBearerToken  string `yaml:"google_bearer_token" json:"google_bearer_token"`
	GoogleProjectID    string `yaml:"google_project_id" json:"google_project_id"`
	ProxyURL           string `yaml:"proxy_url" json:"proxy_url"`
	OAuthClientID      string `yaml:"oauth_client_id" json:"oauth_client_id"`
	OAuthClientSecret  string `yaml:"oauth_client_secret" json:"oauth_client_secret"`
	OAuthRedirectURL   string `yaml:"oauth_redirect_url" json:"oauth_redirect_url"`

	// Behavior settings
	CallsPerRotation        int      `yaml:"calls_per_rotation" json:"calls_per_rotation"`
	RetryEnabled            bool     `yaml:"retry_enabled" json:"retry_enabled"`
	RetryMax                int      `yaml:"retry_max" json:"retry_max"`
	RetryIntervalSec        int      `yaml:"retry_interval_sec" json:"retry_interval_sec"`
	RetryMaxIntervalSec     int      `yaml:"retry_max_interval_sec" json:"retry_max_interval_sec"`
	RetryOn5xx              bool     `yaml:"retry_on_5xx" json:"retry_on_5xx"`
	RetryOnNetworkError     bool     `yaml:"retry_on_network_error" json:"retry_on_network_error"`
	AntiTruncationMax       int      `yaml:"anti_truncation_max" json:"anti_truncation_max"`
	AntiTruncationEnabled   bool     `yaml:"anti_truncation_enabled" json:"anti_truncation_enabled"`
	RequestLog              bool     `yaml:"request_log" json:"request_log"`
	DisabledModels          []string `yaml:"disabled_models" json:"disabled_models"`
	UsageResetIntervalHours int      `yaml:"usage_reset_interval_hours" json:"usage_reset_interval_hours"`
	UsageResetTimezone      string   `yaml:"usage_reset_timezone" json:"usage_reset_timezone"`
	UsageResetHourLocal     int      `yaml:"usage_reset_hour_local" json:"usage_reset_hour_local"`
	CompatibilityMode       bool     `yaml:"compatibility_mode" json:"compatibility_mode"` // Convert system messages to user messages
	AutoBanEnabled          bool     `yaml:"auto_ban_enabled" json:"auto_ban_enabled"`
	AutoBan429Threshold     int      `yaml:"auto_ban_429_threshold" json:"auto_ban_429_threshold"`
	AutoBan403Threshold     int      `yaml:"auto_ban_403_threshold" json:"auto_ban_403_threshold"`
	AutoBan401Threshold     int      `yaml:"auto_ban_401_threshold" json:"auto_ban_401_threshold"`
	AutoBan5xxThreshold     int      `yaml:"auto_ban_5xx_threshold" json:"auto_ban_5xx_threshold"`
	AutoBanConsecutiveFails int      `yaml:"auto_ban_consecutive_fails" json:"auto_ban_consecutive_fails"`
	AutoRecoveryEnabled     bool     `yaml:"auto_recovery_enabled" json:"auto_recovery_enabled"`
	AutoRecoveryIntervalMin int      `yaml:"auto_recovery_interval_min" json:"auto_recovery_interval_min"`

	// Routing state persistence
	PersistRoutingState       bool `yaml:"persist_routing_state" json:"persist_routing_state"`
	RoutingPersistIntervalSec int  `yaml:"routing_persist_interval_sec" json:"routing_persist_interval_sec"`

	// Feature toggles
	OpenAIImagesIncludeMime bool                `yaml:"openai_images_include_mime" json:"openai_images_include_mime"`
	ToolArgsDeltaChunk      int                 `yaml:"tool_args_delta_chunk" json:"tool_args_delta_chunk"`
	SanitizerEnabled        bool                `yaml:"sanitizer_enabled" json:"sanitizer_enabled"`
	SanitizerPatterns       []string            `yaml:"sanitizer_patterns" json:"sanitizer_patterns"`
	PreferredBaseModels     []string            `yaml:"preferred_base_models" json:"preferred_base_models"`
	RegexReplacements       []RegexReplacement  `yaml:"regex_replacements" json:"regex_replacements"`

	// Fake streaming
	FakeStreamingEnabled   bool `yaml:"fake_streaming_enabled" json:"fake_streaming_enabled"`
	FakeStreamingChunkSize int  `yaml:"fake_streaming_chunk_size" json:"fake_streaming_chunk_size"`
	FakeStreamingDelayMs   int  `yaml:"fake_streaming_delay_ms" json:"fake_streaming_delay_ms"`
	AutoImagePlaceholder   bool `yaml:"auto_image_placeholder" json:"auto_image_placeholder"`

	// Transport settings
	DialTimeoutSec           int `yaml:"dial_timeout_sec" json:"dial_timeout_sec"`
	TLSHandshakeTimeoutSec   int `yaml:"tls_handshake_timeout_sec" json:"tls_handshake_timeout_sec"`
	ResponseHeaderTimeoutSec int `yaml:"response_header_timeout_sec" json:"response_header_timeout_sec"`
	ExpectContinueTimeoutSec int `yaml:"expect_continue_timeout_sec" json:"expect_continue_timeout_sec"`

	// Rate limiting
	RateLimitEnabled bool `yaml:"rate_limit_enabled" json:"rate_limit_enabled"`
	RateLimitRPS     int  `yaml:"rate_limit_rps" json:"rate_limit_rps"`
	RateLimitBurst   int  `yaml:"rate_limit_burst" json:"rate_limit_burst"`

	// Upstream header behavior
	HeaderPassThrough bool `yaml:"header_passthrough" json:"header_passthrough"`

	// Auto probe (liveness)
	AutoProbeEnabled             bool   `yaml:"auto_probe_enabled" json:"auto_probe_enabled"`
	AutoProbeHourUTC             int    `yaml:"auto_probe_hour_utc" json:"auto_probe_hour_utc"`
	AutoProbeModel               string `yaml:"auto_probe_model" json:"auto_probe_model"`
	AutoProbeTimeoutSec          int    `yaml:"auto_probe_timeout_sec" json:"auto_probe_timeout_sec"`
	AutoProbeDisableThresholdPct int    `yaml:"auto_probe_disable_threshold_pct" json:"auto_probe_disable_threshold_pct"`

	// Environment credential support
	AutoLoadEnvCreds bool `yaml:"auto_load_env_creds" json:"auto_load_env_creds"`
}

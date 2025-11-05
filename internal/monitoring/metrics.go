package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP请求指标（兼容 middleware 原有标签维度）
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"server", "method", "path", "status_class"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gcli2api_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"server", "method", "path", "status_class"},
	)

	// HTTP 并发请求数
	HTTPInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_http_inflight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// 凭证相关指标
	CredentialRotationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_credential_rotations_total",
			Help: "Total number of credential rotations",
		},
		[]string{"credential"},
	)

	CredentialErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_credential_errors_total",
			Help: "Total number of credential errors",
		},
		[]string{"credential", "error_code"},
	)

	CredentialRefreshes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_credential_refreshes_total",
			Help: "Total number of credential token refreshes",
		},
		[]string{"credential", "status"},
	)

	// 上游API调用指标（兼容 middleware 原有标签维度）
	UpstreamRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_upstream_requests_total",
			Help: "Total number of upstream API requests",
		},
		[]string{"provider", "status_class"},
	)

	UpstreamRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gcli2api_upstream_request_duration_seconds",
			Help:    "Upstream API request latency in seconds",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"provider"},
	)

	UpstreamRequestDurationByServer = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gcli2api_upstream_request_duration_server_seconds",
			Help:    "Upstream API request latency by server in seconds",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
		[]string{"provider", "server"},
	)

	UpstreamErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_upstream_errors_total",
			Help: "Total number of upstream errors by reason",
		},
		[]string{"provider", "reason"},
	)

	UpstreamRetryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_upstream_retry_attempts_total",
			Help: "Total number of upstream retry attempts",
		},
		[]string{"provider", "outcome"},
	)

	UpstreamModelRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_upstream_model_requests_total",
			Help: "Total number of upstream requests by model",
		},
		[]string{"provider", "model", "status_class"},
	)

	// 流式传输指标（兼容 middleware SSE 指标）
	SSELinesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_sse_lines_total",
			Help: "Total number of SSE lines sent",
		},
		[]string{"server", "path"},
	)

	SSEDisconnectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_sse_disconnects_total",
			Help: "Total number of SSE disconnects by reason",
		},
		[]string{"server", "path", "reason"},
	)

	ToolCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_tool_calls_total",
			Help: "Total number of tool calls",
		},
		[]string{"server", "path"},
	)

	AntiTruncationAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_anti_truncation_attempts_total",
			Help: "Total number of anti-truncation continuation attempts",
		},
		[]string{"server", "path"},
	)

	ModelFallbacksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_model_fallbacks_total",
			Help: "Total number of model fallback hits",
		},
		[]string{"server", "path", "from_model", "to_model"},
	)

	ThinkingRemovedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_thinking_removed_total",
			Help: "Total number of thinking config removals",
		},
		[]string{"server", "path", "model"},
	)

	ManagementAccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_management_access_total",
			Help: "Total number of management access decisions",
		},
		[]string{"route", "result", "source"},
	)

	RateLimitKeysGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_ratelimit_keys",
			Help: "Current number of per-key rate limiters",
		},
	)

	RateLimitSweepsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gcli2api_ratelimit_sweeps_total",
			Help: "Total number of rate limiter TTL cache sweeps",
		},
	)

	// Assembly / routing operations
	AssemblyOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_assembly_operations_total",
			Help: "Total number of assembly/routing administrative operations",
		},
		[]string{"action", "status", "actor"},
	)

	// 系统指标
	ActiveCredentials = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_active_credentials",
			Help: "Number of active credentials",
		},
	)

	DisabledCredentials = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_disabled_credentials",
			Help: "Number of disabled credentials",
		},
	)

	// Token使用指标
	TokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_tokens_used_total",
			Help: "Total number of tokens used",
		},
		[]string{"model", "type"}, // type: prompt, completion, total
	)

	// 自动探活指标
	AutoProbeRunsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_auto_probe_runs_total",
			Help: "Total number of credential probe runs",
		},
		[]string{"source", "status", "model"}, // source: auto/manual, status: all_ok/partial/all_failed/empty/error
	)

	AutoProbeDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gcli2api_auto_probe_duration_seconds",
			Help:    "Credential probe latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source", "model"},
	)

	AutoProbeSuccessRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gcli2api_auto_probe_success_ratio",
			Help: "Success ratio of last credential probe run",
		},
		[]string{"source", "model"},
	)

	AutoProbeCredentialCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gcli2api_auto_probe_target_credentials",
			Help: "Number of credentials evaluated in last probe run",
		},
		[]string{"source", "model"},
	)

	AutoProbeLastSuccess = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gcli2api_auto_probe_last_success_unix",
			Help: "Unix timestamp of the last successful credential probe run",
		},
		[]string{"source", "model"},
	)

	// 上游同步指标
	UpstreamDiscoveryCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gcli2api_upstream_discovery_cache_hits_total",
			Help: "Total number of cache hits when serving upstream discovery data",
		},
	)

	UpstreamDiscoveryFetchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_upstream_discovery_fetch_total",
			Help: "Total number of upstream discovery refresh attempts",
		},
		[]string{"result"}, // result: success/no_credentials/empty/error/no_credential_manager
	)

	UpstreamDiscoveryFetchDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gcli2api_upstream_discovery_fetch_duration_seconds",
			Help:    "Duration of upstream discovery refresh attempts in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	UpstreamDiscoveryBases = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_upstream_discovery_known_bases",
			Help: "Number of base models currently cached from upstream",
		},
	)

	UpstreamDiscoveryCacheExpiry = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_upstream_discovery_cache_expires_unix",
			Help: "Unix timestamp when cached upstream discovery data expires",
		},
	)

	UpstreamDiscoveryLastSuccess = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_upstream_discovery_last_success_unix",
			Help: "Unix timestamp of the last successful upstream discovery refresh",
		},
	)

	// 路由策略指标
	RoutingStickyHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_routing_sticky_hits_total",
			Help: "Total number of sticky routing hits",
		},
		[]string{"source"}, // source: session|auth
	)

	RoutingCooldownEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gcli2api_routing_cooldown_events_total",
			Help: "Total number of cooldown set/extend events",
		},
		[]string{"status"},
	)

	RoutingStickySize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_routing_sticky_size",
			Help: "Current number of sticky routing entries",
		},
	)

	RoutingCooldownSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "gcli2api_routing_cooldown_size",
			Help: "Current number of cooldown entries",
		},
	)

	// Cooldown remaining seconds histogram (observed on snapshots)
	RoutingCooldownRemainingSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "gcli2api_routing_cooldown_remaining_seconds",
			Help:    "Distribution of remaining cooldown time per credential (seconds)",
			Buckets: []float64{0, 1, 2, 5, 10, 20, 30, 60, 120, 300, 600},
		},
	)
)

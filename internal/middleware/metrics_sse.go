package middleware

import (
	"gcli2api-go/internal/monitoring"
)

// RecordSSELines adds to the SSE lines counter for a server/path.
func RecordSSELines(server, path string, n int) {
	if n <= 0 {
		return
	}
	monitoring.SSELinesTotal.WithLabelValues(server, path).Add(float64(n))
}

// RecordToolCalls adds to the tool calls counter for a server/path.
func RecordToolCalls(server, path string, n int) {
	if n <= 0 {
		return
	}
	monitoring.ToolCallsTotal.WithLabelValues(server, path).Add(float64(n))
}

// RecordSSEClose increments an SSE disconnect reason counter for a server/path/reason.
func RecordSSEClose(server, path, reason string) {
	if reason == "" {
		reason = "other"
	}
	monitoring.SSEDisconnectsTotal.WithLabelValues(server, path, reason).Inc()
}

// RecordFallback records a model fallback hit from->to for this route
func RecordFallback(server, path, from, to string) {
	if from == to || to == "" {
		return
	}
	monitoring.ModelFallbacksTotal.WithLabelValues(server, path, from, to).Inc()
}

// RecordThinkingRemoved increments a counter when thinkingConfig is stripped for a model
func RecordThinkingRemoved(server, path, model string) {
	if model == "" {
		return
	}
	monitoring.ThinkingRemovedTotal.WithLabelValues(server, path, model).Inc()
}

// RecordAntiTruncAttempt adds anti-truncation continuation attempts for this route
func RecordAntiTruncAttempt(server, path string, n int) {
	if n <= 0 {
		return
	}
	monitoring.AntiTruncationAttemptsTotal.WithLabelValues(server, path).Add(float64(n))
}

// RecordManagementAccess tracks allow/deny decisions for management guard.
func RecordManagementAccess(route, result, source string) {
	if route == "" {
		route = "/"
	}
	if result == "" {
		result = "unknown"
	}
	if source == "" {
		source = "unknown"
	}
	monitoring.ManagementAccessTotal.WithLabelValues(route, result, source).Inc()
}

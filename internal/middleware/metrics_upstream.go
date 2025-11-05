package middleware

import (
	"math"
	"time"

	"gcli2api-go/internal/monitoring"
)

// RecordUpstream records upstream request duration and status classification with provider label.
func RecordUpstream(provider string, dur time.Duration, status int, networkErr bool) {
	cls := statusClass(status)
	if networkErr {
		cls = "network_error"
	}
	durSec := dur.Seconds()
	if math.IsNaN(durSec) || math.IsInf(durSec, 0) {
		durSec = 0
	}
	monitoring.UpstreamRequestsTotal.WithLabelValues(provider, cls).Inc()
	monitoring.UpstreamRequestDuration.WithLabelValues(provider).Observe(durSec)
}

// RecordUpstreamWithServer is like RecordUpstream, but also tags histograms with server label.
func RecordUpstreamWithServer(provider, server string, dur time.Duration, status int, networkErr bool) {
	cls := statusClass(status)
	if networkErr {
		cls = "network_error"
	}
	durSec := dur.Seconds()
	if math.IsNaN(durSec) || math.IsInf(durSec, 0) {
		durSec = 0
	}
	monitoring.UpstreamRequestsTotal.WithLabelValues(provider, cls).Inc()
	monitoring.UpstreamRequestDuration.WithLabelValues(provider).Observe(durSec)
	monitoring.UpstreamRequestDurationByServer.WithLabelValues(provider, server).Observe(durSec)
}

// RecordUpstreamRetry adds retry attempt counts (attempts beyond the first) by provider/outcome.
func RecordUpstreamRetry(provider string, attempts int, success bool) {
	if attempts <= 0 {
		return
	}
	outcome := "error"
	if success {
		outcome = "success"
	}
	monitoring.UpstreamRetryAttempts.WithLabelValues(provider, outcome).Add(float64(attempts))
}

// RecordUpstreamError increments upstream error by reason
func RecordUpstreamError(provider, reason string) {
	if reason == "" {
		reason = "other"
	}
	monitoring.UpstreamErrors.WithLabelValues(provider, reason).Inc()
}

// RecordUpstreamModel increments per-model upstream counters by provider/model/status class.
func RecordUpstreamModel(provider, model string, status int, networkErr bool) {
	if model == "" {
		model = "unknown"
	}
	cls := statusClass(status)
	if networkErr {
		cls = "network_error"
	}
	monitoring.UpstreamModelRequests.WithLabelValues(provider, model, cls).Inc()
}

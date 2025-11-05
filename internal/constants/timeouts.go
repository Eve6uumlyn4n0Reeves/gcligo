package constants

import "time"

const (
	// UpstreamStreamTimeout enforces max duration for streaming requests.
	UpstreamStreamTimeout = 180 * time.Second
	// UpstreamGenerateTimeout enforces max duration for non-stream requests.
	UpstreamGenerateTimeout = 180 * time.Second
	// CredentialRefreshInterval controls how frequently credentials auto-refresh.
	CredentialRefreshInterval = 5 * time.Minute
	// ServerShutdownTimeout bounds graceful HTTP server shutdown.
	ServerShutdownTimeout = 30 * time.Second
	// ServerGracefulWait defines post-shutdown wait window for cleanup.
	ServerGracefulWait = 2 * time.Second
)

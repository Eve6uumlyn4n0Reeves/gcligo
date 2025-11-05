package common

import (
	"context"

	"gcli2api-go/internal/constants"
)

// WithUpstreamTimeout returns a context with standard upstream timeouts.
func WithUpstreamTimeout(parent context.Context, stream bool) (context.Context, context.CancelFunc) {
	timeout := constants.UpstreamGenerateTimeout
	if stream {
		timeout = constants.UpstreamStreamTimeout
	}
	return context.WithTimeout(parent, timeout)
}

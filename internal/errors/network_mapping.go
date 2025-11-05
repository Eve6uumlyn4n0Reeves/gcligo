package errors

import (
	"net/http"
	"strings"
)

// MapNetworkError maps network errors to standardized APIError objects.
func MapNetworkError(err error) *APIError {
	errMsg := err.Error()
	msg := "Network error: " + errMsg

	switch {
	case strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded"):
		return New(http.StatusGatewayTimeout, "timeout", "timeout_error", "Request timeout: "+errMsg)
	case strings.Contains(errMsg, "connection refused"):
		return New(http.StatusBadGateway, "connection_error", "server_error", "Connection refused: "+errMsg)
	case strings.Contains(errMsg, "EOF") || strings.Contains(errMsg, "connection reset"):
		return New(http.StatusBadGateway, "connection_error", "server_error", "Connection error: "+errMsg)
	case strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "name resolution"):
		return New(http.StatusBadGateway, "dns_error", "server_error", "DNS resolution error: "+errMsg)
	case strings.Contains(errMsg, "certificate") || strings.Contains(errMsg, "tls"):
		return New(http.StatusBadGateway, "tls_error", "server_error", "TLS/Certificate error: "+errMsg)
	case strings.Contains(errMsg, "context canceled"):
		return New(http.StatusRequestTimeout, "request_canceled", "timeout_error", "Request was canceled: "+errMsg)
	default:
		return New(http.StatusBadGateway, "network_error", "server_error", msg)
	}
}

package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// MapHTTPError maps HTTP status codes and upstream payloads to standardized errors.
func MapHTTPError(statusCode int, upstreamBody []byte) *APIError {
	upstreamMsg := extractUpstreamMessage(upstreamBody)

	switch statusCode {
	case http.StatusBadRequest:
		return New(statusCode, "invalid_request_error", "invalid_request_error", firstNonEmpty(upstreamMsg, "Invalid request"))
	case http.StatusUnauthorized:
		return New(statusCode, "invalid_api_key", "authentication_error", firstNonEmpty(upstreamMsg, "Invalid authentication"))
	case http.StatusForbidden:
		return New(statusCode, "permission_denied", "permission_error", firstNonEmpty(upstreamMsg, "Permission denied"))
	case http.StatusNotFound:
		return New(statusCode, "not_found", "invalid_request_error", firstNonEmpty(upstreamMsg, "Resource not found"))
	case http.StatusTooManyRequests:
		return New(statusCode, "rate_limit_exceeded", "rate_limit_error", firstNonEmpty(upstreamMsg, "Rate limit exceeded"))
	case http.StatusInternalServerError:
		return New(statusCode, "server_error", "server_error", firstNonEmpty(upstreamMsg, "Internal server error"))
	case http.StatusBadGateway:
		return New(statusCode, "bad_gateway", "server_error", firstNonEmpty(upstreamMsg, "Bad gateway"))
	case http.StatusServiceUnavailable:
		return New(statusCode, "service_unavailable", "server_error", firstNonEmpty(upstreamMsg, "Service temporarily unavailable"))
	case http.StatusGatewayTimeout:
		return New(statusCode, "timeout", "timeout_error", firstNonEmpty(upstreamMsg, "Request timeout"))
	default:
		return New(statusCode, "unknown_error", "server_error", firstNonEmpty(upstreamMsg, fmt.Sprintf("HTTP %d error", statusCode)))
	}
}

func extractUpstreamMessage(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var jsonErr map[string]interface{}
	if err := json.Unmarshal(body, &jsonErr); err == nil {
		if errObj, ok := jsonErr["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				return msg
			}
		}
	}
	msg := string(body)
	if len(msg) > 200 {
		return msg[:200] + "..."
	}
	return msg
}

func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

package errors

import (
	"encoding/json"
	"net/http"
)

func (e *APIError) ToJSON(format ErrorFormat) ([]byte, error) {
	switch format {
	case FormatOpenAI:
		return e.toOpenAIJSON()
	case FormatGemini:
		return e.toGeminiJSON()
	default:
		return e.toOpenAIJSON()
	}
}

func (e *APIError) toOpenAIJSON() ([]byte, error) {
	errObj := OpenAIError{}
	errObj.Error.Message = e.Message
	errObj.Error.Type = e.Type
	errObj.Error.Code = e.Code
	if e.Details != nil {
		errObj.Error.Details = e.Details
	}
	return json.Marshal(errObj)
}

func (e *APIError) toGeminiJSON() ([]byte, error) {
	errObj := GeminiError{}
	errObj.Error.Code = e.HTTPStatus
	errObj.Error.Message = e.Message
	errObj.Error.Status = e.toGeminiStatus()
	if e.Details != nil {
		errObj.Error.Details = e.Details
	}
	return json.Marshal(errObj)
}

func (e *APIError) toGeminiStatus() string {
	switch e.HTTPStatus {
	case http.StatusBadRequest:
		return "INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusTooManyRequests:
		return "RESOURCE_EXHAUSTED"
	case http.StatusInternalServerError:
		return "INTERNAL"
	case http.StatusServiceUnavailable:
		return "UNAVAILABLE"
	case http.StatusGatewayTimeout:
		return "DEADLINE_EXCEEDED"
	default:
		return "UNKNOWN"
	}
}

func New(httpStatus int, code, errType, message string) *APIError {
	return &APIError{HTTPStatus: httpStatus, Code: code, Type: errType, Message: message}
}

func (e *APIError) WithDetails(details map[string]interface{}) *APIError {
	e.Details = details
	return e
}

func (e *APIError) IsRetryable() bool {
	switch e.HTTPStatus {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusRequestTimeout:
		return true
	}
	switch e.Code {
	case "timeout", "connection_error", "network_error", "dns_error":
		return true
	}
	return false
}

func (e *APIError) GetRetryAfter() int {
	if e.Details != nil {
		if retryAfter, ok := e.Details["retry_after"].(int); ok {
			return retryAfter
		}
		if retryAfter, ok := e.Details["retry_after"].(float64); ok {
			return int(retryAfter)
		}
	}
	switch e.HTTPStatus {
	case http.StatusTooManyRequests:
		return 60
	case http.StatusServiceUnavailable:
		return 30
	case http.StatusBadGateway, http.StatusGatewayTimeout:
		return 15
	default:
		return 5
	}
}

func (e *APIError) IsCritical() bool {
	switch e.HTTPStatus {
	case http.StatusUnauthorized, http.StatusForbidden:
		return true
	}
	switch e.Code {
	case "invalid_api_key", "permission_denied":
		return true
	}
	return false
}

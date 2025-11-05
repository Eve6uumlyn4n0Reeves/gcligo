package logging

// ErrorKind normalizes error categories for logs/metrics.
// It maps HTTP status codes and presence of error to a short string label.
func ErrorKind(status int, hasErr bool) string {
	if hasErr && status == 0 {
		return "network_error"
	}
	switch {
	case status == 429:
		return "upstream_429"
	case status == 401:
		return "upstream_401"
	case status == 403:
		return "upstream_403"
	case status >= 500 && status < 600:
		return "upstream_5xx"
	case status >= 400 && status < 500:
		return "upstream_4xx"
	}
	if hasErr {
		return "error"
	}
	return "ok"
}

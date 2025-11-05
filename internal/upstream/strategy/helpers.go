package strategy

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

func stickyKeyFromHeaders(hdr http.Header) string {
	if hdr == nil {
		return ""
	}
	if v := strings.TrimSpace(hdr.Get("X-Session-ID")); v != "" {
		sum := sha256.Sum256([]byte(v))
		return hex.EncodeToString(sum[:])
	}
	auth := strings.TrimSpace(hdr.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[7:])
		if token != "" {
			sum := sha256.Sum256([]byte(token))
			return hex.EncodeToString(sum[:])
		}
	}
	return ""
}

func stickyKeyAndSourceFromHeaders(hdr http.Header) (string, string) {
	if hdr == nil {
		return "", ""
	}
	if v := strings.TrimSpace(hdr.Get("X-Session-ID")); v != "" {
		sum := sha256.Sum256([]byte(v))
		return hex.EncodeToString(sum[:]), "session"
	}
	auth := strings.TrimSpace(hdr.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		token := strings.TrimSpace(auth[7:])
		if token != "" {
			sum := sha256.Sum256([]byte(token))
			return hex.EncodeToString(sum[:]), "auth"
		}
	}
	return "", ""
}

func toStatusLabel(code int) string {
	switch {
	case code == 429:
		return "429"
	case code == 403:
		return "403"
	case code >= 500 && code <= 599:
		return "5xx"
	default:
		return "other"
	}
}

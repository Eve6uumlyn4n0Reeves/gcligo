package netutil

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ExtractClientIP returns the client IP using common proxy headers.
func ExtractClientIP(c *gin.Context) net.IP {
	if c == nil {
		return nil
	}
	return ExtractIPFromRequest(c.Request)
}

// ExtractIPFromRequest extracts an IP from HTTP headers or remote address.
func ExtractIPFromRequest(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		if len(parts) > 0 {
			if ip := net.ParseIP(strings.TrimSpace(parts[0])); ip != nil {
				return ip
			}
		}
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		if ip := net.ParseIP(strings.TrimSpace(xr)); ip != nil {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

// ClassifyClientSource categorizes the IP origin.
func ClassifyClientSource(ip net.IP) string {
	if ip == nil {
		return "unknown"
	}
	if ip.IsLoopback() {
		return "loopback"
	}
	if IsDockerBridgeIP(ip) {
		return "docker_bridge"
	}
	if ip.IsPrivate() {
		return "private"
	}
	return "public"
}

// IsDockerBridgeIP detects Docker default bridge range.
func IsDockerBridgeIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 172 && ip4[1] == 17
	}
	return false
}

// IPString returns the textual representation or empty string.
func IPString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

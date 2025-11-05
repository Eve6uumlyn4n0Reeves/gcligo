package server

import (
	"net"
	"strings"
	"time"

	"gcli2api-go/internal/config"
	mw "gcli2api-go/internal/middleware"
	netx "gcli2api-go/internal/netutil"
	"github.com/gin-gonic/gin"
)

// managementReadOnlyGuard blocks write operations when read-only mode is enabled.
func managementReadOnlyGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case "GET", "HEAD", "OPTIONS":
			c.Next()
			return
		default:
			c.AbortWithStatusJSON(403, gin.H{"error": "management api is read-only"})
			return
		}
	}
}

// managementRemoteGuard enforces local-only access by default; when remote is allowed,
// it optionally restricts by IP/CIDR whitelist. Records decisions via metrics.
func managementRemoteGuard(routePrefix string, cfg *config.Config) gin.HandlerFunc {
	nets := parseIPNets(cfg.Security.ManagementRemoteAllowIPs)
	created := time.Now()
	return func(c *gin.Context) {
		// Prefer Gin's ClientIP (honors TrustedProxies). Avoid trusting XFF by default.
		cip := c.ClientIP()
		ip := net.ParseIP(strings.TrimSpace(cip))
		src := netx.ClassifyClientSource(ip)
		if src == "loopback" {
			mw.RecordManagementAccess(routePrefix, "allow", src)
			c.Next()
			return
		}
		if !cfg.Security.ManagementAllowRemote {
			mw.RecordManagementAccess(routePrefix, "deny", src)
			c.AbortWithStatusJSON(403, gin.H{"error": "remote management disabled"})
			return
		}
		if cfg.Security.ManagementRemoteTTlHours > 0 {
			if time.Since(created) >= time.Duration(cfg.Security.ManagementRemoteTTlHours)*time.Hour {
				mw.RecordManagementAccess(routePrefix, "deny", src)
				c.AbortWithStatusJSON(403, gin.H{"error": "remote management TTL expired"})
				return
			}
		}
		if len(nets) > 0 && !ipInNets(ip, nets) {
			mw.RecordManagementAccess(routePrefix, "deny", src)
			c.AbortWithStatusJSON(403, gin.H{"error": "ip not allowed for management"})
			return
		}
		mw.RecordManagementAccess(routePrefix, "allow", src)
		c.Next()
	}
}

func parseIPNets(list []string) []*net.IPNet {
	out := make([]*net.IPNet, 0)
	for _, s := range list {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ipnet, err := net.ParseCIDR(s); err == nil {
			out = append(out, ipnet)
			continue
		}
		if ip := net.ParseIP(s); ip != nil {
			var ipnet *net.IPNet
			if ip.To4() != nil {
				_, ipnet, _ = net.ParseCIDR(ip.String() + "/32")
			} else {
				_, ipnet, _ = net.ParseCIDR(ip.String() + "/128")
			}
			if ipnet != nil {
				out = append(out, ipnet)
			}
		}
	}
	return out
}

func ipInNets(ip net.IP, nets []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n != nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

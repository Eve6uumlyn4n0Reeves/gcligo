package server

import (
	"net"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
)

func TestParseIPNets(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected int
	}{
		{
			name:     "Empty list",
			input:    []string{},
			expected: 0,
		},
		{
			name:     "Single CIDR",
			input:    []string{"192.168.1.0/24"},
			expected: 1,
		},
		{
			name:     "Single IP",
			input:    []string{"192.168.1.1"},
			expected: 1,
		},
		{
			name:     "IPv6 CIDR",
			input:    []string{"2001:db8::/32"},
			expected: 1,
		},
		{
			name:     "IPv6 address",
			input:    []string{"2001:db8::1"},
			expected: 1,
		},
		{
			name:     "Mixed valid and invalid",
			input:    []string{"192.168.1.0/24", "invalid", "10.0.0.1"},
			expected: 2,
		},
		{
			name:     "Empty strings",
			input:    []string{"", "  ", "192.168.1.0/24"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPNets(tt.input)
			if len(result) != tt.expected {
				t.Errorf("parseIPNets() returned %d nets, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestIPInNets(t *testing.T) {
	_, net1, _ := net.ParseCIDR("192.168.1.0/24")
	_, net2, _ := net.ParseCIDR("10.0.0.0/8")
	nets := []*net.IPNet{net1, net2}

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IP in first network", "192.168.1.100", true},
		{"IP in second network", "10.1.2.3", true},
		{"IP not in any network", "172.16.0.1", false},
		{"Nil IP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}
			result := ipInNets(ip, nets)
			if result != tt.expected {
				t.Errorf("ipInNets(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}

	t.Run("Nil nets", func(t *testing.T) {
		ip := net.ParseIP("192.168.1.1")
		result := ipInNets(ip, nil)
		if result {
			t.Error("Expected false for nil nets")
		}
	})
}

func TestManagementReadOnlyGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	guard := managementReadOnlyGuard()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		shouldAbort    bool
	}{
		{"GET allowed", "GET", 200, false},
		{"HEAD allowed", "HEAD", 200, false},
		{"OPTIONS allowed", "OPTIONS", 200, false},
		{"POST blocked", "POST", 403, true},
		{"PUT blocked", "PUT", 403, true},
		{"DELETE blocked", "DELETE", 403, true},
		{"PATCH blocked", "PATCH", 403, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(tt.method, "/test", nil)

			guard(c)

			if tt.shouldAbort {
				if !c.IsAborted() {
					t.Error("Expected request to be aborted")
				}
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}
			} else {
				if c.IsAborted() {
					t.Error("Expected request not to be aborted")
				}
			}
		})
	}
}

func TestManagementRemoteGuard(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Loopback always allowed", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.ManagementAllowRemote = false
		cfg.SyncFromDomains()
		guard := managementRemoteGuard("/test", cfg)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "127.0.0.1:12345"

		guard(c)

		if c.IsAborted() {
			t.Error("Expected loopback to be allowed")
		}
	})

	t.Run("Remote denied when not allowed", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.ManagementAllowRemote = false
		cfg.SyncFromDomains()
		guard := managementRemoteGuard("/test", cfg)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		guard(c)

		if !c.IsAborted() {
			t.Error("Expected remote access to be denied")
		}
		if w.Code != 403 {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("Remote allowed when configured", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.ManagementAllowRemote = true
		cfg.SyncFromDomains()
		guard := managementRemoteGuard("/test", cfg)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		guard(c)

		if c.IsAborted() {
			t.Error("Expected remote access to be allowed")
		}
	})

	t.Run("IP whitelist enforcement", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Security.ManagementAllowRemote = true
		cfg.Security.ManagementRemoteAllowIPs = []string{"10.0.0.0/8"}
		cfg.SyncFromDomains()
		guard := managementRemoteGuard("/test", cfg)

		// IP in whitelist
		w1 := httptest.NewRecorder()
		c1, _ := gin.CreateTestContext(w1)
		c1.Request = httptest.NewRequest("GET", "/test", nil)
		c1.Request.RemoteAddr = "10.1.2.3:12345"

		guard(c1)

		if c1.IsAborted() {
			t.Error("Expected whitelisted IP to be allowed")
		}

		// IP not in whitelist
		w2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(w2)
		c2.Request = httptest.NewRequest("GET", "/test", nil)
		c2.Request.RemoteAddr = "192.168.1.100:12345"

		guard(c2)

		if !c2.IsAborted() {
			t.Error("Expected non-whitelisted IP to be denied")
		}
		if w2.Code != 403 {
			t.Errorf("Expected status 403, got %d", w2.Code)
		}
	})
}

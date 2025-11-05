package management

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gcli2api-go/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUpdateConfigApplies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Initialize global config manager
	_ = config.LoadWithFile("")
	cfg := config.Load()
	h := NewAdminAPIHandler(cfg, nil, nil, nil, nil)
	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	payload := map[string]any{"header_passthrough": true}
	b, _ := json.Marshal(payload)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PUT", "/routes/api/management/config", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify config manager reflects change
	cm := config.GetConfigManager()
	fc := cm.GetConfig()
	assert.True(t, fc.HeaderPassThrough)
}

func TestSessionLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		ManagementKey: "secret-key",
	}
	h := NewAdminAPIHandler(cfg, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/login", h.SessionLogin)
	body := map[string]any{"key": "secret-key", "ttl_hours": 1}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Token     string `json:"token"`
		SessionID string `json:"session_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Token == "" || resp.SessionID == "" {
		t.Fatalf("login response missing token/session_id: %v", rec.Body.String())
	}

	assert.True(t, h.ValidateToken(resp.Token), "token should be accepted immediately after login")

	logoutRouter := gin.New()
	logoutRouter.POST("/logout", h.SessionLogout)
	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+resp.Token)
	logoutRec := httptest.NewRecorder()
	logoutRouter.ServeHTTP(logoutRec, logoutReq)
	assert.Equal(t, http.StatusOK, logoutRec.Code)

	assert.False(t, h.ValidateToken(resp.Token), "token should be rejected after logout")
}

// Cookie Secure branch when behind HTTPS proxy: X-Forwarded-Proto=https should mark cookie Secure
func TestSessionCookieSecureWithForwardedProto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{ManagementKey: "secret-key"}
	h := NewAdminAPIHandler(cfg, nil, nil, nil, nil)

	r := gin.New()
	r.POST("/login", h.SessionLogin)

	payload := map[string]any{"key": "secret-key", "ttl_hours": 0.05}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// Simulate HTTPS via reverse proxy
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}
	// Check Set-Cookie has Secure
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "mgmt_session" {
			if !c.Secure {
				t.Fatalf("mgmt_session must be Secure when X-Forwarded-Proto=https")
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("mgmt_session cookie not set")
	}
}

package management

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gcli2api-go/internal/credential"
	oauth "gcli2api-go/internal/oauth"
	"github.com/gin-gonic/gin"
)

// GetFeatures returns feature flags
func (h *AdminAPIHandler) GetFeatures(c *gin.Context) {
	features := gin.H{
		"fake_streaming":  h.cfg.FakeStreamingEnabled,
		"anti_truncation": h.cfg.AntiTruncationMax > 0,
		"retry":           h.cfg.RetryEnabled,
		"rate_limit":      h.cfg.RateLimitEnabled,
		"request_logging": h.cfg.RequestLogEnabled,
		"pprof":           h.cfg.PprofEnabled,
	}
	c.JSON(http.StatusOK, gin.H{"features": features})
}

// UpdateFeature updates a feature flag
func (h *AdminAPIHandler) UpdateFeature(c *gin.Context) {
	feature := c.Param("feature")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	switch strings.ToLower(feature) {
	case "fake_streaming":
		h.cfg.FakeStreamingEnabled = req.Enabled
	case "request_logging":
		h.cfg.RequestLogEnabled = req.Enabled
	default:
		respondError(c, http.StatusBadRequest, "unsupported feature")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated", "feature": feature, "enabled": req.Enabled})
}

// GetOAuthStatus returns OAuth session status
func (h *AdminAPIHandler) GetOAuthStatus(c *gin.Context) {
	// Return OAuth status (placeholder)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// OnboardingStatus reports friendly status for a specific credential (token/project/email availability)
func (h *AdminAPIHandler) OnboardingStatus(c *gin.Context) {
	id := c.Query("credential_id")
	cred := findCredentialByID(h.credMgr, id)
	if cred == nil {
		respondError(c, http.StatusNotFound, "credential not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         cred.ID,
		"email":      cred.Email,
		"project_id": cred.ProjectID,
		"has_token":  strings.TrimSpace(cred.AccessToken) != "",
	})
}

// OnboardingEnableAPIs enables required Google APIs for the given credential's project and returns per-API results
func (h *AdminAPIHandler) OnboardingEnableAPIs(c *gin.Context) {
	var req struct {
		CredentialID string `json:"credential_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid json")
		return
	}
	cred := findCredentialByID(h.credMgr, req.CredentialID)
	if cred == nil {
		respondError(c, http.StatusNotFound, "credential not found")
		return
	}
	if strings.TrimSpace(cred.ProjectID) == "" {
		respondError(c, http.StatusBadRequest, "missing project_id in credential")
		return
	}
	if strings.TrimSpace(cred.AccessToken) == "" {
		respondError(c, http.StatusBadRequest, "missing access_token in credential")
		return
	}
	apis := []string{"generativelanguage.googleapis.com", "aiplatform.googleapis.com", "cloudresourcemanager.googleapis.com", "cloudaicompanion.googleapis.com"}
	results := make([]gin.H, 0, len(apis))
	pd := oauth.NewProjectDetector()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	for _, svc := range apis {
		err := pd.EnableAPI(ctx, cred.AccessToken, cred.ProjectID, svc)
		item := gin.H{"service": svc, "ok": err == nil}
		if err != nil {
			item["error"] = err.Error()
		}
		results = append(results, item)
	}
	c.JSON(http.StatusOK, gin.H{"project_id": cred.ProjectID, "results": results})
}

// findCredentialByID finds a credential by id; if id is empty and only one exists, returns it
func findCredentialByID(mgr *credential.Manager, id string) *credential.Credential {
	if mgr == nil {
		return nil
	}
	creds := mgr.GetAllCredentials()
	if id == "" {
		if len(creds) == 1 {
			return creds[0]
		}
		return nil
	}
	for _, c := range creds {
		if c.ID == id {
			return c
		}
	}
	return nil
}

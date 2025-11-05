package management

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ListCredentials returns all credentials (sanitized)
func (h *AdminAPIHandler) ListCredentials(c *gin.Context) {
	creds := h.credMgr.GetAllCredentials()

	sanitized := make([]gin.H, len(creds))
	for i, cred := range creds {
		score := cred.GetScore()
		successRate := float64(0)
		if cred.TotalRequests > 0 {
			successRate = float64(cred.SuccessCount) / float64(cred.TotalRequests)
		}
		sanitized[i] = gin.H{
			"id":                cred.ID,
			"filename":          cred.ID,
			"type":              cred.Type,
			"email":             cred.Email,
			"project_id":        cred.ProjectID,
			"disabled":          cred.Disabled,
			"auto_banned":       cred.AutoBanned,
			"banned_reason":     cred.BannedReason,
			"ban_until":         cred.BanUntil,
			"healthy":           cred.IsHealthy(),
			"score":             score,
			"health_score":      score,
			"failure_weight":    cred.FailureWeight,
			"total_requests":    cred.TotalRequests,
			"success_count":     cred.SuccessCount,
			"failure_count":     cred.FailureCount,
			"consecutive_fails": cred.ConsecutiveFails,
			"last_error_code":   cred.LastErrorCode,
			"success_rate":      successRate,
			"last_success":      cred.LastSuccess,
			"last_failure":      cred.LastFailure,
		}
	}

	c.JSON(http.StatusOK, gin.H{"credentials": sanitized})
}

// GetCredential returns a specific credential
func (h *AdminAPIHandler) GetCredential(c *gin.Context) {
	id := c.Param("id")

	creds := h.credMgr.GetAllCredentials()
	for _, cred := range creds {
		if cred.ID == id {
			score := cred.GetScore()
			successRate := float64(0)
			if cred.TotalRequests > 0 {
				successRate = float64(cred.SuccessCount) / float64(cred.TotalRequests)
			}
			c.JSON(http.StatusOK, gin.H{
				"id":                cred.ID,
				"filename":          cred.ID,
				"type":              cred.Type,
				"email":             cred.Email,
				"project_id":        cred.ProjectID,
				"disabled":          cred.Disabled,
				"auto_banned":       cred.AutoBanned,
				"banned_reason":     cred.BannedReason,
				"ban_until":         cred.BanUntil,
				"healthy":           cred.IsHealthy(),
				"score":             score,
				"health_score":      score,
				"failure_weight":    cred.FailureWeight,
				"total_requests":    cred.TotalRequests,
				"success_count":     cred.SuccessCount,
				"failure_count":     cred.FailureCount,
				"consecutive_fails": cred.ConsecutiveFails,
				"success_rate":      successRate,
				"last_error_code":   cred.LastErrorCode,
				"last_success":      cred.LastSuccess,
				"last_failure":      cred.LastFailure,
				"failure_reason":    cred.FailureReason,
			})
			return
		}
	}

	respondError(c, http.StatusNotFound, "Credential not found")
}

// DisableCredential disables a credential
func (h *AdminAPIHandler) DisableCredential(c *gin.Context) {
	id := c.Param("id")

	if err := h.credMgr.DisableCredential(id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	h.audit(c, "credential.disable", log.Fields{"id": id})
	c.JSON(http.StatusOK, gin.H{"message": "Credential disabled"})
}

// EnableCredential enables a credential
func (h *AdminAPIHandler) EnableCredential(c *gin.Context) {
	id := c.Param("id")

	if err := h.credMgr.EnableCredential(id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	h.audit(c, "credential.enable", log.Fields{"id": id})
	c.JSON(http.StatusOK, gin.H{"message": "Credential enabled"})
}

// ReloadCredentials reloads credentials from disk
func (h *AdminAPIHandler) ReloadCredentials(c *gin.Context) {
	if err := h.credMgr.LoadCredentials(); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	h.audit(c, "credential.reload", nil)
	c.JSON(http.StatusOK, gin.H{"message": "Credentials reloaded"})
}

// RecoverAllCredentials force recovers all auto-banned credentials
func (h *AdminAPIHandler) RecoverAllCredentials(c *gin.Context) {
	if h.credMgr == nil {
		respondError(c, http.StatusInternalServerError, "credential manager not configured")
		return
	}
	n := h.credMgr.ForceRecoverAll(c.Request.Context())
	h.audit(c, "credential.recover_all", log.Fields{"count": n})
	c.JSON(http.StatusOK, gin.H{"message": "recovered", "count": n})
}

// RecoverCredential force recovers a specific credential
func (h *AdminAPIHandler) RecoverCredential(c *gin.Context) {
	if h.credMgr == nil {
		respondError(c, http.StatusInternalServerError, "credential manager not configured")
		return
	}
	id := c.Param("id")
	if id == "" {
		respondError(c, http.StatusBadRequest, "missing id")
		return
	}
	if err := h.credMgr.ForceRecoverOne(c.Request.Context(), id); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	h.audit(c, "credential.recover_one", log.Fields{"id": id})
	c.JSON(http.StatusOK, gin.H{"message": "recovered", "id": id})
}

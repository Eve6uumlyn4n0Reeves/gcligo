package common

import (
	"net/http"
	"strings"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/oauth"
	upstream "gcli2api-go/internal/upstream"
	upgem "gcli2api-go/internal/upstream/gemini"
	"github.com/gin-gonic/gin"
)

// ShouldRefreshAhead decides whether a credential should be proactively refreshed
// based on access token expiry and configured ahead seconds.
func ShouldRefreshAhead(cfg *config.Config, c *credential.Credential) bool {
	if c == nil || c.Type != "oauth" {
		return false
	}
	if strings.TrimSpace(c.RefreshToken) == "" {
		return false
	}
	if strings.TrimSpace(c.AccessToken) == "" {
		return true
	}
	if c.ExpiresAt.IsZero() {
		return true
	}
	ahead := time.Duration(cfg.RefreshAheadSeconds) * time.Second
	if ahead <= 0 {
		ahead = 180 * time.Second
	}
	return time.Until(c.ExpiresAt) <= ahead
}

// UpstreamClientFor returns an upstream client bound to a specific credential.
// Caller string is used to tag the client for metrics/debugging.
func UpstreamClientFor(cfg *config.Config, cred *credential.Credential, caller string) *upgem.Client {
	if cred == nil {
		return upgem.New(cfg).WithCaller(caller)
	}
	oc := &oauth.Credentials{AccessToken: cred.AccessToken, ProjectID: cred.ProjectID}
	return upgem.NewWithCredential(cfg, oc).WithCaller(caller)
}

// ResultNotifier abstracts router-like components that consume credential results.
type ResultNotifier interface {
	OnResult(credID string, status int)
}

// MarkCredentialFailure records credential/router failure safely.
func MarkCredentialFailure(credMgr *credential.Manager, router ResultNotifier, cred *credential.Credential, reason string, status int) {
	if credMgr != nil && cred != nil {
		credMgr.MarkFailure(cred.ID, reason, status)
	}
	if router != nil && cred != nil {
		router.OnResult(cred.ID, status)
	}
}

// MarkCredentialSuccess records credential success and notifies router.
func MarkCredentialSuccess(credMgr *credential.Manager, router ResultNotifier, cred *credential.Credential, status int) {
	if credMgr != nil && cred != nil {
		credMgr.MarkSuccess(cred.ID)
	}
	if router != nil && cred != nil {
		router.OnResult(cred.ID, status)
	}
}

// HandleUpstreamErrorAbort centralizes upstream error propagation for HTTP handlers.
// Returns true if the error has been handled and the caller should stop processing.
func HandleUpstreamErrorAbort(c *gin.Context, resp *http.Response, err error, cred *credential.Credential, credMgr *credential.Manager, router ResultNotifier, failureReason string) bool {
	if err != nil {
		AbortWithError(c, http.StatusBadGateway, failureReason, err.Error())
		return true
	}
	if resp != nil && resp.StatusCode >= http.StatusBadRequest {
		body, _ := upstream.ReadAll(resp)
		if cred != nil {
			MarkCredentialFailure(credMgr, router, cred, failureReason, resp.StatusCode)
		}
		AbortWithUpstreamError(c, http.StatusBadGateway, failureReason, "upstream error", body)
		return true
	}
	return false
}

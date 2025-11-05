package strategy

import (
	"context"
	"time"

	"gcli2api-go/internal/credential"
)

// PrepareCredential 执行到期前预刷新（如需）。返回可能更新后的凭证。
func (s *Strategy) PrepareCredential(ctx context.Context, c *credential.Credential) *credential.Credential {
	if c == nil || s.credMgr == nil {
		return c
	}
	if s.shouldRefreshAhead(c) {
		if err := s.credMgr.RefreshCredential(ctx, c.ID); err == nil {
			s.onRefresh(c.ID)
			if fresh, ok := s.credMgr.GetCredentialByID(c.ID); ok && fresh != nil {
				return fresh
			}
		}
	}
	return c
}

// Compensate401 在收到 401 时尝试刷新同一凭证并返回刷新后的凭证。
func (s *Strategy) Compensate401(ctx context.Context, credID string) (*credential.Credential, bool) {
	if s.credMgr == nil || credID == "" {
		return nil, false
	}
	if err := s.credMgr.RefreshCredential(ctx, credID); err != nil {
		return nil, false
	}
	s.onRefresh(credID)
	if fresh, ok := s.credMgr.GetCredentialByID(credID); ok && fresh != nil {
		return fresh, true
	}
	return nil, false
}

func (s *Strategy) shouldRefreshAhead(c *credential.Credential) bool {
	if c == nil || c.Type != "oauth" {
		return false
	}
	if c.RefreshToken == "" {
		return false
	}
	if c.AccessToken == "" {
		return true
	}
	if c.ExpiresAt.IsZero() {
		return true
	}
	ahead := time.Duration(s.cfg.RefreshAheadSeconds) * time.Second
	if ahead <= 0 {
		ahead = 180 * time.Second
	}
	return time.Until(c.ExpiresAt) <= ahead
}

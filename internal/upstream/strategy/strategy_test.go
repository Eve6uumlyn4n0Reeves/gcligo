package strategy

import (
	"context"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"github.com/stretchr/testify/require"
)

type staticSource struct {
	creds []*credential.Credential
}

func (s *staticSource) Name() string { return "static" }

func (s *staticSource) Load(ctx context.Context) ([]*credential.Credential, error) {
	out := make([]*credential.Credential, len(s.creds))
	for i, c := range s.creds {
		if c == nil {
			continue
		}
		out[i] = c.Clone()
	}
	return out, nil
}

func makeCred(id string, opts func(*credential.Credential)) *credential.Credential {
	c := &credential.Credential{
		ID:            id,
		Type:          "api_key",
		AccessToken:   "token-" + id,
		TotalRequests: 1,
		SuccessCount:  1,
		LastSuccess:   time.Now(),
		ErrorCodeCounts: map[int]int{
			200: 1,
		},
	}
	if opts != nil {
		opts(c)
	}
	return c
}

func newTestStrategy(t *testing.T, cfg *config.Config, creds ...*credential.Credential) (*Strategy, *credential.Manager) {
	t.Helper()
	source := &staticSource{creds: creds}
	mgr := credential.NewManager(credential.Options{
		Sources:             []credential.CredentialSource{source},
		RefreshAheadSeconds: 60,
	})
	require.NoError(t, mgr.LoadCredentials())

	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.StickyTTLSeconds == 0 {
		cfg.StickyTTLSeconds = 300
	}
	if cfg.RouterCooldownBaseMS == 0 {
		cfg.RouterCooldownBaseMS = 200
	}
	if cfg.RouterCooldownMaxMS == 0 {
		cfg.RouterCooldownMaxMS = 5_000
	}

	strat := NewStrategy(cfg, mgr, func(string) {})
	return strat, mgr
}

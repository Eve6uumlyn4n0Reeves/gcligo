package strategy

import (
	"context"
	"net/http"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"github.com/stretchr/testify/require"
)

func TestStrategyPickWeightedChoosesHighestScore(t *testing.T) {
	cfg := &config.Config{}
	high := makeCred("cred-high", func(c *credential.Credential) {
		c.TotalRequests = 50
		c.SuccessCount = 45
		c.ConsecutiveFails = 0
		c.LastSuccess = time.Now()
	})
	low := makeCred("cred-low", func(c *credential.Credential) {
		c.TotalRequests = 50
		c.SuccessCount = 5
		c.ConsecutiveFails = 5
		c.DailyLimit = 100
		c.DailyUsage = 90
		c.LastFailure = time.Now()
	})

	strat, _ := newTestStrategy(t, cfg, high, low)
	cred := strat.Pick(context.Background(), http.Header{})
	require.NotNil(t, cred)
	require.Equal(t, "cred-high", cred.ID)
}

func TestStrategyPickStickyOverridesWeighted(t *testing.T) {
	cfg := &config.Config{}
	high := makeCred("cred-high", func(c *credential.Credential) {
		c.TotalRequests = 40
		c.SuccessCount = 35
	})
	low := makeCred("cred-low", func(c *credential.Credential) {
		c.TotalRequests = 20
		c.SuccessCount = 4
		c.DailyLimit = 100
		c.DailyUsage = 95
	})

	strat, _ := newTestStrategy(t, cfg, high, low)

	hdr := http.Header{}
	hdr.Set("X-Session-ID", "sticky-user")
	key := stickyKeyFromHeaders(hdr)
	require.NotEmpty(t, key)

	strat.setSticky(key, "cred-low", time.Minute)

	cred := strat.Pick(context.Background(), hdr)
	require.NotNil(t, cred)
	require.Equal(t, "cred-low", cred.ID, "sticky selection should override weighted score")
}

func TestStrategyPickStickySkipsCooldown(t *testing.T) {
	cfg := &config.Config{}
	high := makeCred("cred-high", func(c *credential.Credential) {
		c.TotalRequests = 30
		c.SuccessCount = 28
	})
	low := makeCred("cred-low", nil)

	strat, _ := newTestStrategy(t, cfg, high, low)

	hdr := http.Header{}
	hdr.Set("X-Session-ID", "cooldown-user")
	key := stickyKeyFromHeaders(hdr)
	strat.setSticky(key, "cred-high", time.Minute)
	strat.SetCooldown("cred-high", 1, time.Now().Add(time.Minute))

	cred := strat.Pick(context.Background(), hdr)
	require.NotNil(t, cred)
	require.Equal(t, "cred-low", cred.ID, "cooldown should force sticky fallback")
}

func TestStrategyPickReturnsNilWhenNoCandidates(t *testing.T) {
	strat, _ := newTestStrategy(t, &config.Config{})
	require.Nil(t, strat.Pick(context.Background(), http.Header{}))
}

func TestStrategyPickWithInfo(t *testing.T) {
	credA := makeCred("cred-a", nil)
	credB := makeCred("cred-b", func(c *credential.Credential) {
		c.TotalRequests = 10
		c.SuccessCount = 1
		c.ConsecutiveFails = 3
	})

	strat, _ := newTestStrategy(t, &config.Config{}, credA, credB)
	cred, log := strat.PickWithInfo(context.Background(), http.Header{})
	require.NotNil(t, cred)
	require.NotNil(t, log)
	require.Equal(t, cred.ID, log.CredID)
	require.NotEmpty(t, log.Reason)
}

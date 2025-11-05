package strategy

import (
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"github.com/stretchr/testify/require"
)

func TestStrategyCooldownOnResultSetsAndClears(t *testing.T) {
	cred := makeCred("cred-1", nil)
	strat, _ := newTestStrategy(t, &config.Config{}, cred)

	strat.OnResult("cred-1", 500)
	require.True(t, strat.isCooledDown("cred-1"))

	strat.OnResult("cred-1", 200)
	require.False(t, strat.isCooledDown("cred-1"))
}

func TestStrategySetAndClearCooldown(t *testing.T) {
	cred := makeCred("cred-1", nil)
	strat, _ := newTestStrategy(t, &config.Config{}, cred)

	strat.SetCooldown("cred-1", 2, time.Now().Add(2*time.Second))
	require.True(t, strat.isCooledDown("cred-1"))

	require.True(t, strat.ClearCooldown("cred-1"))
	require.False(t, strat.isCooledDown("cred-1"))
}

func TestStrategySnapshot(t *testing.T) {
	credA := makeCred("cred-a", nil)
	credB := makeCred("cred-b", nil)
	strat, _ := newTestStrategy(t, &config.Config{}, credA, credB)

	strat.setSticky("sticky-key", credA.ID, time.Minute)
	strat.SetCooldown(credA.ID, 1, time.Now().Add(time.Minute))
	strat.SetCooldown(credB.ID, 2, time.Now().Add(2*time.Minute))

	stickyCount, infos := strat.Snapshot()
	require.Equal(t, 1, stickyCount)
	require.Len(t, infos, 2)

	found := map[string]CooldownInfo{}
	for _, info := range infos {
		found[info.CredID] = info
		require.GreaterOrEqual(t, info.RemainingSec, int64(0))
	}
	require.Contains(t, found, credA.ID)
	require.Contains(t, found, credB.ID)
	require.Equal(t, 2, found[credB.ID].Strikes)
}

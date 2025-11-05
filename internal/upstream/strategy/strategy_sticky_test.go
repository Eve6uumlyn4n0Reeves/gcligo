package strategy

import (
	"net/http"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"github.com/stretchr/testify/require"
)

func TestStickySetAndExpiry(t *testing.T) {
	cred := makeCred("cred-sticky", nil)
	strat, _ := newTestStrategy(t, &config.Config{}, cred)

	strat.setSticky("sticky-key", cred.ID, 5*time.Millisecond)

	id, ok := strat.getSticky("sticky-key")
	require.True(t, ok)
	require.Equal(t, cred.ID, id)

	time.Sleep(10 * time.Millisecond)

	_, ok = strat.getSticky("sticky-key")
	require.False(t, ok, "sticky entry should expire after TTL")
}

func TestStickyKeyExtraction(t *testing.T) {
	hdr := http.Header{}
	hdr.Set("X-Session-ID", "session-123")
	key, src := stickyKeyAndSourceFromHeaders(hdr)
	require.NotEmpty(t, key)
	require.Equal(t, "session", src)

	hdr = http.Header{}
	hdr.Set("Authorization", "Bearer token-456")
	key, src = stickyKeyAndSourceFromHeaders(hdr)
	require.NotEmpty(t, key)
	require.Equal(t, "auth", src)
}

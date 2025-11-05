package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	enh "gcli2api-go/internal/handlers/management"
)

func TestProbeHistoryPersistsToStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := newTempFileBackend(t)

	// prepare a single credential
	tmp := t.TempDir()
	mgr := credential.NewManager(credential.Options{AuthDir: tmp})
	require.NoError(t, mgr.LoadCredentials())

	// upstream server that always 200s
	upstream := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"candidates":[{"content":{"parts":[{"text":"pong"}]}}]}}`))
	}))
	defer upstream.Close()

	cfg := &config.Config{CodeAssist: upstream.URL}
	h := enh.NewAdminAPIHandler(cfg, mgr, nil, nil, st)
	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	// invoke probe endpoint
	body := map[string]any{"model": "gemini-2.5-flash"}
	raw, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/routes/api/management/credentials/probe", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// verify storage contains history key
	v, err := st.GetConfig(context.Background(), "auto_probe_history")
	require.NoError(t, err)
	require.NotNil(t, v)
}

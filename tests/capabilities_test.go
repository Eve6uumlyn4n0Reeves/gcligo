package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gcli2api-go/internal/config"
	enh "gcli2api-go/internal/handlers/management"
	store "gcli2api-go/internal/storage"
)

func TestCapabilitiesEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := store.NewFileBackend(t.TempDir())
	assert.NoError(t, st.Initialize(nil))
	cfg := &config.Config{ManagementKey: "x"}
	h := enh.NewAdminAPIHandler(cfg, nil, nil, nil, st)

	r := gin.New()
	grp := r.Group("/routes/api/management")
	h.RegisterRoutes(grp)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/routes/api/management/capabilities", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"storage\"")
	assert.Contains(t, w.Body.String(), "\"type\"")
}

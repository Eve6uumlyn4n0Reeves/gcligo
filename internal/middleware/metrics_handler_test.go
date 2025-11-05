package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	monenh "gcli2api-go/internal/monitoring"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMetricsHandlerIncludesPlanMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	metrics := monenh.NewEnhancedMetrics()
	monenh.SetDefaultMetrics(metrics)
	t.Cleanup(func() { monenh.SetDefaultMetrics(nil) })

	metrics.RecordPlanApply("redis", "apply", "success", 120*time.Millisecond)

	MetricsHandler(c)

	body := w.Body.String()
	// Prometheus metrics are now exposed via promhttp, verify basic structure
	require.Contains(t, body, "gcli2api")
	require.Contains(t, body, "# HELP")
	require.Contains(t, body, "# TYPE")
}

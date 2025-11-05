package management

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRecordProbeMetrics_AllOK(t *testing.T) {
	h := &AdminAPIHandler{}
	source := "unit"
	model := "test-model"

	beforeRuns := testutil.ToFloat64(monitoring.AutoProbeRunsTotal.WithLabelValues(source, "all_ok", model))
	status, success, total := h.recordProbeMetrics(source, model, 1200*time.Millisecond, []gin.H{
		{"ok": true},
		{"ok": true},
	}, nil)

	assert.Equal(t, "all_ok", status)
	assert.Equal(t, 2, success)
	assert.Equal(t, 2, total)

	afterRuns := testutil.ToFloat64(monitoring.AutoProbeRunsTotal.WithLabelValues(source, "all_ok", model))
	assert.InDelta(t, beforeRuns+1, afterRuns, 0.0001)

	ratio := testutil.ToFloat64(monitoring.AutoProbeSuccessRatio.WithLabelValues(source, model))
	assert.InDelta(t, 1.0, ratio, 0.0001)

	lastSuccess := testutil.ToFloat64(monitoring.AutoProbeLastSuccess.WithLabelValues(source, model))
	assert.Greater(t, lastSuccess, float64(0))
}

func TestRecordProbeMetrics_PartialAndEmpty(t *testing.T) {
	h := &AdminAPIHandler{}

	status, success, total := h.recordProbeMetrics("unit-partial", "model-b", 500*time.Millisecond, []gin.H{
		{"ok": true},
		{"ok": false},
	}, nil)
	assert.Equal(t, "partial", status)
	assert.Equal(t, 1, success)
	assert.Equal(t, 2, total)

	ratio := testutil.ToFloat64(monitoring.AutoProbeSuccessRatio.WithLabelValues("unit-partial", "model-b"))
	assert.InDelta(t, 0.5, ratio, 0.0001)

	statusEmpty, successEmpty, totalEmpty := h.recordProbeMetrics("unit-empty", "model-c", 100*time.Millisecond, nil, nil)
	assert.Equal(t, "empty", statusEmpty)
	assert.Equal(t, 0, successEmpty)
	assert.Equal(t, 0, totalEmpty)

	ratioEmpty := testutil.ToFloat64(monitoring.AutoProbeSuccessRatio.WithLabelValues("unit-empty", "model-c"))
	assert.InDelta(t, 0.0, ratioEmpty, 0.0001)
}

func TestUpstreamSuggestReturnsDescriptors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &AdminAPIHandler{
		cfg: &config.Config{
			PreferredBaseModels: []string{"gemini-2.5-pro", "gemini-2.5-flash-image"},
		},
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodGet, "/routes/api/management/models/upstream-suggest", nil)
	ctx.Request = req

	h.UpstreamSuggest(ctx)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Bases     []string                         `json:"bases"`
		Preferred []string                         `json:"preferred"`
		Meta      map[string]models.BaseDescriptor `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	assert.Contains(t, resp.Bases, "gemini-2.5-pro")
	assert.Contains(t, resp.Preferred, "gemini-2.5-pro")

	desc, ok := resp.Meta["gemini-2.5-pro"]
	if !ok {
		t.Fatalf("expected descriptor for gemini-2.5-pro")
	}
	assert.True(t, desc.SupportsStream)
	assert.Equal(t, "gemini-2.5-pro", desc.Base)
}

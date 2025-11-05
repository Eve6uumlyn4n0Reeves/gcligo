package management

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/models"
	"gcli2api-go/internal/monitoring"
	oauth "gcli2api-go/internal/oauth"
	"gcli2api-go/internal/storage"
	tr "gcli2api-go/internal/translator"
	up "gcli2api-go/internal/upstream/gemini"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	probeHistoryKey          = "auto_probe_history"
	maxProbeHistoryEntries   = 50
	defaultProbeHistoryLimit = 20
)

type probeHistoryEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	Model      string                 `json:"model"`
	TimeoutSec int                    `json:"timeout_sec"`
	DurationMs int64                  `json:"duration_ms"`
	Success    int                    `json:"success"`
	Total      int                    `json:"total"`
	Results    []map[string]any       `json:"results"`
	Error      string                 `json:"error,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

func (h *AdminAPIHandler) loadProbeHistory(ctx context.Context) {
	if h == nil || h.storage == nil {
		return
	}
	// 只在 ctx 为 nil 时创建新的 context.Background()
	// 调用者应该尽可能传递有效的 context
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := h.storage.GetConfig(ctx, probeHistoryKey)
	if err != nil {
		var nf *storage.ErrNotFound
		if errors.As(err, &nf) || isNotSupported(err) {
			return
		}
		log.WithError(err).Warn("failed to load probe history from storage")
		return
	}
	data, err := json.Marshal(raw)
	if err != nil {
		log.WithError(err).Warn("failed to marshal stored probe history")
		return
	}
	var entries []probeHistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.WithError(err).Warn("failed to decode stored probe history")
		return
	}
	if len(entries) > maxProbeHistoryEntries {
		entries = entries[:maxProbeHistoryEntries]
	}
	h.probeHistoryMu.Lock()
	h.probeHistory = entries
	h.probeHistoryMu.Unlock()
}

func (h *AdminAPIHandler) recordProbeHistory(ctx context.Context, source, model string, timeoutSec int, duration time.Duration, results []gin.H, err error) {
	if h == nil {
		return
	}
	// 只在 ctx 为 nil 时创建新的 context.Background()
	// 调用者应该尽可能传递有效的 context
	if ctx == nil {
		ctx = context.Background()
	}
	success := 0
	converted := make([]map[string]any, 0, len(results))
	for _, r := range results {
		if r == nil {
			continue
		}
		if ok, okExists := r["ok"].(bool); okExists && ok {
			success++
		}
		converted = append(converted, cloneResultMap(r))
	}
	entry := probeHistoryEntry{
		Timestamp:  time.Now().UTC(),
		Source:     source,
		Model:      model,
		TimeoutSec: timeoutSec,
		DurationMs: duration.Milliseconds(),
		Success:    success,
		Total:      len(converted),
		Results:    converted,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	if source == "auto" {
		log.WithFields(log.Fields{"component": "audit", "action": "credential.probe.auto", "model": model, "success": success, "total": len(converted)}).Info("auto probe completed")
	}
	h.probeHistoryMu.Lock()
	h.probeHistory = append([]probeHistoryEntry{entry}, h.probeHistory...)
	if len(h.probeHistory) > maxProbeHistoryEntries {
		h.probeHistory = h.probeHistory[:maxProbeHistoryEntries]
	}
	snapshot := append([]probeHistoryEntry(nil), h.probeHistory...)
	h.probeHistoryMu.Unlock()
	if h.storage != nil {
		if storageErr := h.storage.SetConfig(ctx, probeHistoryKey, snapshot); storageErr != nil && !isNotSupported(storageErr) {
			log.WithError(storageErr).Warn("failed to persist probe history")
		}
	}
}

// ProbeCredentials performs a fast upstream liveness check using a cheap model (default: gemini-2.5-flash).
// Accepts optional JSON body {"ids": ["cred.json", ...], "model": "gemini-2.5-flash", "timeout_sec": 10}
func (h *AdminAPIHandler) ProbeCredentials(c *gin.Context) {
	if h.credMgr == nil {
		respondError(c, http.StatusInternalServerError, "credential manager not configured")
		return
	}
	start := time.Now()
	var body struct {
		IDs        []string `json:"ids"`
		Model      string   `json:"model"`
		TimeoutSec int      `json:"timeout_sec"`
	}
	_ = c.ShouldBindJSON(&body)
	model := strings.TrimSpace(body.Model)
	if model == "" {
		model = "gemini-2.5-flash"
	}
	to := body.TimeoutSec
	if to <= 0 || to > 60 {
		to = 10
	}
	results := h.probeInternal(c.Request.Context(), body.IDs, model, to)
	duration := time.Since(start)
	status, success, total := h.recordProbeMetrics("manual", model, duration, results, nil)
	// 使用请求的 context 来记录历史，保持 context 链路完整
	h.recordProbeHistory(c.Request.Context(), "manual", model, to, duration, results, nil)
	log.WithFields(log.Fields{"component": "probe", "source": "manual", "model": model, "status": status, "success": success, "total": total, "duration_ms": duration.Milliseconds()}).Info("credential probe completed")
	h.audit(c, "credential.probe", log.Fields{"model": model, "count": len(results)})
	c.JSON(http.StatusOK, gin.H{"model": model, "results": results})
}

// GetProbeHistory returns recent probe history entries (auto + manual)
func (h *AdminAPIHandler) GetProbeHistory(c *gin.Context) {
	limit := defaultProbeHistoryLimit
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			if parsed > maxProbeHistoryEntries {
				parsed = maxProbeHistoryEntries
			}
			limit = parsed
		}
	}
	h.probeHistoryMu.Lock()
	defer h.probeHistoryMu.Unlock()
	if limit > len(h.probeHistory) {
		limit = len(h.probeHistory)
	}
	history := make([]probeHistoryEntry, 0, limit)
	for i := 0; i < limit; i++ {
		history = append(history, h.probeHistory[i])
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// probeInternal executes probe logic and returns result slice (gin.H items)
func (h *AdminAPIHandler) probeInternal(ctx context.Context, ids []string, model string, timeoutSec int) []gin.H {
	if h.credMgr == nil {
		return nil
	}
	filter := map[string]struct{}{}
	for _, id := range ids {
		if id != "" {
			filter[id] = struct{}{}
		}
	}
	creds := h.credMgr.GetAllCredentials()
	raw := map[string]any{"model": model, "messages": []any{map[string]any{"role": "user", "content": "ping"}}, "stream": false}
	rawJSON, _ := json.Marshal(raw)
	reqJSON := tr.OpenAIToGeminiRequest(models.BaseFromFeature(model), rawJSON, false)
	var gemReq map[string]any
	_ = json.Unmarshal(reqJSON, &gemReq)

	type probeEntry struct {
		cred   *credential.Credential
		result gin.H
	}
	entries := make([]probeEntry, 0, len(creds))
	for _, cr := range creds {
		if len(filter) > 0 {
			if _, ok := filter[cr.ID]; !ok {
				continue
			}
		}
		entry := probeEntry{cred: cr}
		if strings.TrimSpace(cr.AccessToken) == "" {
			entry.result = gin.H{"id": cr.ID, "email": cr.Email, "project_id": cr.ProjectID, "ok": false, "status": 0, "error": "no access_token"}
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil
	}

	cctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	const workerLimit = 4
	sem := make(chan struct{}, workerLimit)
	type probeResult struct {
		idx  int
		data gin.H
	}
	results := make([]gin.H, len(entries))
	baseModel := models.BaseFromFeature(model)

	var wg sync.WaitGroup
	resCh := make(chan probeResult, len(entries))
	for idx, entry := range entries {
		if entry.result != nil {
			results[idx] = entry.result
			continue
		}
		wg.Add(1)
		go func(i int, cred *credential.Credential) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			res := h.probeCredential(cctx, cred, baseModel, gemReq)
			resCh <- probeResult{idx: i, data: res}
		}(idx, entry.cred)
	}
	go func() { wg.Wait(); close(resCh) }()
	for r := range resCh {
		results[r.idx] = r.data
	}
	out := make([]gin.H, 0, len(results))
	for _, r := range results {
		if r != nil {
			out = append(out, r)
		}
	}
	return out
}

func (h *AdminAPIHandler) recordProbeMetrics(source, model string, duration time.Duration, results []gin.H, recErr error) (status string, success, total int) {
	if model == "" {
		model = "unknown"
	}
	total = len(results)
	for _, r := range results {
		if ok, exists := r["ok"].(bool); exists && ok {
			success++
		}
	}
	switch {
	case recErr != nil:
		status = "error"
	case total == 0:
		status = "empty"
	case success == total:
		status = "all_ok"
	case success == 0:
		status = "all_failed"
	default:
		status = "partial"
	}
	monitoring.AutoProbeRunsTotal.WithLabelValues(source, status, model).Inc()
	monitoring.AutoProbeDurationSeconds.WithLabelValues(source, model).Observe(duration.Seconds())
	monitoring.AutoProbeCredentialCount.WithLabelValues(source, model).Set(float64(total))
	ratio := 0.0
	if total > 0 {
		ratio = float64(success) / float64(total)
	}
	monitoring.AutoProbeSuccessRatio.WithLabelValues(source, model).Set(ratio)
	if success > 0 {
		monitoring.AutoProbeLastSuccess.WithLabelValues(source, model).Set(float64(time.Now().Unix()))
	}
	return status, success, total
}

// StartAutoProbe launches a daily probe job using configured defaults.
func (h *AdminAPIHandler) StartAutoProbe(ctx context.Context) {
	h.autoProbeMu.Lock()
	// 只在 ctx 为 nil 时创建新的 context.Background()
	// 调用者应该尽可能传递有效的 context（如服务启动时的 context）
	if ctx == nil {
		ctx = context.Background()
	}
	h.autoProbeBaseCtx = ctx
	h.startAutoProbeLocked()
	h.autoProbeMu.Unlock()
}

func (h *AdminAPIHandler) runAutoProbeOnce(ctx context.Context) error {
	h.autoProbeMu.Lock()
	cfg := h.cfg
	h.autoProbeMu.Unlock()
	if cfg == nil {
		return nil
	}
	model := cfg.AutoProbeModel
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.5-flash"
	}
	to := cfg.AutoProbeTimeoutSec
	if to <= 0 {
		to = 10
	}
	start := time.Now()
	results := h.probeInternal(ctx, nil, model, to)
	duration := time.Since(start)
	status, success, total := h.recordProbeMetrics("auto", model, duration, results, nil)
	// 使用传入的 ctx 来记录历史，保持 context 链路完整
	h.recordProbeHistory(ctx, "auto", model, to, duration, results, nil)
	// 可选：满足阈值则自动禁用该 base 模型，并记录原因
	if cfg := h.cfg; cfg != nil && cfg.AutoProbeDisableThresholdPct > 0 && total > 0 {
		ratio := 0.0
		if total > 0 {
			ratio = float64(success) / float64(total)
		}
		threshold := float64(cfg.AutoProbeDisableThresholdPct) / 100.0
		if ratio < threshold {
			base := models.BaseFromFeature(model)
			// 更新 disabled_models 列表（去重）
			dm := append([]string(nil), cfg.DisabledModels...)
			found := false
			for _, d := range dm {
				if strings.EqualFold(strings.TrimSpace(d), base) {
					found = true
					break
				}
			}
			if !found {
				dm = append(dm, base)
			}
			// 持久化到配置
			_ = config.UpdateConfig(map[string]interface{}{"disabled_models": dm})
			cfg.DisabledModels = dm
			// 写入禁用原因到存储（仅 UI 展示，不影响核心逻辑）
			reason := fmt.Sprintf("auto_probe_low_success: %.0f%% < %.0f%%", ratio*100, threshold*100)
			h.setDisabledModelReason(ctx, base, reason)
			log.WithFields(log.Fields{"component": "probe", "action": "model.auto_disable", "base": base, "reason": reason}).Warn("auto-disabled model due to probe failure rate")
		}
	}
	log.WithFields(log.Fields{"component": "probe", "source": "auto", "model": model, "timeout_sec": to, "status": status, "success": success, "total": total, "duration_ms": duration.Milliseconds()}).Info("credential probe completed")
	return nil
}

// setDisabledModelReason stores a human-readable reason for a disabled model for UI surfaces.
func (h *AdminAPIHandler) setDisabledModelReason(ctx context.Context, base, reason string) {
	if h.storage == nil || strings.TrimSpace(base) == "" {
		return
	}
	key := "disabled_model_reasons"
	var m map[string]string
	if v, err := h.storage.GetConfig(ctx, key); err == nil && v != nil {
		b, _ := json.Marshal(v)
		_ = json.Unmarshal(b, &m)
	}
	if m == nil {
		m = map[string]string{}
	}
	m[strings.ToLower(strings.TrimSpace(base))] = reason
	_ = h.storage.SetConfig(ctx, key, m)
}

func (h *AdminAPIHandler) startAutoProbeLocked() {
	h.stopAutoProbeLocked()
	if h.cfg == nil || !h.cfg.AutoProbeEnabled {
		return
	}
	baseCtx := h.autoProbeBaseCtx
	// 只在 baseCtx 为 nil 时创建新的 context.Background()
	// 通常 baseCtx 应该在 StartAutoProbe 时设置
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	childCtx, cancel := context.WithCancel(baseCtx)
	h.autoProbeCancel = cancel
	go h.autoProbeLoop(childCtx)
}

func (h *AdminAPIHandler) restartAutoProbe() {
	h.autoProbeMu.Lock()
	defer h.autoProbeMu.Unlock()
	// 只在 autoProbeBaseCtx 为 nil 时创建新的 context.Background()
	// 通常应该在服务启动时通过 StartAutoProbe 设置
	if h.autoProbeBaseCtx == nil {
		h.autoProbeBaseCtx = context.Background()
	}
	h.startAutoProbeLocked()
}

func (h *AdminAPIHandler) stopAutoProbeLocked() {
	if h.autoProbeCancel != nil {
		h.autoProbeCancel()
		h.autoProbeCancel = nil
	}
}

func (h *AdminAPIHandler) autoProbeLoop(ctx context.Context) {
	h.autoProbeMu.Lock()
	lastRun := h.autoProbeLastRun
	cfg := h.cfg
	h.autoProbeMu.Unlock()
	now := time.Now().UTC()
	if cfg != nil && cfg.AutoProbeEnabled && h.shouldRunImmediately(now, lastRun) {
		if err := h.runAutoProbeOnce(ctx); err != nil {
			log.WithError(err).Warn("auto probe run failed")
		}
		h.setAutoProbeLastRun(time.Now().UTC())
	}
	for {
		next := h.nextAutoProbeTime(time.Now().UTC())
		delay := time.Until(next)
		if delay <= 0 {
			delay = time.Minute
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := h.runAutoProbeOnce(ctx); err != nil {
				log.WithError(err).Warn("auto probe run failed")
			}
			h.setAutoProbeLastRun(time.Now().UTC())
		}
	}
}

func (h *AdminAPIHandler) setAutoProbeLastRun(ts time.Time) {
	h.autoProbeMu.Lock()
	h.autoProbeLastRun = ts
	h.autoProbeMu.Unlock()
}

func (h *AdminAPIHandler) shouldRunImmediately(now, last time.Time) bool {
	h.autoProbeMu.Lock()
	cfg := h.cfg
	h.autoProbeMu.Unlock()
	if cfg == nil || !cfg.AutoProbeEnabled {
		return false
	}
	if last.IsZero() {
		return true
	}
	if now.Sub(last) < 20*time.Hour {
		return false
	}
	hour := cfg.AutoProbeHourUTC
	if hour < 0 || hour > 23 {
		hour = 7
	}
	scheduled := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	return now.After(scheduled)
}

func (h *AdminAPIHandler) nextAutoProbeTime(now time.Time) time.Time {
	h.autoProbeMu.Lock()
	cfg := h.cfg
	last := h.autoProbeLastRun
	h.autoProbeMu.Unlock()
	if cfg == nil {
		return now.Add(24 * time.Hour)
	}
	hour := cfg.AutoProbeHourUTC
	if hour < 0 || hour > 23 {
		hour = 7
	}
	base := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if !now.Before(base) || (!last.IsZero() && now.Sub(last) < time.Hour && now.After(base)) {
		base = base.Add(24 * time.Hour)
	}
	jitterSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	jitter := time.Duration(jitterSrc.Intn(5*60)) * time.Second
	return base.Add(jitter)
}

func (h *AdminAPIHandler) probeCredential(ctx context.Context, cred *credential.Credential, baseModel string, gemReq map[string]any) gin.H {
	if cred == nil {
		return nil
	}
	oc := &oauth.Credentials{AccessToken: cred.AccessToken, ProjectID: cred.ProjectID}
	client := up.NewWithCredential(h.cfg, oc).WithCaller("mgmt")
	effProject := h.cfg.GoogleProjID
	if cred.ProjectID != "" {
		effProject = cred.ProjectID
	}
	payload := map[string]any{"model": baseModel, "project": effProject, "request": gemReq}
	body, _ := json.Marshal(payload)
	status := 0
	errStr := ""
	ok := false
	if resp, err := client.Generate(ctx, body); err != nil {
		errStr = err.Error()
	} else {
		status = resp.StatusCode
		_ = resp.Body.Close()
		if status >= 200 && status < 300 {
			ok = true
			h.credMgr.MarkSuccess(cred.ID)
		} else {
			h.credMgr.MarkFailure(cred.ID, "probe_failed", status)
		}
	}
	return gin.H{"id": cred.ID, "email": cred.Email, "project_id": cred.ProjectID, "ok": ok, "status": status, "error": errStr}
}

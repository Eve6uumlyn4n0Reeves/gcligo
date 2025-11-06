package usage

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Tracker manages usage statistics collection and persistence
type Tracker struct {
	stats   *Stats
	storage Storage
	mu      sync.RWMutex

	// Background persistence
	persistInterval time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewTracker creates a new usage tracker
func NewTracker(storage Storage) *Tracker {
	return &Tracker{
		stats:           NewStats(),
		storage:         storage,
		persistInterval: 60 * time.Second, // Save every minute
		stopCh:          make(chan struct{}),
	}
}

// Start starts the background persistence worker
func (t *Tracker) Start(ctx context.Context) error {
	// Load existing statistics from storage
	if err := t.loadFromStorage(ctx); err != nil {
		log.WithError(err).Warn("Failed to load usage statistics from storage, starting fresh")
	}

	// Start background persistence worker
	t.wg.Add(1)
	go t.persistWorker(ctx)

	log.Info("Usage tracker started")
	return nil
}

// Stop stops the tracker and persists final statistics
func (t *Tracker) Stop(ctx context.Context) error {
	close(t.stopCh)
	t.wg.Wait()

	// Final persistence
	if err := t.saveToStorage(ctx); err != nil {
		log.WithError(err).Error("Failed to save final usage statistics")
		return err
	}

	log.Info("Usage tracker stopped")
	return nil
}

// Record records a request in the statistics
func (t *Tracker) Record(record *RequestRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Update global counters
	t.stats.TotalRequests++
	if record.Success {
		t.stats.SuccessCount++
	} else {
		t.stats.FailureCount++
	}
	if record.Tokens != nil {
		t.stats.TotalTokens += record.Tokens.TotalTokens
	}

	// Update credential statistics
	if record.CredentialID != "" {
		if _, ok := t.stats.Credentials[record.CredentialID]; !ok {
			t.stats.Credentials[record.CredentialID] = NewCredentialUsage(record.CredentialID)
		}
		cred := t.stats.Credentials[record.CredentialID]

		// Check and reset quota if needed
		if cred.ShouldResetQuota() {
			cred.ResetQuota()
		}

		cred.IncrementUsage(record.Model, record.Tokens, record.Success)
	}

	// Update time-based statistics
	t.updateTimeStats(record)

	// Update API statistics
	t.updateAPIStats(record)
}

// updateTimeStats updates daily and hourly statistics
func (t *Tracker) updateTimeStats(record *RequestRecord) {
	// Daily stats
	dateKey := record.Timestamp.Format("2006-01-02")
	if _, ok := t.stats.DailyStats[dateKey]; !ok {
		t.stats.DailyStats[dateKey] = &DailyStats{Date: dateKey}
	}
	daily := t.stats.DailyStats[dateKey]
	daily.Requests++
	if record.Success {
		daily.Success++
	} else {
		daily.Failure++
	}
	if record.Tokens != nil {
		daily.Tokens += record.Tokens.TotalTokens
	}

	// Hourly stats (aggregated across all days)
	hour := record.Timestamp.Hour()
	if _, ok := t.stats.HourlyStats[hour]; !ok {
		t.stats.HourlyStats[hour] = &HourlyStats{Hour: hour}
	}
	hourly := t.stats.HourlyStats[hour]
	hourly.Requests++
	if record.Success {
		hourly.Success++
	} else {
		hourly.Failure++
	}
	if record.Tokens != nil {
		hourly.Tokens += record.Tokens.TotalTokens
	}
}

// updateAPIStats updates per-API statistics
func (t *Tracker) updateAPIStats(record *RequestRecord) {
	if record.API == "" {
		return
	}

	if _, ok := t.stats.APIs[record.API]; !ok {
		t.stats.APIs[record.API] = NewAPIStats(record.API)
	}
	api := t.stats.APIs[record.API]
	api.TotalRequests++
	if record.Tokens != nil {
		api.TotalTokens += record.Tokens.TotalTokens
	}

	// Update model stats within API
	if record.Model != "" {
		if _, ok := api.Models[record.Model]; !ok {
			api.Models[record.Model] = NewModelStats(record.Model)
		}
		model := api.Models[record.Model]
		model.Calls++
		if record.Tokens != nil {
			model.Tokens += record.Tokens.TotalTokens
			model.InputTokens += record.Tokens.InputTokens
			model.OutputTokens += record.Tokens.OutputTokens
			model.ReasoningTokens += record.Tokens.ReasoningTokens
			model.CachedTokens += record.Tokens.CachedTokens
		}
		model.LastUsed = record.Timestamp
	}
}

// GetStats returns a snapshot of current statistics
func (t *Tracker) GetStats() *Stats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a deep copy
	snapshot := &Stats{
		TotalRequests: t.stats.TotalRequests,
		SuccessCount:  t.stats.SuccessCount,
		FailureCount:  t.stats.FailureCount,
		TotalTokens:   t.stats.TotalTokens,
		Credentials:   make(map[string]*CredentialUsage),
		DailyStats:    make(map[string]*DailyStats),
		HourlyStats:   make(map[int]*HourlyStats),
		APIs:          make(map[string]*APIStats),
	}

	// Copy credentials
	for k, v := range t.stats.Credentials {
		snapshot.Credentials[k] = v.Snapshot()
	}

	// Copy daily stats
	for k, v := range t.stats.DailyStats {
		snapshot.DailyStats[k] = &DailyStats{
			Date:     v.Date,
			Requests: v.Requests,
			Tokens:   v.Tokens,
			Success:  v.Success,
			Failure:  v.Failure,
		}
	}

	// Copy hourly stats
	for k, v := range t.stats.HourlyStats {
		snapshot.HourlyStats[k] = &HourlyStats{
			Hour:     v.Hour,
			Requests: v.Requests,
			Tokens:   v.Tokens,
			Success:  v.Success,
			Failure:  v.Failure,
		}
	}

	// Copy API stats
	for k, v := range t.stats.APIs {
		apiCopy := &APIStats{
			Name:          v.Name,
			TotalRequests: v.TotalRequests,
			TotalTokens:   v.TotalTokens,
			Models:        make(map[string]*ModelStats),
		}
		for mk, mv := range v.Models {
			apiCopy.Models[mk] = &ModelStats{
				ModelName:       mv.ModelName,
				Calls:           mv.Calls,
				Tokens:          mv.Tokens,
				InputTokens:     mv.InputTokens,
				OutputTokens:    mv.OutputTokens,
				ReasoningTokens: mv.ReasoningTokens,
				CachedTokens:    mv.CachedTokens,
				LastUsed:        mv.LastUsed,
			}
		}
		snapshot.APIs[k] = apiCopy
	}

	return snapshot
}

// GetCredentialStats returns statistics for a specific credential
func (t *Tracker) GetCredentialStats(credentialID string) *CredentialUsage {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if cred, ok := t.stats.Credentials[credentialID]; ok {
		return cred.Snapshot()
	}
	return nil
}

// IsQuotaExceeded checks if a credential has exceeded its quota
func (t *Tracker) IsQuotaExceeded(credentialID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if cred, ok := t.stats.Credentials[credentialID]; ok {
		return cred.IsQuotaExceeded()
	}
	return false
}

// persistWorker runs in the background to periodically save statistics
func (t *Tracker) persistWorker(ctx context.Context) {
	defer t.wg.Done()

	ticker := time.NewTicker(t.persistInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := t.saveToStorage(ctx); err != nil {
				log.WithError(err).Error("Failed to persist usage statistics")
			}
		case <-t.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// loadFromStorage loads statistics from storage
func (t *Tracker) loadFromStorage(ctx context.Context) error {
	if t.storage == nil {
		return nil
	}

	stats, err := t.storage.LoadStats(ctx)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats = stats

	log.WithFields(log.Fields{
		"credentials": len(stats.Credentials),
		"daily_stats": len(stats.DailyStats),
	}).Info("Loaded usage statistics from storage")

	return nil
}

// saveToStorage saves statistics to storage
func (t *Tracker) saveToStorage(ctx context.Context) error {
	if t.storage == nil {
		return nil
	}

	t.mu.RLock()
	snapshot := t.stats
	t.mu.RUnlock()

	if err := t.storage.SaveStats(ctx, snapshot); err != nil {
		return err
	}

	return nil
}


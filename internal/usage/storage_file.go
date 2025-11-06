package usage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
)

// FileStorage implements Storage interface using file system
type FileStorage struct {
	dataDir string
	mu      sync.RWMutex
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(dataDir string) (*FileStorage, error) {
	if dataDir == "" {
		dataDir = "./data/usage"
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	return &FileStorage{
		dataDir: dataDir,
	}, nil
}

// LoadStats implements Storage
func (f *FileStorage) LoadStats(ctx context.Context) (*Stats, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	statsFile := filepath.Join(f.dataDir, "stats.json")
	data, err := os.ReadFile(statsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty stats if file doesn't exist
			return NewStats(), nil
		}
		return nil, err
	}

	var stats Stats
	if err := json.Unmarshal(data, &stats); err != nil {
		log.WithError(err).Warn("Failed to unmarshal stats, returning empty stats")
		return NewStats(), nil
	}

	// Initialize maps if nil
	if stats.Credentials == nil {
		stats.Credentials = make(map[string]*CredentialUsage)
	}
	if stats.DailyStats == nil {
		stats.DailyStats = make(map[string]*DailyStats)
	}
	if stats.HourlyStats == nil {
		stats.HourlyStats = make(map[int]*HourlyStats)
	}
	if stats.APIs == nil {
		stats.APIs = make(map[string]*APIStats)
	}

	// Initialize nested maps
	for _, cred := range stats.Credentials {
		if cred.ModelBreakdown == nil {
			cred.ModelBreakdown = make(map[string]*ModelStats)
		}
	}
	for _, api := range stats.APIs {
		if api.Models == nil {
			api.Models = make(map[string]*ModelStats)
		}
	}

	return &stats, nil
}

// SaveStats implements Storage
func (f *FileStorage) SaveStats(ctx context.Context, stats *Stats) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	statsFile := filepath.Join(f.dataDir, "stats.json")
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tempFile := statsFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tempFile, statsFile); err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	return nil
}

// LoadCredentialUsage implements Storage
func (f *FileStorage) LoadCredentialUsage(ctx context.Context, credentialID string) (*CredentialUsage, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	credFile := filepath.Join(f.dataDir, "credentials", credentialID+".json")
	data, err := os.ReadFile(credFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var usage CredentialUsage
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil, err
	}

	// Initialize maps if nil
	if usage.ModelBreakdown == nil {
		usage.ModelBreakdown = make(map[string]*ModelStats)
	}

	return &usage, nil
}

// SaveCredentialUsage implements Storage
func (f *FileStorage) SaveCredentialUsage(ctx context.Context, usage *CredentialUsage) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	credDir := filepath.Join(f.dataDir, "credentials")
	if err := os.MkdirAll(credDir, 0755); err != nil {
		return err
	}

	credFile := filepath.Join(credDir, usage.ID+".json")
	data, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first
	tempFile := credFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tempFile, credFile); err != nil {
		_ = os.Remove(tempFile)
		return err
	}

	return nil
}


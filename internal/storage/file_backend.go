package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	storagecommon "gcli2api-go/internal/storage/common"
)

// FileBackend implements storage using local files
type FileBackend struct {
	baseDir     string
	mu          sync.RWMutex
	credentials map[string]map[string]interface{}
	config      map[string]interface{}
	usage       map[string]map[string]interface{}
}

func (f *FileBackend) replaceCredentialLocked(id string, data map[string]interface{}) {
	if existing, ok := f.credentials[id]; ok {
		storagecommon.ReturnCredentialMap(existing)
	}
	if data == nil {
		delete(f.credentials, id)
		return
	}
	f.credentials[id] = data
}

// NewFileBackend creates a new file-based storage backend
func NewFileBackend(baseDir string) *FileBackend {
	return &FileBackend{
		baseDir:     baseDir,
		credentials: make(map[string]map[string]interface{}),
		config:      make(map[string]interface{}),
		usage:       make(map[string]map[string]interface{}),
	}
}

func (f *FileBackend) Initialize(ctx context.Context) error {
	// Create directories
	dirs := []string{
		filepath.Join(f.baseDir, "credentials"),
		filepath.Join(f.baseDir, "config"),
		filepath.Join(f.baseDir, "usage"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Load existing data
	if err := f.loadAll(); err != nil {
		return fmt.Errorf("failed to load existing data: %w", err)
	}

	return nil
}

func (f *FileBackend) Close() error {
	// Save all data before closing
	return f.saveAll()
}

func (f *FileBackend) Health(ctx context.Context) error {
	// Check if base directory is accessible
	_, err := os.Stat(f.baseDir)
	return err
}

// Credential operations
func (f *FileBackend) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	cred, exists := f.credentials[id]
	if !exists {
		return nil, &ErrNotFound{Key: id}
	}

	return storagecommon.ShallowCopyMap(cred), nil
}

func (f *FileBackend) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	cloned := storagecommon.CloneCredentialMap(data)
	f.replaceCredentialLocked(id, cloned)
	return f.saveCredential(id, cloned)
}

func (f *FileBackend) DeleteCredential(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if cred, ok := f.credentials[id]; ok {
		storagecommon.ReturnCredentialMap(cred)
		delete(f.credentials, id)
	}

	filePath := filepath.Join(f.baseDir, "credentials", id+".json")
	return os.Remove(filePath)
}

func (f *FileBackend) ListCredentials(ctx context.Context) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var ids []string
	for id := range f.credentials {
		ids = append(ids, id)
	}
	return ids, nil
}

// Config operations
func (f *FileBackend) GetConfig(ctx context.Context, key string) (interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	value, exists := f.config[key]
	if !exists {
		return nil, &ErrNotFound{Key: key}
	}
	return value, nil
}

func (f *FileBackend) SetConfig(ctx context.Context, key string, value interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.config[key] = value
	return f.saveConfig()
}

func (f *FileBackend) DeleteConfig(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.config, key)
	return f.saveConfig()
}

func (f *FileBackend) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return storagecommon.ShallowCopyMap(f.config), nil
}

// Usage operations
func (f *FileBackend) IncrementUsage(ctx context.Context, key string, field string, delta int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.usage[key] == nil {
		f.usage[key] = make(map[string]interface{})
	}

	current, _ := f.usage[key][field].(int64)
	f.usage[key][field] = current + delta

	return f.saveUsage(key)
}

func (f *FileBackend) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	usage, exists := f.usage[key]
	if !exists {
		return nil, &ErrNotFound{Key: key}
	}

	return storagecommon.ShallowCopyMap(usage), nil
}

func (f *FileBackend) ResetUsage(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.usage, key)

	filePath := filepath.Join(f.baseDir, "usage", key+".json")
	return os.Remove(filePath)
}

func (f *FileBackend) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return storagecommon.ShallowCopyNestedMap(f.usage), nil
}

// Cache operations (not supported for file backend)
func (f *FileBackend) GetCache(ctx context.Context, key string) ([]byte, error) {
	return nil, &ErrNotSupported{Operation: "GetCache"}
}

func (f *FileBackend) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return &ErrNotSupported{Operation: "SetCache"}
}

func (f *FileBackend) DeleteCache(ctx context.Context, key string) error {
	return &ErrNotSupported{Operation: "DeleteCache"}
}

// Batch operations for performance
func (f *FileBackend) BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]map[string]interface{})

	for _, id := range ids {
		if cred, exists := f.credentials[id]; exists {
			result[id] = storagecommon.ShallowCopyMap(cred)
		}
	}

	return result, nil
}

func (f *FileBackend) BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for id, credData := range data {
		cloned := storagecommon.CloneCredentialMap(credData)
		f.replaceCredentialLocked(id, cloned)
	}

	// Save to disk
	return f.saveAll()
}

func (f *FileBackend) BatchDeleteCredentials(ctx context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, id := range ids {
		if cred, ok := f.credentials[id]; ok {
			storagecommon.ReturnCredentialMap(cred)
			delete(f.credentials, id)
		}
	}

	// Save to disk
	return f.saveAll()
}

// Transaction support (not supported for file backend)
func (f *FileBackend) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, &ErrNotSupported{Operation: "BeginTransaction"}
}

// ExportData exports all data for backup
func (f *FileBackend) ExportData(ctx context.Context) (map[string]interface{}, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	exportData := make(map[string]interface{})

	// Export credentials/configs/usage (deep copy)
	exportData["credentials"] = storagecommon.ShallowCopyNestedMap(f.credentials)
	exportData["configs"] = storagecommon.ShallowCopyMap(f.config)
	exportData["usage"] = storagecommon.ShallowCopyNestedMap(f.usage)

	exportData["exported_at"] = time.Now().UTC()
	exportData["backend"] = "file"

	return exportData, nil
}

// ImportData imports data from backup
func (f *FileBackend) ImportData(ctx context.Context, data map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Import credentials
	if creds, ok := data["credentials"].(map[string]interface{}); ok {
		for id, credDataRaw := range creds {
			if credMap, ok := credDataRaw.(map[string]interface{}); ok {
				cloned := storagecommon.CloneCredentialMap(credMap)
				f.replaceCredentialLocked(id, cloned)
			}
		}
	}

	// Import configs
	if configs, ok := data["configs"].(map[string]interface{}); ok {
		for key, value := range configs {
			f.config[key] = value
		}
	}

	// Import usage
	if usage, ok := data["usage"].(map[string]interface{}); ok {
		for key, usageDataRaw := range usage {
			if usageMap, ok := usageDataRaw.(map[string]interface{}); ok {
				f.usage[key] = storagecommon.ShallowCopyMap(usageMap)
			}
		}
	}

	// Save to disk
	return f.saveAll()
}

// GetStorageStats returns storage statistics
func (f *FileBackend) GetStorageStats(ctx context.Context) (StorageStats, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := StorageStats{
		Backend: "file",
		Healthy: true,
	}

	// Count items
	stats.CredentialCount = len(f.credentials)
	stats.ConfigCount = len(f.config)
	stats.UsageRecordCount = len(f.usage)

	// Calculate total size (approximate)
	var totalSize int64
	for _, cred := range f.credentials {
		if b, err := json.Marshal(cred); err == nil {
			totalSize += int64(len(b))
		}
	}
	for _, cfg := range f.config {
		if b, err := json.Marshal(cfg); err == nil {
			totalSize += int64(len(b))
		}
	}
	stats.TotalSize = totalSize

	return stats, nil
}

// Helper methods
// moved to file_backend_io.go: loadAll

// moved to file_backend_io.go: loadCredentials

// moved to file_backend_io.go: loadConfig

// moved to file_backend_io.go: loadUsage

// moved to file_backend_io.go: saveAll

// moved to file_backend_io.go: saveCredential

// moved to file_backend_io.go: saveConfig

// moved to file_backend_io.go: saveUsage

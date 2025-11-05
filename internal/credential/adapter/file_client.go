package adapter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// FileStorageAdapter 文件存储适配器实现
type FileStorageAdapter struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
	states      map[string]*CredentialState
	credDir     string
	stateDir    string
	config      map[string]interface{}
	watchers    *watcherHub
}

// FileStorageConfig 文件存储配置
type FileStorageConfig struct {
	CredentialsDir string                 `yaml:"credentials_dir" json:"credentials_dir"`
	StatesDir      string                 `yaml:"states_dir" json:"states_dir"`
	WatchInterval  time.Duration          `yaml:"watch_interval" json:"watch_interval"`
	Config         map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// NewFileStorageAdapter 创建新的文件存储适配器
func NewFileStorageAdapter(config *FileStorageConfig) (*FileStorageAdapter, error) {
	if config == nil {
		return nil, fmt.Errorf("file storage config cannot be nil")
	}

	if err := os.MkdirAll(config.CredentialsDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create credentials directory: %w", err)
	}
	if err := os.MkdirAll(config.StatesDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create states directory: %w", err)
	}

	adapter := &FileStorageAdapter{
		credentials: make(map[string]*Credential),
		states:      make(map[string]*CredentialState),
		credDir:     config.CredentialsDir,
		stateDir:    config.StatesDir,
		config:      config.Config,
		watchers:    newWatcherHub(),
	}

	if err := adapter.loadAllCredentials(); err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	if err := adapter.loadAllStates(); err != nil {
		return nil, fmt.Errorf("failed to load states: %w", err)
	}

	if config.WatchInterval > 0 {
		go adapter.startFileWatcher(config.WatchInterval)
	}

	return adapter, nil
}

// Ping 检查存储是否可用（文件存储始终可用）
func (f *FileStorageAdapter) Ping(ctx context.Context) error {
	return nil
}

// Close 关闭存储（文件存储无需关闭）
func (f *FileStorageAdapter) Close() error {
	return nil
}

// GetConfig 获取配置
func (f *FileStorageAdapter) GetConfig() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	cfg := make(map[string]interface{})
	for k, v := range f.config {
		cfg[k] = v
	}
	return cfg
}

// SetConfig 设置配置
func (f *FileStorageAdapter) SetConfig(ctx context.Context, config map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.config = config
	return nil
}

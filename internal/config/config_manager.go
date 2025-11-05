package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gcli2api-go/internal/events"

	log "github.com/sirupsen/logrus"
)

// ConfigManager manages configuration file and hot reload
type ConfigManager struct {
	mu         sync.RWMutex
	config     *FileConfig
	configPath string
	watcher    *time.Ticker
	stopCh     chan struct{}
	onChange   []func(*FileConfig)
	lastMod    time.Time
	publisher  events.Publisher
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) (*ConfigManager, error) {
	if configPath == "" {
		locations := []string{
			"config.yaml",
			"config.yml",
			"config.json",
			filepath.Join(os.Getenv("HOME"), ".gcli2api", "config.yaml"),
			filepath.Join(os.Getenv("HOME"), ".gcli2api", "config.yml"),
			"/etc/gcli2api/config.yaml",
		}

		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				configPath = loc
				break
			}
		}
	}

	if strings.HasPrefix(configPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, configPath[1:])
	}

	cm := &ConfigManager{
		configPath: configPath,
		stopCh:     make(chan struct{}),
		onChange:   make([]func(*FileConfig), 0),
	}

	if err := cm.load(); err != nil {
		if os.IsNotExist(err) || configPath == "" {
			cm.config = cm.defaultConfig()
			log.WithField("path", configPath).Warn("using default configuration (no config file found)")
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	cm.mergeEnvVars()

	if cm.configPath != "" {
		if _, err := os.Stat(cm.configPath); err == nil {
			cm.startWatcher()
		}
	}

	return cm, nil
}

// OnChange registers a callback for configuration changes
func (cm *ConfigManager) OnChange(fn func(*FileConfig)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.onChange = append(cm.onChange, fn)
}

// SetEventPublisher wires the event hub used to broadcast config updates.
func (cm *ConfigManager) SetEventPublisher(p events.Publisher) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.publisher = p
}

// GetConfig returns a copy of the current configuration
func (cm *ConfigManager) GetConfig() *FileConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.config == nil {
		return cm.defaultConfig()
	}

	config := *cm.config
	return &config
}

// UpdateConfig updates the configuration and saves to file
func (cm *ConfigManager) UpdateConfig(updates map[string]interface{}) error {
	cm.mu.Lock()
	var oldCopy FileConfig
	if cm.config == nil {
		cm.config = cm.defaultConfig()
		oldCopy = *cm.config
	} else {
		oldCopy = *cm.config
	}

	for key, value := range updates {
		_ = applyFileConfigUpdate(cm.config, key, value)
	}

	newCopy := *cm.config
	var err error
	if cm.configPath != "" {
		err = cm.save()
	}
	cm.mu.Unlock()
	if err != nil {
		return err
	}

	cm.emitChange(&oldCopy, &newCopy)
	return nil
}

// Close stops the configuration manager
func (cm *ConfigManager) Close() {
	close(cm.stopCh)
	if cm.watcher != nil {
		cm.watcher.Stop()
	}
}

func (cm *ConfigManager) listenersSnapshot() ([]func(*FileConfig), events.Publisher, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	callbacks := make([]func(*FileConfig), len(cm.onChange))
	copy(callbacks, cm.onChange)
	return callbacks, cm.publisher, cm.configPath
}

func (cm *ConfigManager) emitChange(oldCfg, newCfg *FileConfig) {
	callbacks, publisher, path := cm.listenersSnapshot()

	for _, fn := range callbacks {
		fn(newCfg)
	}

	if publisher != nil && newCfg != nil {
		event := ConfigChangeEvent{
			Path:      path,
			UpdatedAt: time.Now().UTC(),
			Config:    *newCfg,
		}
		if oldCfg != nil {
			prev := *oldCfg
			event.Previous = &prev
		}
		publisher.Publish(context.Background(), events.TopicConfigUpdated, event, nil)
	}
}

// ConfigChangeEvent is the payload broadcast when configuration changes.
type ConfigChangeEvent struct {
	Path      string      `json:"path"`
	UpdatedAt time.Time   `json:"updated_at"`
	Config    FileConfig  `json:"config"`
	Previous  *FileConfig `json:"previous,omitempty"`
}

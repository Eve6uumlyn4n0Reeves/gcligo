package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func (cm *ConfigManager) load() error {
	if cm.configPath == "" {
		return os.ErrNotExist
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return err
	}

	var config FileConfig
	ext := strings.ToLower(filepath.Ext(cm.configPath))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		if err := yaml.Unmarshal(data, &config); err != nil {
			if err := json.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("failed to parse config file (tried YAML and JSON)")
			}
		}
	}

	if info, err := os.Stat(cm.configPath); err == nil {
		cm.lastMod = info.ModTime()
	}

	cm.config = &config
	log.WithField("path", cm.configPath).Info("configuration loaded")

	return nil
}

func (cm *ConfigManager) save() error {
	if cm.configPath == "" {
		return fmt.Errorf("no config file path set")
	}

	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var (
		data []byte
		err  error
	)

	ext := strings.ToLower(filepath.Ext(cm.configPath))
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cm.config)
	case ".json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
	default:
		data, err = yaml.Marshal(cm.config)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if info, err := os.Stat(cm.configPath); err == nil {
		cm.lastMod = info.ModTime()
	}

	log.WithField("path", cm.configPath).Info("configuration saved")

	return nil
}

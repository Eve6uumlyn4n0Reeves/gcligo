package config

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExportConfig exports the configuration to a writer
func (cm *ConfigManager) ExportConfig(w io.Writer, format string) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	var (
		data []byte
		err  error
	)

	switch strings.ToLower(format) {
	case "yaml", "yml":
		data, err = yaml.Marshal(cm.config)
	case "json":
		data, err = json.MarshalIndent(cm.config, "", "  ")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = w.Write(data)
	return err
}

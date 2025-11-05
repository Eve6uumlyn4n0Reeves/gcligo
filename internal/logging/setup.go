package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gcli2api-go/internal/config"
	log "github.com/sirupsen/logrus"
)

var (
	logMux        sync.Mutex
	logFileHandle *os.File
)

// Setup configures the global logrus logger using runtime configuration.
// It is idempotent and can be called multiple times; the most recent call wins.
func Setup(cfg *config.Config) error {
	logMux.Lock()
	defer logMux.Unlock()

	var formatter log.Formatter = &log.JSONFormatter{TimestampFormat: time.RFC3339Nano}
	if cfg != nil && cfg.Security.Debug {
		formatter = &log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
		}
	}
	log.SetFormatter(formatter)

	level := log.InfoLevel
	if cfg != nil && cfg.Security.Debug {
		level = log.DebugLevel
	}
	log.SetLevel(level)

	var writers []io.Writer
	writers = append(writers, os.Stdout)

	if logFileHandle != nil {
		_ = logFileHandle.Close()
		logFileHandle = nil
	}

	if cfg != nil && cfg.Security.LogFile != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.Security.LogFile), 0o755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
		file, err := os.OpenFile(cfg.Security.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		logFileHandle = file
		writers = append(writers, file)
	}

	log.SetOutput(io.MultiWriter(writers...))
	return nil
}

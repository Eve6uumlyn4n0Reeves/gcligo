package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

func (cm *ConfigManager) startWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.WithError(err).Warn("failed to create file watcher, falling back to polling")
		cm.startPollingWatcher()
		return
	}

	// Watch the config file
	if err := watcher.Add(cm.configPath); err != nil {
		log.WithError(err).WithField("path", cm.configPath).Warn("failed to watch config file, falling back to polling")
		watcher.Close()
		cm.startPollingWatcher()
		return
	}

	// Also watch the directory to catch atomic writes (rename operations)
	configDir := filepath.Dir(cm.configPath)
	if err := watcher.Add(configDir); err != nil {
		log.WithError(err).WithField("dir", configDir).Warn("failed to watch config directory")
	}

	log.WithField("path", cm.configPath).Info("file watcher started using fsnotify")

	go func() {
		defer watcher.Close()

		// Debounce timer to avoid multiple reloads on rapid changes
		var debounceTimer *time.Timer
		debounceDuration := 100 * time.Millisecond

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only react to Write and Create events on our config file
				if event.Name == cm.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
					// Reset debounce timer
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(debounceDuration, func() {
						cm.checkAndReload()
					})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.WithError(err).Warn("file watcher error")

			case <-cm.stopCh:
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				return
			}
		}
	}()
}

// startPollingWatcher is a fallback when fsnotify is not available
func (cm *ConfigManager) startPollingWatcher() {
	ticker := time.NewTicker(5 * time.Second)
	log.WithField("interval", "5s").Info("file watcher started using polling")

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cm.checkAndReload()
			case <-cm.stopCh:
				return
			}
		}
	}()
}

func (cm *ConfigManager) checkAndReload() {
	if cm.configPath == "" {
		return
	}

	info, err := os.Stat(cm.configPath)
	if err != nil {
		return
	}

	if info.ModTime().After(cm.lastMod) {
		oldConfig := cm.GetConfig()

		if err := cm.load(); err != nil {
			log.WithError(err).WithField("path", cm.configPath).Warn("failed to reload config")
			return
		}

		cm.mergeEnvVars()
		newConfig := cm.GetConfig()

		cm.emitChange(oldConfig, newConfig)
		cm.logConfigChanges(oldConfig, newConfig)
	}
}

func (cm *ConfigManager) logConfigChanges(old, new *FileConfig) {
	if old.OpenAIPort != new.OpenAIPort {
		log.WithFields(log.Fields{"field": "openai_port", "old": old.OpenAIPort, "new": new.OpenAIPort}).Info("config changed")
	}
	if old.Debug != new.Debug {
		log.WithFields(log.Fields{"field": "debug", "old": old.Debug, "new": new.Debug}).Info("config changed")
	}
	if old.RequestLog != new.RequestLog {
		log.WithFields(log.Fields{"field": "request_log", "old": old.RequestLog, "new": new.RequestLog}).Info("config changed")
	}
	if old.CallsPerRotation != new.CallsPerRotation {
		log.WithFields(log.Fields{"field": "calls_per_rotation", "old": old.CallsPerRotation, "new": new.CallsPerRotation}).Info("config changed")
	}
	if old.AuthDir != new.AuthDir {
		log.WithFields(log.Fields{"field": "auth_dir", "old": old.AuthDir, "new": new.AuthDir}).Info("config changed")
	}
}

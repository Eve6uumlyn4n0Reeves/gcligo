package config

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

func (cm *ConfigManager) startWatcher() {
	cm.watcher = time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-cm.watcher.C:
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

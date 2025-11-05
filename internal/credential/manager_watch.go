package credential

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

// WatchAuthDirectory enables hot-reload for credential files within authDir.
func (m *Manager) WatchAuthDirectory(ctx context.Context) {
	if m.authDir == "" {
		return
	}
	m.watchOnce.Do(func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.WithError(err).Warn("credential manager: failed to start file watcher")
			return
		}
		if err := watcher.Add(m.authDir); err != nil {
			log.WithError(err).Warnf("credential manager: failed to watch %s", m.authDir)
			_ = watcher.Close()
			return
		}
		m.watcher = watcher
		go m.reloadLoop(ctx)
		go m.watchLoop(ctx, watcher)
		log.Infof("credential manager: watching %s for changes", m.authDir)
	})
}

func (m *Manager) watchLoop(ctx context.Context, watcher *fsnotify.Watcher) {
	defer watcher.Close()
	for {
		select {
		case evt, ok := <-watcher.Events:
			if !ok {
				return
			}
			if m.shouldReloadForEvent(evt.Name) {
				m.requestReload()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.WithError(err).Warn("credential watcher error")
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) reloadLoop(ctx context.Context) {
	var timer *time.Timer
	var timerCh <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-m.reloadCh:
			if timer == nil {
				timer = time.NewTimer(watchDebounceInterval)
				timerCh = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(watchDebounceInterval)
			}
		case <-timerCh:
			if err := m.LoadCredentials(); err != nil {
				log.WithError(err).Warn("credential manager: auto reload failed")
			}
			timerCh = nil
			timer.Stop()
			timer = nil
		}
	}
}

func (m *Manager) requestReload() {
	select {
	case m.reloadCh <- struct{}{}:
	default:
	}
}

func (m *Manager) shouldReloadForEvent(name string) bool {
	if name == "" {
		return true
	}
	base := strings.ToLower(filepath.Base(name))
	if strings.HasSuffix(base, credentialStateSuffix) {
		return false
	}
	return strings.HasSuffix(base, ".json")
}

func (m *Manager) stateFilePath(id string) string {
	if m.authDir == "" || id == "" {
		return ""
	}
	base := strings.TrimSuffix(id, filepath.Ext(id))
	return filepath.Join(m.authDir, base+credentialStateSuffix)
}

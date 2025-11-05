package adapter

import "sync"

type watcherHub struct {
	mu       sync.RWMutex
	watchers []func([]*Credential)
}

func newWatcherHub() *watcherHub {
	return &watcherHub{
		watchers: make([]func([]*Credential), 0),
	}
}

func (h *watcherHub) Add(watcher func([]*Credential)) {
	if watcher == nil {
		return
	}
	h.mu.Lock()
	h.watchers = append(h.watchers, watcher)
	h.mu.Unlock()
}

func (h *watcherHub) Notify(credentials []*Credential) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.watchers) == 0 {
		return
	}
	for _, watcher := range h.watchers {
		go watcher(credentials)
	}
}

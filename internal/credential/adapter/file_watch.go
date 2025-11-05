package adapter

import (
	"context"
	"time"
)

func (f *FileStorageAdapter) startFileWatcher(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := f.loadAllCredentials(); err != nil {
			continue
		}
		if err := f.loadAllStates(); err != nil {
			continue
		}
		f.notifyWatchers()
	}
}

func (f *FileStorageAdapter) notifyWatchers() {
	if f.watchers == nil {
		return
	}

	creds, err := f.GetAllCredentials(context.Background())
	if err != nil {
		return
	}
	f.watchers.Notify(creds)
}

// AddWatcher 添加凭证变化观察者
func (f *FileStorageAdapter) AddWatcher(watcher func([]*Credential)) {
	if f.watchers == nil {
		f.watchers = newWatcherHub()
	}
	f.watchers.Add(watcher)
}

package adapter

import "context"

// notifyWatchers 通知所有观察者
func (r *RedisStorageAdapter) notifyWatchers() {
	credentials, err := r.GetAllCredentials(context.Background())
	if err != nil {
		return
	}

	if r.watchers != nil {
		r.watchers.Notify(credentials)
	}
}

// AddWatcher 添加凭证变化观察者
func (r *RedisStorageAdapter) AddWatcher(watcher func([]*Credential)) {
	if r.watchers == nil {
		r.watchers = newWatcherHub()
	}
	r.watchers.Add(watcher)
}

package adapter

import "context"

// notifyWatchers 通知所有观察者
func (m *MongoDBStorageAdapter) notifyWatchers() {
	credentials, err := m.GetAllCredentials(context.Background())
	if err != nil {
		return
	}

	if m.watchers != nil {
		m.watchers.Notify(credentials)
	}
}

// AddWatcher 添加凭证变化观察者
func (m *MongoDBStorageAdapter) AddWatcher(watcher func([]*Credential)) {
	if m.watchers == nil {
		m.watchers = newWatcherHub()
	}
	m.watchers.Add(watcher)
}

package adapter

import (
	"sync"
	"testing"
	"time"
)

func TestWatcherHub(t *testing.T) {
	t.Run("Create new hub", func(t *testing.T) {
		hub := newWatcherHub()
		if hub == nil {
			t.Fatal("Expected non-nil hub")
		}

		if hub.watchers == nil {
			t.Error("Expected initialized watchers slice")
		}

		if len(hub.watchers) != 0 {
			t.Errorf("Expected empty watchers, got %d", len(hub.watchers))
		}
	})

	t.Run("Add watcher", func(t *testing.T) {
		hub := newWatcherHub()

		watcher := func(creds []*Credential) {
			// Watcher function
		}

		hub.Add(watcher)

		if len(hub.watchers) != 1 {
			t.Errorf("Expected 1 watcher, got %d", len(hub.watchers))
		}
	})

	t.Run("Add nil watcher", func(t *testing.T) {
		hub := newWatcherHub()
		hub.Add(nil)

		if len(hub.watchers) != 0 {
			t.Errorf("Expected 0 watchers after adding nil, got %d", len(hub.watchers))
		}
	})

	t.Run("Notify watchers", func(t *testing.T) {
		hub := newWatcherHub()
		var wg sync.WaitGroup
		callCount := 0
		var mu sync.Mutex

		watcher := func(creds []*Credential) {
			mu.Lock()
			callCount++
			mu.Unlock()
			wg.Done()
		}

		hub.Add(watcher)
		hub.Add(watcher)

		credentials := []*Credential{
			{ID: "cred1"},
			{ID: "cred2"},
		}

		wg.Add(2)
		hub.Notify(credentials)
		wg.Wait()

		mu.Lock()
		if callCount != 2 {
			t.Errorf("Expected 2 watcher calls, got %d", callCount)
		}
		mu.Unlock()
	})

	t.Run("Notify with no watchers", func(t *testing.T) {
		hub := newWatcherHub()
		credentials := []*Credential{{ID: "cred1"}}

		// Should not panic
		hub.Notify(credentials)
	})

	t.Run("Notify with empty credentials", func(t *testing.T) {
		hub := newWatcherHub()
		called := false
		var wg sync.WaitGroup

		watcher := func(creds []*Credential) {
			called = true
			if len(creds) != 0 {
				t.Error("Expected empty credentials slice")
			}
			wg.Done()
		}

		hub.Add(watcher)
		wg.Add(1)
		hub.Notify([]*Credential{})
		wg.Wait()

		if !called {
			t.Error("Expected watcher to be called")
		}
	})

	t.Run("Concurrent add and notify", func(t *testing.T) {
		hub := newWatcherHub()
		var wg sync.WaitGroup

		// Add watchers concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hub.Add(func(creds []*Credential) {})
			}()
		}

		wg.Wait()

		if len(hub.watchers) != 10 {
			t.Errorf("Expected 10 watchers, got %d", len(hub.watchers))
		}

		// Notify concurrently
		for i := 0; i < 5; i++ {
			go hub.Notify([]*Credential{{ID: "test"}})
		}

		// Give goroutines time to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Watcher receives correct credentials", func(t *testing.T) {
		hub := newWatcherHub()
		var wg sync.WaitGroup
		var receivedCreds []*Credential
		var mu sync.Mutex

		watcher := func(creds []*Credential) {
			mu.Lock()
			receivedCreds = creds
			mu.Unlock()
			wg.Done()
		}

		hub.Add(watcher)

		credentials := []*Credential{
			{ID: "cred1", Name: "Test 1"},
			{ID: "cred2", Name: "Test 2"},
		}

		wg.Add(1)
		hub.Notify(credentials)
		wg.Wait()

		mu.Lock()
		if len(receivedCreds) != 2 {
			t.Errorf("Expected 2 credentials, got %d", len(receivedCreds))
		}

		if receivedCreds[0].ID != "cred1" {
			t.Errorf("Expected cred1, got %s", receivedCreds[0].ID)
		}

		if receivedCreds[1].Name != "Test 2" {
			t.Errorf("Expected 'Test 2', got %s", receivedCreds[1].Name)
		}
		mu.Unlock()
	})
}

package common

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBatchProcessor_BatchGet(t *testing.T) {
	bp := NewBatchProcessor(5)
	ctx := context.Background()

	// Mock get function
	getFunc := func(ctx context.Context, id string) (map[string]interface{}, error) {
		if id == "error" {
			return nil, errors.New("get error")
		}
		return map[string]interface{}{"id": id, "value": "test"}, nil
	}

	tests := []struct {
		name     string
		ids      []string
		wantSucc int
		wantFail int
	}{
		{
			name:     "all success",
			ids:      []string{"id1", "id2", "id3"},
			wantSucc: 3,
			wantFail: 0,
		},
		{
			name:     "mixed results",
			ids:      []string{"id1", "error", "id3"},
			wantSucc: 2,
			wantFail: 1,
		},
		{
			name:     "empty ids",
			ids:      []string{},
			wantSucc: 0,
			wantFail: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := bp.BatchGet(ctx, tt.ids, getFunc)

			if len(results) != len(tt.ids) {
				t.Errorf("BatchGet() returned %d results, want %d", len(results), len(tt.ids))
			}

			success, failed, _ := bp.SummarizeResults(results)
			if success != tt.wantSucc {
				t.Errorf("BatchGet() success = %d, want %d", success, tt.wantSucc)
			}
			if failed != tt.wantFail {
				t.Errorf("BatchGet() failed = %d, want %d", failed, tt.wantFail)
			}
		})
	}
}

func TestBatchProcessor_BatchSet(t *testing.T) {
	bp := NewBatchProcessor(5)
	ctx := context.Background()

	// Mock set function
	setFunc := func(ctx context.Context, id string, data map[string]interface{}) error {
		if id == "error" {
			return errors.New("set error")
		}
		return nil
	}

	tests := []struct {
		name     string
		items    map[string]map[string]interface{}
		wantSucc int
		wantFail int
	}{
		{
			name: "all success",
			items: map[string]map[string]interface{}{
				"id1": {"value": "test1"},
				"id2": {"value": "test2"},
			},
			wantSucc: 2,
			wantFail: 0,
		},
		{
			name: "mixed results",
			items: map[string]map[string]interface{}{
				"id1":   {"value": "test1"},
				"error": {"value": "test2"},
			},
			wantSucc: 1,
			wantFail: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := bp.BatchSet(ctx, tt.items, setFunc)

			success, failed, _ := bp.SummarizeResults(results)
			if success != tt.wantSucc {
				t.Errorf("BatchSet() success = %d, want %d", success, tt.wantSucc)
			}
			if failed != tt.wantFail {
				t.Errorf("BatchSet() failed = %d, want %d", failed, tt.wantFail)
			}
		})
	}
}

func TestBatchProcessor_BatchDelete(t *testing.T) {
	bp := NewBatchProcessor(5)
	ctx := context.Background()

	// Mock delete function
	deleteFunc := func(ctx context.Context, id string) error {
		if id == "error" {
			return errors.New("delete error")
		}
		return nil
	}

	tests := []struct {
		name     string
		ids      []string
		wantSucc int
		wantFail int
	}{
		{
			name:     "all success",
			ids:      []string{"id1", "id2", "id3"},
			wantSucc: 3,
			wantFail: 0,
		},
		{
			name:     "mixed results",
			ids:      []string{"id1", "error", "id3"},
			wantSucc: 2,
			wantFail: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := bp.BatchDelete(ctx, tt.ids, deleteFunc)

			success, failed, _ := bp.SummarizeResults(results)
			if success != tt.wantSucc {
				t.Errorf("BatchDelete() success = %d, want %d", success, tt.wantSucc)
			}
			if failed != tt.wantFail {
				t.Errorf("BatchDelete() failed = %d, want %d", failed, tt.wantFail)
			}
		})
	}
}

func TestBatchProcessor_FilterResults(t *testing.T) {
	bp := NewBatchProcessor(5)

	results := []BatchResult{
		{ID: "id1", Success: true},
		{ID: "id2", Success: false, Error: errors.New("error")},
		{ID: "id3", Success: true},
	}

	successful := bp.FilterSuccessful(results)
	if len(successful) != 2 {
		t.Errorf("FilterSuccessful() returned %d results, want 2", len(successful))
	}

	failed := bp.FilterFailed(results)
	if len(failed) != 1 {
		t.Errorf("FilterFailed() returned %d results, want 1", len(failed))
	}

	successIDs := bp.GetSuccessfulIDs(results)
	if len(successIDs) != 2 {
		t.Errorf("GetSuccessfulIDs() returned %d IDs, want 2", len(successIDs))
	}

	failedIDs := bp.GetFailedIDs(results)
	if len(failedIDs) != 1 {
		t.Errorf("GetFailedIDs() returned %d IDs, want 1", len(failedIDs))
	}
}

func TestBatchProcessor_ChunkIDs(t *testing.T) {
	bp := NewBatchProcessor(5)

	tests := []struct {
		name       string
		ids        []string
		chunkSize  int
		wantChunks int
	}{
		{
			name:       "exact chunks",
			ids:        []string{"1", "2", "3", "4", "5", "6"},
			chunkSize:  2,
			wantChunks: 3,
		},
		{
			name:       "uneven chunks",
			ids:        []string{"1", "2", "3", "4", "5"},
			chunkSize:  2,
			wantChunks: 3,
		},
		{
			name:       "single chunk",
			ids:        []string{"1", "2"},
			chunkSize:  10,
			wantChunks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := bp.ChunkIDs(tt.ids, tt.chunkSize)
			if len(chunks) != tt.wantChunks {
				t.Errorf("ChunkIDs() returned %d chunks, want %d", len(chunks), tt.wantChunks)
			}
		})
	}
}

func TestBatchProcessor_Concurrency(t *testing.T) {
	bp := NewBatchProcessor(2) // Low concurrency to test limiting
	ctx := context.Background()

	callCount := 0
	var mu sync.Mutex

	getFunc := func(ctx context.Context, id string) (map[string]interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate work
		return map[string]interface{}{"id": id}, nil
	}

	ids := []string{"1", "2", "3", "4", "5"}
	results := bp.BatchGet(ctx, ids, getFunc)

	if len(results) != len(ids) {
		t.Errorf("BatchGet() returned %d results, want %d", len(results), len(ids))
	}

	mu.Lock()
	if callCount != len(ids) {
		t.Errorf("getFunc called %d times, want %d", callCount, len(ids))
	}
	mu.Unlock()
}

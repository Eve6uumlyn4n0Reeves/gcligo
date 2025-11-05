package management

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type batchJobStatus string

const (
	jobStatusPending   batchJobStatus = "pending"
	jobStatusRunning   batchJobStatus = "running"
	jobStatusCompleted batchJobStatus = "completed"
	jobStatusFailed    batchJobStatus = "failed"
	jobStatusCancelled batchJobStatus = "cancelled"
)

type batchJob struct {
	id          string
	operation   batchOperation
	status      batchJobStatus
	total       int
	completed   int
	success     int
	failure     int
	createdAt   time.Time
	startedAt   *time.Time
	completedAt *time.Time
	results     []batchResult
	errMsg      string

	ctx    context.Context
	cancel context.CancelFunc

	mu sync.RWMutex
}

func (t *batchJob) snapshot(includeResults bool) BatchTaskSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	progress := 0.0
	if t.total > 0 {
		progress = float64(t.completed) / float64(t.total) * 100
	}
	snap := BatchTaskSnapshot{
		ID:          t.id,
		Operation:   string(t.operation),
		Status:      string(t.status),
		Total:       t.total,
		Completed:   t.completed,
		Success:     t.success,
		Failure:     t.failure,
		Progress:    progress,
		CreatedAt:   t.createdAt,
		StartedAt:   t.startedAt,
		CompletedAt: t.completedAt,
		Error:       t.errMsg,
	}
	if includeResults {
		snap.Results = make([]batchResult, len(t.results))
		copy(snap.Results, t.results)
	}
	return snap
}

func (t *batchJob) markRunning() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.status = jobStatusRunning
	t.startedAt = &now
}

func (t *batchJob) updateProgress(completed, success, failure int, result batchResult) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.completed = completed
	t.success = success
	t.failure = failure
	if result.id != "" {
		t.results = append(t.results, result)
	}
}

func (t *batchJob) complete(output batchProcessOutput) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = jobStatusCompleted
	now := time.Now()
	t.completedAt = &now
	t.completed = output.successCount + output.failureCount
	t.success = output.successCount
	t.failure = output.failureCount
	t.results = output.results
}

func (t *batchJob) fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = jobStatusFailed
	now := time.Now()
	t.completedAt = &now
	if err != nil {
		t.errMsg = err.Error()
	}
}

func (t *batchJob) cancelTask() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status == jobStatusCompleted || t.status == jobStatusFailed {
		return
	}
	t.status = jobStatusCancelled
	now := time.Now()
	t.completedAt = &now
	if t.cancel != nil {
		t.cancel()
	}
}

type BatchTaskSnapshot struct {
	ID          string        `json:"id"`
	Operation   string        `json:"operation"`
	Status      string        `json:"status"`
	Total       int           `json:"total"`
	Completed   int           `json:"completed"`
	Success     int           `json:"success"`
	Failure     int           `json:"failure"`
	Progress    float64       `json:"progress"`
	CreatedAt   time.Time     `json:"created_at"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Error       string        `json:"error,omitempty"`
	Results     []batchResult `json:"results,omitempty"`
}

type BatchTaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*batchJob
}

func NewBatchTaskManager() *BatchTaskManager {
	return &BatchTaskManager{
		tasks: make(map[string]*batchJob),
	}
}

func (m *BatchTaskManager) CreateTask(op batchOperation, total int) *batchJob {
	ctx, cancel := context.WithCancel(context.Background())
	task := &batchJob{
		id:        uuid.NewString(),
		operation: op,
		status:    jobStatusPending,
		total:     total,
		createdAt: time.Now(),
		results:   make([]batchResult, 0, total),
		ctx:       ctx,
		cancel:    cancel,
	}
	m.mu.Lock()
	m.tasks[task.id] = task
	m.mu.Unlock()
	return task
}

func (m *BatchTaskManager) GetTask(id string) (*batchJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	task, ok := m.tasks[id]
	return task, ok
}

func (m *BatchTaskManager) MarkRunning(id string) {
	if task, ok := m.GetTask(id); ok {
		task.markRunning()
	}
}

func (m *BatchTaskManager) UpdateProgress(id string, completed, success, failure int, result batchResult) {
	if task, ok := m.GetTask(id); ok {
		task.updateProgress(completed, success, failure, result)
	}
}

func (m *BatchTaskManager) CompleteTask(id string, output batchProcessOutput) {
	if task, ok := m.GetTask(id); ok {
		task.complete(output)
	}
}

func (m *BatchTaskManager) FailTask(id string, err error) {
	if task, ok := m.GetTask(id); ok {
		task.fail(err)
	}
}

func (m *BatchTaskManager) CancelTask(id string) error {
	task, ok := m.GetTask(id)
	if !ok {
		return errors.New("task not found")
	}
	task.cancelTask()
	return nil
}

func (m *BatchTaskManager) Snapshot(id string, includeResults bool) (BatchTaskSnapshot, error) {
	task, ok := m.GetTask(id)
	if !ok {
		return BatchTaskSnapshot{}, errors.New("task not found")
	}
	return task.snapshot(includeResults), nil
}

func (m *BatchTaskManager) ListSnapshots() []BatchTaskSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snapshots := make([]BatchTaskSnapshot, 0, len(m.tasks))
	for _, task := range m.tasks {
		snapshots = append(snapshots, task.snapshot(false))
	}
	return snapshots
}

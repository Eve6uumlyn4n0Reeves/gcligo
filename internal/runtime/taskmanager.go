package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Task represents a background task
type Task struct {
	Name        string
	Description string
	StartTime   time.Time
	Status      TaskStatus
	Error       error
	cancel      context.CancelFunc
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusStopped  TaskStatus = "stopped"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusCanceled TaskStatus = "canceled"
)

// TaskFunc is a function that runs as a background task
type TaskFunc func(ctx context.Context) error

// TaskManager manages background tasks and their lifecycle
type TaskManager struct {
	tasks  map[string]*Task
	mu     sync.RWMutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTaskManager creates a new task manager
func NewTaskManager(ctx context.Context) *TaskManager {
	ctx, cancel := context.WithCancel(ctx)
	return &TaskManager{
		tasks:  make(map[string]*Task),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts a new background task
func (tm *TaskManager) Start(name, description string, fn TaskFunc) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[name]; exists {
		return fmt.Errorf("task %s already exists", name)
	}

	taskCtx, taskCancel := context.WithCancel(tm.ctx)
	task := &Task{
		Name:        name,
		Description: description,
		StartTime:   time.Now(),
		Status:      TaskStatusRunning,
		cancel:      taskCancel,
	}
	tm.tasks[name] = task

	tm.wg.Add(1)
	go func() {
		defer tm.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(log.Fields{
					"task":  name,
					"panic": r,
				}).Error("Task panicked")
				tm.mu.Lock()
				task.Status = TaskStatusFailed
				task.Error = fmt.Errorf("panic: %v", r)
				tm.mu.Unlock()
			}
		}()

		log.WithFields(log.Fields{
			"task":        name,
			"description": description,
		}).Info("Task started")

		err := fn(taskCtx)

		tm.mu.Lock()
		if err != nil {
			if taskCtx.Err() == context.Canceled {
				task.Status = TaskStatusCanceled
			} else {
				task.Status = TaskStatusFailed
				task.Error = err
				log.WithFields(log.Fields{
					"task":  name,
					"error": err,
				}).Error("Task failed")
			}
		} else {
			task.Status = TaskStatusStopped
			log.WithFields(log.Fields{
				"task": name,
			}).Info("Task stopped")
		}
		tm.mu.Unlock()
	}()

	return nil
}

// Stop stops a specific task
func (tm *TaskManager) Stop(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[name]
	if !exists {
		return fmt.Errorf("task %s not found", name)
	}

	if task.Status != TaskStatusRunning {
		return fmt.Errorf("task %s is not running", name)
	}

	task.cancel()
	return nil
}

// StopAll stops all running tasks
func (tm *TaskManager) StopAll() {
	tm.cancel()
}

// Wait waits for all tasks to complete
func (tm *TaskManager) Wait() {
	tm.wg.Wait()
}

// GetTask returns information about a specific task
func (tm *TaskManager) GetTask(name string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[name]
	if !exists {
		return nil, fmt.Errorf("task %s not found", name)
	}

	// Return a copy to avoid race conditions
	return &Task{
		Name:        task.Name,
		Description: task.Description,
		StartTime:   task.StartTime,
		Status:      task.Status,
		Error:       task.Error,
	}, nil
}

// ListTasks returns a list of all tasks
func (tm *TaskManager) ListTasks() []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, &Task{
			Name:        task.Name,
			Description: task.Description,
			StartTime:   task.StartTime,
			Status:      task.Status,
			Error:       task.Error,
		})
	}
	return tasks
}

// GetStats returns statistics about tasks
func (tm *TaskManager) GetStats() TaskStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := TaskStats{
		Total: len(tm.tasks),
	}

	for _, task := range tm.tasks {
		switch task.Status {
		case TaskStatusRunning:
			stats.Running++
		case TaskStatusStopped:
			stats.Stopped++
		case TaskStatusFailed:
			stats.Failed++
		case TaskStatusCanceled:
			stats.Canceled++
		}
	}

	return stats
}

// TaskStats contains statistics about tasks
type TaskStats struct {
	Total    int `json:"total"`
	Running  int `json:"running"`
	Stopped  int `json:"stopped"`
	Failed   int `json:"failed"`
	Canceled int `json:"canceled"`
}

// StartPeriodic starts a periodic task that runs at the specified interval
func (tm *TaskManager) StartPeriodic(name, description string, interval time.Duration, fn func(ctx context.Context) error) error {
	return tm.Start(name, description, func(ctx context.Context) error {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run immediately
		if err := fn(ctx); err != nil {
			log.WithFields(log.Fields{
				"task":  name,
				"error": err,
			}).Warn("Periodic task execution failed")
		}

		for {
			select {
			case <-ticker.C:
				if err := fn(ctx); err != nil {
					log.WithFields(log.Fields{
						"task":  name,
						"error": err,
					}).Warn("Periodic task execution failed")
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
}

// StartDelayed starts a task after a delay
func (tm *TaskManager) StartDelayed(name, description string, delay time.Duration, fn TaskFunc) error {
	return tm.Start(name, description, func(ctx context.Context) error {
		select {
		case <-time.After(delay):
			return fn(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}


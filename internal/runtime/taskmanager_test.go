package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewTaskManager(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)
	if tm == nil {
		t.Fatal("NewTaskManager returned nil")
	}
	if tm.tasks == nil {
		t.Error("tasks map not initialized")
	}
}

func TestTaskManager_Start(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	called := false
	err := tm.Start("test-task", "Test task", func(ctx context.Context) error {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	// Wait a bit for task to execute
	time.Sleep(100 * time.Millisecond)

	if !called {
		t.Error("Task function was not called")
	}

	task, err := tm.GetTask("test-task")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if task.Name != "test-task" {
		t.Errorf("Expected task name 'test-task', got '%s'", task.Name)
	}
}

func TestTaskManager_StartDuplicate(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	err := tm.Start("test-task", "Test task", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to start first task: %v", err)
	}

	// Try to start duplicate
	err = tm.Start("test-task", "Test task", func(ctx context.Context) error {
		return nil
	})
	if err == nil {
		t.Error("Expected error when starting duplicate task")
	}

	tm.StopAll()
	tm.Wait()
}

func TestTaskManager_Stop(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	err := tm.Start("test-task", "Test task", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	// Wait a bit for task to start
	time.Sleep(50 * time.Millisecond)

	err = tm.Stop("test-task")
	if err != nil {
		t.Fatalf("Failed to stop task: %v", err)
	}

	// Wait for task to finish
	tm.Wait()

	task, err := tm.GetTask("test-task")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if task.Status != TaskStatusCanceled {
		t.Errorf("Expected status 'canceled', got '%s'", task.Status)
	}
}

func TestTaskManager_StopAll(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	// Start multiple tasks
	for i := 0; i < 5; i++ {
		name := "test-task-" + string(rune('0'+i))
		err := tm.Start(name, "Test task", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil {
			t.Fatalf("Failed to start task %s: %v", name, err)
		}
	}

	// Wait a bit for tasks to start
	time.Sleep(50 * time.Millisecond)

	tm.StopAll()
	tm.Wait()

	stats := tm.GetStats()
	if stats.Total != 5 {
		t.Errorf("Expected 5 total tasks, got %d", stats.Total)
	}
	if stats.Canceled != 5 {
		t.Errorf("Expected 5 canceled tasks, got %d", stats.Canceled)
	}
}

func TestTaskManager_TaskError(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	expectedErr := errors.New("task error")
	err := tm.Start("test-task", "Test task", func(ctx context.Context) error {
		return expectedErr
	})
	if err != nil {
		t.Fatalf("Failed to start task: %v", err)
	}

	// Wait for task to finish
	time.Sleep(100 * time.Millisecond)

	task, err := tm.GetTask("test-task")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	if task.Status != TaskStatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", task.Status)
	}
	if task.Error == nil {
		t.Error("Expected task error to be set")
	}
}

func TestTaskManager_ListTasks(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	// Start multiple tasks
	for i := 0; i < 3; i++ {
		name := "test-task-" + string(rune('0'+i))
		err := tm.Start(name, "Test task", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil {
			t.Fatalf("Failed to start task %s: %v", name, err)
		}
	}

	tasks := tm.ListTasks()
	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	tm.StopAll()
	tm.Wait()
}

func TestTaskManager_GetStats(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	// Start a running task
	err := tm.Start("running-task", "Running task", func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if err != nil {
		t.Fatalf("Failed to start running task: %v", err)
	}

	// Start a task that will fail
	err = tm.Start("failing-task", "Failing task", func(ctx context.Context) error {
		return errors.New("task error")
	})
	if err != nil {
		t.Fatalf("Failed to start failing task: %v", err)
	}

	// Wait for failing task to finish
	time.Sleep(100 * time.Millisecond)

	stats := tm.GetStats()
	if stats.Total != 2 {
		t.Errorf("Expected 2 total tasks, got %d", stats.Total)
	}
	if stats.Running != 1 {
		t.Errorf("Expected 1 running task, got %d", stats.Running)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed task, got %d", stats.Failed)
	}

	tm.StopAll()
	tm.Wait()
}

func TestTaskManager_StartPeriodic(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	count := 0
	err := tm.StartPeriodic("periodic-task", "Periodic task", 50*time.Millisecond, func(ctx context.Context) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to start periodic task: %v", err)
	}

	// Wait for a few executions
	time.Sleep(200 * time.Millisecond)

	if count < 3 {
		t.Errorf("Expected at least 3 executions, got %d", count)
	}

	tm.StopAll()
	tm.Wait()
}

func TestTaskManager_StartDelayed(t *testing.T) {
	ctx := context.Background()
	tm := NewTaskManager(ctx)

	executed := false
	startTime := time.Now()
	err := tm.StartDelayed("delayed-task", "Delayed task", 100*time.Millisecond, func(ctx context.Context) error {
		executed = true
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to start delayed task: %v", err)
	}

	// Wait for task to execute
	tm.Wait()

	if !executed {
		t.Error("Delayed task was not executed")
	}

	elapsed := time.Since(startTime)
	if elapsed < 100*time.Millisecond {
		t.Errorf("Task executed too early: %v", elapsed)
	}
}


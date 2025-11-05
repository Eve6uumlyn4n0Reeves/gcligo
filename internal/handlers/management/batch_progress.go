package management

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *AdminAPIHandler) ensureTaskManager() *BatchTaskManager {
	if h.taskManager == nil {
		h.taskManager = NewBatchTaskManager()
	}
	return h.taskManager
}

// ListBatchTasks returns summaries for all active/complete batch tasks.
func (h *AdminAPIHandler) ListBatchTasks(c *gin.Context) {
	manager := h.ensureTaskManager()
	snapshots := manager.ListSnapshots()
	c.JSON(http.StatusOK, gin.H{
		"tasks": snapshots,
		"total": len(snapshots),
	})
}

// GetBatchTask returns a specific batch task snapshot.
func (h *AdminAPIHandler) GetBatchTask(c *gin.Context) {
	manager := h.ensureTaskManager()
	taskID := c.Param("taskId")
	snapshot, err := manager.Snapshot(taskID, false)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// GetBatchTaskResult returns final results (only when task is done).
func (h *AdminAPIHandler) GetBatchTaskResult(c *gin.Context) {
	manager := h.ensureTaskManager()
	taskID := c.Param("taskId")
	snapshot, err := manager.Snapshot(taskID, true)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	switch snapshot.Status {
	case string(jobStatusCompleted), string(jobStatusFailed), string(jobStatusCancelled):
		c.JSON(http.StatusOK, snapshot)
	default:
		respondError(c, http.StatusBadRequest, "task not completed yet")
	}
}

// StreamBatchTaskProgress emits SSE updates for a given task.
func (h *AdminAPIHandler) StreamBatchTaskProgress(c *gin.Context) {
	manager := h.ensureTaskManager()
	taskID := c.Param("taskId")
	if _, err := manager.Snapshot(taskID, false); err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	streamCtx := c.Request.Context()
	for {
		select {
		case <-streamCtx.Done():
			return
		case <-ticker.C:
			snapshot, err := manager.Snapshot(taskID, false)
			if err != nil {
				c.SSEvent("error", gin.H{"error": err.Error()})
				c.Writer.Flush()
				return
			}
			c.SSEvent("progress", snapshot)
			c.Writer.Flush()
			if snapshot.Status == string(jobStatusCompleted) ||
				snapshot.Status == string(jobStatusFailed) ||
				snapshot.Status == string(jobStatusCancelled) {
				c.SSEvent("done", snapshot)
				c.Writer.Flush()
				return
			}
		}
	}
}

// CancelBatchTask cancels a running task.
func (h *AdminAPIHandler) CancelBatchTask(c *gin.Context) {
	manager := h.ensureTaskManager()
	taskID := c.Param("taskId")
	if err := manager.CancelTask(taskID); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"task_id": taskID, "status": "cancelled"})
}

package management

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/storage"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	defaultBatchConcurrency = 10
	maxBatchConcurrency     = 50
	defaultBatchTimeout     = 5 * time.Minute
	progressBuckets         = 5
	asyncBatchThreshold     = 50
	batchChunkSize          = 25
)

type batchOperation string

const (
	batchOpEnable  batchOperation = "enable"
	batchOpDisable batchOperation = "disable"
	batchOpDelete  batchOperation = "delete"
	batchOpRecover batchOperation = "recover"
)

type batchTask struct {
	start int
	ids   []string
}

type batchResult struct {
	index   int
	id      string
	success bool
	errMsg  string
}

type batchProgress struct {
	Completed    int       `json:"completed"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	Timestamp    time.Time `json:"timestamp"`
}

type batchProcessOutput struct {
	results               []batchResult
	progress              []batchProgress
	duration              time.Duration
	successCount          int
	failureCount          int
	cancelledDueToTimeout bool
}

// BatchEnableCredentials enables multiple credentials at once (concurrent version with rate limiting).
func (h *AdminAPIHandler) BatchEnableCredentials(c *gin.Context) {
	var req struct {
		IDs         []string `json:"ids" binding:"required"`
		Concurrency *int     `json:"concurrency,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if h.batchLimiter == nil {
		h.batchLimiter = NewBatchLimiter(DefaultBatchLimitConfig)
	}

	if allowed, msg, retryAfter := h.batchLimiter.CheckRequest(string(batchOpEnable), len(req.IDs)); !allowed {
		setRetryAfter(c, retryAfter)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "rate_limit_exceeded",
			"message":     msg,
			"retry_after": retryAfter.Seconds(),
		})
		return
	}

	concurrency := selectConcurrency(req.Concurrency, len(req.IDs))

	operation := func(ctx context.Context, ids []string) []credential.BatchOperationResult {
		return h.credMgr.BatchEnableCredentials(ctx, ids)
	}

	if h.shouldRunAsync(len(req.IDs)) {
		h.startAsyncBatch(c, req.IDs, concurrency, batchOpEnable, operation)
		h.batchLimiter.RecordSuccess(string(batchOpEnable), len(req.IDs))
		return
	}

	output := h.processBatchConcurrently(
		c.Request.Context(),
		req.IDs,
		concurrency,
		batchOpEnable,
		operation,
		nil,
	)
	h.batchLimiter.RecordSuccess(string(batchOpEnable), len(req.IDs))
	sendBatchResponse(c, batchOpEnable, concurrency, output)
}

// BatchDisableCredentials disables multiple credentials at once (concurrent version with rate limiting).
func (h *AdminAPIHandler) BatchDisableCredentials(c *gin.Context) {
	var req struct {
		IDs         []string `json:"ids" binding:"required"`
		Concurrency *int     `json:"concurrency,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if h.batchLimiter == nil {
		h.batchLimiter = NewBatchLimiter(DefaultBatchLimitConfig)
	}

	if allowed, msg, retryAfter := h.batchLimiter.CheckRequest(string(batchOpDisable), len(req.IDs)); !allowed {
		setRetryAfter(c, retryAfter)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "rate_limit_exceeded",
			"message":     msg,
			"retry_after": retryAfter.Seconds(),
		})
		return
	}

	concurrency := selectConcurrency(req.Concurrency, len(req.IDs))

	operation := func(ctx context.Context, ids []string) []credential.BatchOperationResult {
		return h.credMgr.BatchDisableCredentials(ctx, ids)
	}

	if h.shouldRunAsync(len(req.IDs)) {
		h.startAsyncBatch(c, req.IDs, concurrency, batchOpDisable, operation)
		h.batchLimiter.RecordSuccess(string(batchOpDisable), len(req.IDs))
		return
	}

	output := h.processBatchConcurrently(
		c.Request.Context(),
		req.IDs,
		concurrency,
		batchOpDisable,
		operation,
		nil,
	)
	h.batchLimiter.RecordSuccess(string(batchOpDisable), len(req.IDs))
	sendBatchResponse(c, batchOpDisable, concurrency, output)
}

// BatchDeleteCredentials deletes multiple credentials at once (concurrent version with rate limiting).
func (h *AdminAPIHandler) BatchDeleteCredentials(c *gin.Context) {
	var req struct {
		IDs         []string `json:"ids" binding:"required"`
		Concurrency *int     `json:"concurrency,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if h.batchLimiter == nil {
		h.batchLimiter = NewBatchLimiter(DefaultBatchLimitConfig)
	}

	if allowed, msg, retryAfter := h.batchLimiter.CheckRequest(string(batchOpDelete), len(req.IDs)); !allowed {
		setRetryAfter(c, retryAfter)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "rate_limit_exceeded",
			"message":     msg,
			"retry_after": retryAfter.Seconds(),
		})
		return
	}

	concurrency := selectConcurrency(req.Concurrency, len(req.IDs))

	operation := func(ctx context.Context, ids []string) []credential.BatchOperationResult {
		return h.credMgr.BatchDeleteCredentials(ctx, ids)
	}

	if h.shouldRunAsync(len(req.IDs)) {
		h.startAsyncBatch(c, req.IDs, concurrency, batchOpDelete, operation)
		h.batchLimiter.RecordSuccess(string(batchOpDelete), len(req.IDs))
		return
	}

	output := h.processBatchConcurrently(
		c.Request.Context(),
		req.IDs,
		concurrency,
		batchOpDelete,
		operation,
		nil,
	)

	h.flushBatchDelete(c.Request.Context(), collectSuccessIDs(output.results))
	h.batchLimiter.RecordSuccess(string(batchOpDelete), len(req.IDs))
	sendBatchResponse(c, batchOpDelete, concurrency, output)
}

// BatchRecoverCredentials recovers multiple credentials at once (concurrent version with rate limiting).
func (h *AdminAPIHandler) BatchRecoverCredentials(c *gin.Context) {
	var req struct {
		IDs         []string `json:"ids" binding:"required"`
		Concurrency *int     `json:"concurrency,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if h.batchLimiter == nil {
		h.batchLimiter = NewBatchLimiter(DefaultBatchLimitConfig)
	}

	if allowed, msg, retryAfter := h.batchLimiter.CheckRequest(string(batchOpRecover), len(req.IDs)); !allowed {
		setRetryAfter(c, retryAfter)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "rate_limit_exceeded",
			"message":     msg,
			"retry_after": retryAfter.Seconds(),
		})
		return
	}

	concurrency := selectConcurrency(req.Concurrency, len(req.IDs))

	operation := func(ctx context.Context, ids []string) []credential.BatchOperationResult {
		return h.credMgr.BatchRecoverCredentials(ctx, ids)
	}

	if h.shouldRunAsync(len(req.IDs)) {
		h.startAsyncBatch(c, req.IDs, concurrency, batchOpRecover, operation)
		h.batchLimiter.RecordSuccess(string(batchOpRecover), len(req.IDs))
		return
	}

	output := h.processBatchConcurrently(
		c.Request.Context(),
		req.IDs,
		concurrency,
		batchOpRecover,
		operation,
		nil,
	)
	h.batchLimiter.RecordSuccess(string(batchOpRecover), len(req.IDs))
	sendBatchResponse(c, batchOpRecover, concurrency, output)
}

func (h *AdminAPIHandler) processBatchConcurrently(
	ctx context.Context,
	ids []string,
	concurrency int,
	op batchOperation,
	operation func(ctx context.Context, ids []string) []credential.BatchOperationResult,
	onProgress func(completed, success, failure int, result batchResult),
) batchProcessOutput {
	total := len(ids)
	if total == 0 {
		return batchProcessOutput{}
	}

	if concurrency <= 0 {
		concurrency = 1
	}
	if concurrency > total {
		concurrency = total
	}

	chunks := buildBatchTasks(ids)
	if concurrency > len(chunks) {
		concurrency = len(chunks)
	}

	tasks := make(chan batchTask, len(chunks))
	resultsChan := make(chan batchResult, total)

	ctx, cancel := context.WithTimeout(ctx, defaultBatchTimeout)
	defer cancel()

	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range tasks {
				select {
				case <-ctx.Done():
					for idx := range task.ids {
						resultsChan <- batchResult{
							index:   task.start + idx,
							id:      task.ids[idx],
							success: false,
							errMsg:  ctx.Err().Error(),
						}
					}
					continue
				default:
				}

				chunkResults := operation(ctx, task.ids)
				for idx, res := range chunkResults {
					br := batchResult{
						index:   task.start + idx,
						id:      res.ID,
						success: res.Success,
					}
					if res.Err != nil {
						br.errMsg = res.Err.Error()
						log.Warnf("[batch:%s worker:%d] failed for %s: %v", op, workerID, res.ID, res.Err)
					}
					resultsChan <- br
				}
			}
		}(i + 1)
	}

	for _, chunk := range chunks {
		tasks <- chunk
	}
	close(tasks)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	results := make([]batchResult, total)
	progress := make([]batchProgress, 0, progressBuckets)
	completed := 0
	success := 0
	failure := 0
	reportStep := determineProgressStep(total)
	cancelled := false

	for result := range resultsChan {
		results[result.index] = result
		completed++
		if result.success {
			success++
		} else {
			failure++
		}

		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				cancelled = true
			}
		}

		if onProgress != nil {
			onProgress(completed, success, failure, result)
		}

		if completed == total || completed%reportStep == 0 {
			progress = append(progress, batchProgress{
				Completed:    completed,
				SuccessCount: success,
				FailureCount: failure,
				Timestamp:    time.Now(),
			})
		}
	}

	duration := time.Since(start)
	return batchProcessOutput{
		results:               results,
		progress:              progress,
		duration:              duration,
		successCount:          success,
		failureCount:          failure,
		cancelledDueToTimeout: cancelled && ctx.Err() != nil,
	}
}

func buildBatchTasks(ids []string) []batchTask {
	if len(ids) == 0 {
		return nil
	}
	chunk := batchChunkSize
	if chunk <= 0 {
		chunk = 1
	}
	tasks := make([]batchTask, 0, (len(ids)+chunk-1)/chunk)
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		task := batchTask{
			start: i,
			ids:   ids[i:end],
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func selectConcurrency(custom *int, total int) int {
	concurrency := defaultBatchConcurrency
	if custom != nil && *custom > 0 {
		concurrency = *custom
	}
	if concurrency > maxBatchConcurrency {
		concurrency = maxBatchConcurrency
	}
	if concurrency > total {
		concurrency = total
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	return concurrency
}

func determineProgressStep(total int) int {
	if total <= 0 {
		return 1
	}
	step := total / progressBuckets
	if step <= 0 {
		step = 1
	}
	return step
}

func (h *AdminAPIHandler) shouldRunAsync(count int) bool {
	return count >= asyncBatchThreshold
}

func (h *AdminAPIHandler) startAsyncBatch(
	c *gin.Context,
	ids []string,
	concurrency int,
	op batchOperation,
	operation func(ctx context.Context, ids []string) []credential.BatchOperationResult,
) {
	manager := h.ensureTaskManager()
	task := manager.CreateTask(op, len(ids))
	go h.runAsyncBatch(task, ids, concurrency, op, operation)
	c.JSON(http.StatusAccepted, gin.H{
		"task_id": task.id,
		"status":  string(task.status),
		"total":   len(ids),
	})
}

func (h *AdminAPIHandler) runAsyncBatch(
	task *batchJob,
	ids []string,
	concurrency int,
	op batchOperation,
	operation func(ctx context.Context, ids []string) []credential.BatchOperationResult,
) {
	manager := h.ensureTaskManager()
	manager.MarkRunning(task.id)
	output := h.processBatchConcurrently(
		task.ctx,
		ids,
		concurrency,
		op,
		operation,
		func(completed, success, failure int, result batchResult) {
			manager.UpdateProgress(task.id, completed, success, failure, result)
		},
	)
	if task.ctx.Err() != nil {
		manager.FailTask(task.id, task.ctx.Err())
		return
	}
	manager.CompleteTask(task.id, output)
	if op == batchOpDelete {
		h.flushBatchDelete(context.Background(), collectSuccessIDs(output.results))
	}
}

func collectSuccessIDs(results []batchResult) []string {
	successIDs := make([]string, 0, len(results))
	for _, r := range results {
		if r.success {
			successIDs = append(successIDs, r.id)
		}
	}
	return successIDs
}

func (h *AdminAPIHandler) flushBatchDelete(ctx context.Context, ids []string) {
	if len(ids) == 0 || h.storage == nil {
		return
	}
	if err := h.storage.BatchDeleteCredentials(ctx, ids); err != nil {
		var unsupported *storage.ErrNotSupported
		if errors.As(err, &unsupported) {
			log.Debugf("storage backend does not support BatchDeleteCredentials: %v", err)
			return
		}
		log.Warnf("storage batch delete fallback failed: %v", err)
	}
}

func sendBatchResponse(c *gin.Context, op batchOperation, concurrency int, output batchProcessOutput) {
	results := make([]gin.H, len(output.results))
	for i, r := range output.results {
		row := gin.H{
			"id":      r.id,
			"success": r.success,
		}
		if r.errMsg != "" {
			row["error"] = r.errMsg
		}
		results[i] = row
	}

	progress := make([]gin.H, len(output.progress))
	for i, p := range output.progress {
		progress[i] = gin.H{
			"completed":     p.Completed,
			"success_count": p.SuccessCount,
			"failure_count": p.FailureCount,
			"timestamp":     p.Timestamp,
		}
	}

	response := gin.H{
		"operation":     string(op),
		"results":       results,
		"total":         len(output.results),
		"success_count": output.successCount,
		"failure_count": output.failureCount,
		"concurrency":   concurrency,
		"duration_ms":   output.duration.Milliseconds(),
		"progress":      progress,
	}

	if output.cancelledDueToTimeout {
		response["warning"] = "operation exceeded timeout; remaining items marked as failed"
	}

	c.JSON(http.StatusOK, response)
}

func setRetryAfter(c *gin.Context, retryAfter time.Duration) {
	if retryAfter <= 0 {
		return
	}
	c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
}

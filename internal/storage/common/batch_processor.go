package common

import (
	"context"
	"fmt"
	"sync"
)

// BatchProcessor 提供批量操作的通用处理逻辑
type BatchProcessor struct {
	maxConcurrency int
	serializer     *Serializer
	errorMapper    *ErrorMapper
}

// NewBatchProcessor 创建新的批量处理器
func NewBatchProcessor(maxConcurrency int) *BatchProcessor {
	if maxConcurrency <= 0 {
		maxConcurrency = 10 // 默认并发数
	}

	return &BatchProcessor{
		maxConcurrency: maxConcurrency,
		serializer:     NewSerializer(),
		errorMapper:    NewErrorMapper(),
	}
}

// BatchResult 批量操作的结果
type BatchResult struct {
	ID      string
	Success bool
	Error   error
	Data    map[string]interface{}
}

// BatchGetFunc 批量获取的函数签名
type BatchGetFunc func(ctx context.Context, id string) (map[string]interface{}, error)

// BatchSetFunc 批量设置的函数签名
type BatchSetFunc func(ctx context.Context, id string, data map[string]interface{}) error

// BatchDeleteFunc 批量删除的函数签名
type BatchDeleteFunc func(ctx context.Context, id string) error

// BatchGet 并发批量获取数据
func (bp *BatchProcessor) BatchGet(ctx context.Context, ids []string, getFunc BatchGetFunc) []BatchResult {
	if len(ids) == 0 {
		return []BatchResult{}
	}

	results := make([]BatchResult, len(ids))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, bp.maxConcurrency)

	for i, id := range ids {
		wg.Add(1)
		go func(index int, itemID string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			data, err := getFunc(ctx, itemID)
			results[index] = BatchResult{
				ID:      itemID,
				Success: err == nil,
				Error:   err,
				Data:    data,
			}
		}(i, id)
	}

	wg.Wait()
	return results
}

// BatchSet 并发批量设置数据
func (bp *BatchProcessor) BatchSet(ctx context.Context, items map[string]map[string]interface{}, setFunc BatchSetFunc) []BatchResult {
	if len(items) == 0 {
		return []BatchResult{}
	}

	results := make([]BatchResult, 0, len(items))
	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, bp.maxConcurrency)

	for id, data := range items {
		wg.Add(1)
		go func(itemID string, itemData map[string]interface{}) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := setFunc(ctx, itemID, itemData)

			mu.Lock()
			results = append(results, BatchResult{
				ID:      itemID,
				Success: err == nil,
				Error:   err,
				Data:    itemData,
			})
			mu.Unlock()
		}(id, data)
	}

	wg.Wait()
	return results
}

// BatchDelete 并发批量删除数据
func (bp *BatchProcessor) BatchDelete(ctx context.Context, ids []string, deleteFunc BatchDeleteFunc) []BatchResult {
	if len(ids) == 0 {
		return []BatchResult{}
	}

	results := make([]BatchResult, len(ids))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, bp.maxConcurrency)

	for i, id := range ids {
		wg.Add(1)
		go func(index int, itemID string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := deleteFunc(ctx, itemID)
			results[index] = BatchResult{
				ID:      itemID,
				Success: err == nil,
				Error:   err,
			}
		}(i, id)
	}

	wg.Wait()
	return results
}

// SummarizeResults 汇总批量操作结果
func (bp *BatchProcessor) SummarizeResults(results []BatchResult) (success, failed int, errors []error) {
	for _, r := range results {
		if r.Success {
			success++
		} else {
			failed++
			if r.Error != nil {
				errors = append(errors, fmt.Errorf("%s: %w", r.ID, r.Error))
			}
		}
	}
	return
}

// FilterSuccessful 过滤出成功的结果
func (bp *BatchProcessor) FilterSuccessful(results []BatchResult) []BatchResult {
	var successful []BatchResult
	for _, r := range results {
		if r.Success {
			successful = append(successful, r)
		}
	}
	return successful
}

// FilterFailed 过滤出失败的结果
func (bp *BatchProcessor) FilterFailed(results []BatchResult) []BatchResult {
	var failed []BatchResult
	for _, r := range results {
		if !r.Success {
			failed = append(failed, r)
		}
	}
	return failed
}

// GetSuccessfulIDs 获取成功操作的 ID 列表
func (bp *BatchProcessor) GetSuccessfulIDs(results []BatchResult) []string {
	var ids []string
	for _, r := range results {
		if r.Success {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

// GetFailedIDs 获取失败操作的 ID 列表
func (bp *BatchProcessor) GetFailedIDs(results []BatchResult) []string {
	var ids []string
	for _, r := range results {
		if !r.Success {
			ids = append(ids, r.ID)
		}
	}
	return ids
}

// ChunkIDs 将 ID 列表分块（用于大批量操作）
func (bp *BatchProcessor) ChunkIDs(ids []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		chunkSize = 100 // 默认块大小
	}

	var chunks [][]string
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[i:end])
	}
	return chunks
}

//go:build legacy_migration
// +build legacy_migration

package migration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gcli2api-go/internal/credential"
	"gcli2api-go/internal/storage"
)

// Migrator 数据迁移器
type Migrator struct {
	source      storage.Backend
	destination storage.Backend
	batchSize   int
	workers     int
	dryRun      bool
	validate    bool
	progress    *MigrationProgress
}

// MigrationProgress 迁移进度
type MigrationProgress struct {
	mu               sync.RWMutex
	TotalItems       int       `json:"total_items"`
	ProcessedItems   int       `json:"processed_items"`
	SuccessItems     int       `json:"success_items"`
	FailedItems      int       `json:"failed_items"`
	SkippedItems     int       `json:"skipped_items"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time,omitempty"`
	CurrentPhase     string    `json:"current_phase"`
	Errors           []string  `json:"errors,omitempty"`
	ValidationIssues []string  `json:"validation_issues,omitempty"`
}

// MigratorConfig 迁移器配置
type MigratorConfig struct {
	Source      storage.Backend
	Destination storage.Backend
	BatchSize   int
	Workers     int
	DryRun      bool
	Validate    bool
}

// NewMigrator 创建迁移器
func NewMigrator(config MigratorConfig) *Migrator {
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.Workers <= 0 {
		config.Workers = 4
	}

	return &Migrator{
		source:      config.Source,
		destination: config.Destination,
		batchSize:   config.BatchSize,
		workers:     config.Workers,
		dryRun:      config.DryRun,
		validate:    config.Validate,
		progress: &MigrationProgress{
			StartTime:    time.Now(),
			CurrentPhase: "initialized",
		},
	}
}

// Migrate 执行迁移
func (m *Migrator) Migrate(ctx context.Context) error {
	m.progress.mu.Lock()
	m.progress.CurrentPhase = "discovering"
	m.progress.mu.Unlock()

	// 1. 发现源数据
	sourceKeys, err := m.source.DiscoverCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover source credentials: %w", err)
	}

	m.progress.mu.Lock()
	m.progress.TotalItems = len(sourceKeys)
	m.progress.CurrentPhase = "migrating"
	m.progress.mu.Unlock()

	if len(sourceKeys) == 0 {
		m.progress.mu.Lock()
		m.progress.CurrentPhase = "completed"
		m.progress.EndTime = time.Now()
		m.progress.mu.Unlock()
		return nil
	}

	// 2. 批量迁移
	batches := m.createBatches(sourceKeys)

	// 使用 worker pool 并发迁移
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.workers)
	errChan := make(chan error, len(batches))

	for _, batch := range batches {
		wg.Add(1)
		go func(keys []string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := m.migrateBatch(ctx, keys); err != nil {
				errChan <- err
			}
		}(batch)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// 3. 验证（如果启用）
	if m.validate && !m.dryRun {
		m.progress.mu.Lock()
		m.progress.CurrentPhase = "validating"
		m.progress.mu.Unlock()

		if err := m.validateMigration(ctx, sourceKeys); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	m.progress.mu.Lock()
	m.progress.CurrentPhase = "completed"
	m.progress.EndTime = time.Now()
	m.progress.mu.Unlock()

	if len(errors) > 0 {
		return fmt.Errorf("migration completed with %d errors", len(errors))
	}

	return nil
}

// migrateBatch 迁移一批数据
func (m *Migrator) migrateBatch(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if err := m.migrateCredential(ctx, key); err != nil {
			m.progress.mu.Lock()
			m.progress.FailedItems++
			m.progress.Errors = append(m.progress.Errors, fmt.Sprintf("key=%s: %v", key, err))
			m.progress.mu.Unlock()
			continue
		}

		m.progress.mu.Lock()
		m.progress.ProcessedItems++
		m.progress.SuccessItems++
		m.progress.mu.Unlock()
	}

	return nil
}

// migrateCredential 迁移单个凭证
func (m *Migrator) migrateCredential(ctx context.Context, key string) error {
	// 从源读取
	cred, err := m.source.LoadCredential(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to load from source: %w", err)
	}

	// 检查目标是否已存在
	existing, err := m.destination.LoadCredential(ctx, key)
	if err == nil && existing != nil {
		// 已存在，跳过
		m.progress.mu.Lock()
		m.progress.SkippedItems++
		m.progress.mu.Unlock()
		return nil
	}

	// Dry run 模式不实际写入
	if m.dryRun {
		return nil
	}

	// 写入目标
	if err := m.destination.StoreCredential(ctx, cred); err != nil {
		return fmt.Errorf("failed to store to destination: %w", err)
	}

	// 迁移状态
	state, err := m.source.GetCredentialState(ctx, key)
	if err == nil && state != nil {
		if err := m.destination.UpdateCredentialState(ctx, key, *state); err != nil {
			return fmt.Errorf("failed to migrate state: %w", err)
		}
	}

	return nil
}

// validateMigration 验证迁移结果
func (m *Migrator) validateMigration(ctx context.Context, keys []string) error {
	var issues []string

	for _, key := range keys {
		sourceCred, err := m.source.LoadCredential(ctx, key)
		if err != nil {
			issues = append(issues, fmt.Sprintf("key=%s: failed to load from source: %v", key, err))
			continue
		}

		destCred, err := m.destination.LoadCredential(ctx, key)
		if err != nil {
			issues = append(issues, fmt.Sprintf("key=%s: failed to load from destination: %v", key, err))
			continue
		}

		if !m.credentialsEqual(sourceCred, destCred) {
			issues = append(issues, fmt.Sprintf("key=%s: credentials do not match", key))
		}
	}

	m.progress.mu.Lock()
	m.progress.ValidationIssues = issues
	m.progress.mu.Unlock()

	if len(issues) > 0 {
		return fmt.Errorf("found %d validation issues", len(issues))
	}

	return nil
}

// credentialsEqual 比较两个凭证是否相等
func (m *Migrator) credentialsEqual(a, b *credential.Credential) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Key == b.Key &&
		a.Model == b.Model &&
		a.Enabled == b.Enabled
}

// createBatches 创建批次
func (m *Migrator) createBatches(keys []string) [][]string {
	var batches [][]string
	for i := 0; i < len(keys); i += m.batchSize {
		end := i + m.batchSize
		if end > len(keys) {
			end = len(keys)
		}
		batches = append(batches, keys[i:end])
	}
	return batches
}

// GetProgress 获取迁移进度
func (m *Migrator) GetProgress() MigrationProgress {
	m.progress.mu.RLock()
	defer m.progress.mu.RUnlock()

	// 返回副本
	return MigrationProgress{
		TotalItems:       m.progress.TotalItems,
		ProcessedItems:   m.progress.ProcessedItems,
		SuccessItems:     m.progress.SuccessItems,
		FailedItems:      m.progress.FailedItems,
		SkippedItems:     m.progress.SkippedItems,
		StartTime:        m.progress.StartTime,
		EndTime:          m.progress.EndTime,
		CurrentPhase:     m.progress.CurrentPhase,
		Errors:           append([]string{}, m.progress.Errors...),
		ValidationIssues: append([]string{}, m.progress.ValidationIssues...),
	}
}

// GetProgressPercentage 获取进度百分比
func (m *Migrator) GetProgressPercentage() float64 {
	m.progress.mu.RLock()
	defer m.progress.mu.RUnlock()

	if m.progress.TotalItems == 0 {
		return 0
	}

	return float64(m.progress.ProcessedItems) / float64(m.progress.TotalItems) * 100
}

// GetEstimatedTimeRemaining 获取预计剩余时间
func (m *Migrator) GetEstimatedTimeRemaining() time.Duration {
	m.progress.mu.RLock()
	defer m.progress.mu.RUnlock()

	if m.progress.ProcessedItems == 0 {
		return 0
	}

	elapsed := time.Since(m.progress.StartTime)
	avgTimePerItem := elapsed / time.Duration(m.progress.ProcessedItems)
	remainingItems := m.progress.TotalItems - m.progress.ProcessedItems

	return avgTimePerItem * time.Duration(remainingItems)
}

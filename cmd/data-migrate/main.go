//go:build legacy_migration
// +build legacy_migration

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"gcli2api-go/internal/config"
	"gcli2api-go/internal/storage"
	"gcli2api-go/internal/storage/migration"
)

func main() {
	var (
		sourceType   = flag.String("source", "", "Source storage type (file/redis/mongodb/postgres)")
		destType     = flag.String("dest", "", "Destination storage type (file/redis/mongodb/postgres)")
		batchSize    = flag.Int("batch", 100, "Batch size for migration")
		workers      = flag.Int("workers", 4, "Number of concurrent workers")
		dryRun       = flag.Bool("dry-run", false, "Dry run mode (no actual writes)")
		validate     = flag.Bool("validate", true, "Validate migration results")
		configFile   = flag.String("config", "config.yaml", "Configuration file path")
		showProgress = flag.Bool("progress", true, "Show progress updates")
	)

	flag.Parse()

	if *sourceType == "" || *destType == "" {
		fmt.Println("Usage: data-migrate -source <type> -dest <type> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	// 创建源存储后端
	sourceBackend, err := createBackend(ctx, *sourceType, cfg)
	if err != nil {
		log.Fatalf("Failed to create source backend: %v", err)
	}
	defer sourceBackend.Close(ctx)

	// 创建目标存储后端
	destBackend, err := createBackend(ctx, *destType, cfg)
	if err != nil {
		log.Fatalf("Failed to create destination backend: %v", err)
	}
	defer destBackend.Close(ctx)

	// 创建迁移器
	migrator := migration.NewMigrator(migration.MigratorConfig{
		Source:      sourceBackend,
		Destination: destBackend,
		BatchSize:   *batchSize,
		Workers:     *workers,
		DryRun:      *dryRun,
		Validate:    *validate,
	})

	// 启动进度显示
	if *showProgress {
		go showProgressUpdates(migrator)
	}

	// 执行迁移
	fmt.Printf("Starting migration from %s to %s...\n", *sourceType, *destType)
	if *dryRun {
		fmt.Println("DRY RUN MODE - No actual writes will be performed")
	}

	if err := migrator.Migrate(ctx); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// 显示最终结果
	progress := migrator.GetProgress()
	fmt.Println("\n=== Migration Complete ===")
	fmt.Printf("Total Items:     %d\n", progress.TotalItems)
	fmt.Printf("Processed:       %d\n", progress.ProcessedItems)
	fmt.Printf("Success:         %d\n", progress.SuccessItems)
	fmt.Printf("Failed:          %d\n", progress.FailedItems)
	fmt.Printf("Skipped:         %d\n", progress.SkippedItems)
	fmt.Printf("Duration:        %v\n", progress.EndTime.Sub(progress.StartTime))

	if len(progress.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(progress.Errors))
		for i, err := range progress.Errors {
			if i >= 10 {
				fmt.Printf("... and %d more errors\n", len(progress.Errors)-10)
				break
			}
			fmt.Printf("  - %s\n", err)
		}
	}

	if len(progress.ValidationIssues) > 0 {
		fmt.Printf("\nValidation Issues (%d):\n", len(progress.ValidationIssues))
		for i, issue := range progress.ValidationIssues {
			if i >= 10 {
				fmt.Printf("... and %d more issues\n", len(progress.ValidationIssues)-10)
				break
			}
			fmt.Printf("  - %s\n", issue)
		}
	}

	if progress.FailedItems > 0 || len(progress.ValidationIssues) > 0 {
		os.Exit(1)
	}
}

func createBackend(ctx context.Context, backendType string, cfg *config.Config) (storage.Backend, error) {
	switch backendType {
	case "file":
		return storage.NewFileBackend(cfg.Storage.File.Path)
	case "redis":
		return storage.NewRedisBackend(ctx, storage.RedisConfig{
			Addr:     cfg.Storage.Redis.Addr,
			Password: cfg.Storage.Redis.Password,
			DB:       cfg.Storage.Redis.DB,
		})
	case "mongodb":
		return storage.NewMongoDBBackend(ctx, storage.MongoDBConfig{
			URI:      cfg.Storage.MongoDB.URI,
			Database: cfg.Storage.MongoDB.Database,
		})
	case "postgres":
		return storage.NewPostgresBackend(ctx, storage.PostgresConfig{
			Host:     cfg.Storage.Postgres.Host,
			Port:     cfg.Storage.Postgres.Port,
			User:     cfg.Storage.Postgres.User,
			Password: cfg.Storage.Postgres.Password,
			Database: cfg.Storage.Postgres.Database,
			SSLMode:  cfg.Storage.Postgres.SSLMode,
		})
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}

func showProgressUpdates(migrator *migration.Migrator) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		progress := migrator.GetProgress()

		if progress.CurrentPhase == "completed" {
			return
		}

		percentage := migrator.GetProgressPercentage()
		eta := migrator.GetEstimatedTimeRemaining()

		fmt.Printf("\r[%s] Progress: %.1f%% (%d/%d) | Success: %d | Failed: %d | Skipped: %d | ETA: %v",
			progress.CurrentPhase,
			percentage,
			progress.ProcessedItems,
			progress.TotalItems,
			progress.SuccessItems,
			progress.FailedItems,
			progress.SkippedItems,
			eta.Round(time.Second),
		)
	}
}

package storage

import (
	"context"
	"fmt"
	"time"
)

type credentialLister interface {
	ListCredentials(ctx context.Context) ([]string, error)
}

type credentialBatchGetter interface {
	BatchGetCredentials(ctx context.Context, ids []string) (map[string]map[string]interface{}, error)
}

type credentialBatchSetter interface {
	BatchSetCredentials(ctx context.Context, data map[string]map[string]interface{}) error
}

type configReader interface {
	ListConfigs(ctx context.Context) (map[string]interface{}, error)
}

type configWriter interface {
	SetConfig(ctx context.Context, key string, value interface{}) error
}

type usageReader interface {
	ListUsage(ctx context.Context) (map[string]map[string]interface{}, error)
}

type exportBackend interface {
	credentialLister
	credentialBatchGetter
	configReader
	usageReader
}

type importBackend interface {
	credentialBatchSetter
	configWriter
}

type statsBackend interface {
	credentialLister
	configReader
	usageReader
}

func exportDataCommon(ctx context.Context, backendName string, backend exportBackend) (map[string]interface{}, error) {
	exportData := map[string]interface{}{
		"backend":     backendName,
		"exported_at": time.Now().UTC(),
	}

	credentials := map[string]map[string]interface{}{}
	ids, err := backend.ListCredentials(ctx)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		credentials, err = backend.BatchGetCredentials(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	exportData["credentials"] = credentials

	if configs, err := backend.ListConfigs(ctx); err == nil {
		exportData["configs"] = configs
	}

	if usage, err := backend.ListUsage(ctx); err == nil {
		exportData["usage"] = usage
	}

	return exportData, nil
}

func importDataCommon(ctx context.Context, backend importBackend, data map[string]interface{}) error {
	if creds, ok := data["credentials"].(map[string]interface{}); ok {
		converted := make(map[string]map[string]interface{}, len(creds))
		for id, raw := range creds {
			if m, ok := raw.(map[string]interface{}); ok {
				converted[id] = m
			}
		}
		if len(converted) > 0 {
			if err := backend.BatchSetCredentials(ctx, converted); err != nil {
				return fmt.Errorf("batch set credentials: %w", err)
			}
		}
	}

	if configs, ok := data["configs"].(map[string]interface{}); ok {
		for key, value := range configs {
			if err := backend.SetConfig(ctx, key, value); err != nil {
				return fmt.Errorf("set config %s: %w", key, err)
			}
		}
	}

	return nil
}

func storageStatsCommon(ctx context.Context, backendName string, backend statsBackend) (StorageStats, error) {
	stats := StorageStats{
		Backend: backendName,
		Healthy: true,
	}

	credIDs, err := backend.ListCredentials(ctx)
	if err != nil {
		stats.Healthy = false
		return stats, err
	}
	stats.CredentialCount = len(credIDs)

	if configs, err := backend.ListConfigs(ctx); err == nil {
		stats.ConfigCount = len(configs)
	}

	if usage, err := backend.ListUsage(ctx); err == nil {
		stats.UsageRecordCount = len(usage)
	}

	return stats, nil
}

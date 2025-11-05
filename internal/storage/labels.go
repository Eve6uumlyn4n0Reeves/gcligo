//go:build !stats_isolation

package storage

import (
	"strings"

	"gcli2api-go/internal/config"
)

// DetectBackendLabel returns a normalized label for the configured backend.
func DetectBackendLabel(cfg *config.Config, backend Backend) string {
	if cfg != nil {
		if raw := strings.TrimSpace(strings.ToLower(cfg.StorageBackend)); raw != "" && raw != "auto" {
			return raw
		}
	}
	switch backend.(type) {
	case *PostgresBackend:
		return "postgres"
	case *MongoDBBackend:
		return "mongodb"
	case *RedisBackend:
		return "redis"
	case *FileBackend:
		return "file"
	case *GitBackend:
		return "git"
	default:
		return "unknown"
	}
}

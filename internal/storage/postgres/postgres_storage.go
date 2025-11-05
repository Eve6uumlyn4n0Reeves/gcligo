package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gcli2api-go/internal/migrations"
	"gcli2api-go/internal/oauth"
	storagecommon "gcli2api-go/internal/storage/common"

	pq "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

type PostgresStorage struct {
	db *sql.DB
}

const defaultPGTimeout = 5 * time.Second

// withPGTimeout is deprecated, use storagecommon.WithStorageTimeout instead
func withPGTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return storagecommon.WithStorageTimeout(ctx, defaultPGTimeout)
}

// NewPostgresStorage creates a new PostgreSQL storage backend
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Info("Connected to PostgreSQL storage backend")

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Initialize(ctx context.Context) error {
	if err := migrations.PostgresUp(p.db); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	log.Info("PostgreSQL migrations applied")
	return nil
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

// PoolStats returns current connection pool statistics.
func (p *PostgresStorage) PoolStats() (active int64, idle int64, misses int64) {
	if p == nil || p.db == nil {
		return 0, 0, 0
	}
	s := p.db.Stats()
	return int64(s.InUse), int64(s.Idle), int64(s.WaitCount)
}

func (p *PostgresStorage) ListCredentials(ctx context.Context) ([]string, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	rows, err := p.db.QueryContext(ctx, "SELECT filename FROM credentials ORDER BY filename")
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}
	defer rows.Close()

	var filenames []string
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, fmt.Errorf("failed to scan filename: %w", err)
		}
		filenames = append(filenames, filename)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return filenames, nil
}

// BatchGetCredentials retrieves multiple credentials in a single round trip.
func (p *PostgresStorage) BatchGetCredentials(ctx context.Context, filenames []string) (map[string]*oauth.Credentials, error) {
	if len(filenames) == 0 {
		return map[string]*oauth.Credentials{}, nil
	}

	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	rows, err := p.db.QueryContext(ctx, `SELECT filename, data FROM credentials WHERE filename = ANY($1)`, pq.Array(filenames))
	if err != nil {
		return nil, fmt.Errorf("failed to batch get credentials: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*oauth.Credentials, len(filenames))
	for rows.Next() {
		var filename string
		var dataJSON []byte
		if err := rows.Scan(&filename, &dataJSON); err != nil {
			return nil, fmt.Errorf("scan credential %s: %w", filename, err)
		}
		var creds oauth.Credentials
		if err := json.Unmarshal(dataJSON, &creds); err != nil {
			return nil, fmt.Errorf("unmarshal credential %s: %w", filename, err)
		}
		result[filename] = &creds
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("batch get credentials rows error: %w", err)
	}

	return result, nil
}

func (p *PostgresStorage) GetCredential(ctx context.Context, filename string) (*oauth.Credentials, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	var dataJSON []byte
	err := p.db.QueryRowContext(ctx, "SELECT data FROM credentials WHERE filename = $1", filename).Scan(&dataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get credential: %w", err)
	}

	var creds oauth.Credentials
	if err := json.Unmarshal(dataJSON, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credential: %w", err)
	}

	return &creds, nil
}

func (p *PostgresStorage) SaveCredential(ctx context.Context, filename string, creds *oauth.Credentials) error {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	dataJSON, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credential: %w", err)
	}

	query := `
		INSERT INTO credentials (filename, data, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (filename)
		DO UPDATE SET data = $2, updated_at = CURRENT_TIMESTAMP
	`

	if _, err := p.db.ExecContext(ctx, query, filename, dataJSON); err != nil {
		return fmt.Errorf("failed to save credential: %w", err)
	}

	return nil
}

func (p *PostgresStorage) DeleteCredential(ctx context.Context, filename string) error {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete credential
	if _, err := tx.ExecContext(ctx, "DELETE FROM credentials WHERE filename = $1", filename); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	// Delete state
	if _, err := tx.ExecContext(ctx, "DELETE FROM credential_states WHERE filename = $1", filename); err != nil {
		return fmt.Errorf("failed to delete credential state: %w", err)
	}

	// Delete usage stats
	if _, err := tx.ExecContext(ctx, "DELETE FROM usage_stats WHERE usage_key = $1", filename); err != nil {
		return fmt.Errorf("failed to delete usage stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchSaveCredentials saves multiple credentials atomically within a single transaction.
func (p *PostgresStorage) BatchSaveCredentials(ctx context.Context, items map[string]*oauth.Credentials) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO credentials (filename, data, updated_at)
        VALUES ($1, $2, CURRENT_TIMESTAMP)
        ON CONFLICT (filename)
        DO UPDATE SET data = EXCLUDED.data, updated_at = CURRENT_TIMESTAMP`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for filename, creds := range items {
		dataJSON, err := json.Marshal(creds)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", filename, err)
		}
		if _, err := stmt.ExecContext(ctx, filename, dataJSON); err != nil {
			return fmt.Errorf("upsert %s: %w", filename, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// BatchDeleteCredentials deletes multiple credentials atomically.
func (p *PostgresStorage) BatchDeleteCredentials(ctx context.Context, filenames []string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, name := range filenames {
		if _, err := tx.ExecContext(ctx, "DELETE FROM credentials WHERE filename = $1", name); err != nil {
			return fmt.Errorf("delete credential %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM credential_states WHERE filename = $1", name); err != nil {
			return fmt.Errorf("delete state %s: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// Note: credential state and usage helpers removed to avoid import cycles and unused build failures.

func (p *PostgresStorage) IncrementUsage(ctx context.Context, key string, field string, delta int64) error {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	query := `
        INSERT INTO usage_stats (usage_key, field, value, updated_at)
        VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
        ON CONFLICT (usage_key, field)
        DO UPDATE SET value = usage_stats.value + EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
    `
	_, err := p.db.ExecContext(ctx, query, key, field, delta)
	if err != nil {
		return fmt.Errorf("failed to increment usage: %w", err)
	}
	return nil
}

func (p *PostgresStorage) GetUsage(ctx context.Context, key string) (map[string]interface{}, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	rows, err := p.db.QueryContext(ctx, "SELECT field, value FROM usage_stats WHERE usage_key = $1", key)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage: %w", err)
	}
	defer rows.Close()

	result := make(map[string]interface{})
	for rows.Next() {
		var field string
		var value int64
		if err := rows.Scan(&field, &value); err != nil {
			return nil, fmt.Errorf("failed to scan usage row: %w", err)
		}
		result[field] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usage rows iteration error: %w", err)
	}
	return result, nil
}

func (p *PostgresStorage) ResetUsage(ctx context.Context, key string) error {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	if _, err := p.db.ExecContext(ctx, "DELETE FROM usage_stats WHERE usage_key = $1", key); err != nil {
		return fmt.Errorf("failed to reset usage: %w", err)
	}
	return nil
}

func (p *PostgresStorage) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	return p.db.BeginTx(ctx, opts)
}

func (p *PostgresStorage) ListUsage(ctx context.Context) (map[string]map[string]interface{}, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	rows, err := p.db.QueryContext(ctx, "SELECT usage_key, field, value FROM usage_stats")
	if err != nil {
		return nil, fmt.Errorf("failed to list usage: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]interface{})
	for rows.Next() {
		var key, field string
		var value int64
		if err := rows.Scan(&key, &field, &value); err != nil {
			return nil, fmt.Errorf("failed to scan usage entry: %w", err)
		}
		if _, ok := result[key]; !ok {
			result[key] = make(map[string]interface{})
		}
		result[key][field] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usage list iteration error: %w", err)
	}
	return result, nil
}

func (p *PostgresStorage) SetConfig(ctx context.Context, key string, value interface{}) error {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal config %s: %w", key, err)
	}
	query := `
		INSERT INTO configs (config_key, value, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (config_key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP
	`
	if _, err := p.db.ExecContext(ctx, query, key, data); err != nil {
		return fmt.Errorf("failed to save config %s: %w", key, err)
	}
	return nil
}

func (p *PostgresStorage) GetConfig(ctx context.Context, key string) (interface{}, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	var raw []byte
	err := p.db.QueryRowContext(ctx, "SELECT value FROM configs WHERE config_key = $1", key).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var out interface{}
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config %s: %w", key, err)
	}
	return out, nil
}

func (p *PostgresStorage) DeleteConfig(ctx context.Context, key string) error {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	res, err := p.db.ExecContext(ctx, "DELETE FROM configs WHERE config_key = $1", key)
	if err != nil {
		return fmt.Errorf("failed to delete config %s: %w", key, err)
	}
	if affected, err := res.RowsAffected(); err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (p *PostgresStorage) ListConfigs(ctx context.Context) (map[string]interface{}, error) {
	ctx, cancel := withPGTimeout(ctx)
	defer cancel()
	rows, err := p.db.QueryContext(ctx, "SELECT config_key, value FROM configs")
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]interface{})
	for rows.Next() {
		var key string
		var raw []byte
		if err := rows.Scan(&key, &raw); err != nil {
			return nil, fmt.Errorf("failed to scan config row: %w", err)
		}
		if len(raw) == 0 {
			result[key] = nil
			continue
		}
		var out interface{}
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config %s: %w", key, err)
		}
		result[key] = out
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("config rows iteration error: %w", err)
	}
	return result, nil
}

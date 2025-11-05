//go:build !stats_isolation

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"gcli2api-go/internal/oauth"
)

type postgresTransaction struct {
	backend *PostgresBackend
	tx      *sql.Tx
	closed  bool
}

func (p *PostgresBackend) BeginTransaction(ctx context.Context) (Transaction, error) {
	pgTx, err := p.storage.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &postgresTransaction{backend: p, tx: pgTx}, nil
}

func (t *postgresTransaction) ensureOpen() error {
	if t.closed || t.tx == nil {
		return fmt.Errorf("transaction already closed")
	}
	return nil
}

func (t *postgresTransaction) GetCredential(ctx context.Context, id string) (map[string]interface{}, error) {
	if err := t.ensureOpen(); err != nil {
		return nil, err
	}
	row := t.tx.QueryRowContext(ctx, "SELECT data FROM credentials WHERE filename = $1", id)
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ErrNotFound{Key: id}
		}
		return nil, fmt.Errorf("fetch credential %s: %w", id, err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode credential %s: %w", id, err)
	}
	return out, nil
}

func (t *postgresTransaction) SetCredential(ctx context.Context, id string, data map[string]interface{}) error {
	if err := t.ensureOpen(); err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("encode credential %s: %w", id, err)
	}
	var cred oauth.Credentials
	if err := json.Unmarshal(payload, &cred); err != nil {
		return fmt.Errorf("invalid credential %s: %w", id, err)
	}
	query := `
		INSERT INTO credentials (filename, data, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (filename)
		DO UPDATE SET data = EXCLUDED.data, updated_at = CURRENT_TIMESTAMP`
	if _, err := t.tx.ExecContext(ctx, query, id, payload); err != nil {
		return fmt.Errorf("upsert credential %s: %w", id, err)
	}
	return nil
}

func (t *postgresTransaction) DeleteCredential(ctx context.Context, id string) error {
	if err := t.ensureOpen(); err != nil {
		return err
	}
	if _, err := t.tx.ExecContext(ctx, "DELETE FROM credentials WHERE filename = $1", id); err != nil {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	_, _ = t.tx.ExecContext(ctx, "DELETE FROM credential_states WHERE filename = $1", id)
	_, _ = t.tx.ExecContext(ctx, "DELETE FROM usage_stats WHERE usage_key = $1", id)
	return nil
}

func (t *postgresTransaction) GetConfig(ctx context.Context, key string) (interface{}, error) {
	if err := t.ensureOpen(); err != nil {
		return nil, err
	}
	row := t.tx.QueryRowContext(ctx, "SELECT value FROM configs WHERE config_key = $1", key)
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &ErrNotFound{Key: key}
		}
		return nil, fmt.Errorf("fetch config %s: %w", key, err)
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("decode config %s: %w", key, err)
	}
	return value, nil
}

func (t *postgresTransaction) SetConfig(ctx context.Context, key string, value interface{}) error {
	if err := t.ensureOpen(); err != nil {
		return err
	}
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode config %s: %w", key, err)
	}
	query := `
		INSERT INTO configs (config_key, value, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (config_key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP`
	if _, err := t.tx.ExecContext(ctx, query, key, valueJSON); err != nil {
		return fmt.Errorf("upsert config %s: %w", key, err)
	}
	return nil
}

func (t *postgresTransaction) DeleteConfig(ctx context.Context, key string) error {
	if err := t.ensureOpen(); err != nil {
		return err
	}
	res, err := t.tx.ExecContext(ctx, "DELETE FROM configs WHERE config_key = $1", key)
	if err != nil {
		return fmt.Errorf("delete config %s: %w", key, err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return &ErrNotFound{Key: key}
	}
	return nil
}

func (t *postgresTransaction) Commit(ctx context.Context) error {
	if t.tx == nil || t.closed {
		return nil
	}
	t.closed = true
	return t.tx.Commit()
}

func (t *postgresTransaction) Rollback(ctx context.Context) error {
	if t.tx == nil || t.closed {
		return nil
	}
	t.closed = true
	return t.tx.Rollback()
}

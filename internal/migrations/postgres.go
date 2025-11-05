package migrations

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func postgresMigrator(db *sql.DB) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("postgres driver: %w", err)
	}
	source, err := iofs.New(sqlMigrations, "sql")
	if err != nil {
		return nil, fmt.Errorf("migrations source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return nil, fmt.Errorf("migrate instance: %w", err)
	}
	return m, nil
}

func closeMigrator(m *migrate.Migrate) error {
	if m == nil {
		return nil
	}
	srcErr, dbErr := m.Close()
	return errors.Join(srcErr, dbErr)
}

// PostgresUp applies all pending migrations.
func PostgresUp(db *sql.DB) error {
	m, err := postgresMigrator(db)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations up: %w", err)
	}
	return nil
}

// PostgresDown rolls back the given number of migrations (default 1 if steps <= 0).
func PostgresDown(db *sql.DB, steps int) error {
	if steps <= 0 {
		steps = 1
	}
	m, err := postgresMigrator(db)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	if err := m.Steps(-steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrations down: %w", err)
	}
	return nil
}

// PostgresVersion returns the current migration version.
func PostgresVersion(db *sql.DB) (uint, bool, error) {
	m, err := postgresMigrator(db)
	if err != nil {
		return 0, false, err
	}
	defer closeMigrator(m)

	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, dirty, fmt.Errorf("migrations version: %w", err)
	}
	return version, dirty, nil
}

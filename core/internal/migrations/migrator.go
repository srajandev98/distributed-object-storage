package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

type Migrator struct {
	db *sql.DB
}

func New(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) Apply() error {
	if err := m.ensureMigrationsTable(); err != nil {
		return err
	}

	entries, err := migrationFiles.ReadDir("sql")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := m.isApplied(name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		contents, err := migrationFiles.ReadFile("sql/" + name)
		if err != nil {
			return err
		}

		tx, err := m.db.Begin()
		if err != nil {
			return err
		}

		if _, err = tx.Exec(string(contents)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err = tx.Exec(`
			INSERT INTO schema_migrations(version)
			VALUES ($1)
		`, name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id BIGSERIAL PRIMARY KEY,
			version TEXT NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func (m *Migrator) isApplied(version string) (bool, error) {
	var exists bool
	err := m.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM schema_migrations
			WHERE version = $1
		)
	`, version).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

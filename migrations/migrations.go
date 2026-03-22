// Package migrations provides a versioned schema migration system for SQL databases.
// It supports PostgreSQL, MariaDB (MySQL), and SQLite with automatic tracking
// of applied migrations.
package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// Migration represents a single database schema migration.
type Migration struct {
	// Version is the unique version number for this migration.
	// Versions must be unique and monotonically increasing.
	Version int64

	// Name is a human-readable description of the migration.
	Name string

	// Up contains the SQL statements to apply the migration.
	Up []string

	// Down contains the SQL statements to revert the migration.
	Down []string
}

// MigrationRecord represents a migration that has been applied to the database.
type MigrationRecord struct {
	Version   int64  `db:"version"`
	Name      string `db:"name"`
	AppliedAt string `db:"applied_at"`
}

// DBDriver represents the supported database driver types.
type DBDriver string

const (
	// DriverPostgreSQL represents PostgreSQL databases.
	DriverPostgreSQL DBDriver = "postgres"
	// DriverMySQL represents MySQL/MariaDB databases.
	DriverMySQL DBDriver = "mysql"
	// DriverSQLite represents SQLite databases.
	DriverSQLite DBDriver = "sqlite3"
)

// Runner executes migrations on a database.
type Runner struct {
	db     *sql.DB
	driver DBDriver
}

// NewRunner creates a new migration runner.
func NewRunner(db *sql.DB, driver DBDriver) *Runner {
	return &Runner{
		db:     db,
		driver: driver,
	}
}

// InitSchema creates the schema_migrations table if it doesn't exist.
func (r *Runner) InitSchema(ctx context.Context) error {
	var query string

	switch r.driver {
	case DriverPostgreSQL:
		query = `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version BIGINT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	case DriverMySQL:
		query = `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version BIGINT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	case DriverSQLite:
		query = `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	default:
		return fmt.Errorf("unsupported database driver: %s", r.driver)
	}

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns all migrations that have been applied to the database.
func (r *Runner) GetAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT version, name, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var rec MigrationRecord
		if err := rows.Scan(&rec.Version, &rec.Name, &rec.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration records: %w", err)
	}

	return records, nil
}

// IsMigrationApplied checks if a specific migration version has been applied.
func (r *Runner) IsMigrationApplied(ctx context.Context, version int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	return count > 0, nil
}

// recordMigration adds a migration record to the schema_migrations table.
func (r *Runner) recordMigration(ctx context.Context, m Migration) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
		m.Version, m.Name)
	if err != nil {
		return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
	}
	return nil
}

// removeMigrationRecord removes a migration record from the schema_migrations table.
func (r *Runner) removeMigrationRecord(ctx context.Context, version int64) error {
	_, err := r.db.ExecContext(ctx,
		"DELETE FROM schema_migrations WHERE version = ?", version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record %d: %w", version, err)
	}
	return nil
}

// UpResult contains the result of running migrations.
type UpResult struct {
	Applied []int64
	Skipped []int64
}

// Up runs all pending migrations in order.
func (r *Runner) Up(ctx context.Context, migrations []Migration) (*UpResult, error) {
	result := &UpResult{
		Applied: make([]int64, 0),
		Skipped: make([]int64, 0),
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for _, m := range migrations {
		applied, err := r.IsMigrationApplied(ctx, m.Version)
		if err != nil {
			return result, err
		}

		if applied {
			result.Skipped = append(result.Skipped, m.Version)
			continue
		}

		// Run migration in a transaction
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("failed to begin transaction for migration %d: %w", m.Version, err)
		}

		// Execute all up statements
		for i, stmt := range m.Up {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				tx.Rollback()
				return result, fmt.Errorf("migration %d up statement %d failed: %w", m.Version, i, err)
			}
		}

		// Record migration
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			m.Version, m.Name); err != nil {
			tx.Rollback()
			return result, fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return result, fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
		}

		result.Applied = append(result.Applied, m.Version)
	}

	return result, nil
}

// DownResult contains the result of reverting migrations.
type DownResult struct {
	Reverted []int64
	Skipped  []int64
}

// Down reverts migrations down to the specified target version (inclusive).
// Set target to 0 to revert all migrations.
func (r *Runner) Down(ctx context.Context, migrations []Migration, target int64) (*DownResult, error) {
	result := &DownResult{
		Reverted: make([]int64, 0),
		Skipped:  make([]int64, 0),
	}

	// Get current version
	applied, err := r.GetAppliedMigrations(ctx)
	if err != nil {
		return result, err
	}

	if len(applied) == 0 {
		return result, nil
	}

	// Sort migrations map by version for lookup
	migMap := make(map[int64]Migration)
	for _, m := range migrations {
		migMap[m.Version] = m
	}

	// Revert migrations in reverse order (newest first)
	for i := len(applied) - 1; i >= 0; i-- {
		rec := applied[i]

		// Stop if we've reached the target
		if rec.Version <= target {
			break
		}

		m, exists := migMap[rec.Version]
		if !exists {
			return result, fmt.Errorf("migration %d not found in provided migrations list", rec.Version)
		}

		// Run rollback in a transaction
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("failed to begin transaction for rollback %d: %w", rec.Version, err)
		}

		// Execute all down statements
		for i, stmt := range m.Down {
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				tx.Rollback()
				return result, fmt.Errorf("migration %d down statement %d failed: %w", rec.Version, i, err)
			}
		}

		// Remove migration record
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM schema_migrations WHERE version = ?", rec.Version); err != nil {
			tx.Rollback()
			return result, fmt.Errorf("failed to remove migration record %d: %w", rec.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return result, fmt.Errorf("failed to commit rollback %d: %w", rec.Version, err)
		}

		result.Reverted = append(result.Reverted, rec.Version)
	}

	return result, nil
}

// Version returns the current schema version.
func (r *Runner) Version(ctx context.Context) (int64, error) {
	var version int64
	err := r.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}
	return version, nil
}

// Status returns the migration status information.
type Status struct {
	Version  int64
	Applied  int
	Pending  int
	LastName string
}

// GetStatus returns the current migration status.
func (r *Runner) GetStatus(ctx context.Context, migrations []Migration) (*Status, error) {
	version, err := r.Version(ctx)
	if err != nil {
		return nil, err
	}

	applied, err := r.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	status := &Status{
		Version: version,
		Applied: len(applied),
		Pending: len(migrations) - len(applied),
	}

	if len(applied) > 0 {
		status.LastName = applied[len(applied)-1].Name
	}

	return status, nil
}

// Validate checks if all applied migrations exist in the provided list.
func (r *Runner) Validate(ctx context.Context, migrations []Migration) error {
	applied, err := r.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	migMap := make(map[int64]bool)
	for _, m := range migrations {
		migMap[m.Version] = true
	}

	for _, rec := range applied {
		if !migMap[rec.Version] {
			return fmt.Errorf("applied migration %d (%s) not found in migrations list", rec.Version, rec.Name)
		}
	}

	return nil
}

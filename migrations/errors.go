package migrations

import "errors"

// Common errors returned by the migrations package.
var (
	// ErrNoDriver indicates that no database driver was specified.
	ErrNoDriver = errors.New("no database driver specified")

	// ErrUnsupportedDriver indicates that the specified driver is not supported.
	ErrUnsupportedDriver = errors.New("unsupported database driver")

	// ErrMigrationNotFound indicates that a migration was not found.
	ErrMigrationNotFound = errors.New("migration not found")

	// ErrMigrationAlreadyApplied indicates that a migration has already been applied.
	ErrMigrationAlreadyApplied = errors.New("migration already applied")

	// ErrInvalidVersion indicates that the migration version is invalid.
	ErrInvalidVersion = errors.New("invalid migration version")

	// ErrNilDB indicates that a nil database connection was provided.
	ErrNilDB = errors.New("database connection is nil")
)

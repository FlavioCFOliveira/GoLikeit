package migrations

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func TestNewRunner(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if runner == nil {
		t.Fatal("NewRunner() returned nil")
	}
	if runner.db != db {
		t.Error("runner.db is not the provided database")
	}
	if runner.driver != DriverSQLite {
		t.Errorf("runner.driver = %v, want %v", runner.driver, DriverSQLite)
	}
}

func TestRunner_InitSchema(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)

	err := runner.InitSchema(ctx)
	if err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	// Verify table exists
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations table count = %d, want 1", count)
	}
}

func TestRunner_IsMigrationApplied(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	// Initially, no migrations should be applied
	applied, err := runner.IsMigrationApplied(ctx, 1)
	if err != nil {
		t.Fatalf("IsMigrationApplied() error = %v", err)
	}
	if applied {
		t.Error("IsMigrationApplied(1) = true, want false")
	}
}

func TestRunner_Up(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_test_table",
			Up: []string{
				"CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)",
			},
			Down: []string{
				"DROP TABLE test",
			},
		},
		{
			Version: 2,
			Name:    "add_test_column",
			Up: []string{
				"ALTER TABLE test ADD COLUMN value INTEGER",
			},
			Down: []string{
				// Note: SQLite doesn't support DROP COLUMN easily
				"CREATE TABLE test_new (id INTEGER PRIMARY KEY, name TEXT)",
				"INSERT INTO test_new SELECT id, name FROM test",
				"DROP TABLE test",
				"ALTER TABLE test_new RENAME TO test",
			},
		},
	}

	result, err := runner.Up(ctx, migrations)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if len(result.Applied) != 2 {
		t.Errorf("Applied migrations = %d, want 2", len(result.Applied))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("Skipped migrations = %d, want 0", len(result.Skipped))
	}

	// Verify table exists
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if count != 1 {
		t.Errorf("test table count = %d, want 1", count)
	}

	// Verify column was added
	var colCount int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info('test') WHERE name = 'value'").Scan(&colCount)
	if err != nil {
		t.Fatalf("failed to check column existence: %v", err)
	}
	if colCount != 1 {
		t.Errorf("value column count = %d, want 1", colCount)
	}

	// Running again should skip both migrations
	result2, err := runner.Up(ctx, migrations)
	if err != nil {
		t.Fatalf("Up() second run error = %v", err)
	}

	if len(result2.Applied) != 0 {
		t.Errorf("Second run applied = %d, want 0", len(result2.Applied))
	}
	if len(result2.Skipped) != 2 {
		t.Errorf("Second run skipped = %d, want 2", len(result2.Skipped))
	}
}

func TestRunner_Down(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create_test_table",
			Up: []string{
				"CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)",
			},
			Down: []string{
				"DROP TABLE test",
			},
		},
		{
			Version: 2,
			Name:    "create_another_table",
			Up: []string{
				"CREATE TABLE another (id INTEGER PRIMARY KEY)",
			},
			Down: []string{
				"DROP TABLE another",
			},
		},
	}

	// Apply all migrations
	_, err := runner.Up(ctx, migrations)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Revert to version 1
	result, err := runner.Down(ctx, migrations, 1)
	if err != nil {
		t.Fatalf("Down() error = %v", err)
	}

	if len(result.Reverted) != 1 {
		t.Errorf("Reverted migrations = %d, want 1", len(result.Reverted))
	}

	// Verify 'another' table was dropped
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='another'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if count != 0 {
		t.Errorf("another table count = %d, want 0", count)
	}

	// Verify 'test' table still exists
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if count != 1 {
		t.Errorf("test table count = %d, want 1", count)
	}
}

func TestRunner_Version(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	// Initially, version should be 0
	version, err := runner.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if version != 0 {
		t.Errorf("Initial version = %d, want 0", version)
	}

	// Apply migrations
	migrations := []Migration{
		{Version: 1, Name: "test1", Up: []string{}, Down: []string{}},
		{Version: 2, Name: "test2", Up: []string{}, Down: []string{}},
	}

	_, err = runner.Up(ctx, migrations)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Version should now be 2
	version, err = runner.Version(ctx)
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if version != 2 {
		t.Errorf("Version after migrations = %d, want 2", version)
	}
}

func TestRunner_GetStatus(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	migrations := []Migration{
		{Version: 1, Name: "test1", Up: []string{}, Down: []string{}},
		{Version: 2, Name: "test2", Up: []string{}, Down: []string{}},
		{Version: 3, Name: "test3", Up: []string{}, Down: []string{}},
	}

	// Apply first migration
	_, err := runner.Up(ctx, migrations[:1])
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	status, err := runner.GetStatus(ctx, migrations)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if status.Version != 1 {
		t.Errorf("Status.Version = %d, want 1", status.Version)
	}
	if status.Applied != 1 {
		t.Errorf("Status.Applied = %d, want 1", status.Applied)
	}
	if status.Pending != 2 {
		t.Errorf("Status.Pending = %d, want 2", status.Pending)
	}
	if status.LastName != "test1" {
		t.Errorf("Status.LastName = %s, want test1", status.LastName)
	}
}

func TestRunner_Validate(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	migrations := []Migration{
		{Version: 1, Name: "test1", Up: []string{}, Down: []string{}},
		{Version: 2, Name: "test2", Up: []string{}, Down: []string{}},
	}

	// Apply migrations
	_, err := runner.Up(ctx, migrations)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	// Validation should pass with complete list
	err = runner.Validate(ctx, migrations)
	if err != nil {
		t.Errorf("Validate() with complete list error = %v", err)
	}

	// Validation should fail with incomplete list
	err = runner.Validate(ctx, migrations[:1])
	if err == nil {
		t.Error("Validate() with incomplete list should have returned error")
	}
}

func TestGetMigrationByVersion(t *testing.T) {
	migrations := []Migration{
		{Version: 1, Name: "test1"},
		{Version: 2, Name: "test2"},
		{Version: 3, Name: "test3"},
	}

	m, found := GetMigrationByVersion(migrations, 2)
	if !found {
		t.Error("GetMigrationByVersion(2) not found")
	}
	if m.Name != "test2" {
		t.Errorf("GetMigrationByVersion(2).Name = %s, want test2", m.Name)
	}

	_, found = GetMigrationByVersion(migrations, 99)
	if found {
		t.Error("GetMigrationByVersion(99) should not be found")
	}
}

func TestGetMigrationsUpTo(t *testing.T) {
	migrations := []Migration{
		{Version: 1, Name: "test1"},
		{Version: 2, Name: "test2"},
		{Version: 3, Name: "test3"},
		{Version: 5, Name: "test5"},
	}

	result := GetMigrationsUpTo(migrations, 3)
	if len(result) != 3 {
		t.Errorf("GetMigrationsUpTo(3) = %d migrations, want 3", len(result))
	}

	result = GetMigrationsUpTo(migrations, 4)
	if len(result) != 3 {
		t.Errorf("GetMigrationsUpTo(4) = %d migrations, want 3", len(result))
	}
}

func TestMigrationBuilder(t *testing.T) {
	builder := NewMigration(1, "test_migration").
		Up("CREATE TABLE test (id INTEGER)").
		Up("INSERT INTO test VALUES (1)").
		Down("DROP TABLE test")

	m := builder.Build()

	if m.Version != 1 {
		t.Errorf("Version = %d, want 1", m.Version)
	}
	if m.Name != "test_migration" {
		t.Errorf("Name = %s, want test_migration", m.Name)
	}
	if len(m.Up) != 2 {
		t.Errorf("Up statements = %d, want 2", len(m.Up))
	}
	if len(m.Down) != 1 {
		t.Errorf("Down statements = %d, want 1", len(m.Down))
	}
}

func TestCommonMigrations(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	runner := NewRunner(db, DriverSQLite)
	if err := runner.InitSchema(ctx); err != nil {
		t.Fatalf("InitSchema() error = %v", err)
	}

	// Apply common migrations
	result, err := runner.Up(ctx, CommonMigrations)
	if err != nil {
		t.Fatalf("Up() error = %v", err)
	}

	if len(result.Applied) != len(CommonMigrations) {
		t.Errorf("Applied = %d, want %d", len(result.Applied), len(CommonMigrations))
	}

	// Verify reactions table exists
	var count int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='reactions'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if count != 1 {
		t.Errorf("reactions table count = %d, want 1", count)
	}
}

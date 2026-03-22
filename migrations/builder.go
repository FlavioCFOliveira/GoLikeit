package migrations

// MigrationBuilder helps construct migrations with a fluent API.
type MigrationBuilder struct {
	version int64
	name    string
	up      []string
	down    []string
}

// NewMigration starts building a new migration.
func NewMigration(version int64, name string) *MigrationBuilder {
	return &MigrationBuilder{
		version: version,
		name:    name,
		up:      make([]string, 0),
		down:    make([]string, 0),
	}
}

// Up adds an up migration statement.
func (b *MigrationBuilder) Up(sql string) *MigrationBuilder {
	b.up = append(b.up, sql)
	return b
}

// Down adds a down migration statement.
func (b *MigrationBuilder) Down(sql string) *MigrationBuilder {
	b.down = append(b.down, sql)
	return b
}

// Build creates the Migration.
func (b *MigrationBuilder) Build() Migration {
	return Migration{
		Version: b.version,
		Name:    b.name,
		Up:      b.up,
		Down:    b.down,
	}
}

// CommonMigrations provides pre-defined migrations for the GoLikeit schema.
var CommonMigrations = []Migration{
	{
		Version: 1,
		Name:    "create_reactions_table",
		Up: []string{
			`CREATE TABLE IF NOT EXISTS reactions (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				entity_type TEXT NOT NULL,
				entity_id TEXT NOT NULL,
				reaction_type TEXT NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE (user_id, entity_type, entity_id)
			)`,
			`CREATE INDEX IF NOT EXISTS idx_reactions_user_id ON reactions(user_id)`,
			`CREATE INDEX IF NOT EXISTS idx_reactions_entity ON reactions(entity_type, entity_id)`,
			`CREATE INDEX IF NOT EXISTS idx_reactions_user_entity ON reactions(user_id, entity_type, entity_id)`,
			`CREATE INDEX IF NOT EXISTS idx_reactions_created_at ON reactions(created_at DESC)`,
		},
		Down: []string{
			`DROP INDEX IF EXISTS idx_reactions_created_at`,
			`DROP INDEX IF EXISTS idx_reactions_user_entity`,
			`DROP INDEX IF EXISTS idx_reactions_entity`,
			`DROP INDEX IF EXISTS idx_reactions_user_id`,
			`DROP TABLE IF EXISTS reactions`,
		},
	},
}

// GetMigrationsUpTo returns migrations with versions up to and including the specified version.
func GetMigrationsUpTo(migrations []Migration, target int64) []Migration {
	var result []Migration
	for _, m := range migrations {
		if m.Version <= target {
			result = append(result, m)
		}
	}
	return result
}

// GetMigrationByVersion returns a specific migration by version.
func GetMigrationByVersion(migrations []Migration, version int64) (Migration, bool) {
	for _, m := range migrations {
		if m.Version == version {
			return m, true
		}
	}
	return Migration{}, false
}

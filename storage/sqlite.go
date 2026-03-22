// Package storage provides SQLite storage implementation for the reaction system.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteConfig holds configuration for SQLite connection.
type SQLiteConfig struct {
	// DataSourceName is the SQLite connection string.
	// For in-memory: ":memory:"
	// For file: "/path/to/database.db"
	DataSourceName string

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
}

// DefaultSQLiteConfig returns a default configuration for file-based SQLite.
func DefaultSQLiteConfig() SQLiteConfig {
	return SQLiteConfig{
		DataSourceName:  "golikeit.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
}

// DefaultSQLiteMemoryConfig returns a configuration for in-memory SQLite.
func DefaultSQLiteMemoryConfig() SQLiteConfig {
	return SQLiteConfig{
		DataSourceName: ":memory:",
		MaxOpenConns:   1,
		MaxIdleConns:   1,
	}
}

// SQLiteStorage implements Repository using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance.
func NewSQLiteStorage(config SQLiteConfig) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", config.DataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	return &SQLiteStorage{db: db}, nil
}

// InitSchema initializes the database schema.
func (s *SQLiteStorage) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS reactions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		reaction_type TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (user_id, entity_type, entity_id)
	);

	CREATE INDEX IF NOT EXISTS idx_reactions_user_id ON reactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_reactions_entity ON reactions(entity_type, entity_id);
	CREATE INDEX IF NOT EXISTS idx_reactions_user_entity ON reactions(user_id, entity_type, entity_id);
	CREATE INDEX IF NOT EXISTS idx_reactions_created_at ON reactions(created_at DESC);
	`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// AddReaction adds or replaces a reaction for a user.
func (s *SQLiteStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	// Check for existing reaction
	var existingID string
	var existingType string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, reaction_type FROM reactions
		WHERE user_id = ? AND entity_type = ? AND entity_id = ?
	`, userID, target.EntityType, target.EntityID).Scan(&existingID, &existingType)

	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("failed to check existing reaction: %w", err)
	}

	if err == nil {
		// Update existing reaction
		_, err = s.db.ExecContext(ctx, `
			UPDATE reactions
			SET reaction_type = ?, created_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, reactionType, existingID)
		if err != nil {
			return false, fmt.Errorf("failed to update reaction: %w", err)
		}
		return true, nil
	}

	// Insert new reaction
	id := generateUUID()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reactions (id, user_id, entity_type, entity_id, reaction_type)
		VALUES (?, ?, ?, ?, ?)
	`, id, userID, target.EntityType, target.EntityID, reactionType)
	if err != nil {
		return false, fmt.Errorf("failed to insert reaction: %w", err)
	}

	return false, nil
}

// RemoveReaction removes a user's reaction.
func (s *SQLiteStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM reactions
		WHERE user_id = ? AND entity_type = ? AND entity_id = ?
	`, userID, target.EntityType, target.EntityID)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return golikeit.ErrReactionNotFound
	}

	return nil
}

// GetUserReaction retrieves a user's current reaction type for a target.
func (s *SQLiteStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	var reactionType string
	err := s.db.QueryRowContext(ctx, `
		SELECT reaction_type FROM reactions
		WHERE user_id = ? AND entity_type = ? AND entity_id = ?
	`, userID, target.EntityType, target.EntityID).Scan(&reactionType)

	if err == sql.ErrNoRows {
		return "", golikeit.ErrReactionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user reaction: %w", err)
	}

	return reactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (s *SQLiteStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reactions
			WHERE user_id = ? AND entity_type = ? AND entity_id = ?
		)
	`, userID, target.EntityType, target.EntityID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check user reaction: %w", err)
	}

	return exists, nil
}

// GetEntityCounts retrieves the reaction counts for an entity.
func (s *SQLiteStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT reaction_type, COUNT(*) FROM reactions
		WHERE entity_type = ? AND entity_id = ?
		GROUP BY reaction_type
	`, target.EntityType, target.EntityID)
	if err != nil {
		return golikeit.EntityCounts{}, fmt.Errorf("failed to get entity counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	var total int64

	for rows.Next() {
		var reactionType string
		var count int64
		if err := rows.Scan(&reactionType, &count); err != nil {
			return golikeit.EntityCounts{}, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[reactionType] = count
		total += count
	}

	if err := rows.Err(); err != nil {
		return golikeit.EntityCounts{}, fmt.Errorf("error iterating counts: %w", err)
	}

	return golikeit.EntityCounts{
		Counts: counts,
		Total:  total,
	}, nil
}

// GetUserReactions retrieves all reactions for a user with optional filters and pagination.
func (s *SQLiteStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Build query
	query := `
		SELECT user_id, entity_type, entity_id, reaction_type, created_at
		FROM reactions
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	if filters.EntityType != "" {
		query += " AND entity_type = ?"
		args = append(args, filters.EntityType)
	}
	if filters.ReactionType != "" {
		query += " AND reaction_type = ?"
		args = append(args, filters.ReactionType)
	}
	if filters.Since != nil {
		query += " AND created_at >= ?"
		args = append(args, filters.Since)
	}
	if filters.Until != nil {
		query += " AND created_at <= ?"
		args = append(args, filters.Until)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS sub"
	var total int64
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pag.Limit, pag.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query user reactions: %w", err)
	}
	defer rows.Close()

	var reactions []golikeit.UserReaction
	for rows.Next() {
		var r golikeit.UserReaction
		if err := rows.Scan(&r.UserID, &r.EntityType, &r.EntityID, &r.ReactionType, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan reaction: %w", err)
		}
		reactions = append(reactions, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, total, nil
}

// GetUserReactionCounts returns aggregated counts per reaction type for a user.
func (s *SQLiteStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	query := `
		SELECT reaction_type, COUNT(*) FROM reactions
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	if entityTypeFilter != "" {
		query += " AND entity_type = ?"
		args = append(args, entityTypeFilter)
	}

	query += " GROUP BY reaction_type"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reaction counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var reactionType string
		var count int64
		if err := rows.Scan(&reactionType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count: %w", err)
		}
		counts[reactionType] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counts: %w", err)
	}

	return counts, nil
}

// GetUserReactionsByType retrieves reactions of a specific type for a user.
func (s *SQLiteStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Get total
	var total int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reactions
		WHERE user_id = ? AND reaction_type = ?
	`, userID, reactionType).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get reactions
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, entity_type, entity_id, reaction_type, created_at
		FROM reactions
		WHERE user_id = ? AND reaction_type = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, reactionType, pag.Limit, pag.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query reactions: %w", err)
	}
	defer rows.Close()

	var reactions []golikeit.UserReaction
	for rows.Next() {
		var r golikeit.UserReaction
		if err := rows.Scan(&r.UserID, &r.EntityType, &r.EntityID, &r.ReactionType, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan reaction: %w", err)
		}
		reactions = append(reactions, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, total, nil
}

// GetEntityReactions retrieves all reactions on an entity with pagination.
func (s *SQLiteStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pag pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	// Get total
	var total int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reactions
		WHERE entity_type = ? AND entity_id = ?
	`, target.EntityType, target.EntityID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get reactions
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, reaction_type, created_at
		FROM reactions
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, target.EntityType, target.EntityID, pag.Limit, pag.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query reactions: %w", err)
	}
	defer rows.Close()

	var reactions []golikeit.EntityReaction
	for rows.Next() {
		var r golikeit.EntityReaction
		if err := rows.Scan(&r.UserID, &r.ReactionType, &r.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan reaction: %w", err)
		}
		reactions = append(reactions, r)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, total, nil
}

// GetRecentReactions retrieves recent reactions on an entity.
func (s *SQLiteStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT user_id, reaction_type, created_at
		FROM reactions
		WHERE entity_type = ? AND entity_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, target.EntityType, target.EntityID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent reactions: %w", err)
	}
	defer rows.Close()

	var reactions []golikeit.RecentUserReaction
	for rows.Next() {
		var r golikeit.RecentUserReaction
		if err := rows.Scan(&r.UserID, &r.ReactionType, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan reaction: %w", err)
		}
		reactions = append(reactions, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reactions: %w", err)
	}

	return reactions, nil
}

// GetLastReactionTime retrieves the timestamp of the most recent reaction on an entity.
func (s *SQLiteStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	var lastTimeStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM reactions
		WHERE entity_type = ? AND entity_id = ?
	`, target.EntityType, target.EntityID).Scan(&lastTimeStr)

	if err != nil {
		return nil, fmt.Errorf("failed to get last reaction time: %w", err)
	}

	if !lastTimeStr.Valid {
		return nil, nil
	}

	// SQLite datetime format: "2006-01-02 15:04:05"
	lastTime, err := time.Parse("2006-01-02 15:04:05", lastTimeStr.String)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last reaction time: %w", err)
	}

	return &lastTime, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (s *SQLiteStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	counts, err := s.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := s.GetRecentReactions(ctx, target, maxRecentUsers*10)
		if err != nil {
			return golikeit.EntityReactionDetail{}, err
		}

		// Group by reaction type
		for _, r := range recent {
			recentUsers[r.ReactionType] = append(recentUsers[r.ReactionType], r)
		}

		// Trim to maxRecentUsers per type
		for rt, users := range recentUsers {
			if len(users) > maxRecentUsers {
				recentUsers[rt] = users[:maxRecentUsers]
			}
		}
	}

	lastTime, err := s.GetLastReactionTime(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	return golikeit.EntityReactionDetail{
		EntityType:     target.EntityType,
		EntityID:       target.EntityID,
		TotalReactions: counts.Total,
		CountsByType:   counts.Counts,
		RecentUsers:    recentUsers,
		LastReaction:   lastTime,
	}, nil
}

// Close releases resources held by the storage.
func (s *SQLiteStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// generateUUID generates a simple UUID for SQLite (since SQLite doesn't have built-in UUID).
// In production, use github.com/google/uuid or similar.
func generateUUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// Package storage provides MariaDB storage implementation for the reaction system.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	_ "github.com/go-sql-driver/mysql"
)

// MariaDBConfig holds configuration for MariaDB connection.
type MariaDBConfig struct {
	// ConnectionString is the MariaDB connection string.
	// If provided, other fields are ignored.
	ConnectionString string

	// Host is the database host.
	Host string

	// Port is the database port.
	Port int

	// Database is the database name.
	Database string

	// User is the database user.
	User string

	// Password is the database password.
	Password string

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum lifetime of a connection.
	ConnMaxLifetime time.Duration
}

// DefaultMariaDBConfig returns a default configuration.
func DefaultMariaDBConfig() MariaDBConfig {
	return MariaDBConfig{
		Host:            "localhost",
		Port:            3306,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	}
}

// MariaDBStorage implements Repository using MariaDB.
type MariaDBStorage struct {
	db *sql.DB
}

// NewMariaDBStorage creates a new MariaDB storage instance.
func NewMariaDBStorage(config MariaDBConfig) (*MariaDBStorage, error) {
	connString := config.ConnectionString
	if connString == "" {
		connString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			config.User, config.Password, config.Host, config.Port, config.Database)
	}

	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open MariaDB database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MariaDB database: %w", err)
	}

	return &MariaDBStorage{db: db}, nil
}

// InitSchema initializes the database schema.
func (m *MariaDBStorage) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS reactions (
		id CHAR(36) PRIMARY KEY,
		user_id CHAR(36) NOT NULL,
		entity_type VARCHAR(64) NOT NULL,
		entity_id CHAR(36) NOT NULL,
		reaction_type VARCHAR(64) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY unique_reaction (user_id, entity_type, entity_id),
		INDEX idx_user_id (user_id),
		INDEX idx_entity (entity_type, entity_id),
		INDEX idx_created_at (created_at DESC)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	_, err := m.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// AddReaction adds or replaces a reaction for a user.
func (m *MariaDBStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	// Check for existing reaction
	var existingID string
	var existingType string
	err := m.db.QueryRowContext(ctx, `
		SELECT id, reaction_type FROM reactions
		WHERE user_id = ? AND entity_type = ? AND entity_id = ?
	`, userID, target.EntityType, target.EntityID).Scan(&existingID, &existingType)

	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("failed to check existing reaction: %w", err)
	}

	if err == nil {
		// Update existing reaction
		_, err = m.db.ExecContext(ctx, `
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
	_, err = m.db.ExecContext(ctx, `
		INSERT INTO reactions (id, user_id, entity_type, entity_id, reaction_type)
		VALUES (?, ?, ?, ?, ?)
	`, id, userID, target.EntityType, target.EntityID, reactionType)
	if err != nil {
		return false, fmt.Errorf("failed to insert reaction: %w", err)
	}

	return false, nil
}

// RemoveReaction removes a user's reaction.
func (m *MariaDBStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	result, err := m.db.ExecContext(ctx, `
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
func (m *MariaDBStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	var reactionType string
	err := m.db.QueryRowContext(ctx, `
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
func (m *MariaDBStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	var exists bool
	err := m.db.QueryRowContext(ctx, `
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
func (m *MariaDBStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	rows, err := m.db.QueryContext(ctx, `
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
func (m *MariaDBStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
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
	err := m.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pag.Limit, pag.Offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
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
func (m *MariaDBStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
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

	rows, err := m.db.QueryContext(ctx, query, args...)
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
func (m *MariaDBStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Get total
	var total int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reactions
		WHERE user_id = ? AND reaction_type = ?
	`, userID, reactionType).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get reactions
	rows, err := m.db.QueryContext(ctx, `
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
func (m *MariaDBStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pag pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	// Get total
	var total int64
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reactions
		WHERE entity_type = ? AND entity_id = ?
	`, target.EntityType, target.EntityID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get reactions
	rows, err := m.db.QueryContext(ctx, `
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
func (m *MariaDBStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	rows, err := m.db.QueryContext(ctx, `
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
func (m *MariaDBStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	var lastTime sql.NullTime
	err := m.db.QueryRowContext(ctx, `
		SELECT MAX(created_at) FROM reactions
		WHERE entity_type = ? AND entity_id = ?
	`, target.EntityType, target.EntityID).Scan(&lastTime)

	if err != nil {
		return nil, fmt.Errorf("failed to get last reaction time: %w", err)
	}

	if !lastTime.Valid {
		return nil, nil
	}

	return &lastTime.Time, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (m *MariaDBStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	counts, err := m.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := m.GetRecentReactions(ctx, target, maxRecentUsers*10)
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

	lastTime, err := m.GetLastReactionTime(ctx, target)
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
func (m *MariaDBStorage) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// Ping verifies connectivity to the MariaDB server.
func (m *MariaDBStorage) Ping(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return m.db.PingContext(ctx)
}

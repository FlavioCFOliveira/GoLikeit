// Package storage provides PostgreSQL storage implementation for the reaction system.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQLConfig holds configuration for PostgreSQL connection.
type PostgreSQLConfig struct {
	// ConnectionString is the PostgreSQL connection string.
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

	// SSLMode is the SSL mode (disable, require, verify-ca, verify-full).
	SSLMode string

	// MaxConns is the maximum number of connections in the pool.
	MaxConns int32

	// MinConns is the minimum number of connections in the pool.
	MinConns int32

	// MaxConnLifetime is the maximum lifetime of a connection.
	MaxConnLifetime time.Duration

	// MaxConnIdleTime is the maximum idle time of a connection.
	MaxConnIdleTime time.Duration
}

// DefaultPostgreSQLConfig returns a default configuration.
func DefaultPostgreSQLConfig() PostgreSQLConfig {
	return PostgreSQLConfig{
		Host:            "localhost",
		Port:            5432,
		SSLMode:         "disable",
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// PostgreSQLStorage implements Repository using PostgreSQL.
type PostgreSQLStorage struct {
	pool *pgxpool.Pool
}

// NewPostgreSQLStorage creates a new PostgreSQL storage instance.
func NewPostgreSQLStorage(ctx context.Context, config PostgreSQLConfig) (*PostgreSQLStorage, error) {
	connString := config.ConnectionString
	if connString == "" {
		connString = fmt.Sprintf(
			"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
			config.Host, config.Port, config.Database, config.User, config.Password, config.SSLMode,
		)
	}

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PostgreSQL config: %w", err)
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &PostgreSQLStorage{pool: pool}, nil
}

// InitSchema initializes the database schema.
func (p *PostgreSQLStorage) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS reactions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL,
		entity_type VARCHAR(64) NOT NULL,
		entity_id UUID NOT NULL,
		reaction_type VARCHAR(64) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		UNIQUE (user_id, entity_type, entity_id)
	);

	CREATE INDEX IF NOT EXISTS idx_reactions_user_id ON reactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_reactions_entity ON reactions(entity_type, entity_id);
	CREATE INDEX IF NOT EXISTS idx_reactions_user_entity ON reactions(user_id, entity_type, entity_id);
	CREATE INDEX IF NOT EXISTS idx_reactions_created_at ON reactions(created_at DESC);
	`

	_, err := p.pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// AddReaction adds or replaces a reaction for a user.
func (p *PostgreSQLStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	// Check for existing reaction
	var existingID string
	var existingType string
	err := p.pool.QueryRow(ctx, `
		SELECT id, reaction_type FROM reactions
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, target.EntityType, target.EntityID).Scan(&existingID, &existingType)

	if err != nil && err != pgx.ErrNoRows {
		return false, fmt.Errorf("failed to check existing reaction: %w", err)
	}

	if err == nil {
		// Update existing reaction
		_, err = p.pool.Exec(ctx, `
			UPDATE reactions
			SET reaction_type = $1, created_at = NOW()
			WHERE id = $2
		`, reactionType, existingID)
		if err != nil {
			return false, fmt.Errorf("failed to update reaction: %w", err)
		}
		return true, nil
	}

	// Insert new reaction
	_, err = p.pool.Exec(ctx, `
		INSERT INTO reactions (user_id, entity_type, entity_id, reaction_type)
		VALUES ($1, $2, $3, $4)
	`, userID, target.EntityType, target.EntityID, reactionType)
	if err != nil {
		return false, fmt.Errorf("failed to insert reaction: %w", err)
	}

	return false, nil
}

// RemoveReaction removes a user's reaction.
func (p *PostgreSQLStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	result, err := p.pool.Exec(ctx, `
		DELETE FROM reactions
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, target.EntityType, target.EntityID)
	if err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	if result.RowsAffected() == 0 {
		return golikeit.ErrReactionNotFound
	}

	return nil
}

// GetUserReaction retrieves a user's current reaction type for a target.
func (p *PostgreSQLStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	var reactionType string
	err := p.pool.QueryRow(ctx, `
		SELECT reaction_type FROM reactions
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, target.EntityType, target.EntityID).Scan(&reactionType)

	if err == pgx.ErrNoRows {
		return "", golikeit.ErrReactionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user reaction: %w", err)
	}

	return reactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (p *PostgreSQLStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM reactions
			WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
		)
	`, userID, target.EntityType, target.EntityID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check user reaction: %w", err)
	}

	return exists, nil
}

// GetEntityCounts retrieves the reaction counts for an entity.
func (p *PostgreSQLStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT reaction_type, COUNT(*) FROM reactions
		WHERE entity_type = $1 AND entity_id = $2
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
func (p *PostgreSQLStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Build query
	query := `
		SELECT user_id, entity_type, entity_id, reaction_type, created_at
		FROM reactions
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if filters.EntityType != "" {
		argCount++
		query += fmt.Sprintf(" AND entity_type = $%d", argCount)
		args = append(args, filters.EntityType)
	}
	if filters.ReactionType != "" {
		argCount++
		query += fmt.Sprintf(" AND reaction_type = $%d", argCount)
		args = append(args, filters.ReactionType)
	}
	if filters.Since != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filters.Since)
	}
	if filters.Until != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filters.Until)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS sub"
	var total int64
	err := p.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, pag.Limit)
	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, pag.Offset)

	rows, err := p.pool.Query(ctx, query, args...)
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
func (p *PostgreSQLStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	query := `
		SELECT reaction_type, COUNT(*) FROM reactions
		WHERE user_id = $1
	`
	args := []interface{}{userID}

	if entityTypeFilter != "" {
		query += " AND entity_type = $2"
		args = append(args, entityTypeFilter)
	}

	query += " GROUP BY reaction_type"

	rows, err := p.pool.Query(ctx, query, args...)
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
func (p *PostgreSQLStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Get total
	var total int64
	err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM reactions
		WHERE user_id = $1 AND reaction_type = $2
	`, userID, reactionType).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get reactions
	rows, err := p.pool.Query(ctx, `
		SELECT user_id, entity_type, entity_id, reaction_type, created_at
		FROM reactions
		WHERE user_id = $1 AND reaction_type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
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
func (p *PostgreSQLStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pag pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	// Get total
	var total int64
	err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM reactions
		WHERE entity_type = $1 AND entity_id = $2
	`, target.EntityType, target.EntityID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get reactions
	rows, err := p.pool.Query(ctx, `
		SELECT user_id, reaction_type, created_at
		FROM reactions
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
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
func (p *PostgreSQLStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT user_id, reaction_type, created_at
		FROM reactions
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3
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
func (p *PostgreSQLStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	var lastTime *time.Time
	err := p.pool.QueryRow(ctx, `
		SELECT MAX(created_at) FROM reactions
		WHERE entity_type = $1 AND entity_id = $2
	`, target.EntityType, target.EntityID).Scan(&lastTime)

	if err != nil {
		return nil, fmt.Errorf("failed to get last reaction time: %w", err)
	}

	return lastTime, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (p *PostgreSQLStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	counts, err := p.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := p.GetRecentReactions(ctx, target, maxRecentUsers*10)
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

	lastTime, err := p.GetLastReactionTime(ctx, target)
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
func (p *PostgreSQLStorage) Close() error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

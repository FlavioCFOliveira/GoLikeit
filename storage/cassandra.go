// Package storage provides Cassandra storage implementation for the reaction system.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"github.com/gocql/gocql"
)

// CassandraConfig holds configuration for Cassandra connection.
type CassandraConfig struct {
	// Hosts is a list of Cassandra node addresses.
	Hosts []string

	// Keyspace is the keyspace name.
	Keyspace string

	// Consistency is the default consistency level.
	Consistency string

	// Timeout is the query timeout.
	Timeout time.Duration

	// ConnectTimeout is the connection timeout.
	ConnectTimeout time.Duration
}

// DefaultCassandraConfig returns a default configuration.
func DefaultCassandraConfig() CassandraConfig {
	return CassandraConfig{
		Hosts:          []string{"localhost:9042"},
		Keyspace:       "golikeit",
		Consistency:    "QUORUM",
		Timeout:        5 * time.Second,
		ConnectTimeout: 10 * time.Second,
	}
}

// CassandraStorage implements Repository using Cassandra.
type CassandraStorage struct {
	session *gocql.Session
}

// NewCassandraStorage creates a new Cassandra storage instance.
func NewCassandraStorage(config CassandraConfig) (*CassandraStorage, error) {
	cluster := gocql.NewCluster(config.Hosts...)
	cluster.Keyspace = config.Keyspace
	cluster.Timeout = config.Timeout
	cluster.ConnectTimeout = config.ConnectTimeout

	// Set consistency level
	switch config.Consistency {
	case "ONE":
		cluster.Consistency = gocql.One
	case "TWO":
		cluster.Consistency = gocql.Two
	case "THREE":
		cluster.Consistency = gocql.Three
	case "QUORUM":
		cluster.Consistency = gocql.Quorum
	case "ALL":
		cluster.Consistency = gocql.All
	case "LOCAL_QUORUM":
		cluster.Consistency = gocql.LocalQuorum
	case "EACH_QUORUM":
		cluster.Consistency = gocql.EachQuorum
	case "LOCAL_ONE":
		cluster.Consistency = gocql.LocalOne
	default:
		cluster.Consistency = gocql.Quorum
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create Cassandra session: %w", err)
	}

	return &CassandraStorage{session: session}, nil
}

// InitSchema initializes the database schema.
func (c *CassandraStorage) InitSchema(ctx context.Context) error {
	// Create keyspace if not exists
	keyspaceCQL := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}
	`, "golikeit")

	if err := c.session.Query(keyspaceCQL).Exec(); err != nil {
		return fmt.Errorf("failed to create keyspace: %w", err)
	}

	// Create reactions table (by user)
	reactionsByUserCQL := `
		CREATE TABLE IF NOT EXISTS reactions_by_user (
			user_id text,
			entity_type text,
			entity_id text,
			reaction_type text,
			created_at timestamp,
			PRIMARY KEY ((user_id), entity_type, entity_id)
		)
	`
	if err := c.session.Query(reactionsByUserCQL).Exec(); err != nil {
		return fmt.Errorf("failed to create reactions_by_user table: %w", err)
	}

	// Create reactions table (by entity)
	reactionsByEntityCQL := `
		CREATE TABLE IF NOT EXISTS reactions_by_entity (
			entity_type text,
			entity_id text,
			user_id text,
			reaction_type text,
			created_at timestamp,
			PRIMARY KEY ((entity_type, entity_id), created_at, user_id)
		)
		WITH CLUSTERING ORDER BY (created_at DESC)
	`
	if err := c.session.Query(reactionsByEntityCQL).Exec(); err != nil {
		return fmt.Errorf("failed to create reactions_by_entity table: %w", err)
	}

	// Create entity counts table (materialized view)
	entityCountsCQL := `
		CREATE TABLE IF NOT EXISTS entity_counts (
			entity_type text,
			entity_id text,
			reaction_type text,
			count counter,
			PRIMARY KEY ((entity_type, entity_id), reaction_type)
		)
	`
	if err := c.session.Query(entityCountsCQL).Exec(); err != nil {
		return fmt.Errorf("failed to create entity_counts table: %w", err)
	}

	return nil
}

// AddReaction adds or replaces a reaction for a user.
func (c *CassandraStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	// Check if reaction exists
	var existingType string
	var existingCreatedAt time.Time
	err := c.session.Query(
		"SELECT reaction_type, created_at FROM reactions_by_user WHERE user_id = ? AND entity_type = ? AND entity_id = ?",
		userID, target.EntityType, target.EntityID,
	).WithContext(ctx).Scan(&existingType, &existingCreatedAt)

	isReplacement := err == nil

	if isReplacement {
		// Update: decrement old count
		if err := c.session.Query(
			"UPDATE entity_counts SET count = count - 1 WHERE entity_type = ? AND entity_id = ? AND reaction_type = ?",
			target.EntityType, target.EntityID, existingType,
		).WithContext(ctx).Exec(); err != nil {
			return false, fmt.Errorf("failed to decrement old count: %w", err)
		}

		// Remove old entry from reactions_by_entity
		if err := c.session.Query(
			"DELETE FROM reactions_by_entity WHERE entity_type = ? AND entity_id = ? AND created_at = ? AND user_id = ?",
			target.EntityType, target.EntityID, existingCreatedAt, userID,
		).WithContext(ctx).Exec(); err != nil {
			return false, fmt.Errorf("failed to delete old reaction: %w", err)
		}

		// Update reactions_by_user
		if err := c.session.Query(
			"UPDATE reactions_by_user SET reaction_type = ?, created_at = ? WHERE user_id = ? AND entity_type = ? AND entity_id = ?",
			reactionType, time.Now().UTC(), userID, target.EntityType, target.EntityID,
		).WithContext(ctx).Exec(); err != nil {
			return false, fmt.Errorf("failed to update reaction: %w", err)
		}
	} else {
		// Insert new reaction
		now := time.Now().UTC()
		if err := c.session.Query(
			"INSERT INTO reactions_by_user (user_id, entity_type, entity_id, reaction_type, created_at) VALUES (?, ?, ?, ?, ?)",
			userID, target.EntityType, target.EntityID, reactionType, now,
		).WithContext(ctx).Exec(); err != nil {
			return false, fmt.Errorf("failed to insert reaction: %w", err)
		}
	}

	// Increment new count
	if err := c.session.Query(
		"UPDATE entity_counts SET count = count + 1 WHERE entity_type = ? AND entity_id = ? AND reaction_type = ?",
		target.EntityType, target.EntityID, reactionType,
	).WithContext(ctx).Exec(); err != nil {
		return false, fmt.Errorf("failed to increment count: %w", err)
	}

	// Insert into reactions_by_entity
	if err := c.session.Query(
		"INSERT INTO reactions_by_entity (entity_type, entity_id, user_id, reaction_type, created_at) VALUES (?, ?, ?, ?, ?)",
		target.EntityType, target.EntityID, userID, reactionType, time.Now().UTC(),
	).WithContext(ctx).Exec(); err != nil {
		return false, fmt.Errorf("failed to insert entity reaction: %w", err)
	}

	return isReplacement, nil
}

// RemoveReaction removes a user's reaction.
func (c *CassandraStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	// Get existing reaction
	var reactionType string
	var createdAt time.Time
	err := c.session.Query(
		"SELECT reaction_type, created_at FROM reactions_by_user WHERE user_id = ? AND entity_type = ? AND entity_id = ?",
		userID, target.EntityType, target.EntityID,
	).WithContext(ctx).Scan(&reactionType, &createdAt)

	if err == gocql.ErrNotFound {
		return golikeit.ErrReactionNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get existing reaction: %w", err)
	}

	// Delete from reactions_by_user
	if err := c.session.Query(
		"DELETE FROM reactions_by_user WHERE user_id = ? AND entity_type = ? AND entity_id = ?",
		userID, target.EntityType, target.EntityID,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to delete reaction: %w", err)
	}

	// Delete from reactions_by_entity
	if err := c.session.Query(
		"DELETE FROM reactions_by_entity WHERE entity_type = ? AND entity_id = ? AND created_at = ? AND user_id = ?",
		target.EntityType, target.EntityID, createdAt, userID,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to delete entity reaction: %w", err)
	}

	// Decrement count
	if err := c.session.Query(
		"UPDATE entity_counts SET count = count - 1 WHERE entity_type = ? AND entity_id = ? AND reaction_type = ?",
		target.EntityType, target.EntityID, reactionType,
	).WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("failed to decrement count: %w", err)
	}

	return nil
}

// GetUserReaction retrieves a user's current reaction type for a target.
func (c *CassandraStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	var reactionType string
	err := c.session.Query(
		"SELECT reaction_type FROM reactions_by_user WHERE user_id = ? AND entity_type = ? AND entity_id = ?",
		userID, target.EntityType, target.EntityID,
	).WithContext(ctx).Scan(&reactionType)

	if err == gocql.ErrNotFound {
		return "", golikeit.ErrReactionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user reaction: %w", err)
	}

	return reactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (c *CassandraStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	var count int
	err := c.session.Query(
		"SELECT COUNT(*) FROM reactions_by_user WHERE user_id = ? AND entity_type = ? AND entity_id = ?",
		userID, target.EntityType, target.EntityID,
	).WithContext(ctx).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check user reaction: %w", err)
	}

	return count > 0, nil
}

// GetEntityCounts retrieves the reaction counts for an entity.
func (c *CassandraStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	iter := c.session.Query(
		"SELECT reaction_type, count FROM entity_counts WHERE entity_type = ? AND entity_id = ?",
		target.EntityType, target.EntityID,
	).WithContext(ctx).Iter()

	counts := make(map[string]int64)
	var total int64

	var reactionType string
	var count int64
	for iter.Scan(&reactionType, &count) {
		if count > 0 {
			counts[reactionType] = count
			total += count
		}
	}

	if err := iter.Close(); err != nil {
		return golikeit.EntityCounts{}, fmt.Errorf("failed to get entity counts: %w", err)
	}

	return golikeit.EntityCounts{
		Counts: counts,
		Total:  total,
	}, nil
}

// GetUserReactions retrieves all reactions for a user with optional filters and pagination.
func (c *CassandraStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Build query based on filters
	query := "SELECT entity_type, entity_id, reaction_type, created_at FROM reactions_by_user WHERE user_id = ?"
	args := []interface{}{userID}

	if filters.EntityType != "" {
		query += " AND entity_type = ?"
		args = append(args, filters.EntityType)
	}

	iter := c.session.Query(query, args...).WithContext(ctx).Iter()

	var reactions []golikeit.UserReaction
	var entityType, entityID, reactionType string
	var createdAt time.Time

	for iter.Scan(&entityType, &entityID, &reactionType, &createdAt) {
		// Apply reaction type filter in memory (Cassandra doesn't support OR in WHERE)
		if filters.ReactionType != "" && reactionType != filters.ReactionType {
			continue
		}
		// Apply date filters in memory
		if filters.Since != nil && createdAt.Before(*filters.Since) {
			continue
		}
		if filters.Until != nil && createdAt.After(*filters.Until) {
			continue
		}

		reactions = append(reactions, golikeit.UserReaction{
			UserID:       userID,
			EntityType:   entityType,
			EntityID:     entityID,
			ReactionType: reactionType,
			CreatedAt:    createdAt,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, 0, fmt.Errorf("failed to get user reactions: %w", err)
	}

	total := int64(len(reactions))

	// Apply pagination in memory
	start := pag.Offset
	if start > len(reactions) {
		start = len(reactions)
	}
	end := start + pag.Limit
	if end > len(reactions) {
		end = len(reactions)
	}

	return reactions[start:end], total, nil
}

// GetUserReactionCounts returns aggregated counts per reaction type for a user.
func (c *CassandraStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	query := "SELECT entity_type, reaction_type FROM reactions_by_user WHERE user_id = ?"
	args := []interface{}{userID}

	if entityTypeFilter != "" {
		query += " AND entity_type = ?"
		args = append(args, entityTypeFilter)
	}

	iter := c.session.Query(query, args...).WithContext(ctx).Iter()

	counts := make(map[string]int64)
	var entityType, reactionType string

	for iter.Scan(&entityType, &reactionType) {
		if entityTypeFilter != "" && entityType != entityTypeFilter {
			continue
		}
		counts[reactionType]++
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get user reaction counts: %w", err)
	}

	return counts, nil
}

// GetUserReactionsByType retrieves reactions of a specific type for a user.
func (c *CassandraStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	// Cassandra doesn't support filtering by reaction_type in the WHERE clause
	// when it's not part of the primary key, so we filter in memory
	iter := c.session.Query(
		"SELECT entity_type, entity_id, reaction_type, created_at FROM reactions_by_user WHERE user_id = ?",
		userID,
	).WithContext(ctx).Iter()

	var reactions []golikeit.UserReaction
	var entityType, entityID, rt string
	var createdAt time.Time

	for iter.Scan(&entityType, &entityID, &rt, &createdAt) {
		if rt == reactionType {
			reactions = append(reactions, golikeit.UserReaction{
				UserID:       userID,
				EntityType:   entityType,
				EntityID:     entityID,
				ReactionType: rt,
				CreatedAt:    createdAt,
			})
		}
	}

	if err := iter.Close(); err != nil {
		return nil, 0, fmt.Errorf("failed to get user reactions: %w", err)
	}

	total := int64(len(reactions))

	// Apply pagination
	start := pag.Offset
	if start > len(reactions) {
		start = len(reactions)
	}
	end := start + pag.Limit
	if end > len(reactions) {
		end = len(reactions)
	}

	return reactions[start:end], total, nil
}

// GetEntityReactions retrieves all reactions on an entity with pagination.
func (c *CassandraStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pag pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	// Cassandra pagination using token or LIMIT/OFFSET pattern
	iter := c.session.Query(
		"SELECT user_id, reaction_type, created_at FROM reactions_by_entity WHERE entity_type = ? AND entity_id = ? LIMIT ?",
		target.EntityType, target.EntityID, pag.Limit+pag.Offset,
	).WithContext(ctx).Iter()

	var reactions []golikeit.EntityReaction
	var userID, reactionType string
	var createdAt time.Time

	for iter.Scan(&userID, &reactionType, &createdAt) {
		reactions = append(reactions, golikeit.EntityReaction{
			UserID:       userID,
			ReactionType: reactionType,
			CreatedAt:    createdAt,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, 0, fmt.Errorf("failed to get entity reactions: %w", err)
	}

	total := int64(len(reactions))

	// Apply offset
	start := pag.Offset
	if start > len(reactions) {
		start = len(reactions)
	}
	end := start + pag.Limit
	if end > len(reactions) {
		end = len(reactions)
	}

	return reactions[start:end], total, nil
}

// GetRecentReactions retrieves recent reactions on an entity.
func (c *CassandraStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	iter := c.session.Query(
		"SELECT user_id, reaction_type, created_at FROM reactions_by_entity WHERE entity_type = ? AND entity_id = ? LIMIT ?",
		target.EntityType, target.EntityID, limit,
	).WithContext(ctx).Iter()

	var reactions []golikeit.RecentUserReaction
	var userID, reactionType string
	var createdAt time.Time

	for iter.Scan(&userID, &reactionType, &createdAt) {
		reactions = append(reactions, golikeit.RecentUserReaction{
			UserID:       userID,
			ReactionType: reactionType,
			Timestamp:    createdAt,
		})
	}

	if err := iter.Close(); err != nil {
		return nil, fmt.Errorf("failed to get recent reactions: %w", err)
	}

	return reactions, nil
}

// GetLastReactionTime retrieves the timestamp of the most recent reaction on an entity.
func (c *CassandraStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	var createdAt time.Time
	err := c.session.Query(
		"SELECT created_at FROM reactions_by_entity WHERE entity_type = ? AND entity_id = ? LIMIT 1",
		target.EntityType, target.EntityID,
	).WithContext(ctx).Scan(&createdAt)

	if err == gocql.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last reaction time: %w", err)
	}

	return &createdAt, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (c *CassandraStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	counts, err := c.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := c.GetRecentReactions(ctx, target, maxRecentUsers*10)
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

	lastTime, err := c.GetLastReactionTime(ctx, target)
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
func (c *CassandraStorage) Close() error {
	if c.session != nil {
		c.session.Close()
	}
	return nil
}

// Ping verifies connectivity to the Cassandra cluster.
func (c *CassandraStorage) Ping(_ context.Context) error {
	if c.session == nil {
		return fmt.Errorf("session is nil")
	}
	// A lightweight query that validates connectivity without touching data.
	return c.session.Query("SELECT now() FROM system.local").Exec()
}

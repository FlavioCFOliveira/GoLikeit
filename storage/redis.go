// Package storage provides Redis storage implementation for the reaction system.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"github.com/redis/go-redis/v9"
)

// RedisConfig holds configuration for Redis connection.
type RedisConfig struct {
	// Address is the Redis server address (host:port).
	Address string

	// Password is the Redis password (optional).
	Password string

	// DB is the Redis database number.
	DB int

	// PoolSize is the connection pool size.
	PoolSize int

	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int

	// MaxRetries is the maximum number of retries.
	MaxRetries int
}

// DefaultRedisConfig returns a default configuration.
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Address:      "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		MaxRetries:   3,
	}
}

// RedisStorage implements Repository using Redis.
type RedisStorage struct {
	client *redis.Client
}

// NewRedisStorage creates a new Redis storage instance.
func NewRedisStorage(config RedisConfig) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		MaxRetries:   config.MaxRetries,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return &RedisStorage{client: client}, nil
}

// redisReactionKey generates a key for a user reaction.
// Pattern: reaction:{user_id}:{entity_type}:{entity_id}
func redisReactionKey(userID, entityType, entityID string) string {
	return fmt.Sprintf("reaction:%s:%s:%s", userID, entityType, entityID)
}

// countsKey generates a key for entity counts.
// Pattern: counts:{{entity_type}:{entity_id}}
func countsKey(entityType, entityID string) string {
	return fmt.Sprintf("counts:{%s:%s}", entityType, entityID)
}

// userReactionsKey generates a key for user's reactions list.
func userReactionsKey(userID string) string {
	return fmt.Sprintf("user:%s:reactions", userID)
}

// entityReactionsKey generates a key for entity's reactions list.
func entityReactionsKey(entityType, entityID string) string {
	return fmt.Sprintf("entity:{%s:%s}:reactions", entityType, entityID)
}

// AddReaction adds or replaces a reaction for a user.
func (r *RedisStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	reactionK := redisReactionKey(userID, target.EntityType, target.EntityID)
	countsK := countsKey(target.EntityType, target.EntityID)
	userReactionsK := userReactionsKey(userID)
	entityReactionsK := entityReactionsKey(target.EntityType, target.EntityID)

	// Check for existing reaction
	existingType, err := r.client.HGet(ctx, reactionK, "type").Result()
	isReplacement := err == nil && existingType != ""

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	if isReplacement {
		// Decrement old type count
		pipe.HIncrBy(ctx, countsK, existingType, -1)
	}

	// Set new reaction
	pipe.HSet(ctx, reactionK, map[string]interface{}{
		"type":       reactionType,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})

	// Increment new type count
	pipe.HIncrBy(ctx, countsK, reactionType, 1)

	// Add to user's reactions set
	pipe.ZAdd(ctx, userReactionsK, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: fmt.Sprintf("%s:%s", target.EntityType, target.EntityID),
	})

	// Add to entity's reactions set
	pipe.ZAdd(ctx, entityReactionsK, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: userID,
	})

	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to add reaction: %w", err)
	}

	return isReplacement, nil
}

// RemoveReaction removes a user's reaction.
func (r *RedisStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	reactionK := redisReactionKey(userID, target.EntityType, target.EntityID)
	countsK := countsKey(target.EntityType, target.EntityID)
	userReactionsK := userReactionsKey(userID)
	entityReactionsK := entityReactionsKey(target.EntityType, target.EntityID)

	// Get existing reaction type
	existingType, err := r.client.HGet(ctx, reactionK, "type").Result()
	if err == redis.Nil {
		return golikeit.ErrReactionNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get existing reaction: %w", err)
	}

	// Use pipeline for atomic operations
	pipe := r.client.Pipeline()

	// Delete reaction
	pipe.Del(ctx, reactionK)

	// Decrement count
	pipe.HIncrBy(ctx, countsK, existingType, -1)

	// Remove from user's reactions
	pipe.ZRem(ctx, userReactionsK, fmt.Sprintf("%s:%s", target.EntityType, target.EntityID))

	// Remove from entity's reactions
	pipe.ZRem(ctx, entityReactionsK, userID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}

	return nil
}

// GetUserReaction retrieves a user's current reaction type for a target.
func (r *RedisStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	reactionK := redisReactionKey(userID, target.EntityType, target.EntityID)

	reactionType, err := r.client.HGet(ctx, reactionK, "type").Result()
	if err == redis.Nil {
		return "", golikeit.ErrReactionNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get user reaction: %w", err)
	}

	return reactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (r *RedisStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	reactionK := redisReactionKey(userID, target.EntityType, target.EntityID)

	exists, err := r.client.Exists(ctx, reactionK).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check user reaction: %w", err)
	}

	return exists > 0, nil
}

// GetEntityCounts retrieves the reaction counts for an entity.
func (r *RedisStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	countsK := countsKey(target.EntityType, target.EntityID)

	counts, err := r.client.HGetAll(ctx, countsK).Result()
	if err != nil {
		return golikeit.EntityCounts{}, fmt.Errorf("failed to get entity counts: %w", err)
	}

	result := make(map[string]int64)
	var total int64

	for reactionType, countStr := range counts {
		count, _ := parseInt64(countStr)
		if count > 0 {
			result[reactionType] = count
			total += count
		}
	}

	return golikeit.EntityCounts{
		Counts: result,
		Total:  total,
	}, nil
}

// GetUserReactions retrieves all reactions for a user with optional filters and pagination.
func (r *RedisStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	userReactionsK := userReactionsKey(userID)

	// Get all reactions for user
	members, err := r.client.ZRevRange(ctx, userReactionsK, 0, -1).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user reactions: %w", err)
	}

	var reactions []golikeit.UserReaction
	for _, member := range members {
		var entityType, entityID string
		fmt.Sscanf(member, "%s:%s", &entityType, &entityID)

		reactionType, err := r.GetUserReaction(ctx, userID, golikeit.EntityTarget{EntityType: entityType, EntityID: entityID})
		if err != nil {
			continue
		}

		// Apply filters
		if filters.EntityType != "" && entityType != filters.EntityType {
			continue
		}
		if filters.ReactionType != "" && reactionType != filters.ReactionType {
			continue
		}

		reactions = append(reactions, golikeit.UserReaction{
			UserID:       userID,
			EntityType:   entityType,
			EntityID:     entityID,
			ReactionType: reactionType,
		})
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

// GetUserReactionCounts returns aggregated counts per reaction type for a user.
func (r *RedisStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	userReactionsK := userReactionsKey(userID)

	members, err := r.client.ZRevRange(ctx, userReactionsK, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user reactions: %w", err)
	}

	counts := make(map[string]int64)

	for _, member := range members {
		var entityType, entityID string
		fmt.Sscanf(member, "%s:%s", &entityType, &entityID)

		if entityTypeFilter != "" && entityType != entityTypeFilter {
			continue
		}

		reactionType, err := r.GetUserReaction(ctx, userID, golikeit.EntityTarget{EntityType: entityType, EntityID: entityID})
		if err != nil {
			continue
		}

		counts[reactionType]++
	}

	return counts, nil
}

// GetUserReactionsByType retrieves reactions of a specific type for a user.
func (r *RedisStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pag pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	userReactionsK := userReactionsKey(userID)

	members, err := r.client.ZRevRange(ctx, userReactionsK, 0, -1).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user reactions: %w", err)
	}

	var reactions []golikeit.UserReaction
	for _, member := range members {
		var entityType, entityID string
		fmt.Sscanf(member, "%s:%s", &entityType, &entityID)

		currentType, err := r.GetUserReaction(ctx, userID, golikeit.EntityTarget{EntityType: entityType, EntityID: entityID})
		if err != nil {
			continue
		}

		if currentType == reactionType {
			reactions = append(reactions, golikeit.UserReaction{
				UserID:       userID,
				EntityType:   entityType,
				EntityID:     entityID,
				ReactionType: reactionType,
			})
		}
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
func (r *RedisStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pag pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	entityReactionsK := entityReactionsKey(target.EntityType, target.EntityID)

	members, err := r.client.ZRevRange(ctx, entityReactionsK, 0, -1).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get entity reactions: %w", err)
	}

	var reactions []golikeit.EntityReaction
	for _, userID := range members {
		reactionType, err := r.GetUserReaction(ctx, userID, target)
		if err != nil {
			continue
		}

		reactions = append(reactions, golikeit.EntityReaction{
			UserID:       userID,
			ReactionType: reactionType,
		})
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

// GetRecentReactions retrieves recent reactions on an entity.
func (r *RedisStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	entityReactionsK := entityReactionsKey(target.EntityType, target.EntityID)

	membersWithScores, err := r.client.ZRevRangeWithScores(ctx, entityReactionsK, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get recent reactions: %w", err)
	}

	var reactions []golikeit.RecentUserReaction
	for _, z := range membersWithScores {
		userID := z.Member.(string)
		reactionType, err := r.GetUserReaction(ctx, userID, target)
		if err != nil {
			continue
		}

		reactions = append(reactions, golikeit.RecentUserReaction{
			UserID:       userID,
			ReactionType: reactionType,
			Timestamp:    time.Unix(int64(z.Score), 0),
		})
	}

	return reactions, nil
}

// GetLastReactionTime retrieves the timestamp of the most recent reaction on an entity.
func (r *RedisStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	entityReactionsK := entityReactionsKey(target.EntityType, target.EntityID)

	result, err := r.client.ZRevRangeWithScores(ctx, entityReactionsK, 0, 0).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get last reaction time: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	lastTime := time.Unix(int64(result[0].Score), 0)
	return &lastTime, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (r *RedisStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	counts, err := r.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := r.GetRecentReactions(ctx, target, maxRecentUsers*10)
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

	lastTime, err := r.GetLastReactionTime(ctx, target)
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
func (r *RedisStorage) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// parseInt64 parses a string to int64, returning 0 on error.
func parseInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

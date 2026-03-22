package cache

import (
	"fmt"
	"time"
)

// UserReaction represents a user's reaction to an entity.
type UserReaction struct {
	// ReactionType is the type of reaction.
	ReactionType string

	// Timestamp is when the reaction was recorded.
	Timestamp time.Time
}

// EntityCounts represents the aggregated reaction counts for an entity.
type EntityCounts struct {
	// CountsByType maps reaction types to their counts.
	CountsByType map[string]int64

	// TotalReactions is the total number of reactions.
	TotalReactions int64

	// Timestamp is when the counts were computed.
	Timestamp time.Time
}

// ReactionCache is a specialized cache for reaction data.
// It provides separate caches for user reactions and entity counts.
type ReactionCache struct {
	userCache    Cache
	entityCache  Cache
	enabled      bool
	userTTL      time.Duration
	entityTTL    time.Duration
}

// ReactionCacheConfig configures the reaction cache.
type ReactionCacheConfig struct {
	// Enabled controls whether caching is enabled.
	Enabled bool

	// MaxUserEntries is the maximum number of user reaction entries.
	MaxUserEntries int

	// MaxEntityEntries is the maximum number of entity count entries.
	MaxEntityEntries int

	// UserTTL is the TTL for user reaction entries.
	UserTTL time.Duration

	// EntityTTL is the TTL for entity count entries.
	EntityTTL time.Duration
}

// DefaultReactionCacheConfig returns a default configuration.
func DefaultReactionCacheConfig() ReactionCacheConfig {
	return ReactionCacheConfig{
		Enabled:          true,
		MaxUserEntries:   10000,
		MaxEntityEntries: 10000,
		UserTTL:          60 * time.Second,
		EntityTTL:        300 * time.Second,
	}
}

// NewReactionCache creates a new reaction cache with the given configuration.
func NewReactionCache(config ReactionCacheConfig) *ReactionCache {
	if !config.Enabled {
		return &ReactionCache{
			userCache:   New(1),
			entityCache: New(1),
			enabled:     false,
			userTTL:     config.UserTTL,
			entityTTL:   config.EntityTTL,
		}
	}

	return &ReactionCache{
		userCache:   New(config.MaxUserEntries),
		entityCache: New(config.MaxEntityEntries),
		enabled:     true,
		userTTL:     config.UserTTL,
		entityTTL:   config.EntityTTL,
	}
}

// IsEnabled returns true if caching is enabled.
func (c *ReactionCache) IsEnabled() bool {
	return c.enabled
}

// GetUserReaction retrieves a user's reaction from the cache.
func (c *ReactionCache) GetUserReaction(userID, entityType, entityID string) (UserReaction, bool) {
	if !c.enabled {
		return UserReaction{}, false
	}

	key := userReactionKey(userID, entityType, entityID)
	value, ok := c.userCache.Get(key)
	if !ok {
		return UserReaction{}, false
	}

	reaction, ok := value.(UserReaction)
	return reaction, ok
}

// SetUserReaction stores a user's reaction in the cache.
func (c *ReactionCache) SetUserReaction(userID, entityType, entityID string, reaction UserReaction) {
	if !c.enabled {
		return
	}

	key := userReactionKey(userID, entityType, entityID)
	c.userCache.Set(key, reaction, c.userTTL)
}

// DeleteUserReaction removes a user's reaction from the cache.
func (c *ReactionCache) DeleteUserReaction(userID, entityType, entityID string) {
	if !c.enabled {
		return
	}

	key := userReactionKey(userID, entityType, entityID)
	c.userCache.Delete(key)
}

// GetEntityCounts retrieves the reaction counts for an entity from the cache.
func (c *ReactionCache) GetEntityCounts(entityType, entityID string) (EntityCounts, bool) {
	if !c.enabled {
		return EntityCounts{}, false
	}

	key := entityCountsKey(entityType, entityID)
	value, ok := c.entityCache.Get(key)
	if !ok {
		return EntityCounts{}, false
	}

	counts, ok := value.(EntityCounts)
	return counts, ok
}

// SetEntityCounts stores the reaction counts for an entity in the cache.
func (c *ReactionCache) SetEntityCounts(entityType, entityID string, counts EntityCounts) {
	if !c.enabled {
		return
	}

	key := entityCountsKey(entityType, entityID)
	c.entityCache.Set(key, counts, c.entityTTL)
}

// DeleteEntityCounts removes the reaction counts for an entity from the cache.
func (c *ReactionCache) DeleteEntityCounts(entityType, entityID string) {
	if !c.enabled {
		return
	}

	key := entityCountsKey(entityType, entityID)
	c.entityCache.Delete(key)
}

// InvalidateByEntity invalidates all cache entries related to an entity.
// This includes:
// - Entity counts for the entity
// - All user reactions for the entity (matching prefix)
func (c *ReactionCache) InvalidateByEntity(entityType, entityID string) {
	if !c.enabled {
		return
	}

	// Delete by entity prefix - this covers both counts and user reactions
	prefix := entityPrefix(entityType, entityID)
	c.userCache.DeleteByPrefix(prefix)
	c.entityCache.DeleteByPrefix(prefix)
}

// Clear clears all entries from the cache.
func (c *ReactionCache) Clear() {
	if !c.enabled {
		return
	}

	c.userCache.Clear()
	c.entityCache.Clear()
}

// Stats returns combined cache statistics.
func (c *ReactionCache) Stats() Stats {
	if !c.enabled {
		return Stats{}
	}

	userStats := c.userCache.Stats()
	entityStats := c.entityCache.Stats()

	return Stats{
		Hits:       userStats.Hits + entityStats.Hits,
		Misses:     userStats.Misses + entityStats.Misses,
		Entries:    userStats.Entries + entityStats.Entries,
		MaxEntries: userStats.MaxEntries + entityStats.MaxEntries,
	}
}

// UserCacheStats returns statistics for the user reaction cache.
func (c *ReactionCache) UserCacheStats() Stats {
	if !c.enabled {
		return Stats{}
	}
	return c.userCache.Stats()
}

// EntityCacheStats returns statistics for the entity counts cache.
func (c *ReactionCache) EntityCacheStats() Stats {
	if !c.enabled {
		return Stats{}
	}
	return c.entityCache.Stats()
}

// userReactionKey generates a cache key for a user reaction.
func userReactionKey(userID, entityType, entityID string) string {
	return fmt.Sprintf("entity:%s:%s:user:%s", entityType, entityID, userID)
}

// entityCountsKey generates a cache key for entity counts.
func entityCountsKey(entityType, entityID string) string {
	return fmt.Sprintf("entity:%s:%s:counts", entityType, entityID)
}

// entityPrefix generates a prefix for all entries related to an entity.
func entityPrefix(entityType, entityID string) string {
	return fmt.Sprintf("entity:%s:%s:", entityType, entityID)
}

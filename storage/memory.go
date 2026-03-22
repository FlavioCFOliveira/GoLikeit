package storage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
)

// MemoryStorage implements Repository with in-memory storage.
// Data is lost when the instance is closed. Safe for concurrent use.
type MemoryStorage struct {
	mu        sync.RWMutex
	reactions map[string]*golikeit.UserReaction // Key: "user_id:entity_type:entity_id"
	counts    map[string]map[string]int64       // Key: "entity_type:entity_id" -> map[reaction_type]count
	closed    bool
}

// NewMemoryStorage creates a new in-memory storage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		reactions: make(map[string]*golikeit.UserReaction),
		counts:    make(map[string]map[string]int64),
	}
}

// reactionKey generates a unique key for a user reaction.
func reactionKey(userID string, target golikeit.EntityTarget) string {
	return fmt.Sprintf("%s:%s:%s", userID, target.EntityType, target.EntityID)
}

// entityKey generates a unique key for an entity.
func entityKey(target golikeit.EntityTarget) string {
	return target.String()
}

// AddReaction adds or replaces a reaction for a user.
func (m *MemoryStorage) AddReaction(ctx context.Context, userID string, target golikeit.EntityTarget, reactionType string) (bool, error) {
	if m.closed {
		return false, golikeit.ErrStorageUnavailable
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := reactionKey(userID, target)
	existing, exists := m.reactions[key]

	now := time.Now().UTC()

	if exists {
		// Update existing reaction
		oldType := existing.ReactionType
		existing.ReactionType = reactionType
		existing.UpdatedAt = now

		// Update counts: decrement old type, increment new type
		entityKey := entityKey(target)
		if m.counts[entityKey] != nil {
			m.counts[entityKey][oldType]--
			if m.counts[entityKey][oldType] <= 0 {
				delete(m.counts[entityKey], oldType)
			}
		}
		m.counts[entityKey][reactionType]++

		return true, nil
	}

	// Create new reaction
	m.reactions[key] = &golikeit.UserReaction{
		UserID:       userID,
		EntityType:   target.EntityType,
		EntityID:     target.EntityID,
		ReactionType: reactionType,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Update counts
	entityKey := entityKey(target)
	if m.counts[entityKey] == nil {
		m.counts[entityKey] = make(map[string]int64)
	}
	m.counts[entityKey][reactionType]++

	return false, nil
}

// RemoveReaction removes a user's reaction.
func (m *MemoryStorage) RemoveReaction(ctx context.Context, userID string, target golikeit.EntityTarget) error {
	if m.closed {
		return golikeit.ErrStorageUnavailable
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := reactionKey(userID, target)
	reaction, exists := m.reactions[key]
	if !exists {
		return golikeit.ErrReactionNotFound
	}

	// Update counts
	entityKey := entityKey(target)
	if m.counts[entityKey] != nil {
		m.counts[entityKey][reaction.ReactionType]--
		if m.counts[entityKey][reaction.ReactionType] <= 0 {
			delete(m.counts[entityKey], reaction.ReactionType)
		}
		if len(m.counts[entityKey]) == 0 {
			delete(m.counts, entityKey)
		}
	}

	delete(m.reactions, key)
	return nil
}

// GetUserReaction retrieves a user's current reaction type for a target.
func (m *MemoryStorage) GetUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (string, error) {
	if m.closed {
		return "", golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := reactionKey(userID, target)
	reaction, exists := m.reactions[key]
	if !exists {
		return "", golikeit.ErrReactionNotFound
	}

	return reaction.ReactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (m *MemoryStorage) HasUserReaction(ctx context.Context, userID string, target golikeit.EntityTarget) (bool, error) {
	if m.closed {
		return false, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := reactionKey(userID, target)
	_, exists := m.reactions[key]
	return exists, nil
}

// GetEntityCounts retrieves the reaction counts for an entity.
func (m *MemoryStorage) GetEntityCounts(ctx context.Context, target golikeit.EntityTarget) (golikeit.EntityCounts, error) {
	if m.closed {
		return golikeit.EntityCounts{}, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	entityKey := entityKey(target)
	counts := m.counts[entityKey]
	if counts == nil {
		return golikeit.EntityCounts{
			Counts: make(map[string]int64),
			Total:  0,
		}, nil
	}

	// Make a copy of counts
	countsCopy := make(map[string]int64, len(counts))
	var total int64
	for rt, count := range counts {
		countsCopy[rt] = count
		total += count
	}

	return golikeit.EntityCounts{
		Counts: countsCopy,
		Total:  total,
	}, nil
}

// GetUserReactions retrieves all reactions for a user with optional filters and pagination.
func (m *MemoryStorage) GetUserReactions(ctx context.Context, userID string, filters Filters, pagination pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	if m.closed {
		return nil, 0, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all reactions for this user
	var results []golikeit.UserReaction
	for _, reaction := range m.reactions {
		if reaction.UserID != userID {
			continue
		}

		// Apply filters
		if filters.EntityType != "" && reaction.EntityType != filters.EntityType {
			continue
		}
		if filters.ReactionType != "" && reaction.ReactionType != filters.ReactionType {
			continue
		}
		if filters.Since != nil && reaction.CreatedAt.Before(*filters.Since) {
			continue
		}
		if filters.Until != nil && reaction.CreatedAt.After(*filters.Until) {
			continue
		}

		results = append(results, *reaction)
	}

	// Sort by timestamp (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := int64(len(results))

	// Apply pagination
	start := pagination.Offset
	if start > len(results) {
		start = len(results)
	}
	end := start + pagination.Limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// GetUserReactionCounts returns aggregated counts per reaction type for a user.
func (m *MemoryStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	if m.closed {
		return nil, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	counts := make(map[string]int64)
	for _, reaction := range m.reactions {
		if reaction.UserID != userID {
			continue
		}
		if entityTypeFilter != "" && reaction.EntityType != entityTypeFilter {
			continue
		}
		counts[reaction.ReactionType]++
	}

	return counts, nil
}

// GetUserReactionsByType retrieves reactions of a specific type for a user.
func (m *MemoryStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pagination pagination.Pagination) ([]golikeit.UserReaction, int64, error) {
	if m.closed {
		return nil, 0, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect reactions matching criteria
	var results []golikeit.UserReaction
	for _, reaction := range m.reactions {
		if reaction.UserID != userID || reaction.ReactionType != reactionType {
			continue
		}
		results = append(results, *reaction)
	}

	// Sort by timestamp (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := int64(len(results))

	// Apply pagination
	start := pagination.Offset
	if start > len(results) {
		start = len(results)
	}
	end := start + pagination.Limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// GetEntityReactions retrieves all reactions on an entity with pagination.
func (m *MemoryStorage) GetEntityReactions(ctx context.Context, target golikeit.EntityTarget, pagination pagination.Pagination) ([]golikeit.EntityReaction, int64, error) {
	if m.closed {
		return nil, 0, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect reactions for this entity
	var results []golikeit.EntityReaction
	for _, reaction := range m.reactions {
		if reaction.EntityType != target.EntityType || reaction.EntityID != target.EntityID {
			continue
		}
		results = append(results, golikeit.EntityReaction{
			UserID:       reaction.UserID,
			ReactionType: reaction.ReactionType,
			CreatedAt:    reaction.CreatedAt,
		})
	}

	// Sort by timestamp (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	total := int64(len(results))

	// Apply pagination
	start := pagination.Offset
	if start > len(results) {
		start = len(results)
	}
	end := start + pagination.Limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], total, nil
}

// GetRecentReactions retrieves recent reactions on an entity.
func (m *MemoryStorage) GetRecentReactions(ctx context.Context, target golikeit.EntityTarget, limit int) ([]golikeit.RecentUserReaction, error) {
	if m.closed {
		return nil, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect reactions
	var results []golikeit.RecentUserReaction
	for _, reaction := range m.reactions {
		if reaction.EntityType != target.EntityType || reaction.EntityID != target.EntityID {
			continue
		}
		results = append(results, golikeit.RecentUserReaction{
			UserID:       reaction.UserID,
			ReactionType: reaction.ReactionType,
			Timestamp:    reaction.CreatedAt,
		})
	}

	// Sort by timestamp (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}

// GetLastReactionTime retrieves the timestamp of the most recent reaction on an entity.
func (m *MemoryStorage) GetLastReactionTime(ctx context.Context, target golikeit.EntityTarget) (*time.Time, error) {
	if m.closed {
		return nil, golikeit.ErrStorageUnavailable
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastTime *time.Time
	for _, reaction := range m.reactions {
		if reaction.EntityType != target.EntityType || reaction.EntityID != target.EntityID {
			continue
		}
		if lastTime == nil || reaction.CreatedAt.After(*lastTime) {
			t := reaction.CreatedAt
			lastTime = &t
		}
	}

	return lastTime, nil
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (m *MemoryStorage) GetEntityReactionDetail(ctx context.Context, target golikeit.EntityTarget, maxRecentUsers int) (golikeit.EntityReactionDetail, error) {
	if m.closed {
		return golikeit.EntityReactionDetail{}, golikeit.ErrStorageUnavailable
	}

	counts, err := m.GetEntityCounts(ctx, target)
	if err != nil {
		return golikeit.EntityReactionDetail{}, err
	}

	recentUsers := make(map[string][]golikeit.RecentUserReaction)
	if maxRecentUsers > 0 {
		recent, err := m.GetRecentReactions(ctx, target, maxRecentUsers*10) // Get extra for grouping
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
func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	m.reactions = nil
	m.counts = nil
	return nil
}

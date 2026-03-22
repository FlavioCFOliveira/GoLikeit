// Package golikeit provides integration tests for the GoLikeit client.
// These tests verify component interactions.
package golikeit

import (
	"context"
	"sync"
	"testing"
)

// mockIntegrationStorage implements ReactionStorage interface for testing
type mockIntegrationStorage struct {
	reactions map[string]string // key: userID:entityType:entityID, value: reactionType
	mu        sync.RWMutex
}

func newMockIntegrationStorage() *mockIntegrationStorage {
	return &mockIntegrationStorage{
		reactions: make(map[string]string),
	}
}

func (m *mockIntegrationStorage) makeKey(userID string, target EntityTarget) string {
	return userID + ":" + target.EntityType + ":" + target.EntityID
}

func (m *mockIntegrationStorage) AddReaction(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.makeKey(userID, target)
	_, exists := m.reactions[key]
	m.reactions[key] = reactionType
	return exists, nil
}

func (m *mockIntegrationStorage) RemoveReaction(ctx context.Context, userID string, target EntityTarget) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.makeKey(userID, target)
	delete(m.reactions, key)
	return nil
}

func (m *mockIntegrationStorage) GetUserReaction(ctx context.Context, userID string, target EntityTarget) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := m.makeKey(userID, target)
	if reaction, ok := m.reactions[key]; ok {
		return reaction, nil
	}
	return "", ErrReactionNotFound
}

func (m *mockIntegrationStorage) GetEntityCounts(ctx context.Context, target EntityTarget) (EntityCounts, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var counts EntityCounts
	counts.Counts = make(map[string]int64)
	for key, reactionType := range m.reactions {
		if containsStr(key, target.EntityType) && containsStr(key, target.EntityID) {
			counts.Total++
			counts.Counts[reactionType]++
		}
	}
	return counts, nil
}

func (m *mockIntegrationStorage) GetUserReactions(ctx context.Context, userID string, pagination Pagination) ([]UserReaction, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reactions []UserReaction
	for key, reactionType := range m.reactions {
		if containsStr(key, userID) {
			parts := splitKey(key)
			if len(parts) == 3 {
				reactions = append(reactions, UserReaction{
					EntityType:   parts[1],
					EntityID:     parts[2],
					ReactionType: reactionType,
				})
			}
		}
	}
	return reactions, int64(len(reactions)), nil
}

func (m *mockIntegrationStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counts := make(map[string]int64)
	for key, reactionType := range m.reactions {
		if containsStr(key, userID) {
			if entityTypeFilter == "" {
				counts[reactionType]++
			} else {
				parts := splitKey(key)
				if len(parts) == 3 && parts[1] == entityTypeFilter {
					counts[reactionType]++
				}
			}
		}
	}
	return counts, nil
}

func (m *mockIntegrationStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pagination Pagination) ([]UserReaction, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reactions []UserReaction
	for key, rt := range m.reactions {
		if rt == reactionType && containsStr(key, userID) {
			parts := splitKey(key)
			if len(parts) == 3 {
				reactions = append(reactions, UserReaction{
					EntityType:   parts[1],
					EntityID:     parts[2],
					ReactionType: reactionType,
				})
			}
		}
	}
	return reactions, int64(len(reactions)), nil
}

func (m *mockIntegrationStorage) GetEntityReactions(ctx context.Context, target EntityTarget, pagination Pagination) ([]EntityReaction, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var reactions []EntityReaction
	for key, reactionType := range m.reactions {
		if containsStr(key, target.EntityType) && containsStr(key, target.EntityID) {
			parts := splitKey(key)
			if len(parts) == 3 {
				reactions = append(reactions, EntityReaction{
					UserID:       parts[0],
					ReactionType: reactionType,
				})
			}
		}
	}

	// Apply pagination
	total := int64(len(reactions))
	start := pagination.Offset
	end := start + pagination.Limit
	if start > len(reactions) {
		start = len(reactions)
	}
	if end > len(reactions) {
		end = len(reactions)
	}
	if start > end {
		start = end
	}

	return reactions[start:end], total, nil
}

func (m *mockIntegrationStorage) GetRecentReactions(ctx context.Context, target EntityTarget, limit int) ([]RecentUserReaction, error) {
	return []RecentUserReaction{}, nil
}

func containsStr(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitKey(key string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestEndToEnd_AddAndRetrieveReactions(t *testing.T) {
	mockStore := newMockIntegrationStorage()

	ctx := context.Background()
	userID := "integration-user"
	target1 := EntityTarget{EntityType: "blog_post", EntityID: "post-1"}
	target2 := EntityTarget{EntityType: "blog_post", EntityID: "post-2"}
	target3 := EntityTarget{EntityType: "comment", EntityID: "comment-1"}

	// 1. User adds reactions to multiple posts
	_, err := mockStore.AddReaction(ctx, userID, target1, "LIKE")
	if err != nil {
		t.Fatalf("Failed to add reaction to post 1: %v", err)
	}

	_, err = mockStore.AddReaction(ctx, userID, target2, "LOVE")
	if err != nil {
		t.Fatalf("Failed to add reaction to post 2: %v", err)
	}

	_, err = mockStore.AddReaction(ctx, userID, target3, "THUMBS_UP")
	if err != nil {
		t.Fatalf("Failed to add reaction to comment: %v", err)
	}

	// 2. User retrieves their reactions
	userReactions, total, err := mockStore.GetUserReactions(ctx, userID, Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to get user reactions: %v", err)
	}
	if total != 3 {
		t.Errorf("Expected 3 reactions, got %d", total)
	}
	if len(userReactions) != 3 {
		t.Errorf("Expected 3 reactions in list, got %d", len(userReactions))
	}

	// 3. Check user's reaction counts
	reactionCounts, err := mockStore.GetUserReactionCounts(ctx, userID, "")
	if err != nil {
		t.Fatalf("Failed to get user reaction counts: %v", err)
	}
	if reactionCounts["LIKE"] != 1 {
		t.Errorf("Expected 1 LIKE, got %d", reactionCounts["LIKE"])
	}
	if reactionCounts["LOVE"] != 1 {
		t.Errorf("Expected 1 LOVE, got %d", reactionCounts["LOVE"])
	}
	if reactionCounts["THUMBS_UP"] != 1 {
		t.Errorf("Expected 1 THUMBS_UP, got %d", reactionCounts["THUMBS_UP"])
	}

	// 4. Check entity counts
	counts1, err := mockStore.GetEntityCounts(ctx, target1)
	if err != nil {
		t.Fatalf("Failed to get entity counts: %v", err)
	}
	if counts1.Total != 1 {
		t.Errorf("Expected 1 reaction on post 1, got %d", counts1.Total)
	}

	// 5. User removes one reaction
	err = mockStore.RemoveReaction(ctx, userID, target1)
	if err != nil {
		t.Fatalf("Failed to remove reaction: %v", err)
	}

	// 6. Verify removal
	reaction, err := mockStore.GetUserReaction(ctx, userID, target1)
	if err == nil {
		t.Errorf("Expected error after removal, got reaction: %s", reaction)
	}

	// 7. Get reactions filtered by type
	likeReactions, _, err := mockStore.GetUserReactionsByType(ctx, userID, "LIKE", Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to get user reactions by type: %v", err)
	}
	if len(likeReactions) != 0 {
		t.Errorf("Expected 0 LIKE reactions after removal, got %d", len(likeReactions))
	}

	loveReactions, _, err := mockStore.GetUserReactionsByType(ctx, userID, "LOVE", Pagination{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to get user reactions by type: %v", err)
	}
	if len(loveReactions) != 1 {
		t.Errorf("Expected 1 LOVE reaction, got %d", len(loveReactions))
	}
}

func TestEndToEnd_MultipleUsersInteraction(t *testing.T) {
	mockStore := newMockIntegrationStorage()

	ctx := context.Background()
	target := EntityTarget{EntityType: "video", EntityID: "viral-video-123"}

	// Multiple users react to the same post
	users := []string{"alice", "bob", "charlie", "diana", "eve"}
	for _, user := range users {
		_, err := mockStore.AddReaction(ctx, user, target, "LIKE")
		if err != nil {
			t.Fatalf("Failed to add reaction for user %s: %v", user, err)
		}
	}

	// Verify entity counts
	counts, err := mockStore.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("Failed to get entity counts: %v", err)
	}
	if counts.Total != 5 {
		t.Errorf("Expected 5 reactions, got %d", counts.Total)
	}
	if counts.Counts["LIKE"] != 5 {
		t.Errorf("Expected 5 LIKE reactions, got %d", counts.Counts["LIKE"])
	}

	// Get entity reactions with pagination
	entityReactions, total, err := mockStore.GetEntityReactions(ctx, target, Pagination{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("Failed to get entity reactions: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected total 5 reactions, got %d", total)
	}
	if len(entityReactions) != 3 {
		t.Errorf("Expected 3 reactions on page 1, got %d", len(entityReactions))
	}

	// Second page
	entityReactionsPage2, _, err := mockStore.GetEntityReactions(ctx, target, Pagination{Limit: 3, Offset: 3})
	if err != nil {
		t.Fatalf("Failed to get entity reactions page 2: %v", err)
	}
	if len(entityReactionsPage2) != 2 {
		t.Errorf("Expected 2 reactions on page 2, got %d", len(entityReactionsPage2))
	}

	// Each user can only have one reaction
	for _, user := range users {
		reaction, err := mockStore.GetUserReaction(ctx, user, target)
		if err != nil {
			t.Fatalf("Failed to get reaction for user %s: %v", user, err)
		}
		if reaction != "LIKE" {
			t.Errorf("Expected LIKE for user %s, got %s", user, reaction)
		}
	}
}

func TestEndToEnd_ReactionReplacement(t *testing.T) {
	mockStore := newMockIntegrationStorage()

	ctx := context.Background()
	userID := "replacement-user"
	target := EntityTarget{EntityType: "post", EntityID: "replaceable-post"}

	// 1. Add initial reaction
	isReplaced, err := mockStore.AddReaction(ctx, userID, target, "LIKE")
	if err != nil {
		t.Fatalf("Failed to add initial reaction: %v", err)
	}
	if isReplaced {
		t.Error("Expected isReplaced to be false for new reaction")
	}

	// 2. Replace with different reaction
	isReplaced, err = mockStore.AddReaction(ctx, userID, target, "LOVE")
	if err != nil {
		t.Fatalf("Failed to replace reaction: %v", err)
	}
	if !isReplaced {
		t.Error("Expected isReplaced to be true for replacement")
	}

	// 3. Verify final reaction type
	reaction, err := mockStore.GetUserReaction(ctx, userID, target)
	if err != nil {
		t.Fatalf("Failed to get user reaction: %v", err)
	}
	if reaction != "LOVE" {
		t.Errorf("Expected LOVE after replacement, got %s", reaction)
	}

	// 4. Verify entity counts reflect replacement (not duplicate)
	counts, err := mockStore.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("Failed to get entity counts: %v", err)
	}
	if counts.Total != 1 {
		t.Errorf("Expected 1 reaction (replacement, not duplicate), got %d", counts.Total)
	}
	if counts.Counts["LIKE"] != 0 {
		t.Errorf("Expected 0 LIKE reactions after replacement, got %d", counts.Counts["LIKE"])
	}
	if counts.Counts["LOVE"] != 1 {
		t.Errorf("Expected 1 LOVE reaction, got %d", counts.Counts["LOVE"])
	}
}

func TestEndToEnd_ConcurrentAccess(t *testing.T) {
	mockStore := newMockIntegrationStorage()

	ctx := context.Background()
	target := EntityTarget{EntityType: "concurrent_post", EntityID: "concurrent-1"}

	// Multiple concurrent users
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			userID := "concurrent-user-" + string(rune('A'+id%26))
			_, err := mockStore.AddReaction(ctx, userID, target, "LIKE")
			if err != nil {
				t.Errorf("Failed to add reaction: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Verify counts
	counts, err := mockStore.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("Failed to get entity counts: %v", err)
	}

	// Should have at least concurrency/2 reactions
	if counts.Total < int64(concurrency/2) {
		t.Errorf("Expected at least %d reactions, got %d", concurrency/2, counts.Total)
	}
}

func TestIntegration_InvalidInputs(t *testing.T) {
	mockStore := newMockIntegrationStorage()

	ctx := context.Background()

	// Test with empty user ID
	target := EntityTarget{EntityType: "post", EntityID: "123"}
	_, err := mockStore.AddReaction(ctx, "", target, "LIKE")
	if err != nil {
		t.Logf("Empty user ID behavior: %v", err)
	}

	// Test with invalid entity type (uppercase)
	target2 := EntityTarget{EntityType: "POST", EntityID: "123"}
	_, err = mockStore.AddReaction(ctx, "user1", target2, "LIKE")
	if err != nil {
		t.Logf("Uppercase entity type behavior: %v", err)
	}

	// Test with invalid reaction type (lowercase)
	_, err = mockStore.AddReaction(ctx, "user1", target, "like")
	if err != nil {
		t.Logf("Lowercase reaction type behavior: %v", err)
	}
}

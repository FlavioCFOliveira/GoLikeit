// Package e2e provides end-to-end tests for the GoLikeit reaction system.
// These tests validate complete user workflows across all components:
// API, Business Logic, Cache, Storage, Events, and Audit.
//
//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/events"
	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
)

// TestE2EReactionLifecycle validates the complete reaction lifecycle:
// add, get, and remove operations work correctly end-to-end.
func TestE2EReactionLifecycle(t *testing.T) {
	ctx := context.Background()

	// Create client with in-memory storage for E2E test
	client, cleanup := setupE2EClient(t)
	defer cleanup()

	userID := "user-123"
	entityType := "photo"
	entityID := "photo-456"
	reactionType := "LIKE"

	t.Run("add reaction", func(t *testing.T) {
		isReplaced, err := client.AddReaction(ctx, userID, entityType, entityID, reactionType)
		if err != nil {
			t.Fatalf("failed to add reaction: %v", err)
		}
		if isReplaced {
			t.Error("expected isReplaced to be false for new reaction")
		}
	})

	t.Run("get user reaction", func(t *testing.T) {
		reaction, err := client.GetUserReaction(ctx, userID, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get user reaction: %v", err)
		}
		if reaction != reactionType {
			t.Errorf("expected reaction %q, got %q", reactionType, reaction)
		}
	})

	t.Run("get entity counts", func(t *testing.T) {
		counts, total, err := client.GetEntityReactionCounts(ctx, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get entity counts: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if counts[reactionType] != 1 {
			t.Errorf("expected count 1 for %s, got %d", reactionType, counts[reactionType])
		}
	})

	t.Run("has user reaction", func(t *testing.T) {
		hasReaction, err := client.HasUserReaction(ctx, userID, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to check user reaction: %v", err)
		}
		if !hasReaction {
			t.Error("expected user to have reaction")
		}
	})

	t.Run("replace reaction", func(t *testing.T) {
		newReactionType := "LOVE"
		isReplaced, err := client.AddReaction(ctx, userID, entityType, entityID, newReactionType)
		if err != nil {
			t.Fatalf("failed to replace reaction: %v", err)
		}
		if !isReplaced {
			t.Error("expected isReplaced to be true for replacement")
		}

		// Verify new reaction
		reaction, err := client.GetUserReaction(ctx, userID, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get user reaction after replace: %v", err)
		}
		if reaction != newReactionType {
			t.Errorf("expected reaction %q, got %q", newReactionType, reaction)
		}

		// Verify counts updated
		counts, total, err := client.GetEntityReactionCounts(ctx, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get entity counts after replace: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1 after replace, got %d", total)
		}
		if counts["LIKE"] != 0 {
			t.Errorf("expected LIKE count 0 after replace, got %d", counts["LIKE"])
		}
		if counts["LOVE"] != 1 {
			t.Errorf("expected LOVE count 1 after replace, got %d", counts["LOVE"])
		}
	})

	t.Run("remove reaction", func(t *testing.T) {
		err := client.RemoveReaction(ctx, userID, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to remove reaction: %v", err)
		}

		// Verify reaction removed
		reaction, err := client.GetUserReaction(ctx, userID, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get user reaction after remove: %v", err)
		}
		if reaction != "" {
			t.Errorf("expected empty reaction after remove, got %q", reaction)
		}

		// Verify counts updated
		counts, total, err := client.GetEntityReactionCounts(ctx, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get entity counts after remove: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0 after remove, got %d", total)
		}
		if counts["LOVE"] != 0 {
			t.Errorf("expected LOVE count 0 after remove, got %d", counts["LOVE"])
		}
	})

	t.Run("remove non-existent reaction", func(t *testing.T) {
		err := client.RemoveReaction(ctx, userID, entityType, entityID)
		if err == nil {
			t.Error("expected error when removing non-existent reaction")
		}
	})
}

// TestE2ECrossComponentIntegration validates that all components work together:
// API, Cache, Storage, and Events.
func TestE2ECrossComponentIntegration(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupE2EClient(t)
	defer cleanup()

	t.Run("cache integration", func(t *testing.T) {
		userID := "user-cache"
		entityID := "entity-cache"

		// First call hits storage and caches result
		_, err := client.AddReaction(ctx, userID, "post", entityID, "LIKE")
		if err != nil {
			t.Fatalf("failed to add reaction: %v", err)
		}

		// Subsequent calls should use cache
		reaction, err := client.GetUserReaction(ctx, userID, "post", entityID)
		if err != nil {
			t.Fatalf("failed to get user reaction: %v", err)
		}
		if reaction != "LIKE" {
			t.Errorf("expected LIKE, got %s", reaction)
		}

		// Remove should invalidate cache
		err = client.RemoveReaction(ctx, userID, "post", entityID)
		if err != nil {
			t.Fatalf("failed to remove reaction: %v", err)
		}

		// Cache should be invalidated
		reaction, err = client.GetUserReaction(ctx, userID, "post", entityID)
		if err != nil {
			t.Fatalf("failed to get user reaction after remove: %v", err)
		}
		if reaction != "" {
			t.Errorf("expected empty reaction after remove, got %s", reaction)
		}
	})

	t.Run("event emission", func(t *testing.T) {
		userID := "user-event"
		entityID := "entity-event"

		// Subscribe to events
		eventCh := make(chan events.Event, 10)
		client.EventBus().SubscribeSyncFunc(events.Filter{}, func(ctx context.Context, evt events.Event) error {
			eventCh <- evt
			return nil
		})

		// Add reaction
		_, err := client.AddReaction(ctx, userID, "post", entityID, "LOVE")
		if err != nil {
			t.Fatalf("failed to add reaction: %v", err)
		}

		// Wait for event
		select {
		case evt := <-eventCh:
			if evt.Type != events.TypeReactionAdded {
				t.Errorf("expected event type %s, got %s", events.TypeReactionAdded, evt.Type)
			}
		case <-time.After(2 * time.Second):
			t.Error("timeout waiting for add event")
		}

		// Remove reaction
		err = client.RemoveReaction(ctx, userID, "post", entityID)
		if err != nil {
			t.Fatalf("failed to remove reaction: %v", err)
		}

		// Wait for remove event
		select {
		case evt := <-eventCh:
			if evt.Type != events.TypeReactionRemoved {
				t.Errorf("expected event type %s, got %s", events.TypeReactionRemoved, evt.Type)
			}
		case <-time.After(2 * time.Second):
			t.Error("timeout waiting for remove event")
		}
	})
}

// TestE2EConcurrentOperations validates thread safety under concurrent load.
func TestE2EConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupE2EClient(t)
	defer cleanup()

	entityType := "post"
	entityID := "concurrent-post"

	t.Run("concurrent adds from different users", func(t *testing.T) {
		numUsers := 100
		var wg sync.WaitGroup
		errCh := make(chan error, numUsers)

		for i := 0; i < numUsers; i++ {
			wg.Add(1)
			go func(userNum int) {
				defer wg.Done()
				userID := fmt.Sprintf("user-%d", userNum)
				_, err := client.AddReaction(ctx, userID, entityType, entityID, "LIKE")
				if err != nil {
					errCh <- fmt.Errorf("user %d: %w", userNum, err)
				}
			}(i)
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			t.Errorf("concurrent add failed: %v", err)
		}

		// Verify counts
		counts, total, err := client.GetEntityReactionCounts(ctx, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get counts: %v", err)
		}
		if total != int64(numUsers) {
			t.Errorf("expected total %d, got %d", numUsers, total)
		}
		if counts["LIKE"] != int64(numUsers) {
			t.Errorf("expected LIKE count %d, got %d", numUsers, counts["LIKE"])
		}
	})

	t.Run("concurrent operations same user", func(t *testing.T) {
		userID := "concurrent-user"
		entityID := "concurrent-entity"

		var wg sync.WaitGroup
		numOps := 50

		// Concurrent adds and removes from same user
		for i := 0; i < numOps; i++ {
			wg.Add(1)
			go func(opNum int) {
				defer wg.Done()
				if opNum%2 == 0 {
					client.AddReaction(ctx, userID, entityType, entityID, "LIKE")
				} else {
					client.RemoveReaction(ctx, userID, entityType, entityID)
				}
			}(i)
		}

		wg.Wait()

		// Final state should be consistent
		_, err := client.GetUserReaction(ctx, userID, entityType, entityID)
		if err != nil {
			t.Logf("final state error (acceptable due to race): %v", err)
		}
	})

	t.Run("concurrent reads during writes", func(t *testing.T) {
		userID := "reader-user"
		entityID := "reader-entity"

		// Setup initial reaction
		_, err := client.AddReaction(ctx, userID, entityType, entityID, "LIKE")
		if err != nil {
			t.Fatalf("failed to add initial reaction: %v", err)
		}

		var wg sync.WaitGroup
		numReaders := 20
		numWriters := 10

		// Start readers
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_, _, _ = client.GetEntityReactionCounts(ctx, entityType, entityID)
				}
			}()
		}

		// Start writers
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(writerNum int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					client.AddReaction(ctx, userID, entityType, entityID, "LOVE")
					client.AddReaction(ctx, userID, entityType, entityID, "LIKE")
				}
			}(i)
		}

		wg.Wait()
	})
}

// TestE2EBulkOperations validates bulk query operations.
func TestE2EBulkOperations(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupE2EClient(t)
	defer cleanup()

	t.Run("bulk user reactions", func(t *testing.T) {
		userID := "bulk-user"
		entityType := "post"

		// Create reactions on multiple entities
		numEntities := 10
		for i := 0; i < numEntities; i++ {
			entityID := fmt.Sprintf("entity-%d", i)
			reactionType := "LIKE"
			if i%2 == 0 {
				reactionType = "LOVE"
			}
			_, err := client.AddReaction(ctx, userID, entityType, entityID, reactionType)
			if err != nil {
				t.Fatalf("failed to add reaction %d: %v", i, err)
			}
		}

		// Query bulk reactions
		targets := make([]golikeit.EntityTarget, numEntities)
		for i := 0; i < numEntities; i++ {
			targets[i] = golikeit.EntityTarget{
				EntityType: entityType,
				EntityID:   fmt.Sprintf("entity-%d", i),
			}
		}

		reactions, err := client.GetUserReactionsBulk(ctx, userID, targets)
		if err != nil {
			t.Fatalf("failed to get bulk reactions: %v", err)
		}

		if len(reactions) != numEntities {
			t.Errorf("expected %d reactions, got %d", numEntities, len(reactions))
		}

		// Verify reaction types
		for i, target := range targets {
			expectedType := "LIKE"
			if i%2 == 0 {
				expectedType = "LOVE"
			}
			if reactions[target] != expectedType {
				t.Errorf("expected %s for entity-%d, got %s", expectedType, i, reactions[target])
			}
		}
	})

	t.Run("bulk entity counts", func(t *testing.T) {
		entityType := "article"
		numEntities := 5

		// Create reactions from multiple users
		for entityNum := 0; entityNum < numEntities; entityNum++ {
			entityID := fmt.Sprintf("article-%d", entityNum)
			for userNum := 0; userNum < 10; userNum++ {
				userID := fmt.Sprintf("user-%d", userNum)
				reactionType := "LIKE"
				if userNum%2 == 0 {
					reactionType = "LOVE"
				}
				_, err := client.AddReaction(ctx, userID, entityType, entityID, reactionType)
				if err != nil {
					t.Fatalf("failed to add reaction: %v", err)
				}
			}
		}

		// Query bulk counts
		targets := make([]golikeit.EntityTarget, numEntities)
		for i := 0; i < numEntities; i++ {
			targets[i] = golikeit.EntityTarget{
				EntityType: entityType,
				EntityID:   fmt.Sprintf("article-%d", i),
			}
		}

		counts, err := client.GetEntityCountsBulk(ctx, targets)
		if err != nil {
			t.Fatalf("failed to get bulk counts: %v", err)
		}

		if len(counts) != numEntities {
			t.Errorf("expected %d count results, got %d", numEntities, len(counts))
		}

		// Each entity should have 10 reactions
		for _, target := range targets {
			if counts[target].Total != 10 {
				t.Errorf("expected total 10 for %s, got %d", target.EntityID, counts[target].Total)
			}
		}
	})

	t.Run("multiple user reactions on single entity", func(t *testing.T) {
		entityType := "video"
		entityID := "bulk-video"

		userIDs := []string{"user-a", "user-b", "user-c", "user-d", "user-e"}
		expectedReactions := map[string]string{
			"user-a": "LIKE",
			"user-b": "LOVE",
			"user-c": "LIKE",
			"user-d": "LOVE",
			"user-e": "LIKE",
		}

		// Create reactions
		for userID, reactionType := range expectedReactions {
			_, err := client.AddReaction(ctx, userID, entityType, entityID, reactionType)
			if err != nil {
				t.Fatalf("failed to add reaction for %s: %v", userID, err)
			}
		}

		// Query multiple users
		reactions, err := client.GetMultipleUserReactions(ctx, userIDs, entityType, entityID)
		if err != nil {
			t.Fatalf("failed to get multiple user reactions: %v", err)
		}

		if len(reactions) != len(userIDs) {
			t.Errorf("expected %d reactions, got %d", len(userIDs), len(reactions))
		}

		for userID, expectedType := range expectedReactions {
			if reactions[userID] != expectedType {
				t.Errorf("expected %s for %s, got %s", expectedType, userID, reactions[userID])
			}
		}
	})
}

// TestE2EPagination validates pagination for list operations.
func TestE2EPagination(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupE2EClient(t)
	defer cleanup()

	entityType := "content"
	userID := "pagination-user"

	// Create multiple reactions
	numReactions := 25
	for i := 0; i < numReactions; i++ {
		entityID := fmt.Sprintf("content-%d", i)
		reactionType := "LIKE"
		if i%2 == 0 {
			reactionType = "LOVE"
		}
		_, err := client.AddReaction(ctx, userID, entityType, entityID, reactionType)
		if err != nil {
			t.Fatalf("failed to add reaction %d: %v", i, err)
		}
	}

	t.Run("paginated user reactions", func(t *testing.T) {
		pagination := golikeit.Pagination{Limit: 10, Offset: 0}
		result, err := client.GetUserReactions(ctx, userID, pagination)
		if err != nil {
			t.Fatalf("failed to get paginated reactions: %v", err)
		}

		if result.Total != int64(numReactions) {
			t.Errorf("expected total %d, got %d", numReactions, result.Total)
		}
		if len(result.Items) != 10 {
			t.Errorf("expected 10 items, got %d", len(result.Items))
		}
		if !result.HasNext {
			t.Error("expected HasNext to be true")
		}

		// Second page
		pagination.Offset = 10
		result2, err := client.GetUserReactions(ctx, userID, pagination)
		if err != nil {
			t.Fatalf("failed to get second page: %v", err)
		}
		if len(result2.Items) != 10 {
			t.Errorf("expected 10 items on page 2, got %d", len(result2.Items))
		}

		// Third page
		pagination.Offset = 20
		result3, err := client.GetUserReactions(ctx, userID, pagination)
		if err != nil {
			t.Fatalf("failed to get third page: %v", err)
		}
		if len(result3.Items) != 5 {
			t.Errorf("expected 5 items on page 3, got %d", len(result3.Items))
		}
		if result3.HasNext {
			t.Error("expected HasNext to be false on last page")
		}
	})
}

// TestE2EErrorHandling validates error scenarios and recovery.
func TestE2EErrorHandling(t *testing.T) {
	ctx := context.Background()
	client, cleanup := setupE2EClient(t)
	defer cleanup()

	t.Run("invalid reaction type", func(t *testing.T) {
		_, err := client.AddReaction(ctx, "user", "post", "123", "INVALID_TYPE")
		if err == nil {
			t.Error("expected error for invalid reaction type")
		}
	})

	t.Run("empty user ID", func(t *testing.T) {
		_, err := client.AddReaction(ctx, "", "post", "123", "LIKE")
		if err == nil {
			t.Error("expected error for empty user ID")
		}
	})

	t.Run("empty entity ID", func(t *testing.T) {
		_, err := client.AddReaction(ctx, "user", "post", "", "LIKE")
		if err == nil {
			t.Error("expected error for empty entity ID")
		}
	})

	t.Run("closed client operations", func(t *testing.T) {
		// Create a new client just for this test
		testClient, testCleanup := setupE2EClient(t)
		testCleanup() // Close immediately

		// Operations on closed client should fail
		_, err := testClient.AddReaction(ctx, "user", "post", "123", "LIKE")
		if err == nil {
			t.Error("expected error when using closed client")
		}
	})
}

// setupE2EClient creates a configured client for E2E testing.
func setupE2EClient(t *testing.T) (*golikeit.Client, func()) {
	t.Helper()

	config := golikeit.Config{
		ReactionTypes: []string{"LIKE", "LOVE", "HAHA", "WOW", "SAD", "ANGRY"},
		Cache: golikeit.CacheConfig{
			Enabled:         true,
			UserReactionTTL: time.Second,
			EntityCountsTTL: time.Second * 5,
			MaxEntries:      1000,
			EvictionPolicy:  "LRU",
		},
		Pagination: golikeit.PaginationConfig{
			DefaultLimit: 20,
			MaxLimit:     100,
			MaxOffset:    10000,
		},
		Events: events.Config{
			Enabled:        true,
			AsyncQueueSize: 100,
			AsyncWorkers:   2,
		},
	}

	client, err := golikeit.New(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Use in-memory storage for E2E tests
	client.SetStorage(newE2EStorage())

	cleanup := func() {
		if err := client.Close(); err != nil {
			t.Logf("cleanup error: %v", err)
		}
	}

	return client, cleanup
}

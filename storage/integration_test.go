// Package storage provides integration tests for storage backends using testcontainers.
// These tests require Docker to be running.
package storage

import (
	"context"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// ============================================================================
// PostgreSQL Integration Tests
// ============================================================================

func TestPostgreSQLStorage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("golikeit_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Create storage
	config := PostgreSQLConfig{
		ConnectionString: connStr,
		MaxConns:         5,
		MinConns:         2,
		MaxConnLifetime:  time.Minute,
		MaxConnIdleTime:  30 * time.Second,
	}

	storage, err := NewPostgreSQLStorage(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create PostgreSQL storage: %v", err)
	}
	defer storage.Close()

	// Initialize schema
	if err := storage.InitSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Run integration tests
	runStorageIntegrationTests(t, storage)
}

// ============================================================================
// MariaDB Integration Tests
// ============================================================================

func TestMariaDBStorage_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start MariaDB container
	container, err := mysql.Run(ctx, "mariadb:11",
		mysql.WithDatabase("golikeit_test"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("Failed to start MariaDB container: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "parseTime=true")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Create storage
	config := MariaDBConfig{
		ConnectionString: connStr,
		MaxOpenConns:     5,
		MaxIdleConns:     2,
		ConnMaxLifetime:  time.Minute,
	}

	storage, err := NewMariaDBStorage(config)
	if err != nil {
		t.Fatalf("Failed to create MariaDB storage: %v", err)
	}
	defer storage.Close()

	// Initialize schema
	if err := storage.InitSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Run integration tests
	runStorageIntegrationTests(t, storage)
}

// ============================================================================
// SQLite Integration Tests (In-Memory)
// ============================================================================

func TestSQLiteStorage_Integration(t *testing.T) {
	ctx := context.Background()

	// Create in-memory SQLite storage
	config := DefaultSQLiteMemoryConfig()
	storage, err := NewSQLiteStorage(config)
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Initialize schema
	if err := storage.InitSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Run integration tests
	runStorageIntegrationTests(t, storage)
}

// ============================================================================
// Common Integration Test Suite
// ============================================================================

// runStorageIntegrationTests runs a comprehensive test suite against any storage implementation.
func runStorageIntegrationTests(t *testing.T, storage Repository) {
	t.Helper()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "integration-test-123"}

	// Test 1: Add and retrieve reaction
	t.Run("AddAndGetReaction", func(t *testing.T) {
		isReplaced, err := storage.AddReaction(ctx, "user1", target, "LIKE")
		if err != nil {
			t.Fatalf("Failed to add reaction: %v", err)
		}
		if isReplaced {
			t.Error("Expected isReplaced to be false for new reaction")
		}

		reactionType, err := storage.GetUserReaction(ctx, "user1", target)
		if err != nil {
			t.Fatalf("Failed to get user reaction: %v", err)
		}
		if reactionType != "LIKE" {
			t.Errorf("Expected LIKE, got %s", reactionType)
		}
	})

	// Test 2: Replace reaction
	t.Run("ReplaceReaction", func(t *testing.T) {
		isReplaced, err := storage.AddReaction(ctx, "user1", target, "LOVE")
		if err != nil {
			t.Fatalf("Failed to replace reaction: %v", err)
		}
		if !isReplaced {
			t.Error("Expected isReplaced to be true for replacement")
		}

		reactionType, err := storage.GetUserReaction(ctx, "user1", target)
		if err != nil {
			t.Fatalf("Failed to get user reaction: %v", err)
		}
		if reactionType != "LOVE" {
			t.Errorf("Expected LOVE, got %s", reactionType)
		}
	})

	// Test 3: Remove reaction
	t.Run("RemoveReaction", func(t *testing.T) {
		err := storage.RemoveReaction(ctx, "user1", target)
		if err != nil {
			t.Fatalf("Failed to remove reaction: %v", err)
		}

		_, err = storage.GetUserReaction(ctx, "user1", target)
		if err == nil {
			t.Error("Expected error after removal")
		}
	})

	// Test 4: Entity counts
	t.Run("EntityCounts", func(t *testing.T) {
		// Add multiple reactions
		target1 := golikeit.EntityTarget{EntityType: "post", EntityID: "post-1"}
		storage.AddReaction(ctx, "user1", target1, "LIKE")
		storage.AddReaction(ctx, "user2", target1, "LIKE")
		storage.AddReaction(ctx, "user3", target1, "LOVE")

		counts, err := storage.GetEntityCounts(ctx, target1)
		if err != nil {
			t.Fatalf("Failed to get entity counts: %v", err)
		}

		if counts.Total != 3 {
			t.Errorf("Expected total 3, got %d", counts.Total)
		}

		if counts.Counts["LIKE"] != 2 {
			t.Errorf("Expected 2 LIKE reactions, got %d", counts.Counts["LIKE"])
		}

		if counts.Counts["LOVE"] != 1 {
			t.Errorf("Expected 1 LOVE reaction, got %d", counts.Counts["LOVE"])
		}
	})

	// Test 5: Multiple users and entities
	t.Run("MultipleUsersAndEntities", func(t *testing.T) {
		storage.RemoveReaction(ctx, "user1", target)
		storage.RemoveReaction(ctx, "user2", target)

		user1 := "user1"
		user2 := "user2"
		target1 := golikeit.EntityTarget{EntityType: "comment", EntityID: "comment-1"}
		target2 := golikeit.EntityTarget{EntityType: "comment", EntityID: "comment-2"}

		_, err := storage.AddReaction(ctx, user1, target1, "LIKE")
		if err != nil {
			t.Fatalf("Failed to add reaction: %v", err)
		}

		_, err = storage.AddReaction(ctx, user2, target1, "LIKE")
		if err != nil {
			t.Fatalf("Failed to add reaction: %v", err)
		}

		_, err = storage.AddReaction(ctx, user1, target2, "LOVE")
		if err != nil {
			t.Fatalf("Failed to add reaction: %v", err)
		}

		reaction1, err := storage.GetUserReaction(ctx, user1, target1)
		if err != nil {
			t.Fatalf("Failed to get reaction: %v", err)
		}
		if reaction1 != "LIKE" {
			t.Errorf("Expected LIKE, got %s", reaction1)
		}

		reaction2, err := storage.GetUserReaction(ctx, user1, target2)
		if err != nil {
			t.Fatalf("Failed to get reaction: %v", err)
		}
		if reaction2 != "LOVE" {
			t.Errorf("Expected LOVE, got %s", reaction2)
		}

		storage.RemoveReaction(ctx, user1, target1)
		storage.RemoveReaction(ctx, user2, target1)
		storage.RemoveReaction(ctx, user1, target2)
	})

	// Test 6: Pagination for user reactions
	t.Run("UserReactionsPagination", func(t *testing.T) {
		testUser := "pagination-user"
		testTarget := golikeit.EntityTarget{EntityType: "article", EntityID: "article-1"}

		_, err := storage.AddReaction(ctx, testUser, testTarget, "LIKE")
		if err != nil {
			t.Fatalf("Failed to add reaction: %v", err)
		}

		filters := Filters{}
		pag := pagination.Pagination{Limit: 10, Offset: 0}
		reactions, total, err := storage.GetUserReactions(ctx, testUser, filters, pag)
		if err != nil {
			t.Fatalf("Failed to get user reactions: %v", err)
		}

		if total < 1 {
			t.Errorf("Expected at least 1 reaction, got %d", total)
		}

		found := false
		for _, r := range reactions {
			if r.EntityType == "article" && r.EntityID == "article-1" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find the added reaction in paginated results")
		}

		storage.RemoveReaction(ctx, testUser, testTarget)
	})

	// Test 7: Pagination for entity reactions
	t.Run("EntityReactionsPagination", func(t *testing.T) {
		testTarget := golikeit.EntityTarget{EntityType: "video", EntityID: "video-1"}

		for i := 0; i < 3; i++ {
			userID := "entity-pagination-user-" + string(rune('A'+i))
			_, err := storage.AddReaction(ctx, userID, testTarget, "LIKE")
			if err != nil {
				t.Fatalf("Failed to add reaction: %v", err)
			}
		}

		pag := pagination.Pagination{Limit: 2, Offset: 0}
		reactions, total, err := storage.GetEntityReactions(ctx, testTarget, pag)
		if err != nil {
			t.Fatalf("Failed to get entity reactions: %v", err)
		}

		if total < 3 {
			t.Errorf("Expected at least 3 reactions, got %d", total)
		}

		if len(reactions) > 2 {
			t.Errorf("Expected max 2 reactions due to limit, got %d", len(reactions))
		}

		for i := 0; i < 3; i++ {
			userID := "entity-pagination-user-" + string(rune('A'+i))
			storage.RemoveReaction(ctx, userID, testTarget)
		}
	})

	// Test 8: Concurrent access
	t.Run("ConcurrentAccess", func(t *testing.T) {
		testTarget := golikeit.EntityTarget{EntityType: "concurrent", EntityID: "concurrent-1"}

		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func(id int) {
				userID := "concurrent-user-" + string(rune('0'+id))
				_, err := storage.AddReaction(ctx, userID, testTarget, "LIKE")
				if err != nil {
					t.Errorf("Failed to add reaction: %v", err)
				}
				done <- true
			}(i)
		}

		for i := 0; i < 5; i++ {
			<-done
		}

		counts, err := storage.GetEntityCounts(ctx, testTarget)
		if err != nil {
			t.Fatalf("Failed to get entity counts: %v", err)
		}

		if counts.Total != 5 {
			t.Errorf("Expected 5 concurrent reactions, got %d", counts.Total)
		}

		for i := 0; i < 5; i++ {
			userID := "concurrent-user-" + string(rune('0'+i))
			storage.RemoveReaction(ctx, userID, testTarget)
		}
	})
}

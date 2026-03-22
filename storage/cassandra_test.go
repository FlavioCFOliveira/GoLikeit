package storage

import (
	"context"
	"testing"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
)

// skipIfNoCassandra skips the test if Cassandra is not available.
func skipIfNoCassandra(t *testing.T) {
	// For CI environments with Cassandra service
	// Set CASSANDRA_TEST_HOSTS environment variable
	hosts := "localhost:9042"
	if hosts == "" {
		t.Skip("Skipping Cassandra test: CASSANDRA_TEST_HOSTS not set")
	}
}

// newTestCassandraStorage creates a test Cassandra storage.
func newTestCassandraStorage(t *testing.T) *CassandraStorage {
	skipIfNoCassandra(t)

	config := CassandraConfig{
		Hosts:          []string{"localhost:9042"},
		Keyspace:       "golikeit_test",
		Consistency:    "QUORUM",
	}

	storage, err := NewCassandraStorage(config)
	if err != nil {
		t.Skipf("Skipping Cassandra test: %v", err)
	}

	ctx := context.Background()
	if err := storage.InitSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Clean up test data
	storage.session.Query("TRUNCATE reactions_by_user").Exec()
	storage.session.Query("TRUNCATE reactions_by_entity").Exec()
	storage.session.Query("TRUNCATE entity_counts").Exec()

	return storage
}

func TestCassandraStorage_AddReaction(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Add new reaction
	isReplaced, err := storage.AddReaction(ctx, "user1", target, "LIKE")
	if err != nil {
		t.Fatalf("AddReaction failed: %v", err)
	}
	if isReplaced {
		t.Error("Expected isReplaced to be false for new reaction")
	}

	// Add same reaction (should replace)
	isReplaced, err = storage.AddReaction(ctx, "user1", target, "LOVE")
	if err != nil {
		t.Fatalf("AddReaction failed: %v", err)
	}
	if !isReplaced {
		t.Error("Expected isReplaced to be true for replacement")
	}

	// Verify replacement
	reactionType, err := storage.GetUserReaction(ctx, "user1", target)
	if err != nil {
		t.Fatalf("GetUserReaction failed: %v", err)
	}
	if reactionType != "LOVE" {
		t.Errorf("Expected LOVE, got %s", reactionType)
	}
}

func TestCassandraStorage_RemoveReaction(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Try to remove non-existent reaction
	err := storage.RemoveReaction(ctx, "user1", target)
	if err != golikeit.ErrReactionNotFound {
		t.Errorf("Expected ErrReactionNotFound, got %v", err)
	}

	// Add and then remove
	_, err = storage.AddReaction(ctx, "user1", target, "LIKE")
	if err != nil {
		t.Fatalf("AddReaction failed: %v", err)
	}

	err = storage.RemoveReaction(ctx, "user1", target)
	if err != nil {
		t.Errorf("RemoveReaction failed: %v", err)
	}

	// Verify removal
	_, err = storage.GetUserReaction(ctx, "user1", target)
	if err != golikeit.ErrReactionNotFound {
		t.Errorf("Expected reaction to be removed, got %v", err)
	}
}

func TestCassandraStorage_GetUserReaction(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Get non-existent reaction
	_, err := storage.GetUserReaction(ctx, "user1", target)
	if err != golikeit.ErrReactionNotFound {
		t.Errorf("Expected ErrReactionNotFound, got %v", err)
	}

	// Add reaction and get
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	reactionType, err := storage.GetUserReaction(ctx, "user1", target)
	if err != nil {
		t.Fatalf("GetUserReaction failed: %v", err)
	}
	if reactionType != "LIKE" {
		t.Errorf("Expected LIKE, got %s", reactionType)
	}
}

func TestCassandraStorage_HasUserReaction(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Check non-existent
	has, err := storage.HasUserReaction(ctx, "user1", target)
	if err != nil {
		t.Fatalf("HasUserReaction failed: %v", err)
	}
	if has {
		t.Error("Expected false for non-existent reaction")
	}

	// Add and check
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	has, err = storage.HasUserReaction(ctx, "user1", target)
	if err != nil {
		t.Fatalf("HasUserReaction failed: %v", err)
	}
	if !has {
		t.Error("Expected true for existing reaction")
	}
}

func TestCassandraStorage_GetEntityCounts(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Get counts for empty entity
	counts, err := storage.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("GetEntityCounts failed: %v", err)
	}
	if counts.Total != 0 {
		t.Errorf("Expected total 0, got %d", counts.Total)
	}

	// Add reactions from different users
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user3", target, "LOVE")

	counts, err = storage.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("GetEntityCounts failed: %v", err)
	}
	if counts.Total != 3 {
		t.Errorf("Expected total 3, got %d", counts.Total)
	}
	if counts.Counts["LIKE"] != 2 {
		t.Errorf("Expected 2 LIKEs, got %d", counts.Counts["LIKE"])
	}
	if counts.Counts["LOVE"] != 1 {
		t.Errorf("Expected 1 LOVE, got %d", counts.Counts["LOVE"])
	}
}

func TestCassandraStorage_GetUserReactions(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Add reactions for user1
	for i := 1; i <= 5; i++ {
		target := golikeit.EntityTarget{EntityType: "post", EntityID: string(rune('0' + i))}
		_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	}

	pag := pagination.Pagination{Limit: 3, Offset: 0}
	reactions, total, err := storage.GetUserReactions(ctx, "user1", Filters{}, pag)
	if err != nil {
		t.Fatalf("GetUserReactions failed: %v", err)
	}
	if total != 5 {
		t.Errorf("Expected total 5, got %d", total)
	}
	if len(reactions) != 3 {
		t.Errorf("Expected 3 reactions, got %d", len(reactions))
	}
}

func TestCassandraStorage_GetUserReactionCounts(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Add reactions of different types
	for i := 1; i <= 3; i++ {
		target := golikeit.EntityTarget{EntityType: "post", EntityID: string(rune('0' + i))}
		_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	}
	for i := 4; i <= 5; i++ {
		target := golikeit.EntityTarget{EntityType: "post", EntityID: string(rune('0' + i))}
		_, _ = storage.AddReaction(ctx, "user1", target, "LOVE")
	}

	counts, err := storage.GetUserReactionCounts(ctx, "user1", "")
	if err != nil {
		t.Fatalf("GetUserReactionCounts failed: %v", err)
	}
	if counts["LIKE"] != 3 {
		t.Errorf("Expected 3 LIKEs, got %d", counts["LIKE"])
	}
	if counts["LOVE"] != 2 {
		t.Errorf("Expected 2 LOVEs, got %d", counts["LOVE"])
	}
}

func TestCassandraStorage_GetUserReactionsByType(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()

	// Add reactions
	target1 := golikeit.EntityTarget{EntityType: "post", EntityID: "1"}
	target2 := golikeit.EntityTarget{EntityType: "post", EntityID: "2"}
	_, _ = storage.AddReaction(ctx, "user1", target1, "LIKE")
	_, _ = storage.AddReaction(ctx, "user1", target2, "LOVE")

	pag := pagination.Pagination{Limit: 10, Offset: 0}
	reactions, total, err := storage.GetUserReactionsByType(ctx, "user1", "LIKE", pag)
	if err != nil {
		t.Fatalf("GetUserReactionsByType failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}
	if len(reactions) != 1 {
		t.Errorf("Expected 1 reaction, got %d", len(reactions))
	}
	if reactions[0].ReactionType != "LIKE" {
		t.Errorf("Expected LIKE reaction, got %s", reactions[0].ReactionType)
	}
}

func TestCassandraStorage_GetEntityReactions(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Add reactions from different users
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LIKE")

	pag := pagination.Pagination{Limit: 10, Offset: 0}
	reactions, total, err := storage.GetEntityReactions(ctx, target, pag)
	if err != nil {
		t.Fatalf("GetEntityReactions failed: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(reactions) != 2 {
		t.Errorf("Expected 2 reactions, got %d", len(reactions))
	}
}

func TestCassandraStorage_GetRecentReactions(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Add reactions
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LOVE")

	recent, err := storage.GetRecentReactions(ctx, target, 10)
	if err != nil {
		t.Fatalf("GetRecentReactions failed: %v", err)
	}
	if len(recent) != 2 {
		t.Errorf("Expected 2 recent reactions, got %d", len(recent))
	}
}

func TestCassandraStorage_GetLastReactionTime(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Get for empty entity
	lastTime, err := storage.GetLastReactionTime(ctx, target)
	if err != nil {
		t.Fatalf("GetLastReactionTime failed: %v", err)
	}
	if lastTime != nil {
		t.Error("Expected nil for empty entity")
	}

	// Add reactions
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LOVE")

	lastTime, err = storage.GetLastReactionTime(ctx, target)
	if err != nil {
		t.Fatalf("GetLastReactionTime failed: %v", err)
	}
	if lastTime == nil {
		t.Error("Expected non-nil last time")
	}
}

func TestCassandraStorage_GetEntityReactionDetail(t *testing.T) {
	storage := newTestCassandraStorage(t)
	defer storage.Close()

	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "post", EntityID: "123"}

	// Add reactions
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user3", target, "LOVE")

	detail, err := storage.GetEntityReactionDetail(ctx, target, 2)
	if err != nil {
		t.Fatalf("GetEntityReactionDetail failed: %v", err)
	}
	if detail.TotalReactions != 3 {
		t.Errorf("Expected total 3, got %d", detail.TotalReactions)
	}
	if detail.CountsByType["LIKE"] != 2 {
		t.Errorf("Expected 2 LIKEs, got %d", detail.CountsByType["LIKE"])
	}
	if detail.LastReaction == nil {
		t.Error("Expected non-nil last reaction time")
	}
}

func TestCassandraConfig_Default(t *testing.T) {
	config := DefaultCassandraConfig()
	if len(config.Hosts) != 1 || config.Hosts[0] != "localhost:9042" {
		t.Errorf("Expected hosts [localhost:9042], got %v", config.Hosts)
	}
	if config.Keyspace != "golikeit" {
		t.Errorf("Expected keyspace golikeit, got %s", config.Keyspace)
	}
}

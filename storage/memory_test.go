package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/golikeit"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	if storage == nil {
		t.Fatal("NewMemoryStorage() returned nil")
	}
	if storage.reactions == nil {
		t.Error("reactions map not initialized")
	}
	if storage.counts == nil {
		t.Error("counts map not initialized")
	}
}

func TestMemoryStorage_AddReaction(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	tests := []struct {
		name         string
		userID       string
		target       golikeit.EntityTarget
		reactionType string
		wantReplaced bool
		wantErr      bool
	}{
		{
			name:         "add new reaction",
			userID:       "user1",
			target:       golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"},
			reactionType: "LIKE",
			wantReplaced: false,
			wantErr:      false,
		},
		{
			name:         "replace existing reaction",
			userID:       "user1",
			target:       golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"},
			reactionType: "LOVE",
			wantReplaced: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaced, err := storage.AddReaction(ctx, tt.userID, tt.target, tt.reactionType)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if replaced != tt.wantReplaced {
				t.Errorf("AddReaction() replaced = %v, want %v", replaced, tt.wantReplaced)
			}
		})
	}
}

func TestMemoryStorage_RemoveReaction(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Add a reaction first
	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")

	tests := []struct {
		name    string
		userID  string
		target  golikeit.EntityTarget
		wantErr bool
	}{
		{
			name:    "remove existing reaction",
			userID:  "user1",
			target:  target,
			wantErr: false,
		},
		{
			name:    "remove non-existing reaction",
			userID:  "user1",
			target:  target,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.RemoveReaction(ctx, tt.userID, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveReaction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryStorage_GetUserReaction(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")

	tests := []struct {
		name       string
		userID     string
		target     golikeit.EntityTarget
		wantResult string
		wantErr    bool
	}{
		{
			name:       "get existing reaction",
			userID:     "user1",
			target:     target,
			wantResult: "LIKE",
			wantErr:    false,
		},
		{
			name:       "get non-existing reaction",
			userID:     "user2",
			target:     target,
			wantResult: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := storage.GetUserReaction(ctx, tt.userID, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.wantResult {
				t.Errorf("GetUserReaction() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestMemoryStorage_HasUserReaction(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")

	tests := []struct {
		name     string
		userID   string
		target   golikeit.EntityTarget
		want     bool
		wantErr  bool
	}{
		{
			name:    "has reaction",
			userID:  "user1",
			target:  target,
			want:    true,
			wantErr: false,
		},
		{
			name:    "no reaction",
			userID:  "user2",
			target:  target,
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := storage.HasUserReaction(ctx, tt.userID, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasUserReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HasUserReaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMemoryStorage_GetEntityCounts(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user3", target, "LOVE")

	counts, err := storage.GetEntityCounts(ctx, target)
	if err != nil {
		t.Fatalf("GetEntityCounts() error = %v", err)
	}

	if counts.Counts["LIKE"] != 2 {
		t.Errorf("counts['LIKE'] = %d, want 2", counts.Counts["LIKE"])
	}
	if counts.Counts["LOVE"] != 1 {
		t.Errorf("counts['LOVE'] = %d, want 1", counts.Counts["LOVE"])
	}
	if counts.Total != 3 {
		t.Errorf("counts.Total = %d, want 3", counts.Total)
	}
}

func TestMemoryStorage_GetUserReactions(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Add reactions for user1
	target1 := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	target2 := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo2"}
	target3 := golikeit.EntityTarget{EntityType: "video", EntityID: "video1"}
	_, _ = storage.AddReaction(ctx, "user1", target1, "LIKE")
	_, _ = storage.AddReaction(ctx, "user1", target2, "LOVE")
	_, _ = storage.AddReaction(ctx, "user1", target3, "LIKE")

	tests := []struct {
		name         string
		userID       string
		filters      Filters
		pagination   pagination.Pagination
		wantCount    int
		wantTotal    int64
	}{
		{
			name:       "get all reactions for user",
			userID:     "user1",
			pagination: pagination.Pagination{Limit: 10, Offset: 0},
			wantCount:  3,
			wantTotal:  3,
		},
		{
			name:       "filter by entity_type",
			userID:     "user1",
			filters:    Filters{EntityType: "photo"},
			pagination: pagination.Pagination{Limit: 10, Offset: 0},
			wantCount:  2,
			wantTotal:  2,
		},
		{
			name:       "filter by reaction_type",
			userID:     "user1",
			filters:    Filters{ReactionType: "LIKE"},
			pagination: pagination.Pagination{Limit: 10, Offset: 0},
			wantCount:  2,
			wantTotal:  2,
		},
		{
			name:       "user with no reactions",
			userID:     "user2",
			pagination: pagination.Pagination{Limit: 10, Offset: 0},
			wantCount:  0,
			wantTotal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := storage.GetUserReactions(ctx, tt.userID, tt.filters, tt.pagination)
			if err != nil {
				t.Errorf("GetUserReactions() error = %v", err)
				return
			}
			if len(results) != tt.wantCount {
				t.Errorf("GetUserReactions() returned %d results, want %d", len(results), tt.wantCount)
			}
			if total != tt.wantTotal {
				t.Errorf("GetUserReactions() total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestMemoryStorage_GetUserReactionCounts(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Add reactions for user1
	// Note: user1 can only have ONE reaction per target
	target1 := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	target2 := golikeit.EntityTarget{EntityType: "video", EntityID: "video1"}
	target3 := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo2"}
	_, _ = storage.AddReaction(ctx, "user1", target1, "LIKE")  // LIKE on photo1
	_, _ = storage.AddReaction(ctx, "user1", target2, "LIKE")  // LIKE on video1
	_, _ = storage.AddReaction(ctx, "user1", target3, "LOVE")    // LOVE on photo2

	tests := []struct {
		name             string
		userID           string
		entityTypeFilter string
		wantLikeCount    int64
		wantLoveCount    int64
	}{
		{
			name:          "counts for all entity types",
			userID:        "user1",
			wantLikeCount: 2, // photo1 and video1
			wantLoveCount: 1, // photo2
		},
		{
			name:             "counts filtered by entity type photo",
			userID:           "user1",
			entityTypeFilter: "photo",
			wantLikeCount:    1, // photo1
			wantLoveCount:    1, // photo2
		},
		{
			name:             "counts filtered by entity type video",
			userID:           "user1",
			entityTypeFilter: "video",
			wantLikeCount:    1,  // video1
			wantLoveCount:    0,
		},
		{
			name:          "user with no reactions",
			userID:        "user2",
			wantLikeCount: 0,
			wantLoveCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			counts, err := storage.GetUserReactionCounts(ctx, tt.userID, tt.entityTypeFilter)
			if err != nil {
				t.Errorf("GetUserReactionCounts() error = %v", err)
				return
			}
			if counts["LIKE"] != tt.wantLikeCount {
				t.Errorf("GetUserReactionCounts()['LIKE'] = %d, want %d", counts["LIKE"], tt.wantLikeCount)
			}
			if counts["LOVE"] != tt.wantLoveCount {
				t.Errorf("GetUserReactionCounts()['LOVE'] = %d, want %d", counts["LOVE"], tt.wantLoveCount)
			}
		})
	}
}

func TestMemoryStorage_GetRecentReactions(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	_, _ = storage.AddReaction(ctx, "user2", target, "LOVE")
	_, _ = storage.AddReaction(ctx, "user3", target, "ANGRY")

	recent, err := storage.GetRecentReactions(ctx, target, 2)
	if err != nil {
		t.Fatalf("GetRecentReactions() error = %v", err)
	}

	if len(recent) != 2 {
		t.Errorf("GetRecentReactions() returned %d results, want 2", len(recent))
	}
}

func TestMemoryStorage_GetLastReactionTime(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}

	// Add first reaction
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
	time.Sleep(10 * time.Millisecond)

	// Add second reaction
	beforeSecond := time.Now()
	_, _ = storage.AddReaction(ctx, "user2", target, "LOVE")

	lastTime, err := storage.GetLastReactionTime(ctx, target)
	if err != nil {
		t.Fatalf("GetLastReactionTime() error = %v", err)
	}

	if lastTime == nil {
		t.Fatal("GetLastReactionTime() returned nil")
	}

	if lastTime.Before(beforeSecond) {
		t.Error("GetLastReactionTime() returned time before second reaction")
	}
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()

	// Add some data
	ctx := context.Background()
	target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
	_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")

	// Close storage
	err := storage.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify operations fail after close
	_, err = storage.AddReaction(ctx, "user2", target, "LOVE")
	if err != golikeit.ErrStorageUnavailable {
		t.Errorf("AddReaction() after Close() error = %v, want ErrStorageUnavailable", err)
	}
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	storage := NewMemoryStorage()

	// Run concurrent operations
	done := make(chan bool, 3)

	// Writer 1
	go func() {
		for i := 0; i < 50; i++ {
			target := golikeit.EntityTarget{EntityType: "photo", EntityID: fmt.Sprintf("photo%d", i)}
			_, _ = storage.AddReaction(ctx, "user1", target, "LIKE")
		}
		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 50; i++ {
			target := golikeit.EntityTarget{EntityType: "video", EntityID: fmt.Sprintf("video%d", i)}
			_, _ = storage.AddReaction(ctx, "user2", target, "LOVE")
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			target := golikeit.EntityTarget{EntityType: "photo", EntityID: "photo1"}
			_, _ = storage.GetEntityCounts(ctx, target)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for concurrent operations")
		}
	}
}


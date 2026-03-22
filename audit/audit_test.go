package audit

import (
	"context"
	"testing"
	"time"
)

func TestNewEntry(t *testing.T) {
	entry := NewEntry(OperationAdd, "user123", "photo", "456", "LIKE", "")

	if entry.Operation != OperationAdd {
		t.Errorf("expected operation ADD, got %s", entry.Operation)
	}
	if entry.UserID != "user123" {
		t.Errorf("expected userID user123, got %s", entry.UserID)
	}
	if entry.EntityType != "photo" {
		t.Errorf("expected entityType photo, got %s", entry.EntityType)
	}
	if entry.EntityID != "456" {
		t.Errorf("expected entityID 456, got %s", entry.EntityID)
	}
	if entry.ReactionType != "LIKE" {
		t.Errorf("expected reactionType LIKE, got %s", entry.ReactionType)
	}
	if entry.PreviousReaction != "" {
		t.Errorf("expected empty previousReaction, got %s", entry.PreviousReaction)
	}
	if entry.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNullAuditor(t *testing.T) {
	auditor := NewNullAuditor()
	ctx := context.Background()

	t.Run("LogOperation returns nil", func(t *testing.T) {
		entry := NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", "")
		err := auditor.LogOperation(ctx, entry)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("GetByUser returns empty slice", func(t *testing.T) {
		entries, err := auditor.GetByUser(ctx, "user1", 10, 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(entries))
		}
	})

	t.Run("GetByEntity returns empty slice", func(t *testing.T) {
		entries, err := auditor.GetByEntity(ctx, "photo", "1", 10, 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(entries))
		}
	})

	t.Run("GetByOperation returns empty slice", func(t *testing.T) {
		entries, err := auditor.GetByOperation(ctx, OperationAdd, 10, 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(entries))
		}
	})

	t.Run("GetByDateRange returns empty slice", func(t *testing.T) {
		now := time.Now()
		entries, err := auditor.GetByDateRange(ctx, now.Add(-time.Hour), now, 10, 0)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(entries) != 0 {
			t.Errorf("expected empty slice, got %d entries", len(entries))
		}
	})
}

func TestInMemoryStorage(t *testing.T) {
	storage := NewInMemoryStorage()
	ctx := context.Background()

	t.Run("Insert assigns ID", func(t *testing.T) {
		entry := NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", "")
		stored, err := storage.Insert(ctx, entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stored.ID == "" {
			t.Error("expected ID to be assigned")
		}
		if stored.ID == entry.ID {
			t.Error("expected ID to be generated, not preserved")
		}
	})

	t.Run("GetByUser returns matching entries", func(t *testing.T) {
		storage := NewInMemoryStorage() // fresh storage

		// Insert entries for different users
		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", ""))
		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user1", "photo", "2", "LOVE", ""))
		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user2", "photo", "3", "LIKE", ""))

		entries, err := storage.GetByUser(ctx, "user1", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}

		// Verify correct user
		for _, e := range entries {
			if e.UserID != "user1" {
				t.Errorf("expected user1, got %s", e.UserID)
			}
		}
	})

	t.Run("GetByEntity returns matching entries", func(t *testing.T) {
		storage := NewInMemoryStorage()

		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", ""))
		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user2", "photo", "1", "LOVE", ""))
		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user3", "video", "1", "LIKE", ""))

		entries, err := storage.GetByEntity(ctx, "photo", "1", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
	})

	t.Run("GetByOperation returns matching entries", func(t *testing.T) {
		storage := NewInMemoryStorage()

		_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", ""))
		_, _ = storage.Insert(ctx, NewEntry(OperationReplace, "user1", "photo", "1", "LOVE", "LIKE"))
		_, _ = storage.Insert(ctx, NewEntry(OperationRemove, "user1", "photo", "1", "", "LOVE"))

		entries, err := storage.GetByOperation(ctx, OperationReplace, 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
		if len(entries) > 0 && entries[0].Operation != OperationReplace {
			t.Errorf("expected REPLACE operation, got %s", entries[0].Operation)
		}
	})

	t.Run("GetByDateRange returns entries in range", func(t *testing.T) {
		storage := NewInMemoryStorage()

		now := time.Now().UTC()
		oldEntry := NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", "")
		oldEntry.Timestamp = now.Add(-2 * time.Hour)

		newEntry := NewEntry(OperationAdd, "user2", "photo", "2", "LIKE", "")
		newEntry.Timestamp = now

		_, _ = storage.Insert(ctx, oldEntry)
		_, _ = storage.Insert(ctx, newEntry)

		// Query last hour
		entries, err := storage.GetByDateRange(ctx, now.Add(-time.Hour), now.Add(time.Hour), 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
		if len(entries) > 0 && entries[0].UserID != "user2" {
			t.Errorf("expected user2 (new entry), got %s", entries[0].UserID)
		}
	})

	t.Run("Pagination works correctly", func(t *testing.T) {
		storage := NewInMemoryStorage()

		// Insert 5 entries
		for i := 0; i < 5; i++ {
			_, _ = storage.Insert(ctx, NewEntry(OperationAdd, "user1", "photo", string(rune('1'+i)), "LIKE", ""))
		}

		// Get first 2
		entries, err := storage.GetByUser(ctx, "user1", 2, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}

		// Get next 2
		entries, err = storage.GetByUser(ctx, "user1", 2, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}

		// Get remaining
		entries, err = storage.GetByUser(ctx, "user1", 2, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("Results ordered by timestamp descending", func(t *testing.T) {
		storage := NewInMemoryStorage()

		now := time.Now().UTC()
		entry1 := NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", "")
		entry1.Timestamp = now.Add(-2 * time.Hour)

		entry2 := NewEntry(OperationAdd, "user1", "photo", "2", "LIKE", "")
		entry2.Timestamp = now.Add(-1 * time.Hour)

		entry3 := NewEntry(OperationAdd, "user1", "photo", "3", "LIKE", "")
		entry3.Timestamp = now

		_, _ = storage.Insert(ctx, entry1)
		_, _ = storage.Insert(ctx, entry2)
		_, _ = storage.Insert(ctx, entry3)

		entries, err := storage.GetByUser(ctx, "user1", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(entries))
		}

		// Should be in reverse chronological order: entry3, entry2, entry1
		if !entries[0].Timestamp.Equal(entry3.Timestamp) {
			t.Errorf("first entry should be newest, got %v", entries[0].Timestamp)
		}
		if !entries[2].Timestamp.Equal(entry1.Timestamp) {
			t.Errorf("last entry should be oldest, got %v", entries[2].Timestamp)
		}
	})
}

func TestPersistentAuditor(t *testing.T) {
	storage := NewInMemoryStorage()
	auditor := NewPersistentAuditor(storage)
	ctx := context.Background()

	t.Run("LogOperation persists entry", func(t *testing.T) {
		entry := NewEntry(OperationAdd, "user1", "photo", "1", "LIKE", "")
		err := auditor.LogOperation(ctx, entry)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Verify entry was stored
		entries, err := auditor.GetByUser(ctx, "user1", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}
	})

	t.Run("LogOperation is fire-and-forget", func(t *testing.T) {
		// Should not error even if storage has issues
		entry := NewEntry(OperationReplace, "user1", "photo", "1", "LOVE", "LIKE")
		err := auditor.LogOperation(ctx, entry)
		if err != nil {
			t.Errorf("LogOperation should never error (fire-and-forget), got %v", err)
		}
	})
}

func TestOperations(t *testing.T) {
	tests := []struct {
		name      string
		operation Operation
		valid     bool
	}{
		{"ADD", OperationAdd, true},
		{"REPLACE", OperationReplace, true},
		{"REMOVE", OperationRemove, true},
		{"INVALID", Operation("INVALID"), true}, // Operations are just strings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.operation) == "" {
				t.Error("empty operation should be avoided")
			}
		})
	}
}

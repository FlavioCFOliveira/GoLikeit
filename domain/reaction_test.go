package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewReaction(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		entityType    string
		entityID      string
		reactionType  string
		wantErr       bool
		errMsg        string
	}{
		{
			name:         "valid reaction",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "LIKE",
			wantErr:      false,
		},
		{
			name:         "empty user_id",
			userID:       "",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "LIKE",
			wantErr:      true,
			errMsg:       "invalid user_id",
		},
		{
			name:         "empty entity_type",
			userID:       "user-123",
			entityType:   "",
			entityID:     "entity-456",
			reactionType: "LIKE",
			wantErr:      true,
			errMsg:       "invalid entity_type",
		},
		{
			name:         "empty entity_id",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "",
			reactionType: "LIKE",
			wantErr:      true,
			errMsg:       "invalid entity_id",
		},
		{
			name:         "empty reaction_type",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "",
			wantErr:      true,
			errMsg:       "invalid reaction_type",
		},
		{
			name:         "invalid entity_type with uppercase",
			userID:       "user-123",
			entityType:   "Photo",
			entityID:     "entity-456",
			reactionType: "LIKE",
			wantErr:      true,
			errMsg:       "entity_type must match",
		},
		{
			name:         "invalid reaction_type with lowercase",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "like",
			wantErr:      true,
			errMsg:       "reaction_type must match",
		},
		{
			name:         "valid entity_type with underscore",
			userID:       "user-123",
			entityType:   "blog_post",
			entityID:     "entity-456",
			reactionType: "LIKE",
			wantErr:      false,
		},
		{
			name:         "valid reaction_type with underscore and hyphen",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "THUMBS_UP-1",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReaction(tt.userID, tt.entityType, tt.entityID, tt.reactionType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewReaction() error message = %v, should contain %v", err, tt.errMsg)
				}
				return
			}
			if !tt.wantErr {
				if got.ID == "" {
					t.Error("NewReaction() ID should not be empty")
				}
				if got.UserID != tt.userID {
					t.Errorf("NewReaction() UserID = %v, want %v", got.UserID, tt.userID)
				}
				if got.EntityType != tt.entityType {
					t.Errorf("NewReaction() EntityType = %v, want %v", got.EntityType, tt.entityType)
				}
				if got.EntityID != tt.entityID {
					t.Errorf("NewReaction() EntityID = %v, want %v", got.EntityID, tt.entityID)
				}
				if got.ReactionType != tt.reactionType {
					t.Errorf("NewReaction() ReactionType = %v, want %v", got.ReactionType, tt.reactionType)
				}
				if got.CreatedAt.IsZero() {
					t.Error("NewReaction() CreatedAt should not be zero")
				}
			}
		})
	}
}

func TestNewReactionWithID(t *testing.T) {
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		id           string
		userID       string
		entityType   string
		entityID     string
		reactionType string
		createdAt    time.Time
		wantErr      bool
	}{
		{
			name:         "valid with specific ID",
			id:           validUUID,
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "LIKE",
			createdAt:    createdAt,
			wantErr:      false,
		},
		{
			name:         "invalid UUID",
			id:           "not-a-uuid",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "LIKE",
			createdAt:    createdAt,
			wantErr:      true,
		},
		{
			name:         "empty ID",
			id:           "",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "entity-456",
			reactionType: "LIKE",
			createdAt:    createdAt,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReactionWithID(tt.id, tt.userID, tt.entityType, tt.entityID, tt.reactionType, tt.createdAt)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReactionWithID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.ID != tt.id {
					t.Errorf("NewReactionWithID() ID = %v, want %v", got.ID, tt.id)
				}
				if !got.CreatedAt.Equal(tt.createdAt) {
					t.Errorf("NewReactionWithID() CreatedAt = %v, want %v", got.CreatedAt, tt.createdAt)
				}
			}
		})
	}
}

func TestReactionTarget(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityID   string
		wantErr    bool
		wantString string
	}{
		{
			name:       "valid target",
			entityType: "photo",
			entityID:   "123",
			wantErr:    false,
			wantString: "photo:123",
		},
		{
			name:       "empty entity_type",
			entityType: "",
			entityID:   "123",
			wantErr:    true,
		},
		{
			name:       "empty entity_id",
			entityType: "photo",
			entityID:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewReactionTarget(tt.entityType, tt.entityID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewReactionTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.String() != tt.wantString {
					t.Errorf("ReactionTarget.String() = %v, want %v", got.String(), tt.wantString)
				}
				if !got.IsValid() {
					t.Error("ReactionTarget.IsValid() should be true")
				}
			}
		})
	}
}

func TestReactionTarget_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		target  ReactionTarget
		want    bool
	}{
		{
			name:    "valid",
			target:  ReactionTarget{EntityType: "photo", EntityID: "123"},
			want:    true,
		},
		{
			name:    "empty entity_type",
			target:  ReactionTarget{EntityType: "", EntityID: "123"},
			want:    false,
		},
		{
			name:    "empty entity_id",
			target:  ReactionTarget{EntityType: "photo", EntityID: "123"},
			want:    true,
		},
		{
			name:    "both empty",
			target:  ReactionTarget{},
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsValid(); got != tt.want {
				t.Errorf("ReactionTarget.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserReactionKey(t *testing.T) {
	tests := []struct {
		name         string
		userID       string
		entityType   string
		entityID     string
		wantErr      bool
		wantString   string
		wantReaction ReactionTarget
	}{
		{
			name:         "valid key",
			userID:       "user-123",
			entityType:   "photo",
			entityID:     "456",
			wantErr:      false,
			wantString:   "user-123:photo:456",
			wantReaction: ReactionTarget{EntityType: "photo", EntityID: "456"},
		},
		{
			name:       "empty user_id",
			userID:     "",
			entityType: "photo",
			entityID:   "456",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUserReactionKey(tt.userID, tt.entityType, tt.entityID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUserReactionKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.String() != tt.wantString {
					t.Errorf("UserReactionKey.String() = %v, want %v", got.String(), tt.wantString)
				}
				if got.ReactionKey() != tt.wantReaction {
					t.Errorf("UserReactionKey.ReactionKey() = %v, want %v", got.ReactionKey(), tt.wantReaction)
				}
			}
		})
	}
}

func TestValidateFunctions(t *testing.T) {
	t.Run("ValidateUserID", func(t *testing.T) {
		tests := []struct {
			name   string
			userID string
			want   bool // true = no error
		}{
			{"valid", "user-123", true},
			{"empty", "", false},
			{"too long", string(make([]byte, 257)), false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateUserID(tt.userID)
				if (err == nil) != tt.want {
					t.Errorf("ValidateUserID() error = %v, want %v", err, tt.want)
				}
			})
		}
	})

	t.Run("ValidateEntityType", func(t *testing.T) {
		tests := []struct {
			name       string
			entityType string
			want       bool
		}{
			{"valid", "photo", true},
			{"valid with underscore", "blog_post", true},
			{"empty", "", false},
			{"uppercase", "Photo", false},
			{"with hyphen", "blog-post", false},
			{"too long", string(make([]byte, 65)), false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateEntityType(tt.entityType)
				if (err == nil) != tt.want {
					t.Errorf("ValidateEntityType() error = %v, want %v", err, tt.want)
				}
			})
		}
	})

	t.Run("ValidateEntityID", func(t *testing.T) {
		tests := []struct {
			name     string
			entityID string
			want     bool
		}{
			{"valid", "entity-123", true},
			{"empty", "", false},
			{"too long", string(make([]byte, 257)), false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateEntityID(tt.entityID)
				if (err == nil) != tt.want {
					t.Errorf("ValidateEntityID() error = %v, want %v", err, tt.want)
				}
			})
		}
	})

	t.Run("ValidateReactionType", func(t *testing.T) {
		tests := []struct {
			name         string
			reactionType string
			want         bool
		}{
			{"valid", "LIKE", true},
			{"valid with underscore", "THUMBS_UP", true},
			{"valid with hyphen", "THUMBS-UP", true},
			{"empty", "", false},
			{"lowercase", "like", false},
			{"too long", string(make([]byte, 65)), false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := ValidateReactionType(tt.reactionType)
				if (err == nil) != tt.want {
					t.Errorf("ValidateReactionType() error = %v, want %v", err, tt.want)
				}
			})
		}
	})
}

func TestReaction_JSONSerialization(t *testing.T) {
	reaction := MustNewReaction("user-123", "photo", "entity-456", "LIKE")

	// Test marshaling
	data, err := json.Marshal(reaction)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Test unmarshaling
	var got Reaction
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.UserID != reaction.UserID {
		t.Errorf("UserID = %v, want %v", got.UserID, reaction.UserID)
	}
	if got.EntityType != reaction.EntityType {
		t.Errorf("EntityType = %v, want %v", got.EntityType, reaction.EntityType)
	}
	if got.EntityID != reaction.EntityID {
		t.Errorf("EntityID = %v, want %v", got.EntityID, reaction.EntityID)
	}
	if got.ReactionType != reaction.ReactionType {
		t.Errorf("ReactionType = %v, want %v", got.ReactionType, reaction.ReactionType)
	}
}

func TestReactionTarget_MarshalText(t *testing.T) {
	target := MustNewReactionTarget("photo", "123")

	// Test marshaling
	data, err := target.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}

	expected := "photo:123"
	if string(data) != expected {
		t.Errorf("MarshalText() = %v, want %v", string(data), expected)
	}

	// Test unmarshaling
	var got ReactionTarget
	if err := got.UnmarshalText(data); err != nil {
		t.Fatalf("UnmarshalText() error = %v", err)
	}

	if got.EntityType != target.EntityType {
		t.Errorf("EntityType = %v, want %v", got.EntityType, target.EntityType)
	}
	if got.EntityID != target.EntityID {
		t.Errorf("EntityID = %v, want %v", got.EntityID, target.EntityID)
	}
}

func TestReactionTarget_UnmarshalText_InvalidFormat(t *testing.T) {
	var target ReactionTarget
	err := target.UnmarshalText([]byte("invalid-format"))
	if err == nil {
		t.Error("UnmarshalText() should return error for invalid format")
	}
}

func BenchmarkNewReaction(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewReaction("user-123", "photo", "entity-456", "LIKE")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReactionTarget_String(b *testing.B) {
	target := MustNewReactionTarget("photo", "123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = target.String()
	}
}

func BenchmarkValidateEntityType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateEntityType("photo")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

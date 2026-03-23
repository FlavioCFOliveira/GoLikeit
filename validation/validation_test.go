package validation

import (
	"strings"
	"testing"
)

func TestValidateUserID(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple",
			userID:  "user123",
			wantErr: false,
		},
		{
			name:    "valid with special chars",
			userID:  "user@example.com",
			wantErr: false,
		},
		{
			name:    "valid with unicode",
			userID:  "用户_123",
			wantErr: false,
		},
		{
			name:    "valid max length",
			userID:  strings.Repeat("a", 256),
			wantErr: false,
		},
		{
			name:    "empty",
			userID:  "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "exceeds max length",
			userID:  strings.Repeat("a", 257),
			wantErr: true,
			errMsg:  "exceeds maximum length",
		},
		{
			name:    "contains null byte",
			userID:  "user\x00name",
			wantErr: true,
			errMsg:  "null bytes",
		},
		{
			name:    "contains tab",
			userID:  "user\tname",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "contains newline",
			userID:  "user\nname",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "contains carriage return",
			userID:  "user\rname",
			wantErr: true,
			errMsg:  "control characters",
		},
		{
			name:    "contains DEL char",
			userID:  "user\x7Fname",
			wantErr: true,
			errMsg:  "control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUserID(tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateUserID(%q) expected error, got nil", tt.userID)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateUserID(%q) error = %v, expected to contain %q", tt.userID, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateUserID(%q) unexpected error: %v", tt.userID, err)
				}
			}
		})
	}
}

func TestValidateEntityType(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid simple",
			entityType: "post",
			wantErr:    false,
		},
		{
			name:       "valid with underscore",
			entityType: "blog_post",
			wantErr:    false,
		},
		{
			name:       "valid with number",
			entityType: "post_v2",
			wantErr:    false,
		},
		{
			name:       "valid max length",
			entityType: strings.Repeat("a", 64),
			wantErr:    false,
		},
		{
			name:       "empty",
			entityType: "",
			wantErr:    true,
			errMsg:     "cannot be empty",
		},
		{
			name:       "exceeds max length",
			entityType: strings.Repeat("a", 65),
			wantErr:    true,
			errMsg:     "exceeds maximum length",
		},
		{
			name:       "uppercase letter",
			entityType: "Post",
			wantErr:    true,
			errMsg:     "lowercase letters",
		},
		{
			name:       "valid with hyphen",
			entityType: "blog-post",
			wantErr:    false,
		},
		{
			name:       "contains space",
			entityType: "blog post",
			wantErr:    true,
			errMsg:     "lowercase letters",
		},
		{
			name:       "contains special char",
			entityType: "post@123",
			wantErr:    true,
			errMsg:     "lowercase letters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntityType(tt.entityType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateEntityType(%q) expected error, got nil", tt.entityType)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEntityType(%q) error = %v, expected to contain %q", tt.entityType, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEntityType(%q) unexpected error: %v", tt.entityType, err)
				}
			}
		})
	}
}

func TestValidateEntityID(t *testing.T) {
	tests := []struct {
		name     string
		entityID string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid simple",
			entityID: "123",
			wantErr:  false,
		},
		{
			name:     "valid with special chars",
			entityID: "post-123_v2",
			wantErr:  false,
		},
		{
			name:     "valid max length",
			entityID: strings.Repeat("a", 256),
			wantErr:  false,
		},
		{
			name:     "empty",
			entityID: "",
			wantErr:  true,
			errMsg:   "cannot be empty",
		},
		{
			name:     "exceeds max length",
			entityID: strings.Repeat("a", 257),
			wantErr:  true,
			errMsg:   "exceeds maximum length",
		},
		{
			name:     "contains null byte",
			entityID: "entity\x00id",
			wantErr:  true,
			errMsg:   "null bytes",
		},
		{
			name:     "contains tab",
			entityID: "entity\tid",
			wantErr:  true,
			errMsg:   "control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntityID(tt.entityID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateEntityID(%q) expected error, got nil", tt.entityID)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEntityID(%q) error = %v, expected to contain %q", tt.entityID, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEntityID(%q) unexpected error: %v", tt.entityID, err)
				}
			}
		})
	}
}

func TestValidateReactionType(t *testing.T) {
	tests := []struct {
		name         string
		reactionType string
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "valid LIKE",
			reactionType: "LIKE",
			wantErr:      false,
		},
		{
			name:         "valid LOVE",
			reactionType: "LOVE",
			wantErr:      false,
		},
		{
			name:         "valid with underscore",
			reactionType: "SUPER_LIKE",
			wantErr:      false,
		},
		{
			name:         "valid with hyphen",
			reactionType: "THUMBS-UP",
			wantErr:      false,
		},
		{
			name:         "valid with number",
			reactionType: "LIKE2",
			wantErr:      false,
		},
		{
			name:         "valid max length",
			reactionType: strings.Repeat("A", 64),
			wantErr:      false,
		},
		{
			name:         "empty",
			reactionType: "",
			wantErr:      true,
			errMsg:       "cannot be empty",
		},
		{
			name:         "exceeds max length",
			reactionType: strings.Repeat("A", 65),
			wantErr:      true,
			errMsg:       "exceeds maximum length",
		},
		{
			name:         "lowercase letter",
			reactionType: "like",
			wantErr:      true,
			errMsg:       "uppercase letters",
		},
		{
			name:         "mixed case",
			reactionType: "Like",
			wantErr:      true,
			errMsg:       "uppercase letters",
		},
		{
			name:         "contains space",
			reactionType: "SUPER LIKE",
			wantErr:      true,
			errMsg:       "uppercase letters",
		},
		{
			name:         "contains special char",
			reactionType: "LIKE@123",
			wantErr:      true,
			errMsg:       "uppercase letters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReactionType(tt.reactionType)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateReactionType(%q) expected error, got nil", tt.reactionType)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateReactionType(%q) error = %v, expected to contain %q", tt.reactionType, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateReactionType(%q) unexpected error: %v", tt.reactionType, err)
				}
			}
		})
	}
}

// mockEntityTarget is a test implementation of EntityTarget.
type mockEntityTarget struct {
	entityType string
	entityID   string
}

func (m *mockEntityTarget) GetEntityType() string { return m.entityType }
func (m *mockEntityTarget) GetEntityID() string   { return m.entityID }

func TestValidateEntityTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  *mockEntityTarget
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid target",
			target: &mockEntityTarget{
				entityType: "post",
				entityID:   "123",
			},
			wantErr: false,
		},
		{
			name:    "nil target",
			target:  nil,
			wantErr: true,
			errMsg:  "cannot be nil",
		},
		{
			name: "invalid entity type",
			target: &mockEntityTarget{
				entityType: "Post",
				entityID:   "123",
			},
			wantErr: true,
			errMsg:  "entity_type",
		},
		{
			name: "invalid entity id",
			target: &mockEntityTarget{
				entityType: "post",
				entityID:   "",
			},
			wantErr: true,
			errMsg:  "entity_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntityTarget(tt.target)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateEntityTarget() expected error, got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateEntityTarget() error = %v, expected to contain %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEntityTarget() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:  "test_field",
		Reason: "test reason",
	}

	expected := `validation error for field "test_field": test reason`
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}

func BenchmarkValidateUserID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateUserID("user123")
	}
}

func BenchmarkValidateEntityType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateEntityType("post")
	}
}

func BenchmarkValidateReactionType(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ValidateReactionType("LIKE")
	}
}

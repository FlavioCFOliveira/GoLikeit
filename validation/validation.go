// Package validation provides input validation for the GoLikeit system.
package validation

import (
	"fmt"
	"reflect"
	"regexp"
)

// Maximum length constants for input fields.
const (
	UserIDMaxLen       = 256
	EntityTypeMaxLen   = 64
	EntityIDMaxLen     = 256
	ReactionTypeMaxLen = 64
)

// Pre-compiled regex patterns for validation.
var (
	// EntityTypePattern matches lowercase alphanumeric, underscore.
	// Pattern: ^[a-z0-9_]+$
	entityTypeRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

	// ReactionTypePattern matches uppercase alphanumeric, underscore, hyphen.
	// Pattern: ^[A-Z0-9_-]+$
	reactionTypeRegex = regexp.MustCompile(`^[A-Z0-9_-]+$`)
)

// ValidationError represents an input validation error.
type ValidationError struct {
	Field  string
	Reason string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field %q: %s", e.Field, e.Reason)
}

// ValidateUserID validates a user ID according to security policies.
// Requirements:
//   - Must not be empty
//   - Maximum length: 256 characters
//   - No null bytes
//   - No control characters
func ValidateUserID(userID string) error {
	if userID == "" {
		return &ValidationError{
			Field:  "user_id",
			Reason: "cannot be empty",
		}
	}

	if len(userID) > UserIDMaxLen {
		return &ValidationError{
			Field:  "user_id",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters", UserIDMaxLen),
		}
	}

	if containsNullByte(userID) {
		return &ValidationError{
			Field:  "user_id",
			Reason: "contains null bytes",
		}
	}

	if containsControlChar(userID) {
		return &ValidationError{
			Field:  "user_id",
			Reason: "contains control characters",
		}
	}

	return nil
}

// ValidateEntityType validates an entity type according to security policies.
// Requirements:
//   - Must match pattern: ^[a-z0-9_]+$ (lowercase alphanumeric and underscore)
//   - Maximum length: 64 characters
func ValidateEntityType(entityType string) error {
	if entityType == "" {
		return &ValidationError{
			Field:  "entity_type",
			Reason: "cannot be empty",
		}
	}

	if len(entityType) > EntityTypeMaxLen {
		return &ValidationError{
			Field:  "entity_type",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters", EntityTypeMaxLen),
		}
	}

	if !entityTypeRegex.MatchString(entityType) {
		return &ValidationError{
			Field:  "entity_type",
			Reason: "must contain only lowercase letters, numbers, and underscores",
		}
	}

	return nil
}

// ValidateEntityID validates an entity ID according to security policies.
// Requirements:
//   - Must not be empty
//   - Maximum length: 256 characters
//   - No null bytes
//   - No control characters
func ValidateEntityID(entityID string) error {
	if entityID == "" {
		return &ValidationError{
			Field:  "entity_id",
			Reason: "cannot be empty",
		}
	}

	if len(entityID) > EntityIDMaxLen {
		return &ValidationError{
			Field:  "entity_id",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters", EntityIDMaxLen),
		}
	}

	if containsNullByte(entityID) {
		return &ValidationError{
			Field:  "entity_id",
			Reason: "contains null bytes",
		}
	}

	if containsControlChar(entityID) {
		return &ValidationError{
			Field:  "entity_id",
			Reason: "contains control characters",
		}
	}

	return nil
}

// ValidateReactionType validates a reaction type according to security policies.
// Requirements:
//   - Must match pattern: ^[A-Z0-9_-]+$ (uppercase alphanumeric, underscore, hyphen)
//   - Maximum length: 64 characters
func ValidateReactionType(reactionType string) error {
	if reactionType == "" {
		return &ValidationError{
			Field:  "reaction_type",
			Reason: "cannot be empty",
		}
	}

	if len(reactionType) > ReactionTypeMaxLen {
		return &ValidationError{
			Field:  "reaction_type",
			Reason: fmt.Sprintf("exceeds maximum length of %d characters", ReactionTypeMaxLen),
		}
	}

	if !reactionTypeRegex.MatchString(reactionType) {
		return &ValidationError{
			Field:  "reaction_type",
			Reason: "must contain only uppercase letters, numbers, underscores, and hyphens",
		}
	}

	return nil
}

// EntityTarget represents an entity target for validation.
type EntityTarget interface {
	GetEntityType() string
	GetEntityID() string
}

// ValidateEntityTarget validates an entity target by validating both
// entity_type and entity_id fields.
func ValidateEntityTarget(target EntityTarget) error {
	// Check if target is nil or contains nil pointer
	if target == nil || (reflect.ValueOf(target).Kind() == reflect.Ptr && reflect.ValueOf(target).IsNil()) {
		return &ValidationError{
			Field:  "entity_target",
			Reason: "cannot be nil",
		}
	}

	if err := ValidateEntityType(target.GetEntityType()); err != nil {
		return err
	}

	if err := ValidateEntityID(target.GetEntityID()); err != nil {
		return err
	}

	return nil
}

// containsControlChar reports whether s contains any control characters
// (ASCII 0x00-0x1F and 0x7F).
func containsControlChar(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return true
		}
	}
	return false
}

// containsNullByte reports whether s contains any null bytes (0x00).
func containsNullByte(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == 0x00 {
			return true
		}
	}
	return false
}

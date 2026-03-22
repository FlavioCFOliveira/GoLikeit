// Package domain defines the core domain types and models for the GoLikeit reaction system.
// All types are designed as immutable value objects with proper validation and JSON serialization support.
package domain

import (
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// Validation patterns for domain fields.
const (
	// EntityTypePattern matches lowercase alphanumeric with underscores.
	EntityTypePattern = `^[a-z0-9_]+$`

	// ReactionTypePattern matches uppercase alphanumeric with underscores and hyphens.
	ReactionTypePattern = `^[A-Z0-9_-]+$`

	// MaxEntityTypeLength is the maximum length for entity types.
	MaxEntityTypeLength = 64

	// MaxReactionTypeLength is the maximum length for reaction types.
	MaxReactionTypeLength = 64

	// MaxIDLength is the maximum length for user and entity IDs.
	MaxIDLength = 256
)

var (
	entityTypeRegex   = regexp.MustCompile(EntityTypePattern)
	reactionTypeRegex = regexp.MustCompile(ReactionTypePattern)
)

// Reaction represents a user's reaction to a specific entity.
// It is an immutable value object that uniquely identifies a reaction instance.
//
// Fields are ordered by size (8 bytes, 8 bytes, then strings) to minimize memory padding.
type Reaction struct {
	// CreatedAt is the UTC timestamp when the reaction was created.
	// Stored as time.Time (16 bytes on 64-bit systems: 8 bytes for sec + 8 for nsec/location pointer).
	CreatedAt time.Time `json:"created_at"`

	// ID is the unique identifier for this reaction (UUID v4 format).
	ID string `json:"id"`

	// UserID identifies the user who made the reaction.
	UserID string `json:"user_id"`

	// EntityType is the type of entity being reacted to (e.g., "photo", "article").
	EntityType string `json:"entity_type"`

	// EntityID is the unique identifier of the entity being reacted to.
	EntityID string `json:"entity_id"`

	// ReactionType is the type of reaction (e.g., "LIKE", "LOVE").
	ReactionType string `json:"reaction_type"`
}

// NewReaction creates a new Reaction with the specified parameters.
// It generates a new UUID v4 for the ID and sets the creation timestamp to UTC now.
// Returns an error if any validation fails.
func NewReaction(userID, entityType, entityID, reactionType string) (Reaction, error) {
	if err := ValidateUserID(userID); err != nil {
		return Reaction{}, fmt.Errorf("invalid user_id: %w", err)
	}

	if err := ValidateEntityType(entityType); err != nil {
		return Reaction{}, fmt.Errorf("invalid entity_type: %w", err)
	}

	if err := ValidateEntityID(entityID); err != nil {
		return Reaction{}, fmt.Errorf("invalid entity_id: %w", err)
	}

	if err := ValidateReactionType(reactionType); err != nil {
		return Reaction{}, fmt.Errorf("invalid reaction_type: %w", err)
	}

	return Reaction{
		ID:           uuid.New().String(),
		UserID:       userID,
		EntityType:   entityType,
		EntityID:     entityID,
		ReactionType: reactionType,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

// MustNewReaction creates a new Reaction and panics if validation fails.
// This is useful for testing and situations where inputs are known to be valid.
func MustNewReaction(userID, entityType, entityID, reactionType string) Reaction {
	r, err := NewReaction(userID, entityType, entityID, reactionType)
	if err != nil {
		panic(err)
	}
	return r
}

// NewReactionWithID creates a Reaction with a specific ID and timestamp.
// This is useful for reconstructing reactions from storage.
// Returns an error if the ID is not a valid UUID or if other validation fails.
func NewReactionWithID(id, userID, entityType, entityID, reactionType string, createdAt time.Time) (Reaction, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Reaction{}, fmt.Errorf("invalid uuid: %w", err)
	}

	if err := ValidateUserID(userID); err != nil {
		return Reaction{}, fmt.Errorf("invalid user_id: %w", err)
	}

	if err := ValidateEntityType(entityType); err != nil {
		return Reaction{}, fmt.Errorf("invalid entity_type: %w", err)
	}

	if err := ValidateEntityID(entityID); err != nil {
		return Reaction{}, fmt.Errorf("invalid entity_id: %w", err)
	}

	if err := ValidateReactionType(reactionType); err != nil {
		return Reaction{}, fmt.Errorf("invalid reaction_type: %w", err)
	}

	return Reaction{
		ID:           id,
		UserID:       userID,
		EntityType:   entityType,
		EntityID:     entityID,
		ReactionType: reactionType,
		CreatedAt:    createdAt.UTC(),
	}, nil
}

// ReactionTarget identifies a unique entity that can receive reactions.
// It is an immutable value object representing the tuple (entity_type, entity_id).
type ReactionTarget struct {
	// EntityType is the type of entity (e.g., "photo", "article", "comment").
	EntityType string `json:"entity_type"`

	// EntityID is the unique identifier of the entity instance.
	EntityID string `json:"entity_id"`
}

// NewReactionTarget creates a new ReactionTarget with validation.
func NewReactionTarget(entityType, entityID string) (ReactionTarget, error) {
	if err := ValidateEntityType(entityType); err != nil {
		return ReactionTarget{}, err
	}

	if err := ValidateEntityID(entityID); err != nil {
		return ReactionTarget{}, err
	}

	return ReactionTarget{
		EntityType: entityType,
		EntityID:   entityID,
	}, nil
}

// MustNewReactionTarget creates a ReactionTarget and panics if validation fails.
func MustNewReactionTarget(entityType, entityID string) ReactionTarget {
	t, err := NewReactionTarget(entityType, entityID)
	if err != nil {
		panic(err)
	}
	return t
}

// String returns a string representation of the ReactionTarget in the form "entity_type:entity_id".
func (rt ReactionTarget) String() string {
	return fmt.Sprintf("%s:%s", rt.EntityType, rt.EntityID)
}

// IsValid returns true if both EntityType and EntityID are non-empty.
func (rt ReactionTarget) IsValid() bool {
	return rt.EntityType != "" && rt.EntityID != ""
}

// MarshalText implements encoding.TextMarshaler for use as map keys.
func (rt ReactionTarget) MarshalText() (text []byte, err error) {
	return []byte(rt.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for parsing from map keys.
func (rt *ReactionTarget) UnmarshalText(text []byte) error {
	// Parse format "entity_type:entity_id"
	s := string(text)
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			rt.EntityType = s[:i]
			rt.EntityID = s[i+1:]
			return nil
		}
	}
	return fmt.Errorf("invalid ReactionTarget format: %s", s)
}

// UserReactionKey identifies a unique user-entity reaction pair.
// It is an immutable value object representing the tuple (user_id, entity_type, entity_id).
// This can be used as a lookup key for finding a user's reaction on a specific entity.
type UserReactionKey struct {
	// UserID is the identifier of the user.
	UserID string `json:"user_id"`

	// EntityType is the type of entity.
	EntityType string `json:"entity_type"`

	// EntityID is the identifier of the entity.
	EntityID string `json:"entity_id"`
}

// NewUserReactionKey creates a new UserReactionKey with validation.
func NewUserReactionKey(userID, entityType, entityID string) (UserReactionKey, error) {
	if err := ValidateUserID(userID); err != nil {
		return UserReactionKey{}, err
	}

	if err := ValidateEntityType(entityType); err != nil {
		return UserReactionKey{}, err
	}

	if err := ValidateEntityID(entityID); err != nil {
		return UserReactionKey{}, err
	}

	return UserReactionKey{
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
	}, nil
}

// String returns a string representation in the form "user_id:entity_type:entity_id".
func (urk UserReactionKey) String() string {
	return fmt.Sprintf("%s:%s:%s", urk.UserID, urk.EntityType, urk.EntityID)
}

// ReactionKey returns the ReactionTarget portion of this key.
func (urk UserReactionKey) ReactionKey() ReactionTarget {
	return ReactionTarget{
		EntityType: urk.EntityType,
		EntityID:   urk.EntityID,
	}
}

// Validation functions.

// ValidateUserID validates a user ID.
func ValidateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("user_id is empty")
	}
	if len(userID) > MaxIDLength {
		return fmt.Errorf("user_id exceeds %d characters", MaxIDLength)
	}
	return nil
}

// ValidateEntityType validates an entity type.
func ValidateEntityType(entityType string) error {
	if entityType == "" {
		return fmt.Errorf("entity_type is empty")
	}
	if len(entityType) > MaxEntityTypeLength {
		return fmt.Errorf("entity_type exceeds %d characters", MaxEntityTypeLength)
	}
	if !entityTypeRegex.MatchString(entityType) {
		return fmt.Errorf("entity_type must match pattern %s", EntityTypePattern)
	}
	return nil
}

// ValidateEntityID validates an entity ID.
func ValidateEntityID(entityID string) error {
	if entityID == "" {
		return fmt.Errorf("entity_id is empty")
	}
	if len(entityID) > MaxIDLength {
		return fmt.Errorf("entity_id exceeds %d characters", MaxIDLength)
	}
	return nil
}

// ValidateReactionType validates a reaction type.
func ValidateReactionType(reactionType string) error {
	if reactionType == "" {
		return fmt.Errorf("reaction_type is empty")
	}
	if len(reactionType) > MaxReactionTypeLength {
		return fmt.Errorf("reaction_type exceeds %d characters", MaxReactionTypeLength)
	}
	if !reactionTypeRegex.MatchString(reactionType) {
		return fmt.Errorf("reaction_type must match pattern %s", ReactionTypePattern)
	}
	return nil
}

// ValidateUUID validates that a string is a valid UUID v4.
func ValidateUUID(id string) error {
	if id == "" {
		return fmt.Errorf("uuid is empty")
	}
	u, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid uuid: %w", err)
	}
	if u.Version() != 4 {
		return fmt.Errorf("uuid must be version 4, got version %d", u.Version())
	}
	return nil
}

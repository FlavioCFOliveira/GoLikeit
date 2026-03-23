// Package validation provides fuzz testing for validation functions.
//
//go:build gofuzz
// +build gofuzz

package validation

import (
	"testing"
)

// FuzzValidateUserID tests user ID validation with random inputs.
func FuzzValidateUserID(f *testing.F) {
	// Seed corpus
	f.Add("user-123")
	f.Add("user@example.com")
	f.Add("")
	f.Add("a")
	f.Add(string(make([]byte, 300))) // Long string (over 256 limit)
	f.Add("user\x00name")             // null byte
	f.Add("user\nname")               // newline

	f.Fuzz(func(t *testing.T, userID string) {
		// Validate should not panic
		_ = ValidateUserID(userID)
	})
}

// FuzzValidateEntityType tests entity type validation with random inputs.
func FuzzValidateEntityType(f *testing.F) {
	// Seed corpus
	f.Add("photo")
	f.Add("post")
	f.Add("comment")
	f.Add("")
	f.Add("a")
	f.Add("PHOTO")           // uppercase - should fail
	f.Add("photo123")        // alphanumeric
	f.Add("photo_123")       // underscore
	f.Add("photo 123")       // space - should fail
	f.Add("photo\n")         // newline - should fail
	f.Add("photo\x00")       // null byte - should fail
	f.Add(string([]byte{0xFF})) // invalid UTF-8

	f.Fuzz(func(t *testing.T, entityType string) {
		// Validate should not panic
		_ = ValidateEntityType(entityType)
	})
}

// FuzzValidateEntityID tests entity ID validation with random inputs.
func FuzzValidateEntityID(f *testing.F) {
	// Seed corpus
	f.Add("entity-123")
	f.Add("123456")
	f.Add("uuid-1234-5678")
	f.Add("")
	f.Add("a")
	f.Add(string(make([]byte, 1000))) // Very long string
	f.Add("entity\x00id")              // null byte
	f.Add("entity\nid")                // newline
	f.Add("entity\tid")                // tab

	f.Fuzz(func(t *testing.T, entityID string) {
		// Validate should not panic
		_ = ValidateEntityID(entityID)
	})
}

// FuzzValidateReactionType tests reaction type validation with random inputs.
func FuzzValidateReactionType(f *testing.F) {
	// Seed corpus with common reaction types
	f.Add("LIKE")
	f.Add("LOVE")
	f.Add("HAHA")
	f.Add("WOW")
	f.Add("SAD")
	f.Add("ANGRY")
	f.Add("")
	f.Add("A")
	f.Add(string(make([]byte, 100))) // Long string (over 64 limit)
	f.Add("like")                      // lowercase - should fail
	f.Add("LIKE LOVE")                 // space - should fail
	f.Add("LIKE\n")                    // newline - should fail
	f.Add("LIKE\x00")                  // null byte - should fail
	f.Add("LIKE❤️")                    // emoji - should fail
	f.Add("LIKE-\n\t_123")              // mixed

	f.Fuzz(func(t *testing.T, reactionType string) {
		// Validate should not panic
		_ = ValidateReactionType(reactionType)
	})
}

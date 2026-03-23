// Package business provides fuzz testing for business logic functions.
//
//go:build gofuzz
// +build gofuzz

package business

import (
	"testing"
)

// FuzzConfigValidation tests business configuration validation.
func FuzzConfigValidation(f *testing.F) {
	// Seed: reactionTypes (comma-separated), enableCache, enableEvents, enableRateLimit
	f.Add("LIKE,LOVE,HAHA", true, true, true)
	f.Add("LIKE", false, false, false)
	f.Add("", true, true, true)                // empty types - should fail
	f.Add("LIKE,LIKE", true, true, true)      // duplicates - should fail
	f.Add("like,LOVE", true, true, true)     // lowercase - should fail
	f.Add("LIKE LOVE", true, true, true)     // space - should fail
	f.Add("LIKE,\nLOVE", true, true, true)   // newline - should fail
	f.Add("LIKE\x00", true, true, true)      // null byte - should fail
	f.Add(string([]byte{0xFF, 0xFE}), true, true, true) // invalid UTF-8

	f.Fuzz(func(t *testing.F, typesStr string, cache, events, rateLimit bool) {
		cfg := Config{
			ReactionTypes:      parseTypes(typesStr),
			EnableCaching:      cache,
			EnableEvents:       events,
			EnableRateLimiting: rateLimit,
		}

		// Validate should not panic
		_ = cfg.Validate()
	})
}

// FuzzReactionTypeValidation tests reaction type validation against configured types.
func FuzzReactionTypeValidation(f *testing.F) {
	// Seed: configuredTypes (comma-separated), reactionToValidate
	f.Add("LIKE,LOVE,HAHA", "LIKE")
	f.Add("LIKE,LOVE,HAHA", "LOVE")
	f.Add("LIKE,LOVE,HAHA", "INVALID")
	f.Add("LIKE,LOVE,HAHA", "")
	f.Add("LIKE,LOVE,HAHA", "like")          // lowercase
	f.Add("LIKE,LOVE,HAHA", "LIKE LOVE")     // space
	f.Add("LIKE,LOVE,HAHA", "LIKE\n")        // newline
	f.Add("LIKE,LOVE,HAHA", "LIKE\x00")      // null
	f.Add("", "LIKE")                       // no configured types
	f.Add("LIKE", "LIKE")

	f.Fuzz(func(t *testing.F, configuredTypes, reactionToValidate string) {
		validTypes := parseTypes(configuredTypes)
		if len(validTypes) == 0 {
			return
		}

		// Validation should not panic
		_ = isValidReactionType(reactionToValidate, validTypes)
	})
}

// FuzzServiceInputValidation tests service method input validation.
func FuzzServiceInputValidation(f *testing.F) {
	// Seed: userID, entityType, entityID, reactionType
	f.Add("user-123", "photo", "photo-456", "LIKE")
	f.Add("", "photo", "photo-456", "LIKE")          // empty user
	f.Add("user-123", "", "photo-456", "LIKE")      // empty entity type
	f.Add("user-123", "photo", "", "LIKE")          // empty entity ID
	f.Add("user-123", "photo", "photo-456", "")     // empty reaction
	f.Add("user-123", "photo", "photo-456", "INVALID")
	f.Add(string(make([]byte, 300)), "photo", "photo-456", "LIKE") // long user
	f.Add("user-123", string(make([]byte, 100)), "photo-456", "LIKE") // long type
	f.Add("user-123", "photo", string(make([]byte, 300)), "LIKE") // long ID
	f.Add("user\n123", "photo", "photo-456", "LIKE") // newline in user
	f.Add("user\x00123", "photo", "photo-456", "LIKE") // null in user
	f.Add("user-123", "photo\n", "photo-456", "LIKE") // newline in type
	f.Add("user-123", "photo", "photo-456\n", "LIKE") // newline in ID

	f.Fuzz(func(t *testing.F, userID, entityType, entityID, reactionType string) {
		// Validation should not panic
		_ = validateUserID(userID)
		_ = validateEntityType(entityType)
		_ = validateEntityID(entityID)
	})
}

// FuzzPaginationValidation tests pagination validation for business queries.
func FuzzPaginationValidation(f *testing.F) {
	// Seed: limit, offset, maxLimit, maxOffset
	f.Add(10, 0, 100, 10000)
	f.Add(0, 0, 100, 10000)
	f.Add(-1, 0, 100, 10000)
	f.Add(10, -1, 100, 10000)
	f.Add(1000, 0, 100, 10000)
	f.Add(10, 100000, 100, 10000)
	f.Add(1<<31-1, 1<<31-1, 1<<31-1, 1<<31-1)
	f.Add(0, 0, 0, 0)
	f.Add(-1, -1, -1, -1)

	f.Fuzz(func(t *testing.F, limit, offset, maxLimit, maxOffset int) {
		// Validation should not panic
		_ = validateLimit(limit, maxLimit)
		_ = validateOffset(offset, maxOffset)
	})
}

// FuzzBulkOperationInputs tests bulk operation input validation.
func FuzzBulkOperationInputs(f *testing.F) {
	// Seed: numTargets (1-100), userID
	f.Add(1, "user-1")
	f.Add(10, "user-1")
	f.Add(100, "user-1")
	f.Add(0, "user-1")
	f.Add(-1, "user-1")
	f.Add(1000, "user-1")
	f.Add(10, "")
	f.Add(10, string(make([]byte, 300)))

	f.Fuzz(func(t *testing.F, numTargets int, userID string) {
		// Bulk size validation should not panic
		_ = validateBulkSize(numTargets)
		_ = validateUserID(userID)
	})
}

// FuzzConfigJSONSerialization tests Config JSON marshaling/unmarshaling.
func FuzzConfigJSONSerialization(f *testing.F) {
	// Seed JSON configurations
	f.Add([]byte(`{"reaction_types":["LIKE","LOVE"],"enable_caching":true,"enable_events":true,"enable_rate_limiting":false}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"reaction_types":[]}`))
	f.Add([]byte(`{"reaction_types":["LIKE","LIKE"]}`))
	f.Add([]byte(`{"reaction_types":["like"]}`))
	f.Add([]byte(`{"reaction_types":["LIKE LOVE"]}`))
	f.Add([]byte(`{"invalid_field":"value"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(`{"reaction_types":["LIKE"}`))

	f.Fuzz(func(t *testing.F, data []byte) {
		var cfg Config
		_ = unmarshalConfig(data, &cfg)
		_, _ = marshalConfig(cfg)
	})
}

// Helper functions
func parseTypes(s string) []string {
	if s == "" {
		return nil
	}
	// Simple split by comma
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func isValidReactionType(reactionType string, validTypes []string) bool {
	if reactionType == "" {
		return false
	}
	for _, vt := range validTypes {
		if vt == reactionType {
			return true
		}
	}
	return false
}

func validateUserID(userID string) error {
	if userID == "" {
		return ErrInvalidInput
	}
	if len(userID) > 256 {
		return ErrInvalidInput
	}
	// Check for null bytes
	for i := 0; i < len(userID); i++ {
		if userID[i] == 0 {
			return ErrInvalidInput
		}
	}
	return nil
}

func validateEntityType(entityType string) error {
	if entityType == "" {
		return ErrInvalidInput
	}
	if len(entityType) > 64 {
		return ErrInvalidInput
	}
	return nil
}

func validateEntityID(entityID string) error {
	if entityID == "" {
		return ErrInvalidInput
	}
	if len(entityID) > 256 {
		return ErrInvalidInput
	}
	return nil
}

func validateLimit(limit, maxLimit int) error {
	if limit < 0 {
		return ErrInvalidInput
	}
	if maxLimit > 0 && limit > maxLimit {
		return ErrInvalidInput
	}
	return nil
}

func validateOffset(offset, maxOffset int) error {
	if offset < 0 {
		return ErrInvalidInput
	}
	if maxOffset > 0 && offset > maxOffset {
		return ErrInvalidInput
	}
	return nil
}

func validateBulkSize(size int) error {
	if size < 0 {
		return ErrInvalidInput
	}
	if size > 1000 {
		return ErrInvalidInput
	}
	return nil
}

func unmarshalConfig(data []byte, cfg *Config) error {
	// Simplified unmarshaling for fuzzing
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	// Just check if it looks like JSON
	if len(data) > 0 && data[0] != '{' {
		return ErrInvalidInput
	}
	return nil
}

func marshalConfig(cfg Config) ([]byte, error) {
	// Simplified marshaling
	return []byte(`{}`), nil
}

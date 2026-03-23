// Package golikeit provides fuzz testing for domain types and functions.
//
//go:build gofuzz
// +build gofuzz

package golikeit

import (
	"encoding/json"
	"testing"
	"time"
)

// FuzzEntityTargetString tests EntityTarget.String() with random inputs.
func FuzzEntityTargetString(f *testing.F) {
	// Seed corpus
	f.Add("photo", "123")
	f.Add("post", "abc-def")
	f.Add("", "")
	f.Add("a", "b")
	f.Add(string([]byte{0}), "test") // null byte in type
	f.Add("type", string([]byte{0})) // null byte in ID
	f.Add("very-long-entity-type-name-that-exceeds-normal-limits", "very-long-entity-id-that-also-exceeds-normal-limits-123456789")

	f.Fuzz(func(t *testing.T, entityType, entityID string) {
		// Should not panic on any input
		target := EntityTarget{
			EntityType: entityType,
			EntityID:   entityID,
		}
		_ = target.String()
		_ = target.IsValid()
	})
}

// FuzzReactionTypeValidation tests reaction type validation with configured types.
func FuzzReactionTypeValidation(f *testing.F) {
	// Valid reaction types
	validTypes := []string{"LIKE", "LOVE", "HAHA", "WOW", "SAD", "ANGRY"}

	// Seed with valid types
	for _, rt := range validTypes {
		f.Add(rt)
	}
	// Seed with invalid types
	f.Add("")
	f.Add("like")          // lowercase
	f.Add("LIKE LOVE")     // space
	f.Add("LIKE\n")        // newline
	f.Add("LIKE\x00")      // null
	f.Add("❤️")            // emoji
	f.Add(string(make([]byte, 100))) // very long

	f.Fuzz(func(t *testing.T, reactionType string) {
		// Test if reaction type matches expected pattern
		_ = IsValidReactionFormat(reactionType)
	})
}

// FuzzUserReactionSerialization tests JSON marshaling/unmarshaling.
func FuzzUserReactionSerialization(f *testing.F) {
	// Seed with valid JSON
	f.Add([]byte(`{"user_id":"user1","entity_type":"photo","entity_id":"123","reaction_type":"LIKE","created_at":"2024-01-15T10:30:00Z","updated_at":"2024-01-15T10:30:00Z"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"user_id":""}`))
	f.Add([]byte(`{"reaction_type":"INVALID"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`invalid json`))
	f.Add([]byte(`{"extra_field":"value"}`))
	f.Add([]byte(`{"created_at":"invalid-date"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var ur UserReaction
		// Unmarshal should not panic
		_ = json.Unmarshal(data, &ur)

		// Marshal round-trip should not panic
		if ur.UserID != "" || ur.EntityType != "" {
			_, _ = json.Marshal(ur)
		}
	})
}

// FuzzEntityCountsSerialization tests EntityCounts JSON handling.
func FuzzEntityCountsSerialization(f *testing.F) {
	// Seed with valid JSON
	f.Add([]byte(`{"counts":{"LIKE":10,"LOVE":5},"total":15}`))
	f.Add([]byte(`{"counts":{},"total":0}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"counts":null,"total":0}`))
	f.Add([]byte(`{"total":-1}`))
	f.Add([]byte(`{"counts":{"LIKE":-5},"total":-5}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var ec EntityCounts
		// Unmarshal should not panic
		_ = json.Unmarshal(data, &ec)

		// Marshal should not panic
		_, _ = json.Marshal(ec)
	})
}

// FuzzPaginationValidation tests pagination parameter validation.
func FuzzPaginationValidation(f *testing.F) {
	// Seed: limit, offset
	f.Add(10, 0)
	f.Add(0, 0)
	f.Add(-1, 0)
	f.Add(10, -1)
	f.Add(100000, 0)
	f.Add(10, 1000000)

	f.Fuzz(func(t *testing.T, limit, offset int) {
		p := Pagination{Limit: limit, Offset: offset}
		// IsValid should not panic
		_ = p.IsEmpty()

		// Clone should not panic
		_ = p.Clone()

		// CalculateLimit/Offset should not panic
		_ = p.CalculateLimit(25)
		_ = p.CalculateOffset()
	})
}

// FuzzPaginatedResultSerialization tests PaginatedResult JSON handling.
func FuzzPaginatedResultSerialization(f *testing.F) {
	// Seed with various JSON structures
	f.Add([]byte(`{"items":[],"total":0,"total_pages":1,"current_page":1,"limit":25,"offset":0,"has_next":false,"has_prev":false}`))
	f.Add([]byte(`{"items":[{"user_id":"u1"}],"total":1,"total_pages":1,"current_page":1}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"items":null,"total":null}`))
	f.Add([]byte(`{"total":-1,"total_pages":-1}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test with UserReaction
		var result PaginatedResult[UserReaction]
		_ = json.Unmarshal(data, &result)
		_, _ = json.Marshal(result)

		// Test with EntityReaction
		var result2 PaginatedResult[EntityReaction]
		_ = json.Unmarshal(data, &result2)
		_, _ = json.Marshal(result2)
	})
}

// FuzzTimestampParsing tests timestamp parsing with random formats.
func FuzzTimestampParsing(f *testing.F) {
	// Seed with various timestamp formats
	f.Add("2024-01-15T10:30:00Z")
	f.Add("2024-01-15T10:30:00.123456789Z")
	f.Add("2024-01-15T10:30:00+00:00")
	f.Add("2024-01-15T10:30:00-05:30")
	f.Add("2024-01-15")
	f.Add("10:30:00")
	f.Add("")
	f.Add("invalid")
	f.Add("0000-00-00")
	f.Add("9999-99-99T99:99:99Z")
	f.Add(string([]byte{0x00}))
	f.Add("2024-01-15T10:30:00\n")
	f.Add("2024-01-15T10:30:00\x00extra")

	f.Fuzz(func(t *testing.T, ts string) {
		// Parse should not panic
		_, _ = time.Parse(time.RFC3339, ts)
		_, _ = time.Parse(time.RFC3339Nano, ts)
	})
}

// FuzzConfigValidation tests configuration validation.
func FuzzConfigValidation(f *testing.F) {
	// Seed: reactionTypes (JSON array), cacheEnabled, cacheTTL seconds
	f.Add([]byte(`["LIKE","LOVE"]`), true, 60)
	f.Add([]byte(`[]`), true, 60)                    // empty types
	f.Add([]byte(`["LIKE","LIKE"]`), true, 60)     // duplicates
	f.Add([]byte(`["like"]`), true, 60)            // lowercase
	f.Add([]byte(`["LIKE LOVE"]`), true, 60)       // space in type
	f.Add([]byte(`["LIKE"`), true, 60)             // invalid JSON
	f.Add([]byte(`null`), true, 60)                 // null
	f.Add([]byte(`["LIKE"]`), false, -1)          // negative TTL
	f.Add([]byte(`["LIKE"]`), true, 0)             // zero TTL

	f.Fuzz(func(t *testing.F, typesJSON []byte, cacheEnabled bool, cacheTTL int) {
		var types []string
		_ = json.Unmarshal(typesJSON, &types)

		cfg := Config{
			ReactionTypes: types,
			Cache: CacheConfig{
				Enabled: cacheEnabled,
			},
		}

		// Config creation should not panic
		_, _ = New(cfg)
	})
}

// IsValidReactionFormat checks if a reaction type matches the expected format.
func IsValidReactionFormat(reactionType string) bool {
	if reactionType == "" || len(reactionType) > 64 {
		return false
	}
	for _, r := range reactionType {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// IsEmpty returns true if pagination has zero values.
func (p Pagination) IsEmpty() bool {
	return p.Limit == 0 && p.Offset == 0
}

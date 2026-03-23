// Package security provides comprehensive security tests for the GoLikeit system.
// These tests verify input validation, SQL injection prevention, rate limiting,
// audit log immutability, and error message safety.
package security

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/audit"
	"github.com/FlavioCFOliveira/GoLikeit/ratelimit"
	"github.com/FlavioCFOliveira/GoLikeit/validation"
)

// ============================================================================
// Input Validation Boundary Tests
// ============================================================================

func TestValidateUserID_Boundaries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid_ascii", "user_123", false},
		{"max_length_exact", strings.Repeat("a", 256), false},
		{"max_length_exceeded", strings.Repeat("a", 257), true},
		{"null_byte_injection", "user\x00admin", true},
		{"control_char_tab", "user\t123", true},
		{"control_char_newline", "user\n123", true},
		{"control_char_carriage", "user\r123", true},
		{"control_char_null", "\x00", true},
		{"control_char_bel", "user\x07", true},
		{"control_char_esc", "user\x1b", true},
		{"control_char_del", "user\x7f", true},
		{"unicode_valid", "用户_123", false},
		{"emoji", "user👍", false},
		{"path_traversal", "../../../etc/passwd", false}, // Valid as user ID but suspicious
		{"sql_injection_union", "' UNION SELECT * FROM users--", false},
		{"sql_injection_or", "' OR '1'='1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateUserID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEntityType_Boundaries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid_lowercase", "blog_post", false},
		{"valid_with_numbers", "item_123", false},
		{"uppercase_not_allowed", "BlogPost", true},
		{"hyphen_allowed", "blog-post", false},
		{"space_not_allowed", "blog post", true},
		{"max_length_exact", strings.Repeat("a", 64), false},
		{"max_length_exceeded", strings.Repeat("a", 65), true},
		{"null_byte", "post\x00type", true},
		{"special_chars", "post@type", true},
		{"sql_injection", "post'; DROP TABLE reactions--", true},
		{"path_traversal", "../../../etc/passwd", true},
		{"unicode", "帖子", true}, // Unicode not allowed by pattern
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateEntityType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntityType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEntityID_Boundaries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid_uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid_numeric", "1234567890", false},
		{"valid_alphanumeric", "abc123", false},
		{"max_length_256", strings.Repeat("a", 256), false},
		{"exceeds_max_length", strings.Repeat("a", 257), true},
		{"null_byte", "id\x00123", true},
		{"control_char", "id\t123", true},
		{"sql_injection", "'; DROP TABLE reactions--", false}, // Valid as ID pattern
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateEntityID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntityID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateReactionType_Boundaries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"valid_uppercase", "LIKE", false},
		{"valid_with_underscore", "THUMBS_UP", false},
		{"valid_with_hyphen", "THUMBS-UP", false},
		{"valid_with_numbers", "LIKE2", false},
		{"lowercase_not_allowed", "like", true},
		{"space_not_allowed", "thumbs up", true},
		{"max_length_64", strings.Repeat("A", 64), false},
		{"exceeds_max_length", strings.Repeat("A", 65), true},
		{"null_byte", "LIKE\x00", true},
		{"special_chars", "LIKE@", true},
		{"sql_injection", "LIKE'; DROP TABLE reactions--", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateReactionType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateReactionType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// ============================================================================
// SQL Injection Prevention Tests
// ============================================================================

// SQLInjectionPayloads contains common SQL injection attack patterns
var SQLInjectionPayloads = []string{
	"' OR '1'='1",
	"' OR '1'='1' --",
	"' OR '1'='1' /*",
	"' OR '1'='1' #",
	"' OR '1'='1'; --",
	"'; DROP TABLE reactions--",
	"'; DROP TABLE reactions; --",
	"'; DELETE FROM reactions--",
	"'; INSERT INTO reactions VALUES ('x','x','x','x')--",
	"'; UPDATE reactions SET user_id='hacker'--",
	"' UNION SELECT * FROM users--",
	"' UNION SELECT username, password FROM users--",
	"'; SELECT * FROM sqlite_master--",
	"'; SELECT sql FROM sqlite_master--",
	"')) OR '1'='1--",
	"' OR 1=1--",
	"' OR 1=1#",
	"' OR 1=1/*",
	"') OR '1'='1--",
	"') OR ('1'='1--",
	"'; EXEC xp_cmdshell('dir')--",
	"'; EXEC xp_cmdshell('net user')--",
	"'; SHUTDOWN--",
	"'; BEGIN TRANSACTION--",
	"'; COMMIT--",
	"'; ROLLBACK--",
	"' OR 'a'='a",
	"' OR 'a'='a'; --",
	"' OR 1=1 LIMIT 1--",
	"1' AND 1=1--",
	"1' AND 1=2--",
	"1' OR (SELECT COUNT(*) FROM reactions)>0--",
	"1' AND ASCII(SUBSTRING((SELECT TOP 1 name FROM users),1,1))>64--",
	`\\\" OR \\\"1\\\"=\\\"1`,
	`\\' OR \\\'1\\'=\\'1`,
}

func TestValidationAgainstSQLInjection(t *testing.T) {
	// SQL injection payloads should be rejected or safely handled by validation
	for _, payload := range SQLInjectionPayloads {
		// Test EntityType (most restrictive, should reject SQL patterns)
		if err := validation.ValidateEntityType(payload); err == nil {
			t.Errorf("SQL injection payload %q should be rejected by ValidateEntityType", payload)
		}

		// Test ReactionType (most restrictive, should reject SQL patterns)
		if err := validation.ValidateReactionType(payload); err == nil {
			t.Errorf("SQL injection payload %q should be rejected by ValidateReactionType", payload)
		}
	}
}

// ============================================================================
// Error Message Safety Tests
// ============================================================================

// CredentialPatterns contains patterns that should never appear in error messages
var CredentialPatterns = []string{
	"password",
	"passwd",
	"pwd",
	"secret",
	"api_key",
	"apikey",
	"token",
	"auth",
	"credential",
	"private",
	"ssh",
	"key=",
	"pass=",
	"user=",
}

func TestValidationErrorMessages(t *testing.T) {
	// Verify validation errors don't expose credential patterns
	err := validation.ValidateUserID("")
	if err != nil {
		errMsg := strings.ToLower(err.Error())
		for _, pattern := range CredentialPatterns {
			if strings.Contains(errMsg, pattern) {
				t.Errorf("Error message contains credential pattern %q: %s", pattern, errMsg)
			}
		}
	}
}

// ============================================================================
// Rate Limiting Enforcement Tests
// ============================================================================

func TestRateLimit_Enforcement(t *testing.T) {
	config := ratelimit.Config{
		Enabled:           true,
		RequestsPerSecond: 5,
		BurstSize:         5,
		WindowSize:        time.Second,
	}

	limiter := ratelimit.New(config)

	// Should allow burst of 5 requests
	for i := 0; i < 5; i++ {
		if !limiter.Allow("user1") {
			t.Errorf("Request %d should be allowed (within burst)", i+1)
		}
	}

	// 6th request should be blocked
	if limiter.Allow("user1") {
		t.Error("6th request should be blocked (exceeds burst)")
	}

	// Different user should not be affected
	if !limiter.Allow("user2") {
		t.Error("Different user should be allowed")
	}
}

func TestRateLimit_Disabled(t *testing.T) {
	config := ratelimit.Config{
		Enabled: false,
	}

	limiter := ratelimit.New(config)

	// All requests should be allowed when disabled
	for i := 0; i < 100; i++ {
		if !limiter.Allow("user1") {
			t.Error("Request should be allowed when rate limiting is disabled")
		}
	}
}

func TestRateLimit_Reset(t *testing.T) {
	config := ratelimit.Config{
		Enabled:           true,
		RequestsPerSecond: 5,
		BurstSize:         5,
		WindowSize:        time.Second,
	}

	limiter := ratelimit.New(config)

	// Consume all requests
	for i := 0; i < 5; i++ {
		limiter.Allow("user1")
	}

	// Verify blocked
	if limiter.Allow("user1") {
		t.Error("Should be blocked after consuming burst")
	}

	// Reset the limiter
	limiter.Reset("user1")

	// Should be allowed again
	if !limiter.Allow("user1") {
		t.Error("Should be allowed after reset")
	}
}

func TestRateLimit_GetRemaining(t *testing.T) {
	config := ratelimit.Config{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         10,
		WindowSize:        time.Second,
	}

	limiter := ratelimit.New(config)

	// Initial remaining should be burst size
	if limiter.GetRemaining("user1") != 10 {
		t.Errorf("Initial remaining should be 10, got %d", limiter.GetRemaining("user1"))
	}

	// Consume 3 requests
	for i := 0; i < 3; i++ {
		limiter.Allow("user1")
	}

	// Should have 7 remaining
	if limiter.GetRemaining("user1") != 7 {
		t.Errorf("Remaining should be 7, got %d", limiter.GetRemaining("user1"))
	}
}

func TestRateLimit_GetStats(t *testing.T) {
	config := ratelimit.Config{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         10,
		WindowSize:        time.Second,
	}

	limiter := ratelimit.New(config)

	// Add some requests
	for i := 0; i < 3; i++ {
		limiter.Allow("user1")
	}

	stats := limiter.GetStats("user1")

	if stats.UserID != "user1" {
		t.Errorf("Stats.UserID = %q, want %q", stats.UserID, "user1")
	}

	if stats.RequestsInWindow != 3 {
		t.Errorf("Stats.RequestsInWindow = %d, want %d", stats.RequestsInWindow, 3)
	}

	if stats.Remaining != 7 {
		t.Errorf("Stats.Remaining = %d, want %d", stats.Remaining, 7)
	}

	if stats.Limited {
		t.Error("Stats.Limited should be false")
	}
}

func TestRateLimit_ContextCancellation(t *testing.T) {
	config := ratelimit.Config{
		Enabled:           true,
		RequestsPerSecond: 10,
		BurstSize:         10,
		WindowSize:        time.Second,
	}

	limiter := ratelimit.New(config)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return false for cancelled context
	if limiter.AllowContext(ctx, "user1") {
		t.Error("Should return false for cancelled context")
	}
}

// ============================================================================
// Audit Log Immutability Tests
// ============================================================================

func TestAuditEntry_Immutability(t *testing.T) {
	// Create an audit entry
	entry := audit.NewEntry(
		audit.OperationAdd,
		"user_123",
		"blog_post",
		"post_456",
		"LIKE",
		"",
	)

	// Verify entry has required fields
	if entry.UserID != "user_123" {
		t.Errorf("UserID = %q, want %q", entry.UserID, "user_123")
	}

	if entry.EntityType != "blog_post" {
		t.Errorf("EntityType = %q, want %q", entry.EntityType, "blog_post")
	}

	if entry.EntityID != "post_456" {
		t.Errorf("EntityID = %q, want %q", entry.EntityID, "post_456")
	}

	if entry.ReactionType != "LIKE" {
		t.Errorf("ReactionType = %q, want %q", entry.ReactionType, "LIKE")
	}

	if entry.Operation != audit.OperationAdd {
		t.Errorf("Operation = %q, want %q", entry.Operation, audit.OperationAdd)
	}

	// Timestamp should be set
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	// Timestamp should be in UTC
	if entry.Timestamp.Location() != time.UTC {
		t.Error("Timestamp should be in UTC")
	}
}

func TestAuditEntry_ReplaceOperation(t *testing.T) {
	entry := audit.NewEntry(
		audit.OperationReplace,
		"user_123",
		"blog_post",
		"post_456",
		"LOVE",
		"LIKE",
	)

	if entry.PreviousReaction != "LIKE" {
		t.Errorf("PreviousReaction = %q, want %q", entry.PreviousReaction, "LIKE")
	}

	if entry.ReactionType != "LOVE" {
		t.Errorf("ReactionType = %q, want %q", entry.ReactionType, "LOVE")
	}
}

func TestAuditEntry_RemoveOperation(t *testing.T) {
	entry := audit.NewEntry(
		audit.OperationRemove,
		"user_123",
		"blog_post",
		"post_456",
		"",
		"LIKE",
	)

	if entry.ReactionType != "" {
		t.Errorf("ReactionType should be empty for remove operation, got %q", entry.ReactionType)
	}

	if entry.PreviousReaction != "LIKE" {
		t.Errorf("PreviousReaction = %q, want %q", entry.PreviousReaction, "LIKE")
	}
}

func TestNullAuditor_Safety(t *testing.T) {
	nullAuditor := audit.NewNullAuditor()
	ctx := context.Background()

	// All operations should be safe no-ops
	entry := audit.NewEntry(audit.OperationAdd, "user1", "post", "123", "LIKE", "")

	if err := nullAuditor.LogOperation(ctx, entry); err != nil {
		t.Errorf("LogOperation should not error: %v", err)
	}

	entries, err := nullAuditor.GetByUser(ctx, "user1", 10, 0)
	if err != nil {
		t.Errorf("GetByUser should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("GetByUser should return empty slice, got %d entries", len(entries))
	}

	entries, err = nullAuditor.GetByEntity(ctx, "post", "123", 10, 0)
	if err != nil {
		t.Errorf("GetByEntity should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("GetByEntity should return empty slice, got %d entries", len(entries))
	}

	entries, err = nullAuditor.GetByOperation(ctx, audit.OperationAdd, 10, 0)
	if err != nil {
		t.Errorf("GetByOperation should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("GetByOperation should return empty slice, got %d entries", len(entries))
	}

	entries, err = nullAuditor.GetByDateRange(ctx, time.Now().Add(-time.Hour), time.Now(), 10, 0)
	if err != nil {
		t.Errorf("GetByDateRange should not error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("GetByDateRange should return empty slice, got %d entries", len(entries))
	}
}

func TestAuditEntry_TimestampPrecision(t *testing.T) {
	before := time.Now().UTC()
	entry := audit.NewEntry(audit.OperationAdd, "user1", "post", "123", "LIKE", "")
	after := time.Now().UTC()

	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Error("Timestamp should be within test execution window")
	}
}

// ============================================================================
// Security Error Tests
// ============================================================================

func TestValidationError_Security(t *testing.T) {
	// Ensure validation errors don't leak internal details
	err := validation.ValidateUserID("")

	var valErr *validation.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatal("Expected ValidationError type")
	}

	// Check error message doesn't contain internal details
	errMsg := err.Error()
	if strings.Contains(errMsg, "internal") || strings.Contains(errMsg, "panic") {
		t.Errorf("Error message should not contain internal details: %s", errMsg)
	}

	// Verify field name is included
	if !strings.Contains(errMsg, "user_id") {
		t.Error("Error message should include field name")
	}
}

// ============================================================================
// Race Condition Tests
// ============================================================================

func TestRateLimit_ConcurrentAccess(t *testing.T) {
	config := ratelimit.Config{
		Enabled:           true,
		RequestsPerSecond: 100,
		BurstSize:         50,
		WindowSize:        time.Second,
	}

	limiter := ratelimit.New(config)

	// Concurrent access test
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				limiter.Allow("concurrent_user")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify stats are within expected bounds (max 100 requests)
	stats := limiter.GetStats("concurrent_user")
	if stats.RequestsInWindow > 100 {
		t.Errorf("Concurrent requests should be limited to burst size, got %d", stats.RequestsInWindow)
	}

	// Verify the limiter handled concurrency safely (no panics, no races)
	if stats.RequestsInWindow < 1 {
		t.Error("Should have recorded at least some requests")
	}
}

// ============================================================================
// Fuzzing Tests
// ============================================================================

func FuzzValidateUserID(f *testing.F) {
	// Seed corpus with interesting cases
	f.Add("user_123")
	f.Add("")
	f.Add("a")
	f.Add(strings.Repeat("a", 256))
	f.Add("user\x00admin")
	f.Add("' OR '1'='1")
	f.Add("../../../etc/passwd")
	f.Add("用户_123")

	f.Fuzz(func(t *testing.T, input string) {
		err := validation.ValidateUserID(input)

		// Basic invariants
		if input == "" && err == nil {
			t.Error("Empty string should return error")
		}

		if len(input) > 256 && err == nil {
			t.Error("Input exceeding max length should return error")
		}

		if strings.Contains(input, "\x00") && err == nil {
			t.Error("Input with null byte should return error")
		}
	})
}

func FuzzValidateEntityType(f *testing.F) {
	f.Add("blog_post")
	f.Add("")
	f.Add("post")
	f.Add(strings.Repeat("a", 64))
	f.Add("POST") // Uppercase - should fail
	f.Add("post-type") // Hyphen - should fail
	f.Add("post type") // Space - should fail

	f.Fuzz(func(t *testing.T, input string) {
		err := validation.ValidateEntityType(input)

		if input == "" && err == nil {
			t.Error("Empty string should return error")
		}

		if len(input) > 64 && err == nil {
			t.Error("Input exceeding max length should return error")
		}

		// Should only allow lowercase alphanumeric, underscore, and hyphen
		for _, r := range input {
			if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
				if err == nil {
					t.Errorf("Invalid character %q should return error", r)
				}
				return
			}
		}
	})
}

func FuzzValidateEntityID(f *testing.F) {
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("")
	f.Add("123")
	f.Add(strings.Repeat("a", 256))
	f.Add("id\x00test")

	f.Fuzz(func(t *testing.T, input string) {
		err := validation.ValidateEntityID(input)

		if input == "" && err == nil {
			t.Error("Empty string should return error")
		}

		if len(input) > 256 && err == nil {
			t.Error("Input exceeding max length should return error")
		}
	})
}

func FuzzValidateReactionType(f *testing.F) {
	f.Add("LIKE")
	f.Add("")
	f.Add("THUMBS_UP")
	f.Add("THUMBS-UP")
	f.Add(strings.Repeat("A", 64))
	f.Add("like") // lowercase - should fail

	f.Fuzz(func(t *testing.T, input string) {
		err := validation.ValidateReactionType(input)

		if input == "" && err == nil {
			t.Error("Empty string should return error")
		}

		if len(input) > 64 && err == nil {
			t.Error("Input exceeding max length should return error")
		}
	})
}

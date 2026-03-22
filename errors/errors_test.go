// Package errors provides domain-specific error types for the GoLikeit reaction system.
package errors

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrInvalidInput",
			err:  ErrInvalidInput,
			want: "invalid input provided",
		},
		{
			name: "ErrReactionNotFound",
			err:  ErrReactionNotFound,
			want: "reaction not found",
		},
		{
			name: "ErrStorageUnavailable",
			err:  ErrStorageUnavailable,
			want: "storage backend unavailable",
		},
		{
			name: "ErrInvalidReactionType",
			err:  ErrInvalidReactionType,
			want: "invalid reaction type",
		},
		{
			name: "ErrInvalidReactionFormat",
			err:  ErrInvalidReactionFormat,
			want: "invalid reaction type format",
		},
		{
			name: "ErrNoReactionTypes",
			err:  ErrNoReactionTypes,
			want: "no reaction types configured",
		},
		{
			name: "ErrDuplicateReactionType",
			err:  ErrDuplicateReactionType,
			want: "duplicate reaction type found",
		},
		{
			name: "ErrRateLimitExceeded",
			err:  ErrRateLimitExceeded,
			want: "rate limit exceeded",
		},
		{
			name: "ErrCacheUnavailable",
			err:  ErrCacheUnavailable,
			want: "cache unavailable",
		},
		{
			name: "ErrClientClosed",
			err:  ErrClientClosed,
			want: "client is closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("sentinel error is nil")
				return
			}
			if tt.err.Error() != tt.want {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.want)
			}
		})
	}
}

func TestInputError(t *testing.T) {
	t.Run("Error without cause", func(t *testing.T) {
		err := NewInputError("user_id", "test-value", "cannot be empty")

		want := `invalid input for field "user_id": cannot be empty`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		if err.Field != "user_id" {
			t.Errorf("Field = %q, want %q", err.Field, "user_id")
		}
		if err.Value != "test-value" {
			t.Errorf("Value = %q, want %q", err.Value, "test-value")
		}
		if err.Reason != "cannot be empty" {
			t.Errorf("Reason = %q, want %q", err.Reason, "cannot be empty")
		}
	})

	t.Run("Error with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewInputErrorWithCause("entity_id", "123", "invalid format", cause)

		want := `invalid input for field "entity_id": invalid format (cause: underlying error)`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		if err.Cause != cause {
			t.Error("Cause should be set to the underlying error")
		}
	})

	t.Run("Unwrap without cause", func(t *testing.T) {
		err := NewInputError("user_id", "", "cannot be empty")
		unwrapped := err.Unwrap()

		if !errors.Is(unwrapped, ErrInvalidInput) {
			t.Error("Unwrap() should return ErrInvalidInput when no cause is set")
		}
	})

	t.Run("Unwrap with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewInputErrorWithCause("user_id", "", "invalid", cause)
		unwrapped := err.Unwrap()

		if unwrapped != cause {
			t.Error("Unwrap() should return the underlying cause")
		}
	})

	t.Run("Value truncation", func(t *testing.T) {
		longValue := strings.Repeat("a", 100)
		err := NewInputError("field", longValue, "too long")

		if len(err.Value) != 67 { // 64 + "..."
			t.Errorf("Value should be truncated to 67 chars, got %d", len(err.Value))
		}
		if !strings.HasSuffix(err.Value, "...") {
			t.Error("Value should end with ...")
		}
	})
}

func TestNotFoundError(t *testing.T) {
	t.Run("Error formatting", func(t *testing.T) {
		err := NewNotFoundError("user-123", "photo", "456")

		want := `reaction not found for user "user-123" on photo:456`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Field values", func(t *testing.T) {
		err := NewNotFoundError("user-123", "photo", "456")

		if err.UserID != "user-123" {
			t.Errorf("UserID = %q, want %q", err.UserID, "user-123")
		}
		if err.EntityType != "photo" {
			t.Errorf("EntityType = %q, want %q", err.EntityType, "photo")
		}
		if err.EntityID != "456" {
			t.Errorf("EntityID = %q, want %q", err.EntityID, "456")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		err := NewNotFoundError("user-123", "photo", "456")
		unwrapped := err.Unwrap()

		if unwrapped != ErrReactionNotFound {
			t.Error("Unwrap() should return ErrReactionNotFound")
		}
	})

	t.Run("errors.Is", func(t *testing.T) {
		err := NewNotFoundError("user-123", "photo", "456")

		if !errors.Is(err, ErrReactionNotFound) {
			t.Error("errors.Is should match ErrReactionNotFound")
		}
	})
}

func TestStorageError(t *testing.T) {
	t.Run("Error with cause", func(t *testing.T) {
		cause := errors.New("connection refused")
		err := NewStorageError("create", cause)

		want := `storage operation "create" failed: connection refused`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error without cause", func(t *testing.T) {
		err := NewStorageError("read", nil)

		want := `storage operation "read" failed: <nil>`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Field values", func(t *testing.T) {
		cause := errors.New("timeout")
		err := NewStorageError("update", cause)

		if err.Operation != "update" {
			t.Errorf("Operation = %q, want %q", err.Operation, "update")
		}
		if err.Cause != cause {
			t.Error("Cause should be set correctly")
		}
	})

	t.Run("Unwrap with cause", func(t *testing.T) {
		cause := errors.New("timeout")
		err := NewStorageError("update", cause)
		unwrapped := err.Unwrap()

		if unwrapped != cause {
			t.Error("Unwrap() should return the underlying cause")
		}
	})

	t.Run("Unwrap without cause", func(t *testing.T) {
		err := NewStorageError("delete", nil)
		unwrapped := err.Unwrap()

		if unwrapped != ErrStorageUnavailable {
			t.Error("Unwrap() should return ErrStorageUnavailable when no cause")
		}
	})

	t.Run("errors.Is with cause", func(t *testing.T) {
		cause := errors.New("timeout")
		err := NewStorageError("read", cause)

		if !errors.Is(err, cause) {
			t.Error("errors.Is should match the underlying cause")
		}
	})
}

func TestRateLimitError(t *testing.T) {
	t.Run("Error with retry after", func(t *testing.T) {
		err := NewRateLimitError("add_reaction", 5*time.Second)

		want := `rate limit exceeded for operation "add_reaction", retry after 5s`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Error without retry after", func(t *testing.T) {
		err := NewRateLimitError("remove_reaction", 0)

		want := `rate limit exceeded for operation "remove_reaction"`
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
	})

	t.Run("Field values", func(t *testing.T) {
		err := NewRateLimitError("get_counts", 10*time.Second)

		if err.Operation != "get_counts" {
			t.Errorf("Operation = %q, want %q", err.Operation, "get_counts")
		}
		if err.RetryAfter != 10*time.Second {
			t.Errorf("RetryAfter = %v, want %v", err.RetryAfter, 10*time.Second)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		err := NewRateLimitError("add", time.Second)
		unwrapped := err.Unwrap()

		if unwrapped != ErrRateLimitExceeded {
			t.Error("Unwrap() should return ErrRateLimitExceeded")
		}
	})

	t.Run("errors.Is", func(t *testing.T) {
		err := NewRateLimitError("add", time.Second)

		if !errors.Is(err, ErrRateLimitExceeded) {
			t.Error("errors.Is should match ErrRateLimitExceeded")
		}
	})
}

func TestIsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "InputError",
			err:  NewInputError("field", "value", "reason"),
			want: true,
		},
		{
			name: "ErrInvalidInput directly",
			err:  ErrInvalidInput,
			want: true,
		},
		{
			name: "wrapped InputError",
			err:  errors.New("wrapped: " + NewInputError("field", "value", "reason").Error()),
			want: false, // Not wrapped properly
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "different error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInvalidInput(tt.err)
			if got != tt.want {
				t.Errorf("IsInvalidInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "NotFoundError",
			err:  NewNotFoundError("user", "photo", "123"),
			want: true,
		},
		{
			name: "ErrReactionNotFound directly",
			err:  ErrReactionNotFound,
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "different error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsStorageUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "StorageError wrapping ErrStorageUnavailable",
			err:  NewStorageError("read", ErrStorageUnavailable),
			want: true,
		},
		{
			name: "ErrStorageUnavailable directly",
			err:  ErrStorageUnavailable,
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "different error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsStorageUnavailable(tt.err)
			if got != tt.want {
				t.Errorf("IsStorageUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInvalidReactionType(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrInvalidReactionType directly",
			err:  ErrInvalidReactionType,
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "different error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInvalidReactionType(tt.err)
			if got != tt.want {
				t.Errorf("IsInvalidReactionType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRateLimitExceeded(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "RateLimitError",
			err:  NewRateLimitError("add", time.Second),
			want: true,
		},
		{
			name: "ErrRateLimitExceeded directly",
			err:  ErrRateLimitExceeded,
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "different error",
			err:  errors.New("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimitExceeded(tt.err)
			if got != tt.want {
				t.Errorf("IsRateLimitExceeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			s:      "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string",
			s:      "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "empty string",
			s:      "",
			maxLen: 5,
			want:   "",
		},
		{
			name:   "maxLen 0",
			s:      "hello",
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	t.Run("InputError wraps ErrInvalidInput", func(t *testing.T) {
		err := NewInputError("field", "value", "reason")

		if !errors.Is(err, ErrInvalidInput) {
			t.Error("InputError should wrap ErrInvalidInput")
		}
	})

	t.Run("NotFoundError wraps ErrReactionNotFound", func(t *testing.T) {
		err := NewNotFoundError("user", "photo", "123")

		if !errors.Is(err, ErrReactionNotFound) {
			t.Error("NotFoundError should wrap ErrReactionNotFound")
		}
	})

	t.Run("StorageError with cause", func(t *testing.T) {
		cause := errors.New("timeout")
		err := NewStorageError("read", cause)

		if !errors.Is(err, cause) {
			t.Error("StorageError should be the underlying cause")
		}
	})

	t.Run("StorageError without cause wraps ErrStorageUnavailable", func(t *testing.T) {
		err := NewStorageError("read", nil)

		if !errors.Is(err, ErrStorageUnavailable) {
			t.Error("StorageError without cause should wrap ErrStorageUnavailable")
		}
	})

	t.Run("RateLimitError wraps ErrRateLimitExceeded", func(t *testing.T) {
		err := NewRateLimitError("add", time.Second)

		if !errors.Is(err, ErrRateLimitExceeded) {
			t.Error("RateLimitError should wrap ErrRateLimitExceeded")
		}
	})
}

func TestErrorAs(t *testing.T) {
	t.Run("errors.As with InputError", func(t *testing.T) {
		err := NewInputError("user_id", "value", "reason")
		var inputErr *InputError

		if !errors.As(err, &inputErr) {
			t.Error("errors.As should work with InputError")
		}
		if inputErr.Field != "user_id" {
			t.Errorf("Field = %q, want %q", inputErr.Field, "user_id")
		}
	})

	t.Run("errors.As with NotFoundError", func(t *testing.T) {
		err := NewNotFoundError("user", "photo", "123")
		var notFoundErr *NotFoundError

		if !errors.As(err, &notFoundErr) {
			t.Error("errors.As should work with NotFoundError")
		}
		if notFoundErr.UserID != "user" {
			t.Errorf("UserID = %q, want %q", notFoundErr.UserID, "user")
		}
	})

	t.Run("errors.As with StorageError", func(t *testing.T) {
		err := NewStorageError("create", errors.New("timeout"))
		var storageErr *StorageError

		if !errors.As(err, &storageErr) {
			t.Error("errors.As should work with StorageError")
		}
		if storageErr.Operation != "create" {
			t.Errorf("Operation = %q, want %q", storageErr.Operation, "create")
		}
	})

	t.Run("errors.As with RateLimitError", func(t *testing.T) {
		err := NewRateLimitError("add", 5*time.Second)
		var rateLimitErr *RateLimitError

		if !errors.As(err, &rateLimitErr) {
			t.Error("errors.As should work with RateLimitError")
		}
		if rateLimitErr.Operation != "add" {
			t.Errorf("Operation = %q, want %q", rateLimitErr.Operation, "add")
		}
	})
}

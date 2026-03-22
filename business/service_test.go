// Package business provides business layer functionality for the GoLikeit reaction system.
package business

import (
	"errors"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errType error
	}{
		{
			name: "valid config with single type",
			config: Config{
				ReactionTypes: []string{"LIKE"},
			},
			wantErr: false,
		},
		{
			name: "valid config with multiple types",
			config: Config{
				ReactionTypes:      []string{"LIKE", "LOVE", "DISLIKE"},
				EnableCaching:      true,
				EnableEvents:       true,
				EnableRateLimiting: true,
			},
			wantErr: false,
		},
		{
			name: "empty reaction types",
			config: Config{
				ReactionTypes: []string{},
			},
			wantErr: true,
			errType: ErrNoReactionTypes,
		},
		{
			name: "nil reaction types",
			config: Config{
				ReactionTypes: nil,
			},
			wantErr: true,
			errType: ErrNoReactionTypes,
		},
		{
			name: "empty reaction type string",
			config: Config{
				ReactionTypes: []string{"LIKE", ""},
			},
			wantErr: true,
			errType: ErrInvalidReactionType,
		},
		{
			name: "duplicate reaction types",
			config: Config{
				ReactionTypes: []string{"LIKE", "LOVE", "LIKE"},
			},
			wantErr: true,
			errType: ErrDuplicateReactionType,
		},
		{
			name: "case sensitive duplicates",
			config: Config{
				ReactionTypes: []string{"LIKE", "like"},
			},
			wantErr: false, // "LIKE" and "like" are different strings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("Validate() error = %v, should contain %v", err, tt.errType)
				}
			}
		})
	}
}

func TestServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  *ServiceError
		want string
	}{
		{
			name: "error with underlying error",
			err: &ServiceError{
				Op:  "create",
				Err: errors.New("database connection failed"),
			},
			want: "create: database connection failed",
		},
		{
			name: "config error",
			err: &ServiceError{
				Op:  "config",
				Err: errors.New("no reaction types"),
			},
			want: "config: no reaction types",
		},
		{
			name: "storage error",
			err: &ServiceError{
				Op:  "storage",
				Err: errors.New("timeout"),
			},
			want: "storage: timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	t.Run("unwrap with underlying error", func(t *testing.T) {
		underlying := errors.New("connection refused")
		err := &ServiceError{
			Op:  "create",
			Err: underlying,
		}

		unwrapped := err.Unwrap()
		if unwrapped != underlying {
			t.Error("Unwrap() should return the underlying error")
		}
	})

	t.Run("unwrap without underlying error", func(t *testing.T) {
		err := &ServiceError{
			Op:  "create",
			Err: nil,
		}

		unwrapped := err.Unwrap()
		if unwrapped != nil {
			t.Error("Unwrap() should return nil when no underlying error")
		}
	})
}

func TestBusinessError(t *testing.T) {
	t.Run("NewBusinessError creates error", func(t *testing.T) {
		err := NewBusinessError("something went wrong")
		if err == nil {
			t.Fatal("NewBusinessError should not return nil")
		}
		if err.Error() != "something went wrong" {
			t.Errorf("Error() = %q, want %q", err.Error(), "something went wrong")
		}
	})

	t.Run("BusinessError implements error interface", func(t *testing.T) {
		var err error = NewBusinessError("test error")
		if err.Error() != "test error" {
			t.Errorf("Error() = %q, want %q", err.Error(), "test error")
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNoReactionTypes",
			err:      ErrNoReactionTypes,
			expected: "config: no reaction types configured",
		},
		{
			name:     "ErrInvalidReactionType",
			err:      ErrInvalidReactionType,
			expected: "validate: invalid reaction type",
		},
		{
			name:     "ErrDuplicateReactionType",
			err:      ErrDuplicateReactionType,
			expected: "config: duplicate reaction type",
		},
		{
			name:     "ErrInvalidInput",
			err:      ErrInvalidInput,
			expected: "validate: invalid input",
		},
		{
			name:     "ErrReactionNotFound",
			err:      ErrReactionNotFound,
			expected: "query: reaction not found",
		},
		{
			name:     "ErrStorageUnavailable",
			err:      ErrStorageUnavailable,
			expected: "storage: storage unavailable",
		},
		{
			name:     "ErrNotImplemented",
			err:      ErrNotImplemented,
			expected: "method: not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("sentinel error should not be nil")
				return
			}
			if tt.err.Error() != tt.expected {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestSentinelErrors_Is(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		wrapped  error
	}{
		{
			name:     "ErrNoReactionTypes",
			sentinel: ErrNoReactionTypes,
			wrapped:  errNoReactionTypes,
		},
		{
			name:     "ErrInvalidReactionType",
			sentinel: ErrInvalidReactionType,
			wrapped:  errInvalidReactionType,
		},
		{
			name:     "ErrDuplicateReactionType",
			sentinel: ErrDuplicateReactionType,
			wrapped:  errDuplicateReactionType,
		},
		{
			name:     "ErrInvalidInput",
			sentinel: ErrInvalidInput,
			wrapped:  errInvalidInput,
		},
		{
			name:     "ErrReactionNotFound",
			sentinel: ErrReactionNotFound,
			wrapped:  errReactionNotFound,
		},
		{
			name:     "ErrStorageUnavailable",
			sentinel: ErrStorageUnavailable,
			wrapped:  errStorageUnavailable,
		},
		{
			name:     "ErrNotImplemented",
			sentinel: ErrNotImplemented,
			wrapped:  errNotImplemented,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The sentinel errors should wrap the internal errors
			if !errors.Is(tt.sentinel, tt.wrapped) {
				t.Errorf("%v should wrap %v", tt.sentinel, tt.wrapped)
			}
		})
	}
}

func TestConfig_Validate_Consistency(t *testing.T) {
	t.Run("multiple validations return same result", func(t *testing.T) {
		config := Config{
			ReactionTypes: []string{"LIKE", "LOVE"},
		}

		// First validation
		err1 := config.Validate()

		// Second validation - should return same result
		err2 := config.Validate()

		if (err1 == nil) != (err2 == nil) {
			t.Error("Multiple validations should return consistent results")
		}
	})

	t.Run("config is not modified during validation", func(t *testing.T) {
		originalTypes := []string{"LIKE", "LOVE"}
		config := Config{
			ReactionTypes:      append([]string(nil), originalTypes...),
			EnableCaching:      true,
			EnableEvents:       false,
			EnableRateLimiting: true,
		}

		originalCopy := Config{
			ReactionTypes:      append([]string(nil), config.ReactionTypes...),
			EnableCaching:      config.EnableCaching,
			EnableEvents:       config.EnableEvents,
			EnableRateLimiting: config.EnableRateLimiting,
		}

		_ = config.Validate()

		if len(config.ReactionTypes) != len(originalCopy.ReactionTypes) {
			t.Error("Validation should not modify ReactionTypes length")
		}
		for i, rt := range config.ReactionTypes {
			if rt != originalCopy.ReactionTypes[i] {
				t.Error("Validation should not modify ReactionTypes values")
			}
		}
	})
}

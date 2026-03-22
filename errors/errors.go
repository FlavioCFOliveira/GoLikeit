// Package errors defines domain-specific error types for the GoLikeit reaction system.
// These errors provide clear categorization and programmatic error handling capabilities.
package errors

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for common failure scenarios.
// Use errors.Is() to check for these errors in calling code.
var (
	// ErrInvalidInput indicates that one or more input parameters are invalid.
	// This includes empty strings, values exceeding maximum length, or malformed data.
	ErrInvalidInput = errors.New("invalid input provided")

	// ErrReactionNotFound indicates that no reaction exists for the specified user and entity.
	// Returned when attempting to remove or update a non-existent reaction.
	ErrReactionNotFound = errors.New("reaction not found")

	// ErrStorageUnavailable indicates that the storage backend is not accessible.
	// This could be due to network issues, database connection failures, or timeouts.
	ErrStorageUnavailable = errors.New("storage backend unavailable")

	// ErrInvalidReactionType indicates that the reaction type is not recognized
	// or is not in the configured list of allowed reaction types.
	ErrInvalidReactionType = errors.New("invalid reaction type")

	// ErrInvalidReactionFormat indicates that the reaction type format is invalid.
	// Reaction types must match the pattern ^[A-Z0-9_-]+$.
	ErrInvalidReactionFormat = errors.New("invalid reaction type format")

	// ErrNoReactionTypes indicates that no reaction types were configured at initialization.
	// At least one reaction type must be provided.
	ErrNoReactionTypes = errors.New("no reaction types configured")

	// ErrDuplicateReactionType indicates that a duplicate reaction type was found
	// in the configuration. All reaction types must be unique.
	ErrDuplicateReactionType = errors.New("duplicate reaction type found")

	// ErrRateLimitExceeded indicates that the rate limit for an operation has been exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrCacheUnavailable indicates that the cache backend is not accessible.
	ErrCacheUnavailable = errors.New("cache unavailable")

	// ErrClientClosed indicates that the client has been closed and cannot be used.
	ErrClientClosed = errors.New("client is closed")
)

// InputError provides detailed information about validation failures.
// It wraps ErrInvalidInput and includes the field name and specific reason.
type InputError struct {
	Field   string // Field that failed validation
	Value   string // Value that was provided (may be truncated for security)
	Reason  string // Human-readable reason for the failure
	Cause   error  // Underlying cause, if any
}

// Error implements the error interface.
func (e *InputError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("invalid input for field %q: %s (cause: %v)", e.Field, e.Reason, e.Cause)
	}
	return fmt.Sprintf("invalid input for field %q: %s", e.Field, e.Reason)
}

// Unwrap returns the underlying cause for use with errors.Is and errors.As.
func (e *InputError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrInvalidInput
}

// NewInputError creates a new InputError with the specified details.
func NewInputError(field, value, reason string) *InputError {
	return &InputError{
		Field:  field,
		Value:  truncate(value, 64), // Truncate for security
		Reason: reason,
	}
}

// NewInputErrorWithCause creates a new InputError wrapping an underlying cause.
func NewInputErrorWithCause(field, value, reason string, cause error) *InputError {
	return &InputError{
		Field:  field,
		Value:  truncate(value, 64),
		Reason: reason,
		Cause:  cause,
	}
}

// NotFoundError provides detailed information when a reaction is not found.
type NotFoundError struct {
	UserID     string
	EntityType string
	EntityID   string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("reaction not found for user %q on %s:%s", e.UserID, e.EntityType, e.EntityID)
}

// Unwrap returns the underlying error for use with errors.Is.
func (e *NotFoundError) Unwrap() error {
	return ErrReactionNotFound
}

// NewNotFoundError creates a new NotFoundError with the specified details.
func NewNotFoundError(userID, entityType, entityID string) *NotFoundError {
	return &NotFoundError{
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
	}
}

// StorageError provides detailed information about storage failures.
type StorageError struct {
	Operation string // The operation that failed (e.g., "create", "read")
	Cause     error  // The underlying storage error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	return fmt.Sprintf("storage operation %q failed: %v", e.Operation, e.Cause)
}

// Unwrap returns the underlying cause for use with errors.Is.
func (e *StorageError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrStorageUnavailable
}

// NewStorageError creates a new StorageError wrapping an underlying cause.
func NewStorageError(operation string, cause error) *StorageError {
	return &StorageError{
		Operation: operation,
		Cause:     cause,
	}
}

// RateLimitError provides detailed information about rate limit violations.
type RateLimitError struct {
	Operation  string        // The operation that was rate limited
	RetryAfter time.Duration // Time to wait before retrying
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded for operation %q, retry after %v", e.Operation, e.RetryAfter)
	}
	return fmt.Sprintf("rate limit exceeded for operation %q", e.Operation)
}

// Unwrap returns the underlying error for use with errors.Is.
func (e *RateLimitError) Unwrap() error {
	return ErrRateLimitExceeded
}

// NewRateLimitError creates a new RateLimitError with the specified details.
func NewRateLimitError(operation string, retryAfter time.Duration) *RateLimitError {
	return &RateLimitError{
		Operation:  operation,
		RetryAfter: retryAfter,
	}
}

// Helper function to truncate strings for security.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// IsInvalidInput checks if an error is or wraps ErrInvalidInput.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsNotFound checks if an error is or wraps ErrReactionNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrReactionNotFound)
}

// IsStorageUnavailable checks if an error is or wraps ErrStorageUnavailable.
func IsStorageUnavailable(err error) bool {
	return errors.Is(err, ErrStorageUnavailable)
}

// IsInvalidReactionType checks if an error is or wraps ErrInvalidReactionType.
func IsInvalidReactionType(err error) bool {
	return errors.Is(err, ErrInvalidReactionType)
}

// IsRateLimitExceeded checks if an error is or wraps ErrRateLimitExceeded.
func IsRateLimitExceeded(err error) bool {
	return errors.Is(err, ErrRateLimitExceeded)
}
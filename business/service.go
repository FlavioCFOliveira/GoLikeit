// Package business provides configuration validation and domain error types for
// the GoLikeit reaction system.
//
// The canonical business layer implementation is golikeit.Client, which handles
// all input validation, reaction type enforcement, cache management, event emission,
// and storage orchestration. This package provides supporting types used when
// integrating or configuring the library.
package business

// Config holds the configuration for initializing the GoLikeit client.
type Config struct {
	// ReactionTypes is the list of allowed reaction types.
	// At least one type must be provided.
	ReactionTypes []string

	// EnableCaching enables the caching layer.
	EnableCaching bool

	// EnableEvents enables the event system.
	EnableEvents bool

	// EnableRateLimiting enables rate limiting.
	EnableRateLimiting bool
}

// Validate validates the business layer configuration.
func (c Config) Validate() error {
	if len(c.ReactionTypes) == 0 {
		return ErrNoReactionTypes
	}

	seen := make(map[string]struct{}, len(c.ReactionTypes))
	for _, rt := range c.ReactionTypes {
		if rt == "" {
			return ErrInvalidReactionType
		}
		if _, exists := seen[rt]; exists {
			return ErrDuplicateReactionType
		}
		seen[rt] = struct{}{}
	}

	return nil
}

// ServiceError is the common error type for business layer errors.
type ServiceError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	return e.Op + ": " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// Common business layer errors.
// These should be used with errors.Is() for checking.
var (
	// ErrNoReactionTypes indicates no reaction types were configured.
	ErrNoReactionTypes = &ServiceError{Op: "config", Err: errNoReactionTypes}

	// ErrInvalidReactionType indicates an invalid or unknown reaction type.
	ErrInvalidReactionType = &ServiceError{Op: "validate", Err: errInvalidReactionType}

	// ErrDuplicateReactionType indicates a duplicate reaction type in config.
	ErrDuplicateReactionType = &ServiceError{Op: "config", Err: errDuplicateReactionType}

	// ErrInvalidInput indicates invalid input parameters.
	ErrInvalidInput = &ServiceError{Op: "validate", Err: errInvalidInput}

	// ErrReactionNotFound indicates a reaction was not found.
	ErrReactionNotFound = &ServiceError{Op: "query", Err: errReactionNotFound}

	// ErrStorageUnavailable indicates the storage backend is unavailable.
	ErrStorageUnavailable = &ServiceError{Op: "storage", Err: errStorageUnavailable}

	// ErrNotImplemented indicates a method is not yet implemented.
	ErrNotImplemented = &ServiceError{Op: "method", Err: errNotImplemented}
)

// Internal error values for wrapping.
var (
	errNoReactionTypes       = NewBusinessError("no reaction types configured")
	errInvalidReactionType   = NewBusinessError("invalid reaction type")
	errDuplicateReactionType = NewBusinessError("duplicate reaction type")
	errInvalidInput          = NewBusinessError("invalid input")
	errReactionNotFound      = NewBusinessError("reaction not found")
	errStorageUnavailable    = NewBusinessError("storage unavailable")
	errNotImplemented        = NewBusinessError("not implemented")
)

// BusinessError represents a domain-specific error.
type BusinessError struct {
	msg string
}

// Error implements the error interface.
func (e *BusinessError) Error() string {
	return e.msg
}

// NewBusinessError creates a new BusinessError.
func NewBusinessError(msg string) error {
	return &BusinessError{msg: msg}
}

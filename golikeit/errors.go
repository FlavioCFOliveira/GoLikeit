// Package golikeit provides a public API client for the reaction system.
package golikeit

import "errors"

// Exported errors for programmatic error handling.
// These errors can be checked using errors.Is().
var (
	// ErrInvalidInput indicates that one or more input parameters are invalid.
	ErrInvalidInput = errors.New("invalid input")

	// ErrReactionNotFound indicates that no reaction exists for the specified target.
	ErrReactionNotFound = errors.New("reaction not found")

	// ErrStorageUnavailable indicates that the storage backend is not accessible.
	ErrStorageUnavailable = errors.New("storage unavailable")

	// ErrInvalidReactionType indicates that the reaction type is not in the configured registry.
	ErrInvalidReactionType = errors.New("invalid reaction type")

	// ErrInvalidReactionFormat indicates that the reaction type format is invalid.
	// Reaction types must match the pattern ^[A-Z0-9_-]+$.
	ErrInvalidReactionFormat = errors.New("reaction type must match [A-Z0-9_-]+")

	// ErrNoReactionTypes indicates that no reaction types were configured.
	ErrNoReactionTypes = errors.New("at least one reaction type required")

	// ErrDuplicateReactionType indicates that a reaction type was defined more than once.
	ErrDuplicateReactionType = errors.New("duplicate reaction type")

	// ErrClientClosed indicates that the client has been closed and is no longer usable.
	ErrClientClosed = errors.New("client is closed")
)

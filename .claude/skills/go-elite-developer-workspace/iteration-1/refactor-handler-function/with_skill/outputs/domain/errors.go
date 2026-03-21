// Package domain contains the core business models and errors.
package domain

import "errors"

// Domain errors.
var (
	ErrInvalidInput = errors.New("invalid input: name is required")
	ErrNotFound     = errors.New("user not found")
	ErrDatabase     = errors.New("database error")
)

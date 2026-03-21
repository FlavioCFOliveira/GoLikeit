// Package models contains domain models for the application.
package models

import (
	"errors"
	"strings"
)

// Common validation errors.
var (
	ErrUserNameRequired = errors.New("user name is required")
	ErrUserNameTooLong  = errors.New("user name exceeds maximum length of 255 characters")
)

// User represents a user in the system.
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Validate checks if the user data is valid.
// Returns nil if valid, or an error describing the validation failure.
func (u *User) Validate() error {
	u.Name = strings.TrimSpace(u.Name)

	if u.Name == "" {
		return ErrUserNameRequired
	}

	if len(u.Name) > 255 {
		return ErrUserNameTooLong
	}

	return nil
}

// Sanitize cleans and prepares user data.
func (u *User) Sanitize() {
	u.Name = strings.TrimSpace(u.Name)
}

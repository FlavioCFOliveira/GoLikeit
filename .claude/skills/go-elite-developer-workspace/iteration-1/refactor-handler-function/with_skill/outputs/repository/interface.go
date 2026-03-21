// Package repository defines the data access layer interfaces.
package repository

import (
	"context"

	"github.com/example/golikeit/domain"
)

// UserRepository defines the interface for user data access.
// This abstraction allows for easy testing and switching of implementations.
type UserRepository interface {
	// GetAll returns all users from the database.
	GetAll(ctx context.Context) ([]domain.User, error)

	// Create inserts a new user and returns the generated ID.
	Create(ctx context.Context, name string) (int64, error)
}

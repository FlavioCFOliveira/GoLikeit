// Package service contains business logic layer.
package service

import (
	"context"
	"fmt"

	"github.com/example/project/models"
	"github.com/example/project/repository"
)

// UserService handles business logic for user operations.
type UserService struct {
	repo repository.UserRepository
}

// NewUserService creates a new user service with the given repository.
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// ListUsers returns all users.
func (s *UserService) ListUsers(ctx context.Context) ([]models.User, error) {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// CreateUser creates a new user after validation.
func (s *UserService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	// Validate input
	if err := user.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Sanitize input
	user.Sanitize()

	// Create user
	created, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return created, nil
}

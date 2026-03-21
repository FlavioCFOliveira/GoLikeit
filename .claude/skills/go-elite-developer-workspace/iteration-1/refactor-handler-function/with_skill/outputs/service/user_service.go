// Package service contains business logic for user operations.
package service

import (
	"context"
	"fmt"

	"github.com/example/golikeit/domain"
	"github.com/example/golikeit/repository"
)

// UserService handles business logic for user operations.
type UserService struct {
	repo repository.UserRepository
}

// NewUserService creates a new UserService with the given repository.
func NewUserService(repo repository.UserRepository) *UserService {
	if repo == nil {
		panic("repository cannot be nil")
	}
	return &UserService{repo: repo}
}

// ListUsers returns all users.
func (s *UserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

// CreateUser creates a new user after validating the input.
func (s *UserService) CreateUser(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	id, err := s.repo.Create(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &domain.UserCreateResponse{ID: id}, nil
}

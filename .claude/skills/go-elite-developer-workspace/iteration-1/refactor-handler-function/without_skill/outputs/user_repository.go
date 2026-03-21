// Package repository provides data access layer abstractions.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/project/models"
)

// UserRepository defines the interface for user data operations.
// This abstraction allows for easy testing and swapping implementations.
type UserRepository interface {
	// GetAll returns all users from the database.
	GetAll(ctx context.Context) ([]models.User, error)
	// Create inserts a new user and returns the created user with ID.
	Create(ctx context.Context, user *models.User) (*models.User, error)
}

// Ensure SQLUserRepository implements UserRepository.
var _ UserRepository = (*SQLUserRepository)(nil)

// SQLUserRepository implements UserRepository using SQL database.
type SQLUserRepository struct {
	db *sql.DB
}

// NewSQLUserRepository creates a new SQL-based user repository.
func NewSQLUserRepository(db *sql.DB) *SQLUserRepository {
	return &SQLUserRepository{db: db}
}

// GetAll retrieves all users from the database.
func (r *SQLUserRepository) GetAll(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name FROM users")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	return r.scanUsers(rows)
}

// scanUsers scans sql.Rows into a slice of User.
func (r *SQLUserRepository) scanUsers(rows *sql.Rows) ([]models.User, error) {
	var users []models.User

	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, fmt.Errorf("failed to scan user row: %w", err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

// Create inserts a new user into the database.
func (r *SQLUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	// Using PostgreSQL-style placeholder ($1). For MySQL, use "?" instead.
	result, err := r.db.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", user.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	user.ID = id
	return user, nil
}

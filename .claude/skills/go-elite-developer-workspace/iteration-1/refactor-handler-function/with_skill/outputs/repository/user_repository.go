// Package repository provides data access implementations.
package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/example/golikeit/domain"
)

// SQLUserRepository implements UserRepository using a SQL database.
type SQLUserRepository struct {
	db *sql.DB
}

// NewSQLUserRepository creates a new SQL-backed user repository.
// The db parameter must not be nil.
func NewSQLUserRepository(db *sql.DB) *SQLUserRepository {
	if db == nil {
		panic("db cannot be nil")
	}
	return &SQLUserRepository{db: db}
}

// GetAll returns all users from the database.
func (r *SQLUserRepository) GetAll(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name FROM users")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query users: %v", domain.ErrDatabase, err)
	}
	defer rows.Close()

	// Pre-allocate with reasonable capacity to avoid reallocations
	users := make([]domain.User, 0, 100)

	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, fmt.Errorf("%w: failed to scan user: %v", domain.ErrDatabase, err)
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: rows iteration error: %v", domain.ErrDatabase, err)
	}

	return users, nil
}

// Create inserts a new user and returns the generated ID.
func (r *SQLUserRepository) Create(ctx context.Context, name string) (int64, error) {
	result, err := r.db.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", name)
	if err != nil {
		return 0, fmt.Errorf("%w: failed to insert user: %v", domain.ErrDatabase, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("%w: failed to get last insert id: %v", domain.ErrDatabase, err)
	}

	return id, nil
}

// Package domain contains the core business models.
package domain

// User represents a user in the system.
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// UserCreateRequest represents the data needed to create a new user.
type UserCreateRequest struct {
	Name string `json:"name"`
}

// Validate checks if the create request contains valid data.
func (r UserCreateRequest) Validate() error {
	if r.Name == "" {
		return ErrInvalidInput
	}
	return nil
}

// UserResponse represents the response after creating a user.
type UserCreateResponse struct {
	ID int64 `json:"id"`
}

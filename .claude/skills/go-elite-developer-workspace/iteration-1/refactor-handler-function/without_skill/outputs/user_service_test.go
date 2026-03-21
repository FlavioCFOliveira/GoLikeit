package service

import (
	"context"
	"errors"
	"testing"

	"github.com/example/project/models"
)

// mockRepository is a test double for UserRepository.
type mockRepository struct {
	users       []models.User
	nextID      int64
	getAllErr   error
	createErr   error
	lastCreated *models.User
}

func (m *mockRepository) GetAll(ctx context.Context) ([]models.User, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.users, nil
}

func (m *mockRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.nextID++
	user.ID = m.nextID
	m.lastCreated = user
	return user, nil
}

func TestUserService_ListUsers_Success(t *testing.T) {
	mock := &mockRepository{
		users: []models.User{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		},
	}
	service := NewUserService(mock)

	users, err := service.ListUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUserService_ListUsers_RepositoryError(t *testing.T) {
	mock := &mockRepository{getAllErr: errors.New("db error")}
	service := NewUserService(mock)

	_, err := service.ListUsers(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestUserService_CreateUser_Success(t *testing.T) {
	mock := &mockRepository{}
	service := NewUserService(mock)

	user := &models.User{Name: "Charlie"}
	created, err := service.CreateUser(context.Background(), user)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.ID != 1 {
		t.Errorf("expected ID 1, got %d", created.ID)
	}

	if mock.lastCreated.Name != "Charlie" {
		t.Errorf("expected name 'Charlie', got %s", mock.lastCreated.Name)
	}
}

func TestUserService_CreateUser_ValidationError(t *testing.T) {
	mock := &mockRepository{}
	service := NewUserService(mock)

	// Empty name should fail validation
	user := &models.User{Name: ""}
	_, err := service.CreateUser(context.Background(), user)

	if !errors.Is(err, models.ErrUserNameRequired) {
		t.Errorf("expected ErrUserNameRequired, got %v", err)
	}
}

func TestUserService_CreateUser_NameTooLong(t *testing.T) {
	mock := &mockRepository{}
	service := NewUserService(mock)

	// Name exceeding 255 characters should fail
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}

	user := &models.User{Name: string(longName)}
	_, err := service.CreateUser(context.Background(), user)

	if !errors.Is(err, models.ErrUserNameTooLong) {
		t.Errorf("expected ErrUserNameTooLong, got %v", err)
	}
}

func TestUserService_CreateUser_SanitizesInput(t *testing.T) {
	mock := &mockRepository{}
	service := NewUserService(mock)

	// Name with leading/trailing spaces
	user := &models.User{Name: "  Alice  "}
	_, err := service.CreateUser(context.Background(), user)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.lastCreated.Name != "Alice" {
		t.Errorf("expected sanitized name 'Alice', got '%s'", mock.lastCreated.Name)
	}
}

func TestUserService_CreateUser_RepositoryError(t *testing.T) {
	mock := &mockRepository{createErr: errors.New("insert failed")}
	service := NewUserService(mock)

	user := &models.User{Name: "Dave"}
	_, err := service.CreateUser(context.Background(), user)

	if err == nil {
		t.Error("expected error, got nil")
	}
}

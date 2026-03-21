package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/project/models"
	"github.com/example/project/service"
)

// mockUserRepository is a test double for UserRepository.
type mockUserRepository struct {
	users   []models.User
	nextID  int64
	getErr  error
	createErr error
}

func (m *mockUserRepository) GetAll(ctx context.Context) ([]models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.users, nil
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	m.nextID++
	user.ID = m.nextID
	return user, nil
}

// setupHandler creates a handler with mock dependencies for testing.
func setupHandler(t *testing.T) (*UserHandler, *mockUserRepository) {
	t.Helper()

	mockRepo := &mockUserRepository{
		users:  []models.User{},
		nextID: 0,
	}

	userService := service.NewUserService(mockRepo)
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	handler := NewUserHandler(userService, logger)

	return handler, mockRepo
}

func TestUserHandler_GetUsers_Success(t *testing.T) {
	handler, mockRepo := setupHandler(t)
	mockRepo.users = []models.User{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "Bob"},
	}

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var response []models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("expected 2 users, got %d", len(response))
	}

	if response[0].Name != "Alice" {
		t.Errorf("expected first user name 'Alice', got %s", response[0].Name)
	}
}

func TestUserHandler_GetUsers_ServiceError(t *testing.T) {
	handler, mockRepo := setupHandler(t)
	mockRepo.getErr = errors.New("database connection failed")

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	handler, _ := setupHandler(t)

	payload := `{"name": "Charlie"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var response map[string]int64
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["id"] != 1 {
		t.Errorf("expected id 1, got %d", response["id"])
	}
}

func TestUserHandler_CreateUser_ValidationError(t *testing.T) {
	handler, _ := setupHandler(t)

	// Empty name should fail validation
	payload := `{"name": ""}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUserHandler_CreateUser_InvalidJSON(t *testing.T) {
	handler, _ := setupHandler(t)

	payload := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUserHandler_CreateUser_ServiceError(t *testing.T) {
	handler, mockRepo := setupHandler(t)
	mockRepo.createErr = errors.New("database write failed")

	payload := `{"name": "Dave"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestUserHandler_MethodNotAllowed(t *testing.T) {
	handler, _ := setupHandler(t)

	req := httptest.NewRequest(http.MethodPut, "/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

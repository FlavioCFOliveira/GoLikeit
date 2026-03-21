package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/golikeit/domain"
)

// mockUserService is a mock implementation of UserService for testing.
type mockUserService struct {
	listUsersFunc  func(ctx context.Context) ([]domain.User, error)
	createUserFunc func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error)
}

func (m *mockUserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(ctx)
	}
	return nil, errors.New("listUsers not implemented")
}

func (m *mockUserService) CreateUser(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, req)
	}
	return nil, errors.New("createUser not implemented")
}

func TestNewUserHandler_PanicsOnNilService(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil service, but got none")
		}
	}()
	NewUserHandler(nil)
}

func TestUserHandler_ServeHTTP_MethodRouting(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET request",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST request",
			method:     http.MethodPost,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "PUT request not allowed",
			method:     http.MethodPut,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "DELETE request not allowed",
			method:     http.MethodDelete,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockUserService{
				listUsersFunc: func(ctx context.Context) ([]domain.User, error) {
					return []domain.User{}, nil
				},
				createUserFunc: func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
					return &domain.UserCreateResponse{ID: 1}, nil
				},
			}
			handler := NewUserHandler(mock)

			var req *http.Request
			if tt.method == http.MethodPost {
				body := bytes.NewBufferString(`{"name":"Alice"}`)
				req = httptest.NewRequest(tt.method, "/users", body)
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/users", nil)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("ServeHTTP() status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestUserHandler_listUsers(t *testing.T) {
	tests := []struct {
		name           string
		mockListUsers  func(ctx context.Context) ([]domain.User, error)
		wantStatus     int
		wantBody       string
		wantContentType string
	}{
		{
			name: "success - returns users",
			mockListUsers: func(ctx context.Context) ([]domain.User, error) {
				return []domain.User{
					{ID: 1, Name: "Alice"},
					{ID: 2, Name: "Bob"},
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantContentType: "application/json",
		},
		{
			name: "database error",
			mockListUsers: func(ctx context.Context) ([]domain.User, error) {
				return nil, domain.ErrDatabase
			},
			wantStatus:     http.StatusInternalServerError,
			wantContentType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockUserService{listUsersFunc: tt.mockListUsers}
			handler := NewUserHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			rr := httptest.NewRecorder()

			handler.listUsers(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("listUsers() status = %d, want %d", rr.Code, tt.wantStatus)
			}

			contentType := rr.Header().Get("Content-Type")
			if tt.wantContentType == "application/json" && contentType != "application/json" {
				t.Errorf("listUsers() Content-Type = %s, want application/json", contentType)
			}
		})
	}
}

func TestUserHandler_createUser(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		mockCreateUser  func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error)
		wantStatus      int
		wantContentType string
	}{
		{
			name: "success - creates user",
			body: `{"name":"Alice"}`,
			mockCreateUser: func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
				return &domain.UserCreateResponse{ID: 1}, nil
			},
			wantStatus:      http.StatusCreated,
			wantContentType: "application/json",
		},
		{
			name:            "invalid json",
			body:            `{"name":}`,
			mockCreateUser:  nil,
			wantStatus:      http.StatusBadRequest,
			wantContentType: "text/plain",
		},
		{
			name: "validation error",
			body: `{"name":""}`,
			mockCreateUser: func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
				return nil, domain.ErrInvalidInput
			},
			wantStatus:      http.StatusBadRequest,
			wantContentType: "text/plain",
		},
		{
			name: "database error",
			body: `{"name":"Alice"}`,
			mockCreateUser: func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
				return nil, domain.ErrDatabase
			},
			wantStatus:      http.StatusInternalServerError,
			wantContentType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockUserService{createUserFunc: tt.mockCreateUser}
			handler := NewUserHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.createUser(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("createUser() status = %d, want %d", rr.Code, tt.wantStatus)
			}

			contentType := rr.Header().Get("Content-Type")
			if tt.wantContentType == "application/json" && contentType != "application/json" {
				t.Errorf("createUser() Content-Type = %s, want application/json", contentType)
			}
		})
	}
}

func TestUserHandler_createUser_DecodesRequest(t *testing.T) {
	var capturedReq domain.UserCreateRequest
	mock := &mockUserService{
		createUserFunc: func(ctx context.Context, req domain.UserCreateRequest) (*domain.UserCreateResponse, error) {
			capturedReq = req
			return &domain.UserCreateResponse{ID: 1}, nil
		},
	}
	handler := NewUserHandler(mock)

	body := `{"name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.createUser(rr, req)

	if capturedReq.Name != "Alice" {
		t.Errorf("createUser() decoded request.Name = %s, want Alice", capturedReq.Name)
	}

	if rr.Code != http.StatusCreated {
		t.Errorf("createUser() status = %d, want %d", rr.Code, http.StatusCreated)
	}

	var resp domain.UserCreateResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("createUser() response.ID = %d, want 1", resp.ID)
	}
}

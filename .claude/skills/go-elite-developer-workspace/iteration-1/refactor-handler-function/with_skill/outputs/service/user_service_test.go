package service

import (
	"context"
	"errors"
	"testing"

	"github.com/example/golikeit/domain"
)

// mockUserRepository is a mock implementation of repository.UserRepository for testing.
type mockUserRepository struct {
	getAllFunc func(ctx context.Context) ([]domain.User, error)
	createFunc func(ctx context.Context, name string) (int64, error)
}

func (m *mockUserRepository) GetAll(ctx context.Context) ([]domain.User, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}
	return nil, errors.New("getAll not implemented")
}

func (m *mockUserRepository) Create(ctx context.Context, name string) (int64, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, name)
	}
	return 0, errors.New("create not implemented")
}

func TestNewUserService_PanicsOnNilRepo(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil repo, but got none")
		}
	}()
	NewUserService(nil)
}

func TestUserService_ListUsers(t *testing.T) {
	tests := []struct {
		name        string
		mockGetAll  func(ctx context.Context) ([]domain.User, error)
		wantUsers   []domain.User
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success - returns users",
			mockGetAll: func(ctx context.Context) ([]domain.User, error) {
				return []domain.User{
					{ID: 1, Name: "Alice"},
					{ID: 2, Name: "Bob"},
				}, nil
			},
			wantUsers: []domain.User{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
			},
			wantErr: false,
		},
		{
			name: "repository error",
			mockGetAll: func(ctx context.Context) ([]domain.User, error) {
				return nil, domain.ErrDatabase
			},
			wantUsers:   nil,
			wantErr:     true,
			expectedErr: domain.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockUserRepository{getAllFunc: tt.mockGetAll}
			svc := NewUserService(mock)

			users, err := svc.ListUsers(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("ListUsers() expected error but got none")
					return
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("ListUsers() error = %v, expected error containing %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ListUsers() unexpected error = %v", err)
				return
			}

			if len(users) != len(tt.wantUsers) {
				t.Errorf("ListUsers() returned %d users, want %d", len(users), len(tt.wantUsers))
			}

			for i, u := range users {
				if u.ID != tt.wantUsers[i].ID || u.Name != tt.wantUsers[i].Name {
					t.Errorf("ListUsers() user[%d] = %+v, want %+v", i, u, tt.wantUsers[i])
				}
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name        string
		request     domain.UserCreateRequest
		mockCreate  func(ctx context.Context, name string) (int64, error)
		wantID      int64
		wantErr     bool
		expectedErr error
	}{
		{
			name:    "success - creates user",
			request: domain.UserCreateRequest{Name: "Alice"},
			mockCreate: func(ctx context.Context, name string) (int64, error) {
				return 1, nil
			},
			wantID:  1,
			wantErr: false,
		},
		{
			name:        "validation error - empty name",
			request:     domain.UserCreateRequest{Name: ""},
			mockCreate:  nil,
			wantID:      0,
			wantErr:     true,
			expectedErr: domain.ErrInvalidInput,
		},
		{
			name:    "repository error",
			request: domain.UserCreateRequest{Name: "Alice"},
			mockCreate: func(ctx context.Context, name string) (int64, error) {
				return 0, domain.ErrDatabase
			},
			wantID:      0,
			wantErr:     true,
			expectedErr: domain.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockUserRepository{createFunc: tt.mockCreate}
			svc := NewUserService(mock)

			resp, err := svc.CreateUser(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("CreateUser() expected error but got none")
					return
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("CreateUser() error = %v, expected error containing %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateUser() unexpected error = %v", err)
				return
			}

			if resp.ID != tt.wantID {
				t.Errorf("CreateUser() ID = %d, want %d", resp.ID, tt.wantID)
			}
		})
	}
}

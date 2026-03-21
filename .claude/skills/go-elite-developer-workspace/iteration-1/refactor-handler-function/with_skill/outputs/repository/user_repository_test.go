package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/example/golikeit/domain"
)

func TestNewSQLUserRepository_PanicsOnNilDB(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil db, but got none")
		}
	}()
	NewSQLUserRepository(nil)
}

func TestSQLUserRepository_GetAll(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(mock sqlmock.Sqlmock)
		wantUsers   []domain.User
		wantErr     bool
		expectedErr error
	}{
		{
			name: "success - returns users",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(1, "Alice").
					AddRow(2, "Bob")
				mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)
			},
			wantUsers: []domain.User{
				{ID: 1, Name: "Alice"},
				{ID: 2, Name: "Bob"},
			},
			wantErr: false,
		},
		{
			name: "success - empty result",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"})
				mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)
			},
			wantUsers: []domain.User{},
			wantErr:   false,
		},
		{
			name: "database error on query",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT id, name FROM users").
					WillReturnError(sql.ErrConnDone)
			},
			wantUsers:   nil,
			wantErr:     true,
			expectedErr: domain.ErrDatabase,
		},
		{
			name: "scan error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow("invalid", "Alice") // wrong type for id
				mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)
			},
			wantUsers:   nil,
			wantErr:     true,
			expectedErr: domain.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			tt.mockSetup(mock)

			repo := NewSQLUserRepository(db)
			users, err := repo.GetAll(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAll() expected error but got none")
					return
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("GetAll() error = %v, expected error containing %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetAll() unexpected error = %v", err)
				return
			}

			if len(users) != len(tt.wantUsers) {
				t.Errorf("GetAll() returned %d users, want %d", len(users), len(tt.wantUsers))
			}

			for i, u := range users {
				if u.ID != tt.wantUsers[i].ID || u.Name != tt.wantUsers[i].Name {
					t.Errorf("GetAll() user[%d] = %+v, want %+v", i, u, tt.wantUsers[i])
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSQLUserRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		userName    string
		mockSetup   func(mock sqlmock.Sqlmock)
		wantID      int64
		wantErr     bool
		expectedErr error
	}{
		{
			name:     "success - creates user",
			userName: "Alice",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO users").
					WithArgs("Alice").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantID:  1,
			wantErr: false,
		},
		{
			name:     "database error on insert",
			userName: "Alice",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO users").
					WithArgs("Alice").
					WillReturnError(sql.ErrConnDone)
			},
			wantID:      0,
			wantErr:     true,
			expectedErr: domain.ErrDatabase,
		},
		{
			name:     "error getting last insert id",
			userName: "Alice",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO users").
					WithArgs("Alice").
					WillReturnResult(sqlmock.NewErrorResult(sql.ErrNoRows))
			},
			wantID:      0,
			wantErr:     true,
			expectedErr: domain.ErrDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to create mock: %v", err)
			}
			defer db.Close()

			tt.mockSetup(mock)

			repo := NewSQLUserRepository(db)
			id, err := repo.Create(context.Background(), tt.userName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Create() expected error but got none")
					return
				}
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("Create() error = %v, expected error containing %v", err, tt.expectedErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Create() unexpected error = %v", err)
				return
			}

			if id != tt.wantID {
				t.Errorf("Create() = %d, want %d", id, tt.wantID)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

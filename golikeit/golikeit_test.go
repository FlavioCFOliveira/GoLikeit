package golikeit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/logging"
	"github.com/FlavioCFOliveira/GoLikeit/metrics"
	"github.com/FlavioCFOliveira/GoLikeit/pagination"
	"github.com/FlavioCFOliveira/GoLikeit/resilience"
	"github.com/FlavioCFOliveira/GoLikeit/tracing"
)

// mockStorage is a test double for ReactionStorage.
type mockStorage struct {
	addReactionCalled     bool
	removeReactionCalled  bool
	getUserReactionCalled bool
	getEntityCountsCalled bool

	addReactionCallCount  int
	addReactionFn         func(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error)
	addReactionResult     bool
	addReactionErr          error
	removeReactionErr       error
	getUserReactionResult   string
	getUserReactionErr      error
	getEntityCountsResult   EntityCounts
	getEntityCountsErr      error
	getUserReactionsResult       []UserReaction
	getUserReactionsTotal        int64
	getUserReactionsErr          error
	getUserReactionCountsResult  map[string]int64
	getUserReactionCountsErr     error
	getUserReactionsByTypeResult []UserReaction
	getUserReactionsByTypeTotal  int64
	getUserReactionsByTypeErr    error
	getEntityReactionsResult     []EntityReaction
	getEntityReactionsTotal      int64
	getEntityReactionsErr        error
	getRecentReactionsResult     []RecentUserReaction
	getRecentReactionsErr        error
	getLastReactionTimeResult  *time.Time
	getLastReactionTimeErr     error
	closeErr                   error
}

func (m *mockStorage) AddReaction(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error) {
	m.addReactionCalled = true
	m.addReactionCallCount++
	if m.addReactionFn != nil {
		return m.addReactionFn(ctx, userID, target, reactionType)
	}
	return m.addReactionResult, m.addReactionErr
}

func (m *mockStorage) RemoveReaction(ctx context.Context, userID string, target EntityTarget) error {
	m.removeReactionCalled = true
	return m.removeReactionErr
}

func (m *mockStorage) GetUserReaction(ctx context.Context, userID string, target EntityTarget) (string, error) {
	m.getUserReactionCalled = true
	return m.getUserReactionResult, m.getUserReactionErr
}

func (m *mockStorage) GetEntityCounts(ctx context.Context, target EntityTarget) (EntityCounts, error) {
	m.getEntityCountsCalled = true
	return m.getEntityCountsResult, m.getEntityCountsErr
}

func (m *mockStorage) GetUserReactions(ctx context.Context, userID string, pagination Pagination) ([]UserReaction, int64, error) {
	return m.getUserReactionsResult, m.getUserReactionsTotal, m.getUserReactionsErr
}

func (m *mockStorage) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	return m.getUserReactionCountsResult, m.getUserReactionCountsErr
}

func (m *mockStorage) GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pagination Pagination) ([]UserReaction, int64, error) {
	return m.getUserReactionsByTypeResult, m.getUserReactionsByTypeTotal, m.getUserReactionsByTypeErr
}

func (m *mockStorage) GetEntityReactions(ctx context.Context, target EntityTarget, pagination Pagination) ([]EntityReaction, int64, error) {
	return m.getEntityReactionsResult, m.getEntityReactionsTotal, m.getEntityReactionsErr
}

func (m *mockStorage) GetRecentReactions(ctx context.Context, target EntityTarget, limit int) ([]RecentUserReaction, error) {
	return m.getRecentReactionsResult, m.getRecentReactionsErr
}

func (m *mockStorage) GetLastReactionTime(ctx context.Context, target EntityTarget) (*time.Time, error) {
	return m.getLastReactionTimeResult, m.getLastReactionTimeErr
}

func (m *mockStorage) Close() error {
	return m.closeErr
}

// newTestClient creates a client with mock storage for testing.
func newTestClient(t *testing.T, cfg Config) (*Client, *mockStorage) {
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	mock := &mockStorage{}
	client.SetStorage(mock)

	return client, mock
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errType error
	}{
		{
			name: "valid configuration",
			config: Config{
				ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
			},
			wantErr: false,
		},
		{
			name: "empty reaction types",
			config: Config{
				ReactionTypes: []string{},
			},
			wantErr: true,
			errType: ErrInvalidReactionType, // Wrapped error
		},
		{
			name: "invalid reaction format - lowercase",
			config: Config{
				ReactionTypes: []string{"like"},
			},
			wantErr: true,
			errType: ErrInvalidReactionType,
		},
		{
			name: "invalid reaction format - special chars",
			config: Config{
				ReactionTypes: []string{"LIKE!"},
			},
			wantErr: true,
			errType: ErrInvalidReactionType,
		},
		{
			name: "duplicate reaction types",
			config: Config{
				ReactionTypes: []string{"LIKE", "LIKE"},
			},
			wantErr: true,
			errType: ErrInvalidReactionType,
		},
		{
			name: "reaction type too long",
			config: Config{
				ReactionTypes: []string{"LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE_LIKE"},
			},
			wantErr: true,
			errType: ErrInvalidReactionType,
		},
		{
			name: "valid with complex types",
			config: Config{
				ReactionTypes: []string{"THUMBS_UP", "THUMBS-DOWN", "STAR_5", "ANGRY", "WOW"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errType != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("New() error = %v, should contain %v", err, tt.errType)
				}
			}
			if !tt.wantErr && client == nil {
				t.Error("New() returned nil client without error")
			}
		})
	}
}

func TestClient_AddReaction(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name           string
		userID         string
		entityType     string
		entityID       string
		reactionType   string
		mockResult     bool
		mockErr        error
		wantResult     bool
		wantErr        bool
		expectedErrType error
	}{
		{
			name:         "add new reaction",
			userID:       "user123",
			entityType:   "photo",
			entityID:     "photo456",
			reactionType: "LIKE",
			mockResult:   false,
			wantResult:   false,
			wantErr:      false,
		},
		{
			name:         "replace existing reaction",
			userID:       "user123",
			entityType:   "photo",
			entityID:     "photo456",
			reactionType: "LOVE",
			mockResult:   true,
			wantResult:   true,
			wantErr:      false,
		},
		{
			name:            "empty user ID",
			userID:          "",
			entityType:      "photo",
			entityID:        "photo456",
			reactionType:    "LIKE",
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:            "empty entity type",
			userID:          "user123",
			entityType:      "",
			entityID:        "photo456",
			reactionType:    "LIKE",
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:            "empty entity ID",
			userID:          "user123",
			entityType:      "photo",
			entityID:        "",
			reactionType:    "LIKE",
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:            "invalid reaction type",
			userID:          "user123",
			entityType:      "photo",
			entityID:        "photo456",
			reactionType:    "INVALID_TYPE",
			wantErr:         true,
			expectedErrType: ErrInvalidReactionType,
		},
		{
			name:            "storage error",
			userID:          "user123",
			entityType:      "photo",
			entityID:        "photo456",
			reactionType:    "LIKE",
			mockErr:         errors.New("storage failed"),
			wantErr:         true,
			expectedErrType: ErrStorageUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.addReactionResult = tt.mockResult
			mock.addReactionErr = tt.mockErr

			got, err := client.AddReaction(ctx, tt.userID, tt.entityType, tt.entityID, tt.reactionType)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.expectedErrType != nil {
				if !errors.Is(err, tt.expectedErrType) {
					t.Errorf("AddReaction() error type = %T, want %T", err, tt.expectedErrType)
				}
			}
			if !tt.wantErr && got != tt.wantResult {
				t.Errorf("AddReaction() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestClient_RemoveReaction(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name            string
		userID          string
		entityType      string
		entityID        string
		mockErr         error
		wantErr         bool
		expectedErrType error
	}{
		{
			name:       "remove existing reaction",
			userID:     "user123",
			entityType: "photo",
			entityID:   "photo456",
			wantErr:    false,
		},
		{
			name:            "remove non-existing reaction",
			userID:          "user123",
			entityType:      "photo",
			entityID:        "photo456",
			mockErr:         ErrReactionNotFound,
			wantErr:         true,
			expectedErrType: ErrReactionNotFound,
		},
		{
			name:            "empty user ID",
			userID:          "",
			entityType:      "photo",
			entityID:        "photo456",
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:            "empty entity type",
			userID:          "user123",
			entityType:      "",
			entityID:        "photo456",
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:            "storage error",
			userID:          "user123",
			entityType:      "photo",
			entityID:        "photo456",
			mockErr:         ErrReactionNotFound,
			wantErr:         true,
			expectedErrType: ErrReactionNotFound, // Direct pass-through from storage
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.removeReactionErr = tt.mockErr

			err := client.RemoveReaction(ctx, tt.userID, tt.entityType, tt.entityID)

			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.expectedErrType != nil {
				if !errors.Is(err, tt.expectedErrType) {
					t.Errorf("RemoveReaction() error = %v, should contain %v", err, tt.expectedErrType)
				}
			}
		})
	}
}

func TestClient_GetUserReaction(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name           string
		userID         string
		entityType     string
		entityID       string
		mockResult     string
		mockErr        error
		wantResult     string
		wantErr        bool
		expectedErrType error
	}{
		{
			name:       "user has reaction",
			userID:     "user123",
			entityType: "photo",
			entityID:   "photo456",
			mockResult: "LIKE",
			wantResult: "LIKE",
			wantErr:    false,
		},
		{
			name:       "user has no reaction",
			userID:     "user123",
			entityType: "photo",
			entityID:   "photo789",
			mockResult: "",
			mockErr:    ErrReactionNotFound,
			wantResult: "",
			wantErr:    false,
		},
		{
			name:            "empty user ID",
			userID:          "",
			entityType:      "photo",
			entityID:        "photo456",
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:            "storage error",
			userID:          "user123",
			entityType:      "photo",
			entityID:        "photo456",
			mockErr:         errors.New("storage error"),
			wantErr:         true,
			expectedErrType: ErrStorageUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.getUserReactionResult = tt.mockResult
			mock.getUserReactionErr = tt.mockErr

			got, err := client.GetUserReaction(ctx, tt.userID, tt.entityType, tt.entityID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserReaction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantResult {
				t.Errorf("GetUserReaction() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestClient_HasUserReaction(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name       string
		mockResult string
		want       bool
	}{
		{
			name:       "has reaction",
			mockResult: "LIKE",
			want:       true,
		},
		{
			name:       "no reaction",
			mockResult: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.getUserReactionResult = tt.mockResult

			got, err := client.HasUserReaction(ctx, "user123", "photo", "photo456")
			if err != nil {
				t.Errorf("HasUserReaction() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("HasUserReaction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetUserReactionCounts(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name            string
		userID          string
		entityTypeFilter string
		mockResult      map[string]int64
		mockErr         error
		wantResult      map[string]int64
		wantErr         bool
		expectedErrType error
	}{
		{
			name:       "user with reactions",
			userID:     "user123",
			mockResult: map[string]int64{"LIKE": 10, "LOVE": 5},
			wantResult: map[string]int64{"LIKE": 10, "LOVE": 5},
			wantErr:    false,
		},
		{
			name:            "empty user ID",
			userID:          "",
			mockResult:      map[string]int64{},
			wantErr:         true,
			expectedErrType: ErrInvalidInput,
		},
		{
			name:    "storage error",
			userID:  "user123",
			mockErr: errors.New("storage error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.getUserReactionCountsResult = tt.mockResult
			mock.getUserReactionCountsErr = tt.mockErr

			got, err := client.GetUserReactionCounts(ctx, tt.userID, tt.entityTypeFilter)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserReactionCounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.expectedErrType != nil {
				if !errors.Is(err, tt.expectedErrType) {
					t.Errorf("GetUserReactionCounts() error type = %T, want %T", err, tt.expectedErrType)
				}
			}
			if !tt.wantErr {
				for rt, want := range tt.wantResult {
					if got[rt] != want {
						t.Errorf("GetUserReactionCounts() counts[%s] = %v, want %v", rt, got[rt], want)
					}
				}
			}
		})
	}
}

func TestClient_HasUserReactionType(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name         string
		mockResult   string
		reactionType string
		want         bool
		wantErr      bool
	}{
		{
			name:         "has specific reaction type",
			mockResult:   "LIKE",
			reactionType: "LIKE",
			want:         true,
			wantErr:      false,
		},
		{
			name:         "has different reaction type",
			mockResult:   "LIKE",
			reactionType: "LOVE",
			want:         false,
			wantErr:      false,
		},
		{
			name:         "no reaction",
			mockResult:   "",
			reactionType: "LIKE",
			want:         false,
			wantErr:      false,
		},
		{
			name:         "invalid reaction type",
			mockResult:   "LIKE",
			reactionType: "INVALID",
			want:         false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.getUserReactionResult = tt.mockResult

			got, err := client.HasUserReactionType(ctx, "user123", "photo", "photo456", tt.reactionType)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasUserReactionType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("HasUserReactionType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_GetEntityReactionCounts(t *testing.T) {
	ctx := context.Background()
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE", "LOVE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name           string
		mockCounts     map[string]int64
		mockTotal      int64
		mockErr        error
		wantCounts     map[string]int64
		wantTotal      int64
		wantErr        bool
	}{
		{
			name:       "entity with reactions",
			mockCounts: map[string]int64{"LIKE": 10, "DISLIKE": 2, "LOVE": 5},
			mockTotal:  17,
			wantCounts: map[string]int64{"LIKE": 10, "DISLIKE": 2, "LOVE": 5},
			wantTotal:  17,
			wantErr:    false,
		},
		{
			name:       "entity with no reactions",
			mockCounts: map[string]int64{},
			mockTotal:  0,
			wantCounts: map[string]int64{"LIKE": 0, "DISLIKE": 0, "LOVE": 0},
			wantTotal:  0,
			wantErr:    false,
		},
		{
			name:    "storage error",
			mockErr: errors.New("storage error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.getEntityCountsResult = EntityCounts{
				Counts: tt.mockCounts,
				Total:  tt.mockTotal,
			}
			mock.getEntityCountsErr = tt.mockErr

			gotCounts, gotTotal, err := client.GetEntityReactionCounts(ctx, "photo", "photo456")

			if (err != nil) != tt.wantErr {
				t.Errorf("GetEntityReactionCounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotTotal != tt.wantTotal {
					t.Errorf("GetEntityReactionCounts() total = %v, want %v", gotTotal, tt.wantTotal)
				}
				for rt, want := range tt.wantCounts {
					if gotCounts[rt] != want {
						t.Errorf("GetEntityReactionCounts() counts[%s] = %v, want %v", rt, gotCounts[rt], want)
					}
				}
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	validConfig := Config{
		ReactionTypes: []string{"LIKE"},
		Cache:         CacheConfig{Enabled: false},
	}

	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{
			name:    "close succeeds",
			wantErr: false,
		},
		{
			name:    "close with storage error",
			mockErr: errors.New("storage close failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, mock := newTestClient(t, validConfig)
			mock.closeErr = tt.mockErr

			err := client.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify client is closed
			if !client.isClosed() {
				t.Error("Close() did not mark client as closed")
			}

			// Verify operations fail after close
			_, err = client.GetUserReaction(context.Background(), "user", "photo", "123")
			if !errors.Is(err, ErrClientClosed) {
				t.Errorf("operation after Close() error = %v, want ErrClientClosed", err)
			}

			// Verify Close is idempotent
			err = client.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("second Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ConcurrentAccess(t *testing.T) {
	validConfig := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE"},
		Cache:         CacheConfig{Enabled: false},
	}

	client, mock := newTestClient(t, validConfig)
	mock.getUserReactionResult = "LIKE"

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 100; i++ {
			_, _ = client.GetUserReaction(ctx, "user", "photo", "123")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_, _ = client.HasUserReaction(ctx, "user", "photo", "123")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_, _, _ = client.GetEntityReactionCounts(ctx, "photo", "123")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for concurrent operations")
		}
	}
}

func TestPagination_Validate(t *testing.T) {
	defaultConfig := DefaultPaginationConfig()

	tests := []struct {
		name    string
		p       Pagination
		wantErr bool
	}{
		{
			name:    "valid pagination",
			p:       Pagination{Limit: 25, Offset: 0},
			wantErr: false,
		},
		{
			name:    "negative limit",
			p:       Pagination{Limit: -1, Offset: 0},
			wantErr: true,
		},
		{
			name:    "negative offset",
			p:       Pagination{Limit: 25, Offset: -1},
			wantErr: true,
		},
		{
			name:    "limit exceeds max",
			p:       Pagination{Limit: defaultConfig.MaxLimit + 1, Offset: 0},
			wantErr: true,
		},
		{
			name:    "offset exceeds max",
			p:       Pagination{Limit: 25, Offset: defaultConfig.MaxOffset + 1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate(pagination.Config{
				DefaultLimit: defaultConfig.DefaultLimit,
				MaxLimit:     defaultConfig.MaxLimit,
				MaxOffset:    defaultConfig.MaxOffset,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewPaginatedResult(t *testing.T) {
	items := []UserReaction{
		{UserID: "user1", ReactionType: "LIKE"},
		{UserID: "user2", ReactionType: "LOVE"},
		{UserID: "user3", ReactionType: "LIKE"},
	}

	result := NewPaginatedResult(items, 100, 3, 0)

	if result.Total != 100 {
		t.Errorf("Total = %d, want 100", result.Total)
	}
	if len(result.Items) != 3 {
		t.Errorf("len(Items) = %d, want 3", len(result.Items))
	}
	if result.TotalPages != 34 {
		t.Errorf("TotalPages = %d, want 34", result.TotalPages)
	}
	if result.CurrentPage != 1 {
		t.Errorf("CurrentPage = %d, want 1", result.CurrentPage)
	}
	if !result.HasNext {
		t.Error("HasNext should be true")
	}
	if result.HasPrev {
		t.Error("HasPrev should be false")
	}
}

func TestEntityTarget_String(t *testing.T) {
	target := EntityTarget{EntityType: "photo", EntityID: "123"}
	want := "photo:123"
	if got := target.String(); got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestEntityTarget_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		target EntityTarget
		want   bool
	}{
		{
			name:   "valid target",
			target: EntityTarget{EntityType: "photo", EntityID: "123"},
			want:   true,
		},
		{
			name:   "empty entity type",
			target: EntityTarget{EntityType: "", EntityID: "123"},
			want:   false,
		},
		{
			name:   "empty entity ID",
			target: EntityTarget{EntityType: "photo", EntityID: ""},
			want:   false,
		},
		{
			name:   "both empty",
			target: EntityTarget{EntityType: "", EntityID: ""},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.target.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_CacheIntegration(t *testing.T) {
	ctx := context.Background()
	configWithCache := Config{
		ReactionTypes: []string{"LIKE", "DISLIKE"},
		Cache: CacheConfig{
			Enabled:         true,
			UserReactionTTL: time.Hour,
			MaxEntries:      100,
		},
	}

	client, mock := newTestClient(t, configWithCache)

	// First call should hit storage
	mock.getUserReactionResult = "LIKE"
	_, _ = client.GetUserReaction(ctx, "user123", "photo", "photo456")
	if !mock.getUserReactionCalled {
		t.Error("First call should hit storage")
	}

	// Reset mock
	mock.getUserReactionCalled = false

	// Second call should hit cache
	_, _ = client.GetUserReaction(ctx, "user123", "photo", "photo456")
	if mock.getUserReactionCalled {
		t.Error("Second call should hit cache, not storage")
	}
}

// TestCircuitBreaker_InitialState verifies circuit starts closed.
func TestCircuitBreaker_InitialState(t *testing.T) {
	client, _ := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}})
	if got := client.CircuitBreakerState(); got != resilience.StateClosed {
		t.Errorf("initial circuit state = %v, want %v", got, resilience.StateClosed)
	}
}

// TestCircuitBreaker_OpensAfterFailureThreshold verifies Closed → Open transition.
func TestCircuitBreaker_OpensAfterFailureThreshold(t *testing.T) {
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}})
	storageErr := errors.New("storage unavailable")
	mock.addReactionErr = storageErr

	ctx := context.Background()
	// Default failure threshold is 5 — trigger it.
	for i := 0; i < 5; i++ {
		_, _ = client.AddReaction(ctx, "user1", "post", "post1", "LIKE")
	}

	if got := client.CircuitBreakerState(); got != resilience.StateOpen {
		t.Errorf("after 5 failures circuit state = %v, want %v", got, resilience.StateOpen)
	}
}

// TestCircuitBreaker_ReturnsStorageUnavailableWhenOpen verifies ErrStorageUnavailable is returned
// when the circuit is open.
func TestCircuitBreaker_ReturnsStorageUnavailableWhenOpen(t *testing.T) {
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}})
	storageErr := errors.New("storage down")
	mock.addReactionErr = storageErr

	ctx := context.Background()
	// Trip the circuit.
	for i := 0; i < 5; i++ {
		_, _ = client.AddReaction(ctx, "user1", "post", "post1", "LIKE")
	}

	// Next call should return ErrStorageUnavailable without reaching storage.
	mock.addReactionErr = nil
	_, err := client.AddReaction(ctx, "user1", "post", "post1", "LIKE")
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Errorf("open circuit error = %v, want ErrStorageUnavailable", err)
	}
}

// testCounter is a simple counter for testing metrics instrumentation.
type testCounter struct {
	value int64
}

func (c *testCounter) Inc()              { c.value++ }
func (c *testCounter) IncBy(d int64)     { c.value += d }
func (c *testCounter) Value() int64      { return c.value }

// testHistogram is a simple histogram for testing metrics instrumentation.
type testHistogram struct {
	count int64
}

func (h *testHistogram) Record(v float64)      { h.count++ }
func (h *testHistogram) Count() int64          { return h.count }
func (h *testHistogram) Sum() float64          { return 0 }
func (h *testHistogram) Observe([]float64)     {}

// testMetrics records which metric names were used.
type testMetrics struct {
	counters   map[string]*testCounter
	histograms map[string]*testHistogram
}

func newTestMetrics() *testMetrics {
	return &testMetrics{
		counters:   make(map[string]*testCounter),
		histograms: make(map[string]*testHistogram),
	}
}

func (m *testMetrics) Counter(name string, labels metrics.Labels) metrics.Counter {
	if _, ok := m.counters[name]; !ok {
		m.counters[name] = &testCounter{}
	}
	return m.counters[name]
}

func (m *testMetrics) Histogram(name string, buckets []float64, labels metrics.Labels) metrics.Histogram {
	if _, ok := m.histograms[name]; !ok {
		m.histograms[name] = &testHistogram{}
	}
	return m.histograms[name]
}

func (m *testMetrics) Close() error { return nil }

// TestMetrics_AddReactionRecordsLatency verifies AddReaction records a latency observation.
func TestMetrics_AddReactionRecordsLatency(t *testing.T) {
	col := newTestMetrics()
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}, Metrics: col})
	mock.addReactionResult = false

	ctx := context.Background()
	_, err := client.AddReaction(ctx, "user1", "post", "post1", "LIKE")
	if err != nil {
		t.Fatalf("AddReaction error: %v", err)
	}

	h, ok := col.histograms[metrics.OperationLatency]
	if !ok || h.count == 0 {
		t.Error("AddReaction should record latency histogram observation")
	}
}

// TestMetrics_ErrorIncrements verifies that storage errors increment the error counter.
func TestMetrics_ErrorIncrements(t *testing.T) {
	col := newTestMetrics()
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}, Metrics: col})
	mock.addReactionErr = errors.New("storage down")

	ctx := context.Background()
	_, _ = client.AddReaction(ctx, "user1", "post", "post1", "LIKE")

	c, ok := col.counters[metrics.OperationErrors]
	if !ok || c.value == 0 {
		t.Error("AddReaction storage error should increment error counter")
	}
}

// TestMetrics_CacheHitMissCounters verifies cache hit and miss counters.
func TestMetrics_CacheHitMissCounters(t *testing.T) {
	col := newTestMetrics()
	configWithCacheAndMetrics := Config{
		ReactionTypes: []string{"LIKE"},
		Cache:         DefaultCacheConfig(),
		Metrics:       col,
	}
	client, mock := newTestClient(t, configWithCacheAndMetrics)
	mock.getUserReactionResult = "LIKE"

	ctx := context.Background()
	// First call: cache miss → storage
	client.GetUserReaction(ctx, "user1", "post", "post1") //nolint:errcheck

	miss, _ := col.counters[metrics.CacheMisses]
	if miss == nil || miss.value == 0 {
		t.Error("First GetUserReaction should increment cache miss counter")
	}

	// Second call: cache hit
	client.GetUserReaction(ctx, "user1", "post", "post1") //nolint:errcheck

	hit, _ := col.counters[metrics.CacheHits]
	if hit == nil || hit.value == 0 {
		t.Error("Second GetUserReaction should increment cache hit counter")
	}
}

// captureLogger records log entries for assertions.
type captureLogger struct {
	infos  []string
	errors []string
	fields logging.Fields
}

func (l *captureLogger) Debug(msg string, fields logging.Fields)         {}
func (l *captureLogger) Warn(msg string, fields logging.Fields)          {}
func (l *captureLogger) Info(msg string, fields logging.Fields)          { l.infos = append(l.infos, msg) }
func (l *captureLogger) Error(msg string, fields logging.Fields)         { l.errors = append(l.errors, msg) }
func (l *captureLogger) WithFields(fields logging.Fields) logging.Logger { l.fields = fields; return l }
func (l *captureLogger) WithContext(ctx context.Context) logging.Logger  { return l }

// TestLogging_AddReactionLogsInfo verifies that a successful AddReaction emits an INFO log.
func TestLogging_AddReactionLogsInfo(t *testing.T) {
	lg := &captureLogger{}
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}, Logger: lg})
	mock.addReactionResult = false

	ctx := context.Background()
	_, err := client.AddReaction(ctx, "user1", "post", "post1", "LIKE")
	if err != nil {
		t.Fatalf("AddReaction error: %v", err)
	}

	if len(lg.infos) == 0 || lg.infos[0] != "reaction added" {
		t.Errorf("expected INFO log 'reaction added', got %v", lg.infos)
	}
}

// TestLogging_AddReactionLogsError verifies that a failed AddReaction emits an ERROR log.
func TestLogging_AddReactionLogsError(t *testing.T) {
	lg := &captureLogger{}
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}, Logger: lg})
	mock.addReactionErr = errors.New("storage down")

	ctx := context.Background()
	_, _ = client.AddReaction(ctx, "user1", "post", "post1", "LIKE")

	if len(lg.errors) == 0 || lg.errors[0] != "add_reaction failed" {
		t.Errorf("expected ERROR log 'add_reaction failed', got %v", lg.errors)
	}
}

// pingableStorage wraps mockStorage to implement the Pinger interface.
type pingableStorage struct {
	mockStorage
	pingErr error
}

func (p *pingableStorage) Ping(_ context.Context) error { return p.pingErr }

// newHealthClient builds a minimal Client suitable for Health() tests.
func newHealthClient(t *testing.T) *Client {
	t.Helper()
	client, err := New(Config{ReactionTypes: []string{"LIKE"}})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return client
}

// TestHealth_StorageUp verifies that Health() reports StatusUp when Ping succeeds.
func TestHealth_StorageUp(t *testing.T) {
	client := newHealthClient(t)
	client.storage = &pingableStorage{}

	h := client.Health(context.Background())
	if h.Storage.Status != StatusUp {
		t.Errorf("storage status = %v, want UP", h.Storage.Status)
	}
	if h.Overall != StatusUp {
		t.Errorf("overall status = %v, want UP", h.Overall)
	}
}

// TestHealth_StorageDown verifies that Health() reports StatusDown when Ping fails.
func TestHealth_StorageDown(t *testing.T) {
	client := newHealthClient(t)
	client.storage = &pingableStorage{pingErr: errors.New("connection refused")}

	h := client.Health(context.Background())
	if h.Storage.Status != StatusDown {
		t.Errorf("storage status = %v, want DOWN", h.Storage.Status)
	}
	if h.Overall != StatusDown {
		t.Errorf("overall status = %v, want DOWN", h.Overall)
	}
}

// TestHealth_NoStorage verifies Health() reports DOWN when storage is nil.
func TestHealth_NoStorage(t *testing.T) {
	client := newHealthClient(t)
	// storage is nil by default after New()

	h := client.Health(context.Background())
	if h.Storage.Status != StatusDown {
		t.Errorf("nil storage status = %v, want DOWN", h.Storage.Status)
	}
}

// TestRetry_TransientErrorIsRetried verifies that transient storage errors trigger retries.
func TestRetry_TransientErrorIsRetried(t *testing.T) {
	transientErr := errors.New("connection reset by peer")

	// Use zero backoff to make the test fast.
	rp := resilience.RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 0,
		MaxBackoff:     0,
		Jitter:         0,
	}
	client, mock := newTestClient(t, Config{
		ReactionTypes: []string{"LIKE"},
		RetryPolicy:   &rp,
	})
	mock.addReactionFn = func(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error) {
		return false, transientErr
	}

	_, err := client.AddReaction(context.Background(), "user1", "post", "p1", "LIKE")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if mock.addReactionCallCount != 3 {
		t.Errorf("expected 3 attempts, got %d", mock.addReactionCallCount)
	}
}

// TestRetry_NonRetryableErrorNotRetried verifies that ErrReactionNotFound is not retried.
// Uses a custom addReactionFn that returns ErrReactionNotFound and counts calls.
// Even though AddReaction wraps the error as ErrStorageUnavailable, the retry layer
// must NOT retry ErrReactionNotFound, so only 1 storage call is made.
func TestRetry_NonRetryableErrorNotRetried(t *testing.T) {
	rp := resilience.RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 0,
		MaxBackoff:     0,
		Jitter:         0,
	}
	client, mock := newTestClient(t, Config{
		ReactionTypes: []string{"LIKE"},
		RetryPolicy:   &rp,
	})
	mock.addReactionFn = func(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error) {
		return false, ErrReactionNotFound
	}

	_, err := client.AddReaction(context.Background(), "user1", "post", "p1", "LIKE")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// ErrReactionNotFound is non-retryable — only 1 storage attempt should be made.
	if mock.addReactionCallCount != 1 {
		t.Errorf("expected 1 attempt (no retry for ErrReactionNotFound), got %d", mock.addReactionCallCount)
	}
}

// TestRetry_ContextCancellationStopsRetry verifies that context cancellation stops the retry loop.
func TestRetry_ContextCancellationStopsRetry(t *testing.T) {
	rp := resilience.RetryPolicy{
		MaxAttempts:    10,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Jitter:         0,
	}
	client, mock := newTestClient(t, Config{
		ReactionTypes: []string{"LIKE"},
		RetryPolicy:   &rp,
	})
	mock.addReactionFn = func(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error) {
		return false, errors.New("transient")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	_, err := client.AddReaction(ctx, "user1", "post", "p1", "LIKE")
	if err == nil {
		t.Fatal("expected error due to context cancellation")
	}
	// Should not have exhausted all 10 retries within 80ms.
	if mock.addReactionCallCount >= 10 {
		t.Errorf("expected context to stop retries before 10 attempts, got %d", mock.addReactionCallCount)
	}
}

// captureTracer records spans created during operations.
type captureTracer struct {
	spans []*captureSpan
}

type captureSpan struct {
	name   string
	attrs  tracing.Attributes
	ended  bool
	errors []error
}

func (t *captureTracer) Start(ctx context.Context, name string, attrs tracing.Attributes) (context.Context, tracing.Span) {
	span := &captureSpan{name: name, attrs: attrs}
	t.spans = append(t.spans, span)
	return ctx, span
}

func (s *captureSpan) End()                   { s.ended = true }
func (s *captureSpan) RecordError(err error)  { s.errors = append(s.errors, err) }
func (s *captureSpan) SetAttribute(k, v string) {
	if s.attrs == nil {
		s.attrs = make(tracing.Attributes)
	}
	s.attrs[k] = v
}

// TestTracing_AddReactionCreatesSpan verifies AddReaction creates a span with correct attributes.
func TestTracing_AddReactionCreatesSpan(t *testing.T) {
	tr := &captureTracer{}
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}, Tracer: tr})
	mock.addReactionResult = false

	_, err := client.AddReaction(context.Background(), "user1", "post", "p1", "LIKE")
	if err != nil {
		t.Fatalf("AddReaction error: %v", err)
	}

	if len(tr.spans) == 0 {
		t.Fatal("expected at least one span to be created")
	}
	span := tr.spans[0]
	if span.name != "golikeit.AddReaction" {
		t.Errorf("span name = %q, want %q", span.name, "golikeit.AddReaction")
	}
	if !span.ended {
		t.Error("span was not ended")
	}
	if span.attrs["user_id"] != "user1" {
		t.Errorf("user_id attr = %q, want %q", span.attrs["user_id"], "user1")
	}
	if span.attrs["entity_type"] != "post" {
		t.Errorf("entity_type attr = %q, want %q", span.attrs["entity_type"], "post")
	}
}

// TestTracing_RemoveReactionCreatesSpan verifies RemoveReaction creates a span.
func TestTracing_RemoveReactionCreatesSpan(t *testing.T) {
	tr := &captureTracer{}
	client, mock := newTestClient(t, Config{ReactionTypes: []string{"LIKE"}, Tracer: tr})
	mock.removeReactionErr = nil

	err := client.RemoveReaction(context.Background(), "user1", "post", "p1")
	if err != nil {
		t.Fatalf("RemoveReaction error: %v", err)
	}

	if len(tr.spans) == 0 {
		t.Fatal("expected span to be created")
	}
	if tr.spans[0].name != "golikeit.RemoveReaction" {
		t.Errorf("span name = %q, want %q", tr.spans[0].name, "golikeit.RemoveReaction")
	}
}

// TestTracing_NoopTracerIsDefault verifies that no tracer config results in NoopTracer.
func TestTracing_NoopTracerIsDefault(t *testing.T) {
	client, err := New(Config{ReactionTypes: []string{"LIKE"}})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if _, ok := client.tracer.(tracing.NoopTracer); !ok {
		t.Errorf("default tracer = %T, want tracing.NoopTracer", client.tracer)
	}
}

package golikeit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/pagination"
)

// mockStorage is a test double for ReactionStorage.
type mockStorage struct {
	addReactionCalled     bool
	removeReactionCalled  bool
	getUserReactionCalled bool
	getEntityCountsCalled bool

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

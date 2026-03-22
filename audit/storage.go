package audit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Storage defines the interface for audit log persistence.
// Implementations handle the actual storage and retrieval of audit entries.
type Storage interface {
	// Insert persists an audit entry and assigns it a unique ID.
	Insert(ctx context.Context, entry Entry) (Entry, error)

	// GetByUser retrieves entries for a specific user.
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]Entry, error)

	// GetByEntity retrieves entries for a specific entity.
	GetByEntity(ctx context.Context, entityType, entityID string, limit, offset int) ([]Entry, error)

	// GetByOperation retrieves entries of a specific operation type.
	GetByOperation(ctx context.Context, operation Operation, limit, offset int) ([]Entry, error)

	// GetByDateRange retrieves entries within a time range.
	GetByDateRange(ctx context.Context, start, end time.Time, limit, offset int) ([]Entry, error)
}

// InMemoryStorage is an in-memory implementation of Storage.
// Suitable for testing and development. Not intended for production use.
// Thread-safe via sync.RWMutex.
type InMemoryStorage struct {
	mu      sync.RWMutex
	entries []Entry
	nextID  int64
}

// NewInMemoryStorage creates a new in-memory storage instance.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		entries: make([]Entry, 0),
		nextID:  1,
	}
}

// Insert stores an audit entry and assigns it a unique ID.
func (s *InMemoryStorage) Insert(_ context.Context, entry Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry.ID = fmt.Sprintf("audit_%d", s.nextID)
	s.nextID++

	s.entries = append(s.entries, entry)
	return entry, nil
}

// GetByUser retrieves entries for a specific user, ordered by timestamp (most recent first).
func (s *InMemoryStorage) GetByUser(_ context.Context, userID string, limit, offset int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Entry
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].UserID == userID {
			results = append(results, s.entries[i])
		}
	}

	return paginate(results, limit, offset), nil
}

// GetByEntity retrieves entries for a specific entity, ordered by timestamp (most recent first).
func (s *InMemoryStorage) GetByEntity(_ context.Context, entityType, entityID string, limit, offset int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Entry
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].EntityType == entityType && s.entries[i].EntityID == entityID {
			results = append(results, s.entries[i])
		}
	}

	return paginate(results, limit, offset), nil
}

// GetByOperation retrieves entries of a specific operation type, ordered by timestamp (most recent first).
func (s *InMemoryStorage) GetByOperation(_ context.Context, operation Operation, limit, offset int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Entry
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].Operation == operation {
			results = append(results, s.entries[i])
		}
	}

	return paginate(results, limit, offset), nil
}

// GetByDateRange retrieves entries within a time range, ordered by timestamp (most recent first).
func (s *InMemoryStorage) GetByDateRange(_ context.Context, start, end time.Time, limit, offset int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []Entry
	for i := len(s.entries) - 1; i >= 0; i-- {
		ts := s.entries[i].Timestamp
		if (ts.Equal(start) || ts.After(start)) && (ts.Equal(end) || ts.Before(end)) {
			results = append(results, s.entries[i])
		}
	}

	return paginate(results, limit, offset), nil
}

// paginate applies pagination to a slice of entries.
func paginate(entries []Entry, limit, offset int) []Entry {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(entries) {
		return []Entry{}
	}

	end := offset + limit
	if limit <= 0 || end > len(entries) {
		end = len(entries)
	}

	return entries[offset:end]
}

// Ensure InMemoryStorage implements Storage.
var _ Storage = (*InMemoryStorage)(nil)

// Package pagination provides limit-offset pagination functionality for list queries.
// It supports configurable defaults, validation, and result metadata calculation.
package pagination

import (
	"fmt"
)

// Pagination represents pagination parameters for list queries.
// Use Validate() to ensure parameters are within acceptable bounds.
type Pagination struct {
	// Limit is the maximum number of items to return.
	// Must be non-negative and not exceed MaxLimit.
	Limit int

	// Offset is the number of items to skip.
	// Must be non-negative and not exceed MaxOffset.
	Offset int
}

// Config holds configuration for pagination behavior.
type Config struct {
	// DefaultLimit is the default number of items per page when Limit is 0.
	DefaultLimit int

	// MaxLimit is the maximum allowed items per page.
	MaxLimit int

	// MaxOffset is the maximum allowed offset.
	MaxOffset int
}

// DefaultConfig returns a Config with sensible defaults.
// DefaultLimit: 25
// MaxLimit: 100
// MaxOffset: 100
func DefaultConfig() Config {
	return Config{
		DefaultLimit: 25,
		MaxLimit:     100,
		MaxOffset:    100,
	}
}

// Result is a generic container for paginated query results.
// It provides metadata to help clients navigate through pages.
type Result[T any] struct {
	// Items is the slice of result items for the current page.
	Items []T

	// Total is the total number of items matching the query (across all pages).
	Total int64

	// TotalPages is the total number of pages.
	TotalPages int

	// CurrentPage is the 1-based current page number.
	CurrentPage int

	// Limit is the page size used for this query.
	Limit int

	// Offset is the offset used for this query.
	Offset int

	// HasNext indicates if there are more results after this page.
	HasNext bool

	// HasPrev indicates if there are results before this page.
	HasPrev bool
}

// NewResult creates a new paginated result with calculated pagination info.
//
// Example:
//
//	items := []User{user1, user2, user3}
//	result := NewResult(items, 100, 25, 0)
//	// result.TotalPages = 4
//	// result.CurrentPage = 1
//	// result.HasNext = true
//	// result.HasPrev = false
func NewResult[T any](items []T, total int64, limit, offset int) Result[T] {
	if limit <= 0 {
		limit = DefaultConfig().DefaultLimit
	}

	// Calculate total pages
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages < 1 {
		totalPages = 1
	}

	// Calculate current page (1-based)
	currentPage := offset/limit + 1

	return Result[T]{
		Items:       items,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: currentPage,
		Limit:       limit,
		Offset:      offset,
		HasNext:     offset+len(items) < int(total),
		HasPrev:     offset > 0,
	}
}

// Validate validates pagination parameters against the provided configuration.
// Returns an error if parameters are invalid or out of bounds.
//
// Validation rules:
// - Limit must be non-negative
// - Offset must be non-negative
// - If Limit is 0, it will be set to DefaultLimit (validation passes)
// - Limit must not exceed MaxLimit
// - Offset must not exceed MaxOffset
func (p *Pagination) Validate(cfg Config) error {
	if p.Limit < 0 {
		return fmt.Errorf("limit cannot be negative: %d", p.Limit)
	}
	if p.Offset < 0 {
		return fmt.Errorf("offset cannot be negative: %d", p.Offset)
	}
	if p.Limit == 0 {
		p.Limit = cfg.DefaultLimit
	}
	if p.Limit > cfg.MaxLimit {
		return fmt.Errorf("limit exceeds maximum: %d > %d", p.Limit, cfg.MaxLimit)
	}
	if p.Offset > cfg.MaxOffset {
		return fmt.Errorf("offset exceeds maximum: %d > %d", p.Offset, cfg.MaxOffset)
	}
	return nil
}

// ValidateWithDefaults validates pagination using the default configuration.
// This is a convenience method for common use cases.
func (p *Pagination) ValidateWithDefaults() error {
	return p.Validate(DefaultConfig())
}

// ValidatePagination is a helper function that validates a Pagination value with the given Config.
// This is useful when you have a Pagination value (not a pointer) and need to validate it.
func ValidatePagination(p Pagination, cfg Config) error {
	return p.Validate(cfg)
}

// NextPage returns pagination parameters for the next page.
// Returns nil if there is no next page.
func (p Pagination) NextPage(total int64) *Pagination {
	nextOffset := p.Offset + p.Limit
	if nextOffset >= int(total) {
		return nil
	}
	return &Pagination{
		Limit:  p.Limit,
		Offset: nextOffset,
	}
}

// PrevPage returns pagination parameters for the previous page.
// Returns nil if there is no previous page.
func (p Pagination) PrevPage() *Pagination {
	if p.Offset <= 0 {
		return nil
	}
	prevOffset := p.Offset - p.Limit
	if prevOffset < 0 {
		prevOffset = 0
	}
	return &Pagination{
		Limit:  p.Limit,
		Offset: prevOffset,
	}
}

// CalculateLimit returns the effective limit, applying defaults if necessary.
func (p Pagination) CalculateLimit(defaultLimit int) int {
	if p.Limit <= 0 {
		return defaultLimit
	}
	return p.Limit
}

// CalculateOffset returns the effective offset, ensuring it's non-negative.
func (p Pagination) CalculateOffset() int {
	if p.Offset < 0 {
		return 0
	}
	return p.Offset
}

// IsEmpty returns true if the pagination has zero limit and offset.
func (p Pagination) IsEmpty() bool {
	return p.Limit == 0 && p.Offset == 0
}

// Clone returns a copy of the pagination parameters.
func (p Pagination) Clone() Pagination {
	return Pagination{
		Limit:  p.Limit,
		Offset: p.Offset,
	}
}

// LimitOffset is a convenience type for database queries that use LIMIT/OFFSET.
// It provides the validated limit and offset values.
type LimitOffset struct {
	Limit  int
	Offset int
}

// ToLimitOffset converts pagination to LimitOffset with default handling.
func (p Pagination) ToLimitOffset(cfg Config) LimitOffset {
	limit := p.Limit
	if limit <= 0 {
		limit = cfg.DefaultLimit
	}
	if limit > cfg.MaxLimit {
		limit = cfg.MaxLimit
	}

	offset := p.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > cfg.MaxOffset {
		offset = cfg.MaxOffset
	}

	return LimitOffset{
		Limit:  limit,
		Offset: offset,
	}
}

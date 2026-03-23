// Package config provides configuration structures and validation for the GoLikeit system.
// All configuration is immutable after initialization and validated at startup.
package config

import (
	"fmt"
	"regexp"
	"time"
)

// Default configuration values.
const (
	// DefaultDatabaseType is the default database backend type.
	DefaultDatabaseType = "postgresql"

	// DefaultCacheEnabled is the default state for cache.
	DefaultCacheEnabled = true

	// DefaultUserReactionTTL is the default TTL for user reaction cache entries.
	DefaultUserReactionTTL = 60 * time.Second

	// DefaultEntityCountsTTL is the default TTL for entity counts cache entries.
	DefaultEntityCountsTTL = 300 * time.Second

	// DefaultMaxCacheEntries is the default maximum number of cache entries.
	DefaultMaxCacheEntries = 10000

	// DefaultPaginationLimit is the default pagination limit.
	DefaultPaginationLimit = 25

	// MaxPaginationLimit is the maximum allowed pagination limit.
	MaxPaginationLimit = 100

	// MaxPaginationOffset is the maximum allowed pagination offset.
	MaxPaginationOffset = 100
)

// Validation patterns.
var (
	// reactionTypePattern matches uppercase alphanumeric with underscores and hyphens.
	reactionTypePattern = regexp.MustCompile(`^[A-Z0-9_-]+$`)
)

// Config holds the complete configuration for the GoLikeit system.
// This configuration is validated at initialization and immutable after.
type Config struct {
	// Database holds database connection configuration.
	Database DatabaseConfig

	// Cache holds cache configuration (optional, defaults enabled).
	Cache CacheConfig

	// Events holds event system configuration (optional).
	Events EventsConfig

	// RateLimit holds rate limiting configuration (optional).
	RateLimit RateLimitConfig

	// ReactionTypes is the list of allowed reaction types (required).
	// Each type must match the pattern ^[A-Z0-9_-]+$.
	ReactionTypes []string

	// Pagination holds pagination configuration (optional).
	Pagination PaginationConfig
}

// Validate validates the complete configuration.
// Returns an error if any part of the configuration is invalid.
func (c Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	if err := c.Cache.Validate(); err != nil {
		return fmt.Errorf("cache config: %w", err)
	}

	if err := c.Events.Validate(); err != nil {
		return fmt.Errorf("events config: %w", err)
	}

	if err := c.RateLimit.Validate(); err != nil {
		return fmt.Errorf("rate limit config: %w", err)
	}

	if err := c.Pagination.Validate(); err != nil {
		return fmt.Errorf("pagination config: %w", err)
	}

	if err := validateReactionTypes(c.ReactionTypes); err != nil {
		return fmt.Errorf("reaction types: %w", err)
	}

	return nil
}

// DatabaseConfig holds configuration for database connections.
type DatabaseConfig struct {
	// Type is the database type (e.g., "postgresql", "mysql", "sqlite", "mongodb", "cassandra").
	Type string

	// Host is the database host (optional for embedded databases).
	Host string

	// Port is the database port (optional for embedded databases).
	Port int

	// Database is the database name.
	Database string

	// Username is the database username.
	Username string

	// Password is the database password.
	Password string

	// ConnectionString is an optional direct connection string.
	// If provided, other fields may be ignored depending on the driver.
	ConnectionString string

	// MaxOpenConns is the maximum number of open connections to the database.
	// Zero means unlimited.
	MaxOpenConns int

	// MaxIdleConns is the maximum number of connections in the idle connection pool.
	// Zero means default (2).
	MaxIdleConns int

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Zero means connections are not closed due to age.
	ConnMaxLifetime time.Duration

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	// Zero means connections are not closed due to idleness.
	ConnMaxIdleTime time.Duration

	// Timeout is the default timeout for database operations.
	Timeout time.Duration
}

// Validate validates the database configuration.
func (dc DatabaseConfig) Validate() error {
	// Type is optional - if not specified, defaults will be used
	if dc.Type != "" {
		validTypes := map[string]bool{
			"postgresql": true,
			"mysql":      true,
			"mariadb":    true,
			"sqlite":     true,
			"redis":      true,
			"mongodb":    true,
			"cassandra":  true,
			"memory":     true,
		}
		if !validTypes[dc.Type] {
			return fmt.Errorf("invalid database type: %q", dc.Type)
		}
	}

	// Port validation (if specified)
	if dc.Port < 0 || dc.Port > 65535 {
		return fmt.Errorf("invalid port: %d", dc.Port)
	}

	// Connection pool validation
	if dc.MaxOpenConns < 0 {
		return fmt.Errorf("max_open_conns cannot be negative")
	}
	if dc.MaxIdleConns < 0 {
		return fmt.Errorf("max_idle_conns cannot be negative")
	}

	return nil
}

// CacheConfig holds configuration for the caching layer.
type CacheConfig struct {
	// Enabled enables or disables caching. Default: true.
	Enabled bool

	// UserReactionTTL is the TTL for cached user reactions. Default: 60s.
	UserReactionTTL time.Duration

	// EntityCountsTTL is the TTL for cached entity counts. Default: 300s.
	EntityCountsTTL time.Duration

	// MaxEntries is the maximum number of cache entries. Default: 10000.
	MaxEntries int

	// EvictionPolicy is the cache eviction policy (currently only "LRU").
	EvictionPolicy string
}

// Validate validates the cache configuration.
func (cc CacheConfig) Validate() error {
	if cc.UserReactionTTL < 0 {
		return fmt.Errorf("user_reaction_ttl cannot be negative")
	}
	if cc.EntityCountsTTL < 0 {
		return fmt.Errorf("entity_counts_ttl cannot be negative")
	}
	if cc.MaxEntries < 0 {
		return fmt.Errorf("max_entries cannot be negative")
	}

	if cc.EvictionPolicy != "" && cc.EvictionPolicy != "LRU" {
		return fmt.Errorf("unsupported eviction policy: %q (only LRU supported)", cc.EvictionPolicy)
	}

	return nil
}

// DefaultCacheConfig returns a CacheConfig with sensible defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:         DefaultCacheEnabled,
		UserReactionTTL: DefaultUserReactionTTL,
		EntityCountsTTL: DefaultEntityCountsTTL,
		MaxEntries:      DefaultMaxCacheEntries,
		EvictionPolicy:  "LRU",
	}
}

// EventsConfig holds configuration for the event system.
type EventsConfig struct {
	// Enabled enables or disables the event system. Default: true.
	Enabled bool

	// AsyncWorkers is the number of async event workers. Default: 5.
	AsyncWorkers int

	// QueueSize is the size of the async event queue. Default: 1000.
	QueueSize int

	// EventTimeout is the timeout for event delivery. Default: 5s.
	EventTimeout time.Duration

	// EnableSyncSubscriptions enables synchronous event subscriptions.
	EnableSyncSubscriptions bool

	// EnableAsyncSubscriptions enables asynchronous event subscriptions.
	EnableAsyncSubscriptions bool
}

// Validate validates the events configuration.
func (ec EventsConfig) Validate() error {
	if ec.AsyncWorkers < 0 {
		return fmt.Errorf("async_workers cannot be negative")
	}
	if ec.QueueSize < 0 {
		return fmt.Errorf("queue_size cannot be negative")
	}
	if ec.EventTimeout < 0 {
		return fmt.Errorf("event_timeout cannot be negative")
	}

	return nil
}

// DefaultEventsConfig returns an EventsConfig with sensible defaults.
func DefaultEventsConfig() EventsConfig {
	return EventsConfig{
		Enabled:                  true,
		AsyncWorkers:             5,
		QueueSize:                1000,
		EventTimeout:             5 * time.Second,
		EnableSyncSubscriptions:  true,
		EnableAsyncSubscriptions: true,
	}
}

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	// Enabled enables or disables rate limiting. Default: false.
	Enabled bool

	// RequestsPerSecond is the rate limit per user. Zero means unlimited.
	RequestsPerSecond int

	// BurstSize is the maximum burst size. Zero means no burst.
	BurstSize int

	// TTL is the time window for rate limit counters. Default: 60s.
	TTL time.Duration
}

// Validate validates the rate limit configuration.
func (rlc RateLimitConfig) Validate() error {
	if rlc.RequestsPerSecond < 0 {
		return fmt.Errorf("requests_per_second cannot be negative")
	}
	if rlc.BurstSize < 0 {
		return fmt.Errorf("burst_size cannot be negative")
	}
	if rlc.TTL < 0 {
		return fmt.Errorf("ttl cannot be negative")
	}

	return nil
}

// DefaultRateLimitConfig returns a RateLimitConfig with sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:           false,
		RequestsPerSecond: 0, // Unlimited by default
		BurstSize:         0,
		TTL:               60 * time.Second,
	}
}

// PaginationConfig holds configuration for pagination behavior.
type PaginationConfig struct {
	// DefaultLimit is the default number of items per page. Default: 25.
	DefaultLimit int

	// MaxLimit is the maximum allowed items per page. Default: 100.
	MaxLimit int

	// MaxOffset is the maximum allowed offset. Default: 100.
	MaxOffset int
}

// Validate validates the pagination configuration.
func (pc PaginationConfig) Validate() error {
	if pc.DefaultLimit <= 0 {
		return fmt.Errorf("default_limit must be positive")
	}
	if pc.MaxLimit <= 0 {
		return fmt.Errorf("max_limit must be positive")
	}
	if pc.MaxOffset < 0 {
		return fmt.Errorf("max_offset cannot be negative")
	}
	if pc.DefaultLimit > pc.MaxLimit {
		return fmt.Errorf("default_limit (%d) cannot exceed max_limit (%d)", pc.DefaultLimit, pc.MaxLimit)
	}

	return nil
}

// DefaultPaginationConfig returns a PaginationConfig with sensible defaults.
func DefaultPaginationConfig() PaginationConfig {
	return PaginationConfig{
		DefaultLimit: DefaultPaginationLimit,
		MaxLimit:     MaxPaginationLimit,
		MaxOffset:    MaxPaginationOffset,
	}
}

// validateReactionTypes validates a slice of reaction types.
// Requirements:
//   - Must contain at least one reaction type
//   - Each reaction type must match pattern ^[A-Z0-9_-]+$
//   - All reaction types must be unique
func validateReactionTypes(types []string) error {
	if len(types) == 0 {
		return fmt.Errorf("at least one reaction type must be configured")
	}

	seen := make(map[string]struct{}, len(types))
	for _, rt := range types {
		if rt == "" {
			return fmt.Errorf("reaction type cannot be empty")
		}
		if len(rt) > 64 {
			return fmt.Errorf("reaction type %q exceeds 64 characters", rt)
		}
		if !reactionTypePattern.MatchString(rt) {
			return fmt.Errorf("reaction type %q must match pattern ^[A-Z0-9_-]+$", rt)
		}
		if _, exists := seen[rt]; exists {
			return fmt.Errorf("duplicate reaction type: %q", rt)
		}
		seen[rt] = struct{}{}
	}

	return nil
}

// NewConfig creates a new Config with the specified reaction types and sensible defaults.
// This is a convenience function for quick setup.
func NewConfig(reactionTypes []string) (Config, error) {
	cfg := Config{
		Database:      DatabaseConfig{},
		Cache:         DefaultCacheConfig(),
		Events:        DefaultEventsConfig(),
		RateLimit:     DefaultRateLimitConfig(),
		Pagination:    DefaultPaginationConfig(),
		ReactionTypes: reactionTypes,
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// MustNewConfig creates a new Config and panics if validation fails.
func MustNewConfig(reactionTypes []string) Config {
	cfg, err := NewConfig(reactionTypes)
	if err != nil {
		panic(err)
	}
	return cfg
}

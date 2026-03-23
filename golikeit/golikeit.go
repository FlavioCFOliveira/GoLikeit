package golikeit

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/FlavioCFOliveira/GoLikeit/audit"
	"github.com/FlavioCFOliveira/GoLikeit/cache"
	"github.com/FlavioCFOliveira/GoLikeit/events"
	pag "github.com/FlavioCFOliveira/GoLikeit/pagination"
)

// reactionTypePattern is the validation pattern for reaction types.
// Matches uppercase letters, digits, underscores, and hyphens.
const reactionTypePattern = `^[A-Z0-9_-]+$`

var reactionTypeRegex = regexp.MustCompile(reactionTypePattern)

// DatabaseConfig holds configuration for database connections.
type DatabaseConfig struct {
	// Type is the database type (e.g., "postgresql", "mysql", "sqlite").
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
	// If provided, other fields are ignored.
	ConnectionString string
}

// CacheConfig holds configuration for the caching layer.
type CacheConfig struct {
	// Enabled enables or disables caching.
	Enabled bool

	// UserReactionTTL is the TTL for cached user reactions.
	UserReactionTTL time.Duration

	// EntityCountsTTL is the TTL for cached entity counts.
	EntityCountsTTL time.Duration

	// MaxEntries is the maximum number of cache entries.
	MaxEntries int

	// EvictionPolicy is the cache eviction policy (currently only "LRU").
	EvictionPolicy string
}

// DefaultCacheConfig returns a CacheConfig with sensible defaults.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:         true,
		UserReactionTTL: 60 * time.Second,
		EntityCountsTTL: 300 * time.Second,
		MaxEntries:      10000,
		EvictionPolicy:  "LRU",
	}
}

// Config holds the complete configuration for the golikeit client.
type Config struct {
	// Database is the database connection configuration.
	Database DatabaseConfig

	// ReactionTypes is the list of allowed reaction types.
	ReactionTypes []string

	// Cache is the cache configuration (optional, defaults to enabled).
	Cache CacheConfig

	// Pagination is the pagination configuration (optional).
	Pagination PaginationConfig

	// Events is the event bus configuration (optional).
	Events events.Config
}

// validateReactionTypes validates the reaction type configuration.
func validateReactionTypes(types []string) error {
	if len(types) == 0 {
		return ErrNoReactionTypes
	}

	seen := make(map[string]struct{}, len(types))
	for _, rt := range types {
		if rt == "" {
			return fmt.Errorf("%w: empty reaction type", ErrInvalidReactionFormat)
		}
		if len(rt) > 64 {
			return fmt.Errorf("%w: reaction type exceeds 64 characters", ErrInvalidReactionFormat)
		}
		if !reactionTypeRegex.MatchString(rt) {
			return fmt.Errorf("%w: %q does not match [A-Z0-9_-]+", ErrInvalidReactionFormat, rt)
		}
		if _, exists := seen[rt]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicateReactionType, rt)
		}
		seen[rt] = struct{}{}
	}

	return nil
}

// ReactionStorage defines the interface for reaction storage operations.
// This interface is implemented by concrete storage backends.
type ReactionStorage interface {
	// AddReaction adds or replaces a reaction for a user.
	// Returns true if a previous reaction was replaced.
	AddReaction(ctx context.Context, userID string, target EntityTarget, reactionType string) (bool, error)

	// RemoveReaction removes a user's reaction.
	// Returns ErrReactionNotFound if no reaction exists.
	RemoveReaction(ctx context.Context, userID string, target EntityTarget) error

	// GetUserReaction retrieves a user's current reaction type.
	// Returns ("", ErrReactionNotFound) if no reaction exists.
	GetUserReaction(ctx context.Context, userID string, target EntityTarget) (string, error)

	// GetEntityCounts retrieves the reaction counts for an entity.
	GetEntityCounts(ctx context.Context, target EntityTarget) (EntityCounts, error)

	// GetUserReactions retrieves all reactions for a user with pagination.
	GetUserReactions(ctx context.Context, userID string, pagination Pagination) ([]UserReaction, int64, error)

	// GetUserReactionCounts retrieves aggregated counts per reaction type for a user.
	GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error)

	// GetUserReactionsByType retrieves reactions of a specific type for a user.
	GetUserReactionsByType(ctx context.Context, userID string, reactionType string, pagination Pagination) ([]UserReaction, int64, error)

	// GetEntityReactions retrieves all reactions on an entity with pagination.
	GetEntityReactions(ctx context.Context, target EntityTarget, pagination Pagination) ([]EntityReaction, int64, error)

	// GetRecentReactions retrieves recent reactions on an entity.
	GetRecentReactions(ctx context.Context, target EntityTarget, limit int) ([]RecentUserReaction, error)

	// GetLastReactionTime retrieves the timestamp of the most recent reaction.
	GetLastReactionTime(ctx context.Context, target EntityTarget) (*time.Time, error)

	// Close releases any resources held by the storage.
	Close() error
}

// Client is the public API client for the reaction system.
// It is safe for concurrent use by multiple goroutines.
type Client struct {
	config          Config
	reactionTypes   map[string]struct{} // Set of valid reaction types
	reactionTypeList []string           // Ordered list for consistent output
	storage         ReactionStorage
	cache           cache.Cache
	eventBus        *events.Bus
	auditor         audit.Auditor
	paginationCfg   PaginationConfig

	closed    bool
	closedMu  sync.RWMutex
	closeOnce sync.Once
	closeErr  error
}

// New creates a new golikeit client with the provided configuration.
// Returns an error if the configuration is invalid.
func New(config Config) (*Client, error) {
	// Validate reaction types
	if err := validateReactionTypes(config.ReactionTypes); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidReactionType, err)
	}

	// Set defaults
	if config.Pagination.DefaultLimit == 0 {
		config.Pagination = DefaultPaginationConfig()
	}
	if config.Cache.EvictionPolicy == "" {
		config.Cache = DefaultCacheConfig()
	}

	// Build reaction type set for O(1) lookups
	reactionTypeSet := make(map[string]struct{}, len(config.ReactionTypes))
	for _, rt := range config.ReactionTypes {
		reactionTypeSet[rt] = struct{}{}
	}

	// Create event bus
	eventBus := events.NewBus(config.Events)

	// Create cache if enabled
	var reactionCache cache.Cache
	if config.Cache.Enabled {
		reactionCache = cache.New(config.Cache.MaxEntries)
	}

	client := &Client{
		config:           config,
		reactionTypes:    reactionTypeSet,
		reactionTypeList: append([]string(nil), config.ReactionTypes...), // Copy
		eventBus:        eventBus,
		cache:           reactionCache,
		paginationCfg:   config.Pagination,
		auditor:         audit.NewNullAuditor(),
	}

	return client, nil
}

// isClosed reports whether the client has been closed.
func (c *Client) isClosed() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return c.closed
}

// checkClosed returns ErrClientClosed if the client is closed.
func (c *Client) checkClosed() error {
	if c.isClosed() {
		return ErrClientClosed
	}
	return nil
}

// validateReactionType checks if a reaction type is valid.
func (c *Client) validateReactionType(reactionType string) error {
	if reactionType == "" {
		return fmt.Errorf("%w: reaction type is empty", ErrInvalidReactionType)
	}
	if _, ok := c.reactionTypes[reactionType]; !ok {
		return fmt.Errorf("%w: %q is not a configured reaction type", ErrInvalidReactionType, reactionType)
	}
	return nil
}

// validateUserID checks if a user ID is valid.
func validateUserID(userID string) error {
	if userID == "" {
		return fmt.Errorf("%w: user ID is empty", ErrInvalidInput)
	}
	if len(userID) > 256 {
		return fmt.Errorf("%w: user ID exceeds 256 characters", ErrInvalidInput)
	}
	return nil
}

// validateEntityTarget checks if an entity target is valid.
func validateEntityTarget(target EntityTarget) error {
	if target.EntityType == "" {
		return fmt.Errorf("%w: entity type is empty", ErrInvalidInput)
	}
	if target.EntityID == "" {
		return fmt.Errorf("%w: entity ID is empty", ErrInvalidInput)
	}
	return nil
}

// InvalidateCacheForTarget invalidates all cache entries related to a target.
func (c *Client) invalidateCacheForTarget(userID string, target EntityTarget) {
	if c.cache == nil {
		return
	}

	// Invalidate user reaction cache
	userKey := fmt.Sprintf("user:%s:%s", userID, target.String())
	c.cache.Delete(userKey)

	// Invalidate entity counts cache
	entityKey := fmt.Sprintf("counts:%s", target.String())
	c.cache.Delete(entityKey)

	// Invalidate entity detail cache
	detailKey := fmt.Sprintf("detail:%s", target.String())
	c.cache.Delete(detailKey)
}

// AddReaction adds or replaces a user's reaction on a target.
// Returns true if a previous reaction was replaced.
func (c *Client) AddReaction(ctx context.Context, userID, entityType, entityID, reactionType string) (bool, error) {
	if err := c.checkClosed(); err != nil {
		return false, err
	}

	if err := validateUserID(userID); err != nil {
		return false, err
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return false, err
	}

	if err := c.validateReactionType(reactionType); err != nil {
		return false, err
	}

	if c.storage == nil {
		return false, ErrStorageUnavailable
	}

	isReplaced, err := c.storage.AddReaction(ctx, userID, target, reactionType)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}

	// Invalidate cache
	c.invalidateCacheForTarget(userID, target)

	// Fire-and-forget audit log — must not block or fail the primary operation.
	auditOp := audit.OperationAdd
	if isReplaced {
		auditOp = audit.OperationReplace
	}
	auditEntry := audit.NewEntry(auditOp, userID, entityType, entityID, reactionType, "")
	go func() { _ = c.auditor.LogOperation(context.Background(), auditEntry) }()

	// Emit event
	c.eventBus.Emit(events.ReactionEvent{
		Event: events.Event{
			Type:      events.TypeReactionAdded,
			Timestamp: time.Now(),
			Version:   1,
		},
		UserID:       userID,
		EntityType:   entityType,
		EntityID:     entityID,
		ReactionType: reactionType,
	})

	return isReplaced, nil
}

// RemoveReaction removes a user's reaction from a target.
// Returns ErrReactionNotFound if no reaction exists.
func (c *Client) RemoveReaction(ctx context.Context, userID, entityType, entityID string) error {
	if err := c.checkClosed(); err != nil {
		return err
	}

	if err := validateUserID(userID); err != nil {
		return err
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return err
	}

	if c.storage == nil {
		return ErrStorageUnavailable
	}

	if err := c.storage.RemoveReaction(ctx, userID, target); err != nil {
		return err
	}

	// Invalidate cache
	c.invalidateCacheForTarget(userID, target)

	// Fire-and-forget audit log — must not block or fail the primary operation.
	auditEntry := audit.NewEntry(audit.OperationRemove, userID, entityType, entityID, "", "")
	go func() { _ = c.auditor.LogOperation(context.Background(), auditEntry) }()

	// Emit event
	c.eventBus.Emit(events.ReactionEvent{
		Event: events.Event{
			Type:      events.TypeReactionRemoved,
			Timestamp: time.Now(),
			Version:   1,
		},
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
	})

	return nil
}

// GetUserReaction retrieves the current reaction type for a user on a target.
// Returns an empty string if no reaction exists.
func (c *Client) GetUserReaction(ctx context.Context, userID, entityType, entityID string) (string, error) {
	if err := c.checkClosed(); err != nil {
		return "", err
	}

	if err := validateUserID(userID); err != nil {
		return "", err
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return "", err
	}

	if c.storage == nil {
		return "", ErrStorageUnavailable
	}

	// Check cache first
	if c.cache != nil {
		cacheKey := fmt.Sprintf("user:%s:%s", userID, target.String())
		if val, ok := c.cache.Get(cacheKey); ok {
			if reactionType, ok := val.(string); ok {
				return reactionType, nil
			}
		}
	}

	reactionType, err := c.storage.GetUserReaction(ctx, userID, target)
	if err != nil {
		if err == ErrReactionNotFound {
			return "", nil
		}
		return "", err
	}

	// Cache the result
	if c.cache != nil {
		cacheKey := fmt.Sprintf("user:%s:%s", userID, target.String())
		c.cache.Set(cacheKey, reactionType, c.config.Cache.UserReactionTTL)
	}

	return reactionType, nil
}

// HasUserReaction checks if a user has any reaction on a target.
func (c *Client) HasUserReaction(ctx context.Context, userID, entityType, entityID string) (bool, error) {
	reactionType, err := c.GetUserReaction(ctx, userID, entityType, entityID)
	if err != nil {
		return false, err
	}
	return reactionType != "", nil
}

// HasUserReactionType checks if a user has a specific reaction type on a target.
func (c *Client) HasUserReactionType(ctx context.Context, userID, entityType, entityID, reactionType string) (bool, error) {
	if err := c.validateReactionType(reactionType); err != nil {
		return false, err
	}

	currentType, err := c.GetUserReaction(ctx, userID, entityType, entityID)
	if err != nil {
		return false, err
	}
	return currentType == reactionType, nil
}

// GetEntityReactionCounts retrieves the counts per reaction type for an entity.
// Returns counts for all configured reaction types (zero for types with no reactions).
func (c *Client) GetEntityReactionCounts(ctx context.Context, entityType, entityID string) (map[string]int64, int64, error) {
	if err := c.checkClosed(); err != nil {
		return nil, 0, err
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return nil, 0, err
	}

	if c.storage == nil {
		return nil, 0, ErrStorageUnavailable
	}

	// Check cache first
	if c.cache != nil {
		cacheKey := fmt.Sprintf("counts:%s", target.String())
		if val, ok := c.cache.Get(cacheKey); ok {
			if counts, ok := val.(EntityCounts); ok {
				// Return copy with all reaction types
				result := make(map[string]int64, len(c.reactionTypeList))
				for _, rt := range c.reactionTypeList {
					result[rt] = counts.Counts[rt]
				}
				return result, counts.Total, nil
			}
		}
	}

	counts, err := c.storage.GetEntityCounts(ctx, target)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result
	if c.cache != nil {
		cacheKey := fmt.Sprintf("counts:%s", target.String())
		c.cache.Set(cacheKey, counts, c.config.Cache.EntityCountsTTL)
	}

	// Build result with all configured reaction types
	result := make(map[string]int64, len(c.reactionTypeList))
	for _, rt := range c.reactionTypeList {
		result[rt] = counts.Counts[rt]
	}

	return result, counts.Total, nil
}

// ReactionDetailOptions controls the level of detail returned by GetEntityReactionDetail.
type ReactionDetailOptions struct {
	// MaxRecentUsers is the maximum number of recent users to include per reaction type.
	// Zero means no recent users are included.
	MaxRecentUsers int
}

// GetEntityReactionDetail retrieves comprehensive reaction information for an entity.
func (c *Client) GetEntityReactionDetail(ctx context.Context, entityType, entityID string, options ReactionDetailOptions) (EntityReactionDetail, error) {
	if err := c.checkClosed(); err != nil {
		return EntityReactionDetail{}, err
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return EntityReactionDetail{}, err
	}

	if c.storage == nil {
		return EntityReactionDetail{}, ErrStorageUnavailable
	}

	// Get counts
	counts, total, err := c.GetEntityReactionCounts(ctx, entityType, entityID)
	if err != nil {
		return EntityReactionDetail{}, err
	}

	detail := EntityReactionDetail{
		EntityType:     entityType,
		EntityID:       entityID,
		TotalReactions: total,
		CountsByType:   counts,
		RecentUsers:    make(map[string][]RecentUserReaction),
	}

	// Get recent users if requested
	if options.MaxRecentUsers > 0 {
		recent, err := c.storage.GetRecentReactions(ctx, target, options.MaxRecentUsers*len(c.reactionTypeList))
		if err != nil {
			return EntityReactionDetail{}, err
		}

		// Group by reaction type
		for _, r := range recent {
			detail.RecentUsers[r.ReactionType] = append(detail.RecentUsers[r.ReactionType], r)
		}

		// Trim to MaxRecentUsers per type
		for rt, users := range detail.RecentUsers {
			if len(users) > options.MaxRecentUsers {
				detail.RecentUsers[rt] = users[:options.MaxRecentUsers]
			}
		}
	}

	// Get last reaction time
	lastTime, err := c.storage.GetLastReactionTime(ctx, target)
	if err != nil {
		return EntityReactionDetail{}, err
	}
	detail.LastReaction = lastTime

	return detail, nil
}

// GetUserReactions retrieves all reactions for a user with pagination.
func (c *Client) GetUserReactions(ctx context.Context, userID string, pagination Pagination) (PaginatedResult[UserReaction], error) {
	if err := c.checkClosed(); err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	if err := validateUserID(userID); err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	pagCfg := pag.Config{
		DefaultLimit: c.paginationCfg.DefaultLimit,
		MaxLimit:     c.paginationCfg.MaxLimit,
		MaxOffset:    c.paginationCfg.MaxOffset,
	}
	if err := pag.ValidatePagination(pagination, pagCfg); err != nil {
		return PaginatedResult[UserReaction]{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if c.storage == nil {
		return PaginatedResult[UserReaction]{}, ErrStorageUnavailable
	}

	reactions, total, err := c.storage.GetUserReactions(ctx, userID, pagination)
	if err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	return NewPaginatedResult(reactions, total, pagination.Limit, pagination.Offset), nil
}

// GetUserReactionCounts retrieves aggregated counts per reaction type for a user.
// If entityTypeFilter is non-empty, counts are filtered to that entity type.
// Returns a map of reaction type to count.
func (c *Client) GetUserReactionCounts(ctx context.Context, userID string, entityTypeFilter string) (map[string]int64, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	if c.storage == nil {
		return nil, ErrStorageUnavailable
	}

	counts, err := c.storage.GetUserReactionCounts(ctx, userID, entityTypeFilter)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

// GetUserReactionsByType retrieves reactions of a specific type for a user with pagination.
func (c *Client) GetUserReactionsByType(ctx context.Context, userID, reactionType string, pagination Pagination) (PaginatedResult[UserReaction], error) {
	if err := c.checkClosed(); err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	if err := validateUserID(userID); err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	if err := c.validateReactionType(reactionType); err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	pagCfg := pag.Config{
		DefaultLimit: c.paginationCfg.DefaultLimit,
		MaxLimit:     c.paginationCfg.MaxLimit,
		MaxOffset:    c.paginationCfg.MaxOffset,
	}
	if err := pag.ValidatePagination(pagination, pagCfg); err != nil {
		return PaginatedResult[UserReaction]{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if c.storage == nil {
		return PaginatedResult[UserReaction]{}, ErrStorageUnavailable
	}

	reactions, total, err := c.storage.GetUserReactionsByType(ctx, userID, reactionType, pagination)
	if err != nil {
		return PaginatedResult[UserReaction]{}, err
	}

	return NewPaginatedResult(reactions, total, pagination.Limit, pagination.Offset), nil
}

// GetEntityReactions retrieves all reactions on an entity with pagination.
func (c *Client) GetEntityReactions(ctx context.Context, entityType, entityID string, pagination Pagination) (PaginatedResult[EntityReaction], error) {
	if err := c.checkClosed(); err != nil {
		return PaginatedResult[EntityReaction]{}, err
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return PaginatedResult[EntityReaction]{}, err
	}

	pagCfg := pag.Config{
		DefaultLimit: c.paginationCfg.DefaultLimit,
		MaxLimit:     c.paginationCfg.MaxLimit,
		MaxOffset:    c.paginationCfg.MaxOffset,
	}
	if err := pag.ValidatePagination(pagination, pagCfg); err != nil {
		return PaginatedResult[EntityReaction]{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if c.storage == nil {
		return PaginatedResult[EntityReaction]{}, ErrStorageUnavailable
	}

	reactions, total, err := c.storage.GetEntityReactions(ctx, target, pagination)
	if err != nil {
		return PaginatedResult[EntityReaction]{}, err
	}

	return NewPaginatedResult(reactions, total, pagination.Limit, pagination.Offset), nil
}

// GetUserReactionsBulk retrieves reaction states for multiple targets.
// Returns a map from EntityTarget to reaction type (empty string if no reaction).
func (c *Client) GetUserReactionsBulk(ctx context.Context, userID string, targets []EntityTarget) (map[EntityTarget]string, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return make(map[EntityTarget]string), nil
	}

	result := make(map[EntityTarget]string, len(targets))
	for _, target := range targets {
		if err := validateEntityTarget(target); err != nil {
			return nil, err
		}

		reactionType, err := c.GetUserReaction(ctx, userID, target.EntityType, target.EntityID)
		if err != nil {
			return nil, err
		}
		result[target] = reactionType
	}

	return result, nil
}

// GetEntityCountsBulk retrieves counts for multiple targets.
func (c *Client) GetEntityCountsBulk(ctx context.Context, targets []EntityTarget) (map[EntityTarget]EntityCounts, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return make(map[EntityTarget]EntityCounts), nil
	}

	result := make(map[EntityTarget]EntityCounts, len(targets))
	for _, target := range targets {
		if err := validateEntityTarget(target); err != nil {
			return nil, err
		}

		counts, _, err := c.GetEntityReactionCounts(ctx, target.EntityType, target.EntityID)
		if err != nil {
			return nil, err
		}

		var total int64
		for _, c := range counts {
			total += c
		}

		result[target] = EntityCounts{
			Counts: counts,
			Total:  total,
		}
	}

	return result, nil
}

// GetMultipleUserReactions retrieves reactions from multiple users on a single entity.
func (c *Client) GetMultipleUserReactions(ctx context.Context, userIDs []string, entityType, entityID string) (map[string]string, error) {
	if err := c.checkClosed(); err != nil {
		return nil, err
	}

	if len(userIDs) == 0 {
		return make(map[string]string), nil
	}

	target := EntityTarget{EntityType: entityType, EntityID: entityID}
	if err := validateEntityTarget(target); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(userIDs))
	for _, userID := range userIDs {
		if err := validateUserID(userID); err != nil {
			return nil, err
		}

		reactionType, err := c.GetUserReaction(ctx, userID, entityType, entityID)
		if err != nil {
			return nil, err
		}
		result[userID] = reactionType
	}

	return result, nil
}

// EventBus returns the underlying event bus for custom event handling.
func (c *Client) EventBus() *events.Bus {
	return c.eventBus
}

// Close releases all resources held by the client.
// It is safe to call Close multiple times.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.closedMu.Lock()
		c.closed = true
		c.closedMu.Unlock()

		// Close event bus
		if c.eventBus != nil {
			_ = c.eventBus.Close()
		}

		// Close cache
		if c.cache != nil {
			c.cache.Clear()
		}

		// Close storage
		if c.storage != nil {
			c.closeErr = c.storage.Close()
		}
	})

	return c.closeErr
}

// Storage returns the underlying storage (for testing purposes).
func (c *Client) Storage() ReactionStorage {
	return c.storage
}

// SetStorage sets the storage implementation (for testing purposes).
func (c *Client) SetStorage(storage ReactionStorage) {
	c.storage = storage
}

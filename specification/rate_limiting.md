# Rate Limiting Specification

## Overview

The system shall provide configurable rate limiting to prevent API abuse and protect resources from excessive usage. Rate limiting operates on a per-user basis using a sliding window algorithm, ensuring fair resource distribution while preventing spam, brute force attacks, and metric manipulation.

Rate limiting is applied at the module level and is configurable by the consuming application. All limits, window sizes, and enforcement behaviors are externally configurable - no hardcoded values.

## Functional Requirements

### Requirement 1: Per-User Rate Limiting

**Description:** The system shall enforce rate limits on a per-user basis using the user identifier.

**Requirements:**
- Rate limiting is keyed by `user_id` - each user has independent rate limit counters
- Anonymous/unauthenticated users are identified by a configurable fallback key (IP address, session ID, or rejected)
- Rate limit counters are isolated per user - one user's usage does not affect another
- Rate limits apply to all write operations: AddReaction, RemoveReaction

**Configuration:**
```go
type RateLimitConfig struct {
    // Key extraction function for rate limit identification
    KeyExtractor func(ctx context.Context, userID string) string

    // How to handle unauthenticated users
    AnonymousStrategy AnonymousStrategy // REJECT, IP_ADDRESS, SESSION_ID
}
```

### Requirement 2: Sliding Window Algorithm

**Description:** The system shall implement the sliding window algorithm for rate limiting, providing smooth rate enforcement without the burst issues of fixed windows.

**Algorithm Specification:**

**Sliding Window Logic:**
1. Track request timestamps in a sliding window of configurable duration
2. Window "slides" continuously - old requests expire as time progresses
3. Current window includes all requests within `[now - window_duration, now]`
4. Allow request if count in current window < limit
5. Reject request if count in current window >= limit

**Advantages over Fixed Window:**
- No burst attacks at window boundaries
- Smooth rate enforcement over time
- Better distribution of requests

**Implementation Options:**

1. **In-Memory Sliding Window (Default):**
   - Stores timestamps in memory (sorted list or ring buffer)
   - Fast O(log n) or O(1) operations
   - Suitable for single-instance deployments
   - Memory usage: O(limit) per user

2. **Redis Sliding Window (Distributed):**
   - Uses Redis sorted sets (ZADD, ZREMRANGEBYSCORE, ZCARD)
   - Atomic operations via Lua script
   - Suitable for distributed deployments
   - Automatic expiration via Redis TTL

3. **Token Bucket Alternative:**
   - Optional alternative algorithm
   - Configurable burst capacity
   - Constant rate refill

**Algorithm Configuration:**
```go
type SlidingWindowConfig struct {
    Algorithm RateLimitAlgorithm // SLIDING_WINDOW, TOKEN_BUCKET

    // Window duration (e.g., "1m", "1h")
    WindowDuration time.Duration

    // Maximum requests per window
    MaxRequests int

    // Storage backend
    Storage RateLimitStorage // MEMORY, REDIS

    // Redis configuration (if using Redis)
    RedisConfig *RedisConfig
}
```

### Requirement 3: Configurable Limits

**Description:** All rate limiting parameters shall be externally configurable with no hardcoded values.

**Configuration Parameters:**

```go
// Complete rate limiting configuration
type RateLimitingConfig struct {
    // Master switch
    Enabled bool // Default: true

    // Per-operation limits (all optional, nil = no limit)
    AddReaction    *OperationLimit
    RemoveReaction *OperationLimit

    // Global limit across all operations
    Global    *OperationLimit

    // Default limit for unspecified operations
    Default   *OperationLimit
}

type OperationLimit struct {
    // Requests allowed per window
    Requests int // Required, must be > 0

    // Window duration
    Window   time.Duration // Required, must be > 0

    // Algorithm to use
    Algorithm RateLimitAlgorithm // Default: SLIDING_WINDOW
}
```

**Configuration Examples:**

```go
// Example 1: Permissive (100 reactions per minute)
config := golikeit.RateLimitingConfig{
    Enabled: true,
    AddReaction: &golikeit.OperationLimit{
        Requests: 100,
        Window:   time.Minute,
    },
}

// Example 2: Restrictive (10 reactions per 10 seconds)
config := golikeit.RateLimitingConfig{
    Enabled: true,
    Default: &golikeit.OperationLimit{
        Requests: 10,
        Window:   10 * time.Second,
    },
}

// Example 3: Different limits per operation
config := golikeit.RateLimitingConfig{
    Enabled: true,
    AddReaction: &golikeit.OperationLimit{
        Requests: 100,
        Window:   time.Minute,
    },
    RemoveReaction: &golikeit.OperationLimit{
        Requests: 50,
        Window:   time.Minute,
    },
    // Uses default for unspecified operations
    Default: &golikeit.OperationLimit{
        Requests: 60,
        Window:   time.Minute,
    },
}

// Example 4: Disabled
config := golikeit.RateLimitingConfig{
    Enabled: false,
}
```

### Requirement 4: Rate Limit Response

**Description:** When rate limit is exceeded, the system shall return a clear error response with retry information.

**Error Response:**

```go
// ErrRateLimitExceeded is returned when rate limit is exceeded
type ErrRateLimitExceeded struct {
    Operation     string        // Operation that was rate limited
    Limit         int           // Current limit
    Window        time.Duration // Current window duration
    RetryAfter    time.Duration // Time until next request is allowed
    ResetTime     time.Time     // When the window resets
}

func (e *ErrRateLimitExceeded) Error() string {
    return fmt.Sprintf("rate limit exceeded: %d requests per %v, retry after %v",
        e.Limit, e.Window, e.RetryAfter)
}
```

**Error Behavior:**
- Operation is rejected before any storage access
- No partial state changes occur
- HTTP-equivalent status would be 429 (Too Many Requests)
- Error includes retry-after information for client handling

### Requirement 5: Storage Backends

**Description:** The rate limiter shall support multiple storage backends for different deployment scenarios.

**Storage Options:**

1. **In-Memory (Default):**
   - Uses Go maps with mutex or sync.Map
   - No external dependencies
   - Data lost on restart
   - Suitable for single-instance deployments
   - Automatic cleanup of expired entries

2. **Redis:**
   - Uses Redis sorted sets for sliding window
   - Shared state across instances
   - Persistence optional
   - Suitable for distributed deployments

**Storage Configuration:**
```go
type RateLimitStorage string

const (
    StorageMemory RateLimitStorage = "memory"
    StorageRedis   RateLimitStorage = "redis"
)

type RateLimiterStorage interface {
    Increment(ctx context.Context, key string, window time.Duration) (count int, err error)
    GetCount(ctx context.Context, key string, window time.Duration) (count int, err error)
    Reset(ctx context.Context, key string) error
}
```

### Requirement 6: Rate Limit Exclusions

**Description:** The system shall support configurable exclusions from rate limiting.

**Exclusion Types:**

1. **User Exclusions:**
   - Specific user IDs exempt from rate limits
   - Useful for admin users or internal services

2. **Context Exclusions:**
   - Requests with specific context values (e.g., admin context)
   - Bypass rate limiting via context marker

3. **Operation Exclusions:**
   - Specific operations exempt (e.g., only limit AddReaction, not RemoveReaction)
   - Configured per-operation

**Configuration:**
```go
type RateLimitExclusions struct {
    // User IDs exempt from rate limiting
    ExemptUsers []string

    // Function to determine if request is exempt
    IsExempt func(ctx context.Context, userID string, operation string) bool
}
```

### Requirement 7: Rate Limit Headers/Metadata

**Description:** The system shall expose current rate limit status for client awareness.

**Rate Limit Headers (equivalent):**

```go
type RateLimitStatus struct {
    Limit          int           // Maximum allowed requests
    Remaining      int           // Remaining requests in current window
    ResetTime      time.Time     // When the window resets
    WindowDuration time.Duration // Size of the window
}

// GetRateLimitStatus returns current status for a user
func (c *Client) GetRateLimitStatus(ctx context.Context, userID string, operation string) (*RateLimitStatus, error)
```

**Use Cases:**
- Client UI can display remaining quota
- Warning when approaching limit
- Adaptive client behavior

### Requirement 8: Distributed Rate Limiting

**Description:** For distributed deployments, the rate limiter shall provide consistent rate limiting across instances.

**Requirements:**
- Redis storage provides shared state across instances
- Clock skew handling (use Redis server time)
- Atomic operations prevent race conditions
- Eventual consistency acceptable (small window overlap)

**Redis Implementation:**
```lua
-- Sliding window rate limiting with Redis
local key = KEYS[1]
local window = tonumber(ARGV[1]) * 1000 -- convert to milliseconds
local limit = tonumber(ARGV[2])
local now = redis.call('TIME')[1] * 1000 + redis.call('TIME')[2] / 1000

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current entries
local current = redis.call('ZCARD', key)

if current < limit then
    -- Add current request
    redis.call('ZADD', key, now, now .. ':' .. ARGV[3])
    -- Set expiration
    redis.call('PEXPIRE', key, window)
    return {1, limit - current - 1} -- allowed, remaining
else
    return {0, 0} -- denied, remaining
end
```

### Requirement 9: Rate Limiting Performance

**Description:** Rate limiting operations shall have minimal performance impact.

**Performance Requirements:**
- Rate limit check latency: <1ms (p95) for in-memory
- Rate limit check latency: <5ms (p95) for Redis
- Memory usage: O(active users × limit) for in-memory
- No blocking operations in rate limit check path

**Optimization Strategies:**
- Lazy cleanup of expired entries
- Pre-allocated ring buffers for timestamps
- Connection pooling for Redis
- Local cache for Redis responses (brief TTL)

## Configuration Examples

### Example 1: Basic Configuration

```go
client, err := golikeit.New(
    golikeit.WithDatabaseURL("postgres://localhost/reactions"),
    golikeit.WithReactionTypes("LIKE", "DISLIKE"),
    golikeit.WithRateLimiting(golikeit.RateLimitingConfig{
        Enabled: true,
        Default: &golikeit.OperationLimit{
            Requests: 60,
            Window:   time.Minute,
        },
    }),
)
```

### Example 2: Production Configuration

```go
client, err := golikeit.New(
    golikeit.WithDatabaseURL("postgres://localhost/reactions"),
    golikeit.WithReactionTypes("LIKE", "DISLIKE", "LOVE", "ANGRY"),
    golikeit.WithRateLimiting(golikeit.RateLimitingConfig{
        Enabled: true,
        AddReaction: &golikeit.OperationLimit{
            Requests: 100,
            Window:   time.Minute,
        },
        RemoveReaction: &golikeit.OperationLimit{
            Requests: 50,
            Window:   time.Minute,
        },
        Global: &golikeit.OperationLimit{
            Requests: 500,
            Window:   time.Hour,
        },
    }),
    golikeit.WithRateLimitStorage(golikeit.RateLimitRedisConfig{
        Addr: "localhost:6379",
    }),
)
```

### Example 3: Disabled for Internal Services

```go
client, err := golikeit.New(
    golikeit.WithDatabaseURL("postgres://localhost/reactions"),
    golikeit.WithReactionTypes("LIKE", "DISLIKE"),
    golikeit.WithRateLimiting(golikeit.RateLimitingConfig{
        Enabled: false,
    }),
)
```

## Constraints and Limitations

1. **No Hardcoded Limits:** All rate limit values must be externally configured. No defaults that cannot be overridden.

2. **User-Level Only:** Rate limiting is per-user, not per-IP or per-session (unless explicitly configured via key extractor).

3. **Sliding Window Default:** The default algorithm is sliding window. Token bucket is available as alternative.

4. **In-Memory Default:** Default storage is in-memory. Redis required for distributed deployments.

5. **No Rate Limit on Reads:** Rate limiting applies only to write operations (AddReaction, RemoveReaction). Read operations are not rate limited.

6. **Clock Dependency:** Sliding window accuracy depends on clock synchronization in distributed deployments.

## Relationships with Other Functional Blocks

- **[api_interface.md](api_interface.md):** Rate limiting is part of the public API and returns rate limit errors
- **[data_persistence.md](data_persistence.md):** Rate limiting storage is separate from reaction storage
- **[security_policies.md](security_policies.md):** Rate limiting is a security control against abuse

## Change History

| Date | Change | Description |
|------|--------|-------------|
| 2026-03-21 | Initial | First version with Like/Unlike/Dislike/Undislike rate limiting |
| 2026-03-22 | Update | Updated to generic AddReaction/RemoveReaction operations |

## Acceptance Criteria

1. **AC1:** Rate limiting is configurable with no hardcoded values
2. **AC2:** Sliding window algorithm is implemented and documented
3. **AC3:** Per-user rate limiting is enforced via user_id
4. **AC4:** Redis storage backend is supported for distributed deployments
5. **AC5:** In-memory storage backend works for single-instance deployments
6. **AC6:** ErrRateLimitExceeded error includes retry-after information
7. **AC7:** Rate limiting can be disabled via configuration
8. **AC8:** Different limits can be configured per operation
9. **AC9:** Global limit across all operations is supported
10. **AC10:** Rate limit status can be queried for client awareness
11. **AC11:** Exemptions are supported for specific users/contexts
12. **AC12:** Rate limit check latency is <1ms for in-memory (p95)
13. **AC13:** Rate limit check latency is <5ms for Redis (p95)
14. **AC14:** Read operations are not rate limited

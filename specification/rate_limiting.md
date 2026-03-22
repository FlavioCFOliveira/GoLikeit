# Rate Limiting Specification

## Overview

The system provides configurable rate limiting to prevent API abuse. Rate limiting operates on a per-user basis using a sliding window algorithm.

## Functional Requirements

### Requirement 1: Per-User Rate Limiting

Rate limits are enforced on a per-user basis.

- Rate limiting is keyed by `user_id`
- Rate limit counters are isolated per user
- Rate limits apply to write operations: AddReaction, RemoveReaction

### Requirement 2: Sliding Window Algorithm

The system implements sliding window algorithm.

- Track request timestamps in a sliding window
- Current window includes all requests within `[now - window_duration, now]`
- Allow request if count < limit
- Reject request if count >= limit

**Implementation Options:**
- **In-Memory:** Suitable for single-instance deployments
- **Redis:** Suitable for distributed deployments

### Requirement 3: Configurable Limits

All rate limiting parameters are externally configurable.

```go
type RateLimitingConfig struct {
    Enabled    bool
    AddReaction    *OperationLimit
    RemoveReaction *OperationLimit
    Global     *OperationLimit
    Default    *OperationLimit
}

type OperationLimit struct {
    Requests  int
    Window    time.Duration
    Algorithm RateLimitAlgorithm
}
```

### Requirement 4: Rate Limit Response

When rate limit is exceeded, the system returns error with retry information.

```go
type ErrRateLimitExceeded struct {
    Operation  string
    Limit      int
    Window     time.Duration
    RetryAfter time.Duration
    ResetTime  time.Time
}
```

### Requirement 5: Storage Backends

The rate limiter supports multiple storage backends.

- **In-Memory (Default):** Uses Go maps
- **Redis:** Uses sorted sets for sliding window

### Requirement 6: Rate Limit Exclusions

The system supports configurable exclusions.

- **User Exclusions:** Specific user IDs exempt
- **Context Exclusions:** Requests with specific context values
- **Operation Exclusions:** Specific operations exempt

### Requirement 7: Rate Limit Status

The system exposes current rate limit status.

```go
type RateLimitStatus struct {
    Limit          int
    Remaining      int
    ResetTime      time.Time
    WindowDuration time.Duration
}
```

## Constraints and Limitations

1. **No Hardcoded Limits:** All values must be externally configured.
2. **User-Level Only:** Rate limiting is per-user.
3. **No Rate Limit on Reads:** Only write operations are rate limited.
4. **Clock Dependency:** Sliding window depends on clock synchronization.

## Acceptance Criteria

1. **AC1:** Rate limiting is configurable
2. **AC2:** Sliding window algorithm is implemented
3. **AC3:** Per-user rate limiting is enforced
4. **AC4:** Redis storage backend is supported
5. **AC5:** ErrRateLimitExceeded includes retry-after information
6. **AC6:** Rate limiting can be disabled via configuration
7. **AC7:** Different limits can be configured per operation
8. **AC8:** Global limit across all operations is supported

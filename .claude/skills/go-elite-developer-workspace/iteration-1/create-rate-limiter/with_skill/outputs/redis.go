package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBackend implements a distributed token bucket rate limiter using Redis.
type RedisBackend struct {
	client *redis.Client
	script *redis.Script
}

// RedisScript is the Lua script for atomic token bucket operations.
// It uses Redis EVALSHA for optimal performance.
const redisScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local tokens_needed = tonumber(ARGV[4])
local ttl = tonumber(ARGV[5])

local bucket = redis.call('HMGET', key, 'tokens', 'last_access')
local tokens = tonumber(bucket[1])
local last_access = tonumber(bucket[2])

if tokens == nil then
    tokens = burst
    last_access = now
end

-- Calculate tokens to add based on elapsed time
local elapsed = now - last_access
local new_tokens = tokens + (elapsed * rate)
if new_tokens > burst then
    new_tokens = burst
end

-- Check if we have enough tokens
local allowed = 0
local remaining = 0
local reset_time = now

if new_tokens >= tokens_needed then
    new_tokens = new_tokens - tokens_needed
    allowed = 1
    remaining = math.floor(new_tokens)
else
    -- Calculate when enough tokens will be available
    local needed = tokens_needed - new_tokens
    local seconds_until = needed / rate
    reset_time = now + math.ceil(seconds_until)
    remaining = math.floor(new_tokens)
end

-- Update bucket
redis.call('HMSET', key, 'tokens', new_tokens, 'last_access', now)
redis.call('EXPIRE', key, ttl)

return {allowed, remaining, reset_time}
`

// NewRedisBackend creates a new Redis-backed rate limiter.
// The client should be connected and ready to use.
func NewRedisBackend(client *redis.Client) *RedisBackend {
	return &RedisBackend{
		client: client,
		script: redis.NewScript(redisScript),
	}
}

// Take implements the Backend interface.
func (rb *RedisBackend) Take(ctx context.Context, key string, rate float64, burst int, n int) (int, time.Time, error) {
	now := time.Now().Unix()
	ttl := int64(time.Hour.Seconds()) // 1 hour TTL

	result, err := rb.script.Run(ctx, rb.client, []string{key}, rate, burst, now, n, ttl).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, time.Time{}, ErrBackendClosed
		}
		return 0, time.Time{}, fmt.Errorf("redis script execution failed: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 3 {
		return 0, time.Time{}, errors.New("unexpected result from redis script")
	}

	// allowed is 1 if allowed, 0 if not
	allowed := values[0].(int64)
	remaining := int(values[1].(int64))
	resetUnix := values[2].(int64)

	reset := time.Unix(resetUnix, 0)

	// If not allowed, remaining is actually the current tokens
	if allowed == 0 {
		return -1, reset, nil
	}

	return remaining, reset, nil
}

// Close implements the Backend interface.
func (rb *RedisBackend) Close() error {
	return rb.client.Close()
}

// Ping checks connectivity to Redis.
func (rb *RedisBackend) Ping(ctx context.Context) error {
	return rb.client.Ping(ctx).Err()
}

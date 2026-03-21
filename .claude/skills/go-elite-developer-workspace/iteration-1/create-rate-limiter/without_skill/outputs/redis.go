package ratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBackend implements Backend using Redis for distributed rate limiting.
type RedisBackend struct {
	client *redis.Client
	prefix string
}

// RedisOptions contains configuration options for the Redis backend.
type RedisOptions struct {
	// Addr is the Redis server address (host:port).
	Addr string

	// Password is the Redis password (optional).
	Password string

	// DB is the Redis database number.
	DB int

	// Prefix is prepended to all Redis keys.
	Prefix string

	// Client is an optional pre-configured Redis client.
	// If provided, Addr, Password, and DB are ignored.
	Client *redis.Client
}

// NewRedisBackend creates a new Redis-backed rate limiter.
func NewRedisBackend(opts RedisOptions) (*RedisBackend, error) {
	var client *redis.Client

	if opts.Client != nil {
		client = opts.Client
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     opts.Addr,
			Password: opts.Password,
			DB:       opts.DB,
		})
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := opts.Prefix
	if prefix == "" {
		prefix = "ratelimit:"
	}

	return &RedisBackend{
		client: client,
		prefix: prefix,
	}, nil
}

// Take attempts to take tokens from the Redis-backed bucket.
func (r *RedisBackend) Take(ctx context.Context, key string, tokens int, rate Rate) (TakeResult, error) {
	now := time.Now().Unix()
	bucketKey := r.prefix + key

	// Use Lua script for atomic token bucket operation
	script := `
		local key = KEYS[1]
		local tokens_requested = tonumber(ARGV[1])
		local rate_limit = tonumber(ARGV[2])
		local burst = tonumber(ARGV[3])
		local period = tonumber(ARGV[4])
		local now = tonumber(ARGV[5])

		-- Get current state or initialize
		local state = redis.call('HMGET', key, 'tokens', 'last_refill')
		local current_tokens = tonumber(state[1]) or burst
		local last_refill = tonumber(state[2]) or now

		-- Calculate tokens to add based on elapsed time
		local elapsed = now - last_refill
		local rate_per_second = rate_limit / period
		local tokens_to_add = elapsed * rate_per_second

		-- Update token count, capped at burst
		current_tokens = math.min(burst, current_tokens + tokens_to_add)

		-- Check if request can be satisfied
		local allowed = 0
		local remaining = current_tokens
		local retry_after = 0

		if current_tokens >= tokens_requested then
			current_tokens = current_tokens - tokens_requested
			allowed = 1
			remaining = current_tokens
		else
			-- Calculate retry after time
			local needed = tokens_requested - current_tokens
			retry_after = math.ceil(needed / rate_per_second)
		end

		-- Save state
		redis.call('HMSET', key, 'tokens', current_tokens, 'last_refill', now)
		redis.call('EXPIRE', key, 3600) -- Expire after 1 hour of inactivity

		return {allowed, math.floor(remaining), retry_after}
	`

	result, err := r.client.Eval(
		ctx,
		script,
		[]string{bucketKey},
		tokens,
		rate.Limit,
		rate.Burst,
		rate.Period.Seconds(),
		now,
	).Result()

	if err != nil {
		return TakeResult{}, fmt.Errorf("redis eval failed: %w", err)
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	retryAfter := time.Duration(values[2].(int64)) * time.Second

	// Calculate reset time
	missing := rate.Limit - remaining
	ratePerSecond := float64(rate.Limit) / rate.Period.Seconds()
	secondsToFull := float64(missing) / ratePerSecond
	resetTime := now + int64(secondsToFull)

	return TakeResult{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  time.Unix(resetTime, 0),
		RetryAfter: retryAfter,
	}, nil
}

// Close closes the Redis client connection.
func (r *RedisBackend) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

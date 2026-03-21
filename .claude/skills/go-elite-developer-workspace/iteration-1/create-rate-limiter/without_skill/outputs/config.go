package ratelimiter

import (
	"time"
)

// EndpointConfig defines rate limiting configuration for a specific endpoint.
type EndpointConfig struct {
	// Path is the URL path pattern (supports wildcards like "/api/*").
	Path string

	// Rate is the token bucket configuration.
	Rate Rate

	// KeyFunc generates the rate limit key for requests to this endpoint.
	// If nil, DefaultKeyFunc() is used.
	KeyFunc KeyFunc

	// TokensConsumed is the number of tokens to consume per request.
	// Default is 1.
	TokensConsumed int
}

// Config holds the global rate limiter configuration.
type Config struct {
	// Endpoints maps path patterns to their configurations.
	Endpoints []EndpointConfig

	// DefaultRate is used when no endpoint-specific configuration matches.
	// If zero, no rate limiting is applied.
	DefaultRate Rate

	// KeyFunc is the default key generation function.
	DefaultKeyFunc KeyFunc

	// Enabled controls whether rate limiting is active.
	Enabled bool
}

// DefaultConfig returns a default configuration with reasonable limits.
func DefaultConfig() Config {
	return Config{
		Endpoints: []EndpointConfig{
			{
				Path: "/api/*",
				Rate: Rate{
					Limit:  100,
					Burst:  150,
					Period: time.Minute,
				},
				TokensConsumed: 1,
			},
			{
				Path: "/login",
				Rate: Rate{
					Limit:  5,
					Burst:  10,
					Period: time.Minute,
				},
				TokensConsumed: 1,
			},
		},
		DefaultRate: Rate{
			Limit:  1000,
			Burst:  1500,
			Period: time.Minute,
		},
		DefaultKeyFunc: DefaultKeyFunc(),
		Enabled:        true,
	}
}

// GetConfigForPath returns the configuration for the given path.
// If no specific configuration matches, it returns the default configuration.
func (c *Config) GetConfigForPath(path string) EndpointConfig {
	// Check for exact matches first
	for _, ep := range c.Endpoints {
		if ep.Path == path {
			if ep.KeyFunc == nil {
				ep.KeyFunc = c.DefaultKeyFunc
			}
			if ep.TokensConsumed == 0 {
				ep.TokensConsumed = 1
			}
			return ep
		}
	}

	// Check for wildcard matches
	for _, ep := range c.Endpoints {
		if matchWildcard(ep.Path, path) {
			if ep.KeyFunc == nil {
				ep.KeyFunc = c.DefaultKeyFunc
			}
			if ep.TokensConsumed == 0 {
				ep.TokensConsumed = 1
			}
			return ep
		}
	}

	// Return default configuration
	return EndpointConfig{
		Path:           path,
		Rate:           c.DefaultRate,
		KeyFunc:        c.DefaultKeyFunc,
		TokensConsumed: 1,
	}
}

// matchWildcard checks if path matches the pattern.
// Supports * as a wildcard for any sequence of characters.
func matchWildcard(pattern, path string) bool {
	if pattern == "*" {
		return true
	}

	// Handle suffix wildcard: /api/*
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}

	// Handle prefix wildcard: */api
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
	}

	return pattern == path
}

// Builder provides a fluent API for building configurations.
type Builder struct {
	config Config
}

// NewConfigBuilder creates a new configuration builder.
func NewConfigBuilder() *Builder {
	return &Builder{
		config: Config{
			Endpoints:      make([]EndpointConfig, 0),
			DefaultKeyFunc: DefaultKeyFunc(),
			Enabled:        true,
		},
	}
}

// WithDefaultRate sets the default rate.
func (b *Builder) WithDefaultRate(limit, burst int, period time.Duration) *Builder {
	b.config.DefaultRate = Rate{
		Limit:  limit,
		Burst:  burst,
		Period: period,
	}
	return b
}

// WithEndpoint adds an endpoint-specific configuration.
func (b *Builder) WithEndpoint(path string, limit, burst int, period time.Duration) *Builder {
	b.config.Endpoints = append(b.config.Endpoints, EndpointConfig{
		Path: path,
		Rate: Rate{
			Limit:  limit,
			Burst:  burst,
			Period: period,
		},
		TokensConsumed: 1,
	})
	return b
}

// WithEndpointAndKeyFunc adds an endpoint with a custom key function.
func (b *Builder) WithEndpointAndKeyFunc(path string, limit, burst int, period time.Duration, keyFunc KeyFunc) *Builder {
	b.config.Endpoints = append(b.config.Endpoints, EndpointConfig{
		Path: path,
		Rate: Rate{
			Limit:  limit,
			Burst:  burst,
			Period: period,
		},
		KeyFunc:        keyFunc,
		TokensConsumed: 1,
	})
	return b
}

// Build returns the final configuration.
func (b *Builder) Build() Config {
	return b.config
}

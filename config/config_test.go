package config

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				ReactionTypes: []string{"LIKE", "LOVE"},
				Pagination:    DefaultPaginationConfig(),
				Cache:         DefaultCacheConfig(),
				Events:        DefaultEventsConfig(),
				RateLimit:     DefaultRateLimitConfig(),
			},
			wantErr: false,
		},
		{
			name: "empty reaction types",
			config: Config{
				ReactionTypes: []string{},
				Pagination:    DefaultPaginationConfig(),
			},
			wantErr: true,
			errMsg:  "at least one reaction type",
		},
		{
			name: "nil reaction types",
			config: Config{
				ReactionTypes: nil,
				Pagination:    DefaultPaginationConfig(),
			},
			wantErr: true,
			errMsg:  "at least one reaction type",
		},
		{
			name: "invalid reaction type format",
			config: Config{
				ReactionTypes: []string{"like"},
				Pagination:    DefaultPaginationConfig(),
			},
			wantErr: true,
			errMsg:  "must match pattern",
		},
		{
			name: "duplicate reaction types",
			config: Config{
				ReactionTypes: []string{"LIKE", "LIKE"},
				Pagination:    DefaultPaginationConfig(),
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
		{
			name: "empty reaction type in list",
			config: Config{
				ReactionTypes: []string{"LIKE", ""},
				Pagination:    DefaultPaginationConfig(),
			},
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name: "reaction type too long",
			config: Config{
				ReactionTypes: []string{"LIKE", "THIS_IS_A_VERY_LONG_REACTION_TYPE_THAT_EXCEEDS_THE_MAXIMUM_ALLOWED_LENGTH"},
				Pagination:    DefaultPaginationConfig(),
			},
			wantErr: true,
			errMsg:  "exceeds",
		},
		{
			name: "invalid pagination config",
			config: Config{
				ReactionTypes: []string{"LIKE"},
				Pagination: PaginationConfig{
					DefaultLimit: 0,
					MaxLimit:     100,
				},
			},
			wantErr: true,
			errMsg:  "default_limit",
		},
		{
			name: "invalid cache config",
			config: Config{
				ReactionTypes: []string{"LIKE"},
				Pagination:    DefaultPaginationConfig(),
				Cache: CacheConfig{
					UserReactionTTL: -1 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
		{
			name: "invalid events config",
			config: Config{
				ReactionTypes: []string{"LIKE"},
				Pagination:    DefaultPaginationConfig(),
				Events: EventsConfig{
					AsyncWorkers: -1,
				},
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
		{
			name: "invalid rate limit config",
			config: Config{
				ReactionTypes: []string{"LIKE"},
				Pagination:    DefaultPaginationConfig(),
				RateLimit: RateLimitConfig{
					RequestsPerSecond: -1,
				},
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Config.Validate() error message = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DatabaseConfig
		wantErr bool
	}{
		{
			name:    "empty config is valid",
			config:  DatabaseConfig{},
			wantErr: false,
		},
		{
			name: "valid postgresql",
			config: DatabaseConfig{
				Type: "postgresql",
				Host: "localhost",
				Port: 5432,
			},
			wantErr: false,
		},
		{
			name: "valid mysql",
			config: DatabaseConfig{
				Type: "mysql",
				Host: "localhost",
				Port: 3306,
			},
			wantErr: false,
		},
		{
			name: "valid sqlite",
			config: DatabaseConfig{
				Type: "sqlite",
			},
			wantErr: false,
		},
		{
			name: "invalid database type",
			config: DatabaseConfig{
				Type: "oracle",
			},
			wantErr: true,
		},
		{
			name: "invalid port (negative)",
			config: DatabaseConfig{
				Type: "postgresql",
				Port: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid port (too high)",
			config: DatabaseConfig{
				Type: "postgresql",
				Port: 70000,
			},
			wantErr: true,
		},
		{
			name: "negative max open conns",
			config: DatabaseConfig{
				MaxOpenConns: -1,
			},
			wantErr: true,
		},
		{
			name: "negative max idle conns",
			config: DatabaseConfig{
				MaxIdleConns: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DatabaseConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCacheConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CacheConfig
		wantErr bool
	}{
		{
			name:    "default config is valid",
			config:  DefaultCacheConfig(),
			wantErr: false,
		},
		{
			name: "negative user reaction ttl",
			config: CacheConfig{
				UserReactionTTL: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative entity counts ttl",
			config: CacheConfig{
				EntityCountsTTL: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max entries",
			config: CacheConfig{
				MaxEntries: -1,
			},
			wantErr: true,
		},
		{
			name: "unsupported eviction policy",
			config: CacheConfig{
				EvictionPolicy: "FIFO",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CacheConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEventsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  EventsConfig
		wantErr bool
	}{
		{
			name:    "default config is valid",
			config:  DefaultEventsConfig(),
			wantErr: false,
		},
		{
			name: "negative async workers",
			config: EventsConfig{
				AsyncWorkers: -1,
			},
			wantErr: true,
		},
		{
			name: "negative queue size",
			config: EventsConfig{
				QueueSize: -1,
			},
			wantErr: true,
		},
		{
			name: "negative event timeout",
			config: EventsConfig{
				EventTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EventsConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimitConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{
			name:    "default config is valid",
			config:  DefaultRateLimitConfig(),
			wantErr: false,
		},
		{
			name: "negative requests per second",
			config: RateLimitConfig{
				RequestsPerSecond: -1,
			},
			wantErr: true,
		},
		{
			name: "negative burst size",
			config: RateLimitConfig{
				BurstSize: -1,
			},
			wantErr: true,
		},
		{
			name: "negative ttl",
			config: RateLimitConfig{
				TTL: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("RateLimitConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPaginationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  PaginationConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "default config is valid",
			config:  DefaultPaginationConfig(),
			wantErr: false,
		},
		{
			name: "zero default limit",
			config: PaginationConfig{
				DefaultLimit: 0,
				MaxLimit:     100,
			},
			wantErr: true,
			errMsg:  "default_limit must be positive",
		},
		{
			name: "negative default limit",
			config: PaginationConfig{
				DefaultLimit: -1,
				MaxLimit:     100,
			},
			wantErr: true,
		},
		{
			name: "zero max limit",
			config: PaginationConfig{
				DefaultLimit: 25,
				MaxLimit:     0,
			},
			wantErr: true,
		},
		{
			name: "negative max offset",
			config: PaginationConfig{
				DefaultLimit: 25,
				MaxLimit:     100,
				MaxOffset:    -1,
			},
			wantErr: true,
		},
		{
			name: "default limit exceeds max limit",
			config: PaginationConfig{
				DefaultLimit: 50,
				MaxLimit:     25,
			},
			wantErr: true,
			errMsg:  "cannot exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PaginationConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsString(err.Error(), tt.errMsg) {
					t.Errorf("PaginationConfig.Validate() error message = %v, should contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name          string
		reactionTypes []string
		wantErr       bool
	}{
		{
			name:          "valid reaction types",
			reactionTypes: []string{"LIKE", "LOVE"},
			wantErr:       false,
		},
		{
			name:          "empty reaction types",
			reactionTypes: []string{},
			wantErr:       true,
		},
		{
			name:          "invalid reaction type",
			reactionTypes: []string{"like"},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewConfig(tt.reactionTypes)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify defaults are set
				if !cfg.Cache.Enabled {
					t.Error("NewConfig() cache should be enabled by default")
				}
				if cfg.Cache.UserReactionTTL != DefaultUserReactionTTL {
					t.Errorf("NewConfig() user reaction ttl = %v, want %v", cfg.Cache.UserReactionTTL, DefaultUserReactionTTL)
				}
			}
		})
	}
}

func TestMustNewConfig(t *testing.T) {
	// Valid config should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Error("MustNewConfig() should not panic with valid config")
		}
	}()

	cfg := MustNewConfig([]string{"LIKE", "LOVE"})
	if len(cfg.ReactionTypes) != 2 {
		t.Errorf("MustNewConfig() reaction types count = %d, want 2", len(cfg.ReactionTypes))
	}

	// Invalid config should panic
	t.Run("panics on invalid config", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustNewConfig() should panic with invalid config")
			}
		}()
		_ = MustNewConfig([]string{})
	})
}

func BenchmarkConfig_Validate(b *testing.B) {
	cfg := Config{
		ReactionTypes: []string{"LIKE", "LOVE", "ANGRY", "SAD"},
		Cache:         DefaultCacheConfig(),
		Pagination:    DefaultPaginationConfig(),
		Events:        DefaultEventsConfig(),
		RateLimit:     DefaultRateLimitConfig(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}

func BenchmarkNewConfig(b *testing.B) {
	reactionTypes := []string{"LIKE", "LOVE", "ANGRY", "SAD"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewConfig(reactionTypes)
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

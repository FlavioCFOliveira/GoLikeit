package pagination

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultLimit != 25 {
		t.Errorf("DefaultConfig().DefaultLimit = %d, want 25", cfg.DefaultLimit)
	}
	if cfg.MaxLimit != 100 {
		t.Errorf("DefaultConfig().MaxLimit = %d, want 100", cfg.MaxLimit)
	}
	if cfg.MaxOffset != 10000 {
		t.Errorf("DefaultConfig().MaxOffset = %d, want 10000", cfg.MaxOffset)
	}
}

func TestPagination_Validate(t *testing.T) {
	cfg := Config{
		DefaultLimit: 25,
		MaxLimit:     100,
		MaxOffset:    10000,
	}

	tests := []struct {
		name    string
		p       Pagination
		wantErr bool
		want    Pagination // expected values after validation
	}{
		{
			name:    "valid pagination with explicit values",
			p:       Pagination{Limit: 25, Offset: 0},
			wantErr: false,
			want:    Pagination{Limit: 25, Offset: 0},
		},
		{
			name:    "zero limit should use default",
			p:       Pagination{Limit: 0, Offset: 0},
			wantErr: false,
			want:    Pagination{Limit: 25, Offset: 0},
		},
		{
			name:    "valid pagination with offset",
			p:       Pagination{Limit: 50, Offset: 100},
			wantErr: false,
			want:    Pagination{Limit: 50, Offset: 100},
		},
		{
			name:    "negative limit",
			p:       Pagination{Limit: -1, Offset: 0},
			wantErr: true,
		},
		{
			name:    "negative offset",
			p:       Pagination{Limit: 25, Offset: -1},
			wantErr: true,
		},
		{
			name:    "limit exceeds max",
			p:       Pagination{Limit: 101, Offset: 0},
			wantErr: true,
		},
		{
			name:    "offset exceeds max",
			p:       Pagination{Limit: 25, Offset: 10001},
			wantErr: true,
		},
		{
			name:    "limit at max boundary",
			p:       Pagination{Limit: 100, Offset: 0},
			wantErr: false,
			want:    Pagination{Limit: 100, Offset: 0},
		},
		{
			name:    "offset at max boundary",
			p:       Pagination{Limit: 25, Offset: 10000},
			wantErr: false,
			want:    Pagination{Limit: 25, Offset: 10000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.p
			err := p.Validate(cfg)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && p != tt.want {
				t.Errorf("After Validate() = %+v, want %+v", p, tt.want)
			}
		})
	}
}

func TestNewResult(t *testing.T) {
	items := []string{"item1", "item2", "item3"}

	tests := []struct {
		name            string
		items           []string
		total           int64
		limit           int
		offset          int
		wantTotalPages  int
		wantCurrentPage int
		wantHasNext     bool
		wantHasPrev     bool
	}{
		{
			name:            "first page of multiple",
			items:           items,
			total:           100,
			limit:           25,
			offset:          0,
			wantTotalPages:  4,
			wantCurrentPage: 1,
			wantHasNext:     true,
			wantHasPrev:     false,
		},
		{
			name:            "second page",
			items:           items,
			total:           100,
			limit:           25,
			offset:          25,
			wantTotalPages:  4,
			wantCurrentPage: 2,
			wantHasNext:     true,
			wantHasPrev:     true,
		},
		{
			name:            "last page",
			items:           []string{"item1"},
			total:           76,
			limit:           25,
			offset:          75,
			wantTotalPages:  4,
			wantCurrentPage: 4,
			wantHasNext:     false,
			wantHasPrev:     true,
		},
		{
			name:            "single page",
			items:           items,
			total:           3,
			limit:           25,
			offset:          0,
			wantTotalPages:  1,
			wantCurrentPage: 1,
			wantHasNext:     false,
			wantHasPrev:     false,
		},
		{
			name:            "empty result",
			items:           []string{},
			total:           0,
			limit:           25,
			offset:          0,
			wantTotalPages:  1,
			wantCurrentPage: 1,
			wantHasNext:     false,
			wantHasPrev:     false,
		},
		{
			name:            "zero limit uses default",
			items:           items,
			total:           100,
			limit:           0,
			offset:          0,
			wantTotalPages:  4, // 100/25
			wantCurrentPage: 1,
			wantHasNext:     true,
			wantHasPrev:     false,
		},
		{
			name:            "exact page boundary",
			items:           items,
			total:           75,
			limit:           25,
			offset:          50,
			wantTotalPages:  3,
			wantCurrentPage: 3,
			wantHasNext:     true,  // 50+3=53 < 75, so there are more items
			wantHasPrev:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewResult(tt.items, tt.total, tt.limit, tt.offset)

			if result.Total != tt.total {
				t.Errorf("NewResult().Total = %d, want %d", result.Total, tt.total)
			}
			if len(result.Items) != len(tt.items) {
				t.Errorf("len(NewResult().Items) = %d, want %d", len(result.Items), len(tt.items))
			}
			if result.TotalPages != tt.wantTotalPages {
				t.Errorf("NewResult().TotalPages = %d, want %d", result.TotalPages, tt.wantTotalPages)
			}
			if result.CurrentPage != tt.wantCurrentPage {
				t.Errorf("NewResult().CurrentPage = %d, want %d", result.CurrentPage, tt.wantCurrentPage)
			}
			if result.HasNext != tt.wantHasNext {
				t.Errorf("NewResult().HasNext = %v, want %v", result.HasNext, tt.wantHasNext)
			}
			if result.HasPrev != tt.wantHasPrev {
				t.Errorf("NewResult().HasPrev = %v, want %v", result.HasPrev, tt.wantHasPrev)
			}
		})
	}
}

func TestPagination_NextPage(t *testing.T) {
	tests := []struct {
		name      string
		p         Pagination
		total     int64
		wantNil   bool
		wantLimit int
		wantOffset int
	}{
		{
			name:       "has next page",
			p:          Pagination{Limit: 25, Offset: 0},
			total:      100,
			wantNil:    false,
			wantLimit:  25,
			wantOffset: 25,
		},
		{
			name:       "last page has no next",
			p:          Pagination{Limit: 25, Offset: 75},
			total:      100,
			wantNil:    true,
		},
		{
			name:       "exact boundary has no next",
			p:          Pagination{Limit: 25, Offset: 75},
			total:      100,
			wantNil:    true,
		},
		{
			name:       "total less than offset has no next",
			p:          Pagination{Limit: 25, Offset: 100},
			total:      50,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.NextPage(tt.total)

			if tt.wantNil {
				if got != nil {
					t.Errorf("NextPage() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Error("NextPage() = nil, want non-nil")
				return
			}

			if got.Limit != tt.wantLimit {
				t.Errorf("NextPage().Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("NextPage().Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
		})
	}
}

func TestPagination_PrevPage(t *testing.T) {
	tests := []struct {
		name       string
		p          Pagination
		wantNil    bool
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "has previous page",
			p:          Pagination{Limit: 25, Offset: 25},
			wantNil:    false,
			wantLimit:  25,
			wantOffset: 0,
		},
		{
			name:    "first page has no previous",
			p:       Pagination{Limit: 25, Offset: 0},
			wantNil: true,
		},
		{
			name:       "offset less than limit",
			p:          Pagination{Limit: 25, Offset: 10},
			wantNil:    false,
			wantLimit:  25,
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.PrevPage()

			if tt.wantNil {
				if got != nil {
					t.Errorf("PrevPage() = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Error("PrevPage() = nil, want non-nil")
				return
			}

			if got.Limit != tt.wantLimit {
				t.Errorf("PrevPage().Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("PrevPage().Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
		})
	}
}

func TestPagination_CalculateLimit(t *testing.T) {
	tests := []struct {
		name         string
		p            Pagination
		defaultLimit int
		want         int
	}{
		{
			name:         "zero limit uses default",
			p:            Pagination{Limit: 0},
			defaultLimit: 25,
			want:         25,
		},
		{
			name:         "positive limit returns value",
			p:            Pagination{Limit: 50},
			defaultLimit: 25,
			want:         50,
		},
		{
			name:         "negative limit uses default",
			p:            Pagination{Limit: -5},
			defaultLimit: 25,
			want:         25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.CalculateLimit(tt.defaultLimit)
			if got != tt.want {
				t.Errorf("CalculateLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPagination_CalculateOffset(t *testing.T) {
	tests := []struct {
		name string
		p    Pagination
		want int
	}{
		{
			name: "positive offset",
			p:    Pagination{Offset: 100},
			want: 100,
		},
		{
			name: "zero offset",
			p:    Pagination{Offset: 0},
			want: 0,
		},
		{
			name: "negative offset",
			p:    Pagination{Offset: -10},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.CalculateOffset()
			if got != tt.want {
				t.Errorf("CalculateOffset() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPagination_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		p    Pagination
		want bool
	}{
		{
			name: "empty pagination",
			p:    Pagination{Limit: 0, Offset: 0},
			want: true,
		},
		{
			name: "non-empty limit",
			p:    Pagination{Limit: 25, Offset: 0},
			want: false,
		},
		{
			name: "non-empty offset",
			p:    Pagination{Limit: 0, Offset: 25},
			want: false,
		},
		{
			name: "both non-empty",
			p:    Pagination{Limit: 25, Offset: 50},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.IsEmpty()
			if got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPagination_Clone(t *testing.T) {
	original := Pagination{Limit: 25, Offset: 50}
	clone := original.Clone()

	if original != clone {
		t.Errorf("Clone() = %+v, want %+v", clone, original)
	}

	// Modify clone and verify original is unchanged
	clone.Limit = 50
	if original.Limit == clone.Limit {
		t.Error("Clone modification affected original")
	}
}

func TestPagination_ToLimitOffset(t *testing.T) {
	cfg := Config{
		DefaultLimit: 25,
		MaxLimit:     100,
		MaxOffset:    10000,
	}

	tests := []struct {
		name       string
		p          Pagination
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "normal values",
			p:          Pagination{Limit: 50, Offset: 100},
			wantLimit:  50,
			wantOffset: 100,
		},
		{
			name:       "zero limit uses default",
			p:          Pagination{Limit: 0, Offset: 100},
			wantLimit:  25,
			wantOffset: 100,
		},
		{
			name:       "limit capped at max",
			p:          Pagination{Limit: 200, Offset: 100},
			wantLimit:  100,
			wantOffset: 100,
		},
		{
			name:       "offset capped at max",
			p:          Pagination{Limit: 50, Offset: 20000},
			wantLimit:  50,
			wantOffset: 10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.ToLimitOffset(cfg)
			if got.Limit != tt.wantLimit {
				t.Errorf("ToLimitOffset().Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("ToLimitOffset().Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
		})
	}
}

func TestResult_Generic(t *testing.T) {
	// Test with different types
	t.Run("string type", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		result := NewResult(items, 10, 3, 0)
		if len(result.Items) != 3 {
			t.Errorf("len(Items) = %d, want 3", len(result.Items))
		}
	})

	t.Run("int type", func(t *testing.T) {
		items := []int{1, 2, 3}
		result := NewResult(items, 10, 3, 0)
		if result.Items[0] != 1 {
			t.Errorf("Items[0] = %d, want 1", result.Items[0])
		}
	})

	t.Run("struct type", func(t *testing.T) {
		type TestStruct struct {
			ID   int
			Name string
		}
		items := []TestStruct{{ID: 1, Name: "test"}}
		result := NewResult(items, 1, 10, 0)
		if result.Items[0].Name != "test" {
			t.Errorf("Items[0].Name = %s, want test", result.Items[0].Name)
		}
	})
}

func TestPagination_ValidateWithDefaults(t *testing.T) {
	// Test that ValidateWithDefaults uses default configuration
	p := Pagination{Limit: 0, Offset: 0}
	err := p.ValidateWithDefaults()

	if err != nil {
		t.Errorf("ValidateWithDefaults() error = %v, want nil", err)
	}

	// After validation, Limit should be set to default (25)
	if p.Limit != 25 {
		t.Errorf("After ValidateWithDefaults(), Limit = %d, want 25", p.Limit)
	}
}

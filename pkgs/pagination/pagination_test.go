package pagination

import "testing"

func TestPagination_GetPageSize(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero returns default", 0, defaultPageSize},
		{"negative returns default", -5, defaultPageSize},
		{"positive returns itself", 30, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{PageSize: tt.in}
			if got := p.GetPageSize(); got != tt.want {
				t.Errorf("GetPageSize() = %d, want %d", got, tt.want)
			}
			if p.PageSize != tt.in {
				t.Errorf("GetPageSize mutated PageSize: %d -> %d", tt.in, p.PageSize)
			}
		})
	}
}

func TestPagination_GetPage(t *testing.T) {
	t.Run("explicit page size", func(t *testing.T) {
		p := &Pagination{Page: 3, PageSize: 10}
		if got := p.GetPage(); got != 20 {
			t.Errorf("GetPage() = %d, want 20", got)
		}
	})
	t.Run("default page size", func(t *testing.T) {
		p := &Pagination{Page: 2}
		want := 1 * defaultPageSize
		if got := p.GetPage(); got != want {
			t.Errorf("GetPage() = %d, want %d", got, want)
		}
	})
}

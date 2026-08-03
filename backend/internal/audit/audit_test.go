package audit

import "testing"

// TestEntryFilterNormalizeLimit covers the audit-browser listing's limit
// resolution: unset/non-positive falls back to a sane default, anything
// over the hard cap is clamped down to it, and anything in between passes
// through unchanged — this is what keeps GET /audit from ever turning into
// an unbounded full-table scan regardless of what a caller asks for.
func TestEntryFilterNormalizeLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{"zero falls back to the default", 0, defaultEntryFilterLimit},
		{"negative falls back to the default", -5, defaultEntryFilterLimit},
		{"a reasonable positive value passes through unchanged", 10, 10},
		{"exactly the cap passes through unchanged", maxEntryFilterLimit, maxEntryFilterLimit},
		{"over the cap is clamped down to it", maxEntryFilterLimit + 1000, maxEntryFilterLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := EntryFilter{Limit: tt.limit}
			if got := f.NormalizeLimit(); got != tt.want {
				t.Errorf("NormalizeLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

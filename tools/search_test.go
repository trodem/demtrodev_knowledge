package tools

import "testing"

func TestParseSelectionIndex(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want int
		ok   bool
	}{
		{"1", 5, 0, true},
		{"5", 5, 4, true},
		{"0", 5, -1, false},
		{"6", 5, -1, false},
		{"abc", 5, -1, false},
	}

	for _, tt := range tests {
		got, ok := parseSelectionIndex(tt.in, tt.max)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("parseSelectionIndex(%q,%d) => (%d,%v), want (%d,%v)", tt.in, tt.max, got, ok, tt.want, tt.ok)
		}
	}
}

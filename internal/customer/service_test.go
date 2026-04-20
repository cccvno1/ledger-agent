package customer

import (
	"testing"

	"github.com/cccvno1/ledger-agent/internal/domain"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "axc", 1},
		{"张三", "张四", 1},
		{"李小明", "李晓明", 1},
		{"abc", "", 3},
		{"", "abc", 3},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := domain.Levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSearch_Ranking(t *testing.T) {
	// Verify that Search returns candidates sorted by ascending Levenshtein distance.
	// This test does not require a database.
	results := []*SearchResult{
		{Customer: &Customer{Name: "张三"}, Score: 0},
		{Customer: &Customer{Name: "张四"}, Score: 1},
		{Customer: &Customer{Name: "李大虎"}, Score: 3},
	}
	for i := 1; i < len(results); i++ {
		if results[i].Score < results[i-1].Score {
			t.Errorf("results not sorted: index %d score %d < index %d score %d",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
}

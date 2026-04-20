package chat

import (
	"testing"
	"time"
)

func TestParseDateExpr(t *testing.T) {
	// Wednesday 2026-04-15
	now := time.Date(2026, 4, 15, 14, 30, 0, 0, time.Local)

	tests := []struct {
		name    string
		expr    string
		want    string
		wantErr bool
	}{
		{"explicit date", "2026-04-10", "2026-04-10", false},
		{"today", "今天", "2026-04-15", false},
		{"yesterday", "昨天", "2026-04-14", false},
		{"day before yesterday", "前天", "2026-04-13", false},
		{"3 days ago literal", "大前天", "2026-04-12", false},
		{"N days ago", "3天前", "2026-04-12", false},
		{"7 days ago", "7天前", "2026-04-08", false},
		{"last monday", "上周一", "2026-04-06", false},
		{"last tuesday", "上周二", "2026-04-07", false},
		{"last wednesday", "上周三", "2026-04-08", false},
		{"last sunday", "上周日", "2026-04-12", false},
		{"last sunday alt", "上周天", "2026-04-12", false},
		{"last 星期一", "上星期一", "2026-04-06", false},
		{"empty", "", "", true},
		{"unknown", "下周一", "", true},
		{"gibberish", "abc", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateExpr(tt.expr, now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseDateExpr(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			gotStr := got.Format("2006-01-02")
			if gotStr != tt.want {
				t.Errorf("parseDateExpr(%q) = %s, want %s", tt.expr, gotStr, tt.want)
			}
		})
	}
}

func TestParseDateExpr_Sunday(t *testing.T) {
	// Sunday 2026-04-19
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.Local)

	got, err := parseDateExpr("上周一", now)
	if err != nil {
		t.Fatalf("parseDateExpr error = %v", err)
	}
	want := "2026-04-06"
	if got.Format("2006-01-02") != want {
		t.Errorf("parseDateExpr(上周一) from Sunday = %s, want %s", got.Format("2006-01-02"), want)
	}
}

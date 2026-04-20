package chat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reNDaysAgo = regexp.MustCompile(`^(\d+)\s*天前$`)
	weekdayMap = map[string]time.Weekday{
		"一": time.Monday,
		"二": time.Tuesday,
		"三": time.Wednesday,
		"四": time.Thursday,
		"五": time.Friday,
		"六": time.Saturday,
		"日": time.Sunday,
		"天": time.Sunday,
	}
)

// parseDateExpr parses a Chinese date expression relative to now.
// Supported: YYYY-MM-DD, 今天, 昨天, 前天, 大前天, N天前, 上周一~上周日.
func parseDateExpr(expr string, now time.Time) (time.Time, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return time.Time{}, fmt.Errorf("date expression is empty")
	}

	// Try YYYY-MM-DD first
	if t, err := time.Parse("2006-01-02", expr); err == nil {
		return t, nil
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch expr {
	case "今天":
		return today, nil
	case "昨天":
		return today.AddDate(0, 0, -1), nil
	case "前天":
		return today.AddDate(0, 0, -2), nil
	case "大前天":
		return today.AddDate(0, 0, -3), nil
	}

	// N天前
	if m := reNDaysAgo.FindStringSubmatch(expr); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n < 0 || n > 365 {
			return time.Time{}, fmt.Errorf("date expression out of range: %s", expr)
		}
		return today.AddDate(0, 0, -n), nil
	}

	// 上周X
	if strings.HasPrefix(expr, "上周") || strings.HasPrefix(expr, "上星期") || strings.HasPrefix(expr, "上礼拜") {
		suffix := expr
		for _, prefix := range []string{"上礼拜", "上星期", "上周"} {
			if strings.HasPrefix(expr, prefix) {
				suffix = strings.TrimPrefix(expr, prefix)
				break
			}
		}
		wd, ok := weekdayMap[suffix]
		if !ok {
			return time.Time{}, fmt.Errorf("unknown weekday in expression: %s", expr)
		}
		// Find last week's day: go back to last week's Monday, then add offset
		todayWD := today.Weekday()
		if todayWD == time.Sunday {
			todayWD = 7
		}
		targetWD := int(wd)
		if targetWD == 0 {
			targetWD = 7
		}
		// Days from today to last Monday = todayWD - 1 + 7
		daysBack := int(todayWD) - 1 + 7
		lastMonday := today.AddDate(0, 0, -daysBack)
		return lastMonday.AddDate(0, 0, targetWD-1), nil
	}

	return time.Time{}, fmt.Errorf("unrecognized date expression: %s", expr)
}

package period

import (
	"fmt"
	"strings"
	"time"
)

// Interval kinds used in the navbar (Fava-style).
const (
	KindEmpty   = ""
	KindYear    = "year"
	KindQuarter = "quarter"
	KindMonth   = "month"
	KindWeek    = "week"
	KindDay     = "day"
	KindCustom  = "custom"
)

// Kind returns the interval kind of a filter expression.
func Kind(filter string, now time.Time) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return KindEmpty
	}
	if strings.Contains(filter, " - ") {
		return KindCustom
	}
	if reRel.MatchString(filter) {
		m := reRel.FindStringSubmatch(filter)
		return m[1]
	}
	switch {
	case reDay.MatchString(filter):
		return KindDay
	case reWeek.MatchString(filter):
		return KindWeek
	case reMonth.MatchString(filter):
		return KindMonth
	case reQuarter.MatchString(filter):
		return KindQuarter
	case reYear.MatchString(filter):
		return KindYear
	default:
		return KindCustom
	}
}

// DisplayLabel is a human label for the filter (e.g. "March 2024").
func DisplayLabel(filter string, now time.Time) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "All time"
	}
	r, err := Parse(filter, now)
	if err != nil {
		return filter
	}
	kind := Kind(filter, now)
	if r.Start.IsZero() {
		return r.Label()
	}
	switch kind {
	case KindYear:
		return r.Start.Format("2006")
	case KindQuarter:
		q := (int(r.Start.Month())-1)/3 + 1
		return fmt.Sprintf("Q%d %d", q, r.Start.Year())
	case KindMonth:
		return r.Start.Format("January 2006")
	case KindWeek:
		y, w := r.Start.ISOWeek()
		return fmt.Sprintf("Week %d, %d", w, y)
	case KindDay:
		return r.Start.Format("Mon 2 Jan 2006")
	default:
		return r.Label()
	}
}

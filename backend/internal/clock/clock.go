package clock

import (
	"time"
)

// Beijing is GMT+8. All persisted timestamps use this zone's wall clock
// stored as timezone-naive values (Requirements NFR-8).
var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing).Truncate(time.Microsecond)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func Date(t time.Time) time.Time {
	loc := t.In(Beijing)
	y, m, d := loc.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, Beijing)
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

func Parse(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, Beijing)
}

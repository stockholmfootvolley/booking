package calendar

import (
	"time"
)

func TimeToID(date string) string {
	return TimeParse(date).Format("2006-01-02")
}

func TimeParse(date string) time.Time {
	t, _ := time.Parse(time.RFC3339, date)
	return t
}

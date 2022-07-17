package model

import "time"

type Level uint64

const (
	User       string = "user"
	Token      string = "token"
	DateLayout string = "2006-01-02"
)

const (
	Beginner Level = iota
	Medium
	Advanced
)

func (l Level) String() string {
	switch l {
	case Beginner:
		return "BEGINNER"
	case Medium:
		return "MEDIUM"
	case Advanced:
		return "ADVANCED"
	default:
		return "BEGINNER"
	}
}

func StringToLevel(s string) Level {
	switch s {
	case Beginner.String():
		return Beginner
	case Medium.String():
		return Medium
	case Advanced.String():
		return Advanced
	default:
		return Beginner
	}
}

func TimeToID(date string) string {
	return TimeParse(date).Format("2006-01-02")
}

func TimeParse(date string) time.Time {
	t, _ := time.Parse(time.RFC3339, date)
	return t
}

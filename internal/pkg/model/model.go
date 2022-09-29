package model

import (
	"time"
)

type Level uint64

const (
	User                   string = "user"
	Token                  string = "token"
	DateLayout             string = "2006-01-02"
	DefaultMaxParticipants int    = 10
)

const (
	Basic Level = iota
	Medium
	Advanced
)

func (l Level) String() string {
	switch l {
	case Basic:
		return "BASIC"
	case Medium:
		return "MEDIUM"
	case Advanced:
		return "ADVANCED"
	default:
		return "BASIC"
	}
}

func StringToLevel(s string) Level {
	switch s {
	case Basic.String():
		return Basic
	case Medium.String():
		return Medium
	case Advanced.String():
		return Advanced
	default:
		return Basic
	}
}

func TimeToID(date string) string {
	return TimeParse(date).Format("2006-01-02")
}

func TimeParse(date string) *time.Time {
	t, err := time.Parse(time.RFC3339, date)

	if err != nil {
		return nil
	}
	return &t
}

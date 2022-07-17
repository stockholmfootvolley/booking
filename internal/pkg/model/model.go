package model

type Level uint64

const (
	User  string = "user"
	Token string = "token"
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

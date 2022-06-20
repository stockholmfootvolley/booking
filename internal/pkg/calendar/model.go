package calendar

import (
	"time"

	"google.golang.org/api/calendar/v3"
)

type Event struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Date      time.Time `json:"date"`
	Attendees Attendees `json:"attendees"`
	Local     string    `json:"local"`
}

func GoogleEventToEvent(gEvent *calendar.Event) *Event {
	return &Event{
		ID:        TimeToID(gEvent.Start.DateTime),
		Date:      TimeParse(gEvent.Start.DateTime),
		Name:      gEvent.Summary,
		Attendees: getAttendesList(gEvent),
		Local:     gEvent.Location,
	}
}

func TimeToID(date string) string {
	return TimeParse(date).Format("2006-01-02")
}

func TimeParse(date string) time.Time {
	t, _ := time.Parse(time.RFC3339, date)
	return t
}

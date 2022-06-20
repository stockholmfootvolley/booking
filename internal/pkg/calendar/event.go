package calendar

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"
)

const (
	DateLayout = "2006-01-02"
)

type Attendees []Attendee
type Attendee struct {
	Name  string
	Phone string
}

func (c *Client) GetEvents() ([]*Event, error) {
	events, err := c.Service.Events.List(c.CalendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(time.Now().Format(time.RFC3339)).
		MaxResults(10).
		OrderBy("startTime").
		Do()

	if err != nil {
		return nil, err
	}

	var retEvents []*Event
	for _, ev := range events.Items {
		retEvents = append(retEvents, GoogleEventToEvent(ev))
	}
	return retEvents, nil
}

func (c *Client) GetEvent(date string) (*Event, error) {
	dateParsed, err := time.Parse(DateLayout, date)
	if err != nil {
		return nil, err
	}

	event, err := c.Service.Events.List(c.CalendarID).
		ShowDeleted(false).
		//SingleEvents(true).
		TimeMin(dateParsed.Format(time.RFC3339)).
		TimeMax(dateParsed.Add(24 * time.Hour).Format(time.RFC3339)).
		MaxResults(10).
		Do()
	if err != nil {
		return nil, err
	}
	for _, e := range event.Items {
		return GoogleEventToEvent(e), nil
	}
	return nil, err
}

func (c *Client) AddAttendee(event *calendar.Event, name string, phone string) error {

	attendesList := getAttendesList(event)
	attendesList = append(attendesList, Attendee{
		Name:  name,
		Phone: phone,
	})
	event.Description = attendesList.String()
	_, err := c.Service.Events.Update(c.CalendarID, event.Id, event).Do()
	return err
}

func (c *Client) RemoveAttende(event *calendar.Event, name string, phone string) error {

	attendesList := getAttendesList(event)

	for index, _ := range attendesList {
		if attendesList[index].Name == name && attendesList[index].Phone == phone {
			attendesList = append(attendesList[:index], attendesList[index+1:]...)
			break
		}
	}

	event.Description = attendesList.String()
	_, err := c.Service.Events.Update(c.CalendarID, event.Id, event).Do()
	return err
}

func getAttendesList(event *calendar.Event) Attendees {
	attendes := []Attendee{}
	if event.Description == "" {
		return attendes
	}

	attendesString := strings.Split(event.Description, "\n")
	for _, attende := range attendesString {
		attendeProps := strings.Split(attende, "-")
		attendes = append(attendes, Attendee{
			Name:  attendeProps[0],
			Phone: attendeProps[1],
		})
	}
	return attendes
}

func (a Attendees) String() string {
	var attendesList string
	for _, b := range a {
		attendesList += fmt.Sprintln(b.Name, "-", b.Phone)
	}
	return attendesList
}

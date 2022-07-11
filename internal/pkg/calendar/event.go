package calendar

import (
	"time"

	"go.uber.org/zap"
	"google.golang.org/api/calendar/v3"
	"gopkg.in/yaml.v2"
)

const (
	DateLayout = "2006-01-02"
)

type Attendee struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Description struct {
	Price     int        `yaml:"price"`
	Attendees []Attendee `yaml:"attendes"`
}
type Event struct {
	ID        string     `json:"id"`
	Price     int        `json:"price"`
	Name      string     `json:"name"`
	Date      time.Time  `json:"date"`
	Attendees []Attendee `json:"attendees"`
	Local     string     `json:"local"`
}

func GoogleEventToEvent(gEvent *calendar.Event) (*Event, error) {
	description, err := readDescription(gEvent.Description)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:        TimeToID(gEvent.Start.DateTime),
		Date:      TimeParse(gEvent.Start.DateTime),
		Name:      gEvent.Summary,
		Attendees: description.Attendees,
		Price:     description.Price,
		Local:     gEvent.Location,
	}, nil
}

func readDescription(description string) (*Description, error) {
	descObj := &Description{
		Attendees: []Attendee{},
	}
	err := yaml.Unmarshal([]byte(description), descObj)
	if err != nil {
		return nil, err
	}
	return descObj, nil
}

func (d *Description) String() string {
	content, err := yaml.Marshal(d)
	if err != nil {
		return ""
	}
	return string(content)
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

	retEvents := []*Event{}
	for _, ev := range events.Items {
		e, err := GoogleEventToEvent(ev)
		if err != nil {
			return nil, err
		}
		retEvents = append(retEvents, e)
	}
	return retEvents, nil
}

func (c *Client) GetEvent(date string) (*calendar.Event, error) {
	dateParsed, err := time.Parse(DateLayout, date)
	if err != nil {
		return nil, err
	}

	event, err := c.Service.Events.List(c.CalendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(dateParsed.Format(time.RFC3339)).
		TimeMax(dateParsed.Add(24 * time.Hour).Format(time.RFC3339)).
		MaxResults(10).
		Do()
	if err != nil {
		return nil, err
	}

	for _, e := range event.Items {
		return e, nil
	}

	return nil, err
}

func (c *Client) UpdateEvent(eventDate string, newAttende *Attendee) (*Event, error) {
	oldEvent, err := c.GetEvent(eventDate)
	if err != nil {
		return nil, err
	}

	c.Logger.Info("updating event",
		zap.Any("event", oldEvent),
		zap.Any("attendes", newAttende),
	)

	description, err := readDescription(oldEvent.Description)
	if err != nil {
		return nil, err
	}

	if newAttende != nil {
		description.Attendees = append(description.Attendees, *newAttende)
	}

	oldEvent.Description = description.String()
	newEvent, err := c.Service.Events.Update(c.CalendarID, oldEvent.Id, oldEvent).
		Do()
	if err != nil {
		c.Logger.Error("failed to update event", zap.Error(err))
		return nil, err
	}
	return GoogleEventToEvent(newEvent)
}

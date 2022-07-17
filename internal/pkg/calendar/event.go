package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"go.uber.org/zap"
	"google.golang.org/api/calendar/v3"
	"gopkg.in/yaml.v2"
)

const (
	DateLayout = "2006-01-02"
)

type Attendee struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Description struct {
	Price     int        `yaml:"price"`
	Attendees []Attendee `yaml:"attendes"`
	Level     string     `yaml:"level"`
}
type Event struct {
	ID        string     `json:"id"`
	Price     int        `json:"price"`
	Name      string     `json:"name"`
	Date      time.Time  `json:"date"`
	Attendees []Attendee `json:"attendees"`
	Local     string     `json:"local"`
	Level     string     `json:"level"`
}

func GoogleEventToEvent(gEvent *calendar.Event) (*Event, error) {
	description, err := readDescription(gEvent.Description)
	if err != nil {
		return nil, err
	}

	level := model.StringToLevel(description.Level)

	return &Event{
		ID:        TimeToID(gEvent.Start.DateTime),
		Date:      TimeParse(gEvent.Start.DateTime),
		Name:      gEvent.Summary,
		Attendees: description.Attendees,
		Price:     description.Price,
		Local:     gEvent.Location,
		Level:     level.String(),
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

func (c *Client) GetEvents(ctx context.Context) ([]*Event, error) {
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

	user := ctx.Value(model.User)
	userInfo := user.(spreadsheet.User)

	retEvents := []*Event{}
	for _, ev := range events.Items {

		e, err := GoogleEventToEvent(ev)

		if userInfo.Level < model.StringToLevel(e.Level) {
			continue
		}

		if err != nil {
			return nil, err
		}
		retEvents = append(retEvents, e)
	}
	return retEvents, nil
}

func (c *Client) GetEvent(ctx context.Context, date string) (*calendar.Event, error) {
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

func (c *Client) GetSingleEvent(ctx context.Context, eventDate string) (*calendar.Event, *Description, error) {
	oldEvent, err := c.GetEvent(ctx, eventDate)
	if err != nil {
		return nil, nil, err
	}

	description, err := readDescription(oldEvent.Description)
	if err != nil {
		return nil, nil, err
	}

	userInfo := ctx.Value(model.User).(spreadsheet.User)
	if userInfo.Level < model.StringToLevel(description.Level) {
		return nil, nil, errors.New("user has no compatible level")
	}

	return oldEvent, description, nil
}

func (c *Client) AddAttendeeEvent(ctx context.Context, eventDate string, newAttende *Attendee) (*Event, error) {
	oldEvent, description, err := c.GetSingleEvent(ctx, eventDate)
	if err != nil {
		return nil, err
	}

	userInfo := ctx.Value(model.User).(spreadsheet.User)
	if userInfo.Level < model.StringToLevel(description.Level) {
		return nil, errors.New("user has no compatible level")
	}

	c.Logger.Info("updating event",
		zap.Any("event", oldEvent),
		zap.Any("attendes", newAttende),
	)

	for index := range description.Attendees {
		if description.Attendees[index].Name == newAttende.Name && description.Attendees[index].Email == newAttende.Email {
			return GoogleEventToEvent(oldEvent)
		}
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

func (c *Client) RemoveAttendee(ctx context.Context, eventDate string, removeAttendee *Attendee) (*Event, error) {
	oldEvent, description, err := c.GetSingleEvent(ctx, eventDate)
	if err != nil {
		return nil, err
	}

	c.Logger.Info("removing attendee",
		zap.Any("event", oldEvent),
		zap.Any("attendes", removeAttendee),
	)

	userInfo := ctx.Value(model.User).(spreadsheet.User)
	if userInfo.Level < model.StringToLevel(description.Level) {
		return nil, errors.New("user has no compatible level")
	}

	for index := range description.Attendees {
		if description.Attendees[index].Name == removeAttendee.Name && description.Attendees[index].Email == removeAttendee.Email {
			description.Attendees = append(description.Attendees[:index], description.Attendees[index+1:]...)
			break
		}
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

package calendar

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"

	"go.uber.org/zap"
	"google.golang.org/api/calendar/v3"
	"gopkg.in/yaml.v2"
)

type Attendee struct {
	Name     string    `json:"name" yaml:"name"`
	Email    string    `json:"email" yaml:"email"`
	SignTime time.Time `json:"sign_time" yaml:"sign_time"`
}

type Description struct {
	Price           int        `yaml:"price"`
	Attendees       []Attendee `yaml:"attendes"`
	Level           string     `yaml:"level,omitempty"`
	MaxParticipants int        `yaml:"max_participants"`
}
type Event struct {
	ID              string     `json:"id"`
	Price           int        `json:"price"`
	Name            string     `json:"name"`
	Date            time.Time  `json:"date"`
	Attendees       []Attendee `json:"attendees"`
	Local           string     `json:"local"`
	Level           string     `json:"level"`
	MaxParticipants int        `json:"max_participants"`
}

func GoogleEventToEvent(gEvent *calendar.Event) (*Event, error) {
	description, err := readDescription(gEvent.Description)
	if err != nil {
		return nil, err
	}

	level := model.StringToLevel(description.Level)

	maxParticipants := description.MaxParticipants
	if maxParticipants == 0 {
		maxParticipants = model.DefaultMaxParticipants
	}

	return &Event{
		ID:              model.TimeToID(gEvent.Start.DateTime),
		Date:            model.TimeParse(gEvent.Start.DateTime),
		Name:            gEvent.Summary,
		Attendees:       description.Attendees,
		Price:           description.Price,
		Local:           gEvent.Location,
		Level:           level.String(),
		MaxParticipants: maxParticipants,
	}, nil
}

func readDescription(description string) (*Description, error) {
	descObj := &Description{
		Attendees: []Attendee{},
	}

	// remove html
	withBreakline := strings.ReplaceAll(description, "<br>", "\n")
	noNbsp := strings.ReplaceAll(withBreakline, "&nbsp;", " ")

	p := bluemonday.StrictPolicy()
	nonHtml := p.Sanitize(noNbsp)

	err := yaml.Unmarshal([]byte(nonHtml), descObj)
	if err != nil {
		return nil, err
	}

	sort.Slice(descObj.Attendees, func(i, j int) bool {
		return descObj.Attendees[j].SignTime.After(descObj.Attendees[i].SignTime)
	})

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
	dateParsed, err := time.Parse(model.DateLayout, date)
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

func (c *Client) AddAttendeeEvent(ctx context.Context, eventDate string) (*Event, error) {
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
		zap.Any("attendes", userInfo),
	)

	for index := range description.Attendees {
		if description.Attendees[index].Name == userInfo.Name && description.Attendees[index].Email == userInfo.Email {
			return GoogleEventToEvent(oldEvent)
		}
	}
	description.Attendees = append(description.Attendees, Attendee{
		Name:     userInfo.Name,
		Email:    userInfo.Email,
		SignTime: time.Now(),
	})

	oldEvent.Description = description.String()
	newEvent, err := c.Service.Events.Update(c.CalendarID, oldEvent.Id, oldEvent).
		Do()
	if err != nil {
		c.Logger.Error("failed to update event", zap.Error(err))
		return nil, err
	}
	return GoogleEventToEvent(newEvent)
}

func (c *Client) RemoveAttendee(ctx context.Context, eventDate string) (*Event, error) {
	oldEvent, description, err := c.GetSingleEvent(ctx, eventDate)
	if err != nil {
		return nil, err
	}
	userInfo := ctx.Value(model.User).(spreadsheet.User)

	c.Logger.Info("removing attendee",
		zap.Any("event", oldEvent),
		zap.Any("attendes", userInfo),
	)

	if userInfo.Level < model.StringToLevel(description.Level) {
		return nil, errors.New("user has no compatible level")
	}

	for index := range description.Attendees {
		if description.Attendees[index].Name == userInfo.Name && description.Attendees[index].Email == userInfo.Email {
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

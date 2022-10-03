package calendar

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"github.com/microcosm-cc/bluemonday"
	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"

	"google.golang.org/api/calendar/v3"
	"gopkg.in/yaml.v2"
)

type Attendee struct {
	Name     string     `json:"name" yaml:"name"`
	Email    string     `json:"email" yaml:"email"`
	SignTime time.Time  `json:"sign_time" yaml:"sign_time"`
	PaidTime *time.Time `json:"paid_time" yaml:"paid_time"`
}

type Payments []Payment
type Description struct {
	Price           int        `yaml:"price"`
	Attendees       []Attendee `yaml:"attendes"`
	Level           string     `yaml:"level,omitempty"`
	MaxParticipants int        `yaml:"max_participants"`
	Payments        Payments   `yaml:"payments"`
}

type Payment struct {
	Email         string    `yaml:"email"`
	PaidTimestamp time.Time `yaml:"paid_timestamp"`
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
	QrCode          string     `json:"qr_code"`
}

func (c *Client) GoogleEventToEvent(gEvent *calendar.Event) (*Event, error) {
	description, err := readDescription(gEvent.Description)
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not read description",
				"event":   err,
			}},
		)
		return nil, err
	}

	level := model.StringToLevel(description.Level)

	maxParticipants := description.MaxParticipants
	if maxParticipants == 0 {
		maxParticipants = model.DefaultMaxParticipants
	}

	retEvent := Event{
		ID:              model.TimeToID(gEvent.Start.DateTime),
		Date:            *model.TimeParse(gEvent.Start.DateTime),
		Name:            gEvent.Summary,
		Attendees:       description.Attendees,
		Price:           description.Price,
		Local:           gEvent.Location,
		Level:           level.String(),
		MaxParticipants: maxParticipants,
	}

	if description.Price > 0 {
		qrcode, err := c.Swish.GenerateQrCode(description.Price, description.Level, model.TimeToID(gEvent.Start.DateTime))
		if err != nil {
			c.Logger.Log(logging.Entry{
				Severity: logging.Error,
				Payload: map[string]interface{}{
					"message": "could not read description",
					"event":   err,
				}},
			)
			return nil, err
		}
		retEvent.QrCode = qrcode
	}

	return &retEvent, nil
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

	retEvents := []*Event{}
	for _, ev := range events.Items {

		e, err := c.GoogleEventToEvent(ev)

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
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not parse time",
				"error":   err,
			}},
		)
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
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not list events from google",
				"error":   err,
			}},
		)
		return nil, err
	}

	for _, e := range event.Items {
		return e, nil
	}

	return nil, err
}

func (c *Client) GetSingleEvent(ctx context.Context, eventDate string, userInfo *spreadsheet.User) (*calendar.Event, *Description, error) {
	oldEvent, err := c.GetEvent(ctx, eventDate)
	if err != nil {
		return nil, nil, err
	}

	description, err := readDescription(oldEvent.Description)
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not read description",
				"event":   err,
			}},
		)
		return nil, nil, err
	}

	return oldEvent, description, nil
}

func (c *Client) AddAttendeeEvent(ctx context.Context, eventDate string, payment *Payment, userInfo *spreadsheet.User) (*Event, error) {
	oldEvent, description, err := c.GetSingleEvent(ctx, eventDate, userInfo)
	if err != nil {
		return nil, err
	}

	if userInfo.Level < model.StringToLevel(description.Level) {
		return nil, errors.New("user has no compatible level")
	}

	c.Logger.Log(logging.Entry{
		Severity: logging.Info,
		Payload: map[string]interface{}{
			"message":  "updating event",
			"event":    oldEvent,
			"attendes": userInfo,
		}},
	)

	for index := range description.Attendees {
		if description.Attendees[index].Name == userInfo.Name && description.Attendees[index].Email == userInfo.Email {
			return c.GoogleEventToEvent(oldEvent)
		}
	}
	description.Attendees = append(description.Attendees, Attendee{
		Name:     userInfo.Name,
		Email:    userInfo.Email,
		SignTime: time.Now(),
	})

	if payment != nil {
		description.Payments = append(description.Payments, *payment)
	}

	oldEvent.Description = description.String()
	newEvent, err := c.Service.Events.Update(c.CalendarID, oldEvent.Id, oldEvent).
		Do()
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "failed to update event",
				"error":   err,
			}},
		)

		return nil, err
	}
	return c.GoogleEventToEvent(newEvent)
}

func (c *Client) RemoveAttendee(ctx context.Context, eventDate string, userInfo *spreadsheet.User) (*Event, error) {
	oldEvent, description, err := c.GetSingleEvent(ctx, eventDate, userInfo)
	if err != nil {
		return nil, err
	}

	c.Logger.Log(logging.Entry{
		Severity: logging.Info,
		Payload: map[string]interface{}{
			"message":  "removing attendee",
			"event":    oldEvent,
			"attendes": userInfo,
		}},
	)

	if userInfo.Level < model.StringToLevel(description.Level) {
		return nil, errors.New("user has no compatible level")
	}

	for index := range description.Attendees {
		if strings.EqualFold(description.Attendees[index].Email, userInfo.Email) {
			description.Attendees = append(description.Attendees[:index], description.Attendees[index+1:]...)
			break
		}
	}

	oldEvent.Description = description.String()
	newEvent, err := c.Service.Events.
		Update(c.CalendarID, oldEvent.Id, oldEvent).
		Do()
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "failed to update event",
				"error":   err,
			}},
		)
		return nil, err
	}
	return c.GoogleEventToEvent(newEvent)
}

func (p Payments) HasUserPaid(email string) bool {
	for _, payment := range p {
		if strings.EqualFold(email, payment.Email) {
			return true
		}
	}
	return false
}

func (c *Client) UpdateEvent(ctx context.Context, eventDate string, userInfo *spreadsheet.User) (*Event, error) {
	oldEvent, description, err := c.GetSingleEvent(ctx, eventDate, userInfo)
	if err != nil {
		return nil, err
	}

	for index, attendee := range description.Attendees {
		if attendee.Email == userInfo.Email {
			if description.Attendees[index].PaidTime == nil {
				now := time.Now()
				description.Attendees[index].PaidTime = &now
			} else {
				description.Attendees[index].PaidTime = nil
			}

			oldEvent.Description = description.String()
			newEvent, err := c.Service.Events.Update(c.CalendarID, oldEvent.Id, oldEvent).Do()
			if err != nil {
				c.Logger.Log(logging.Entry{
					Severity: logging.Error,
					Payload: map[string]interface{}{
						"message": "failed to update event",
						"error":   err,
					}},
				)
				return nil, err
			}
			return c.GoogleEventToEvent(newEvent)
		}
	}

	return nil, errors.New("user not found")
}

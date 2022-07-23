package calendar

import (
	"context"

	"cloud.google.com/go/logging"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Client struct {
	Service    *calendar.Service
	CalendarID string
	Logger     *logging.Logger
}

type API interface {
	GetEvents(ctx context.Context) ([]*Event, error)
	GetCalendars() (*calendar.CalendarList, error)
	AddAttendeeEvent(ctx context.Context, eventDate string, payment *Payment, userInfo *spreadsheet.User) (*Event, error)
	RemoveAttendee(ctx context.Context, eventDate string, userInfo *spreadsheet.User) (*Event, error)
	GetSingleEvent(ctx context.Context, eventDate string, userInfo *spreadsheet.User) (*calendar.Event, *Description, error)
}

func New(serviceAccount string, calendarID string, logger *logging.Logger) (*Client, error) {
	service, err := getClient(serviceAccount, logger)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "unable to retrieve Calendar client",
				"error":   err,
			}},
		)
		return nil, err
	}

	return &Client{
		CalendarID: calendarID,
		Service:    service,
		Logger:     logger,
	}, nil
}

func getClient(serviceAccount string, logger *logging.Logger) (*calendar.Service, error) {
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(serviceAccount), calendar.CalendarEventsScope)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "unable read credentials",
				"error":   err,
			}},
		)
		return nil, err
	}

	srv, err := calendar.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "unable authenticate to Calendar API",
				"error":   err,
			}},
		)
		return nil, err
	}

	return srv, err
}

func (c *Client) GetCalendars() (*calendar.CalendarList, error) {
	return c.Service.CalendarList.List().Do()
}

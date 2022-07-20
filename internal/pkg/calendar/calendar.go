package calendar

import (
	"context"

	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Client struct {
	Service    *calendar.Service
	CalendarID string
	Logger     *zap.Logger
}

type API interface {
	GetEvents(ctx context.Context) ([]*Event, error)
	GetCalendars() (*calendar.CalendarList, error)
	AddAttendeeEvent(ctx context.Context, eventDate string, payment *Payment, userInfo *spreadsheet.User) (*Event, error)
	RemoveAttendee(ctx context.Context, eventDate string) (*Event, error)
	GetSingleEvent(ctx context.Context, eventDate string) (*calendar.Event, *Description, error)
}

func New(serviceAccount string, calendarID string, logger *zap.Logger) (*Client, error) {
	service, err := getClient(serviceAccount, logger)
	if err != nil {
		logger.Error("unable to retrieve Calendar client", zap.Error(err))
		return nil, err
	}

	return &Client{
		CalendarID: calendarID,
		Service:    service,
		Logger:     logger,
	}, nil
}

func getClient(serviceAccount string, logger *zap.Logger) (*calendar.Service, error) {
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(serviceAccount), calendar.CalendarEventsScope)
	if err != nil {
		logger.Error("unable read credentials", zap.Error(err))
		return nil, err
	}

	srv, err := calendar.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		logger.Error("unable authenticate to Calendar API", zap.Error(err))
		return nil, err
	}

	return srv, err
}

func (c *Client) GetCalendars() (*calendar.CalendarList, error) {
	return c.Service.CalendarList.List().Do()
}

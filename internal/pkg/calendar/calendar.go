package calendar

import (
	"context"
	"log"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Client struct {
	Service    *calendar.Service
	CalendarID string
}

type API interface {
	GetEvents() (*calendar.Events, error)
}

func New(serviceAccount string, calendarID string) *Client {
	service, err := getClient(serviceAccount)
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v", err)
	}
	return &Client{
		CalendarID: calendarID,
		Service:    service,
	}
}

func getClient(serviceAccount string) (*calendar.Service, error) {
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(serviceAccount), calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("unable read credentials: %v", err)
	}

	srv, err := calendar.NewService(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Fatalf("unable authenticate to Calendar API: %v", err)
	}

	return srv, err
}

func (c *Client) GetEvents() (*calendar.Events, error) {
	return c.Service.Events.List(c.CalendarID).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(time.Now().Format(time.RFC3339)).
		MaxResults(10).
		OrderBy("startTime").
		Do()
}
func (c *Client) GetCalendars() (*calendar.CalendarList, error) {
	return c.Service.CalendarList.List().Do()
}

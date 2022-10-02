package main

import (
	"context"
	"encoding/base64"
	"log"

	"cloud.google.com/go/logging"
	"github.com/caarlos0/env"
	"github.com/stockholmfootvolley/booking/internal/app/rest"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"github.com/stockholmfootvolley/booking/internal/pkg/swish"
)

type config struct {
	ServiceAccount string `env:"SERVICE_ACCOUNT,required"`
	CalendarID     string `env:"CALENDAR_ID,required"`
	SpreadsheetID  string `env:"SPREADSHEET_ID,required"`
	ClientID       string `env:"CLIENT_ID,required"`
	Port           string `env:"PORT" envDefault:"8080"`
	ProjectID      string `env:"PROJECT_ID,required"`
	PhoneNumber    string `env:"PHONE_NUMBER" envDefault:"0724675429"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v\n", err)
	}

	serviceAccountPlainText, err := base64.RawStdEncoding.DecodeString(cfg.ServiceAccount)
	if err != nil {
		log.Fatalf("could not parse service account")
	}
	cfg.ServiceAccount = string(serviceAccountPlainText)

	// Creates a client.
	ctx := context.Background()
	client, err := logging.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	logger := client.Logger(cfg.ProjectID)
	if err != nil {
		log.Fatalf("could not start logger")
	}

	swish, err := swish.New(cfg.PhoneNumber, logger)
	if err != nil {
		log.Fatalf("could not swish logger")
	}

	calendarService, err := calendar.New(cfg.ServiceAccount, cfg.CalendarID, logger, swish)
	if err != nil {
		log.Fatalf("could not start calendar service")
	}

	spreadsheetService, err := spreadsheet.New(cfg.ServiceAccount, cfg.SpreadsheetID, logger)
	if err != nil {
		log.Fatalf("could not start spreadsheet service")
	}

	restService := rest.New(
		calendarService,
		spreadsheetService,
		cfg.Port,
		cfg.ClientID,
		logger)
	restService.Serve()
}

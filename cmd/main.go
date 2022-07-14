package main

import (
	"encoding/base64"
	"log"

	"github.com/caarlos0/env"
	"github.com/stockholmfootvolley/booking/internal/app/rest"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"go.uber.org/zap"
)

type config struct {
	ServiceAccount string `env:"SERVICE_ACCOUNT,required"`
	CalendarID     string `env:"CALENDAR_ID,required"`
	SpreadsheetID  string `env:"SPREADSHEET_ID,required"`
	ClientID       string `env:"CLIENT_ID,required"`
	Port           string `env:"PORT" envDefault:"8080"`
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

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("could not start logger")
	}

	calendarService, err := calendar.New(cfg.ServiceAccount, cfg.CalendarID, logger)
	if err != nil {
		log.Fatalf("could not start calendar service")
	}

	spreadsheetService, err := spreadsheet.New(cfg.ServiceAccount, cfg.SpreadsheetID, logger)
	if err != nil {
		log.Fatalf("could not start spreadsheet service")
	}

	restService := rest.New(calendarService, spreadsheetService, cfg.Port, cfg.ClientID, logger)
	restService.Serve()
}

package main

import (
	"log"

	"github.com/caarlos0/env"
	"github.com/stockholmfootvolley/booking/internal/app/rest"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
)

type config struct {
	ServiceAccount string `env:"SERVICE_ACCOUNT"`
	CalendarID     string `env:"CALENDAR_ID"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("%+v\n", err)
	}

	calendarService := calendar.New(cfg.ServiceAccount, cfg.CalendarID)

	restService := rest.New(calendarService)
	restService.Serve()
}

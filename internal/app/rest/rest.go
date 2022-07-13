package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"go.uber.org/zap"
)

var (
	ErrGetEvents   = errors.New("could not retrieve events")
	ErrMarshalJSON = errors.New("could not marshal json")
)

type Server struct {
	calendarService calendar.API
	port            string
	logger          *zap.Logger
}

type API interface {
	Serve()
}

func New(calendarService calendar.API, port string, logger *zap.Logger) API {
	return &Server{
		calendarService: calendarService,
		port:            port,
		logger:          logger,
	}
}

func (s *Server) getEvents(c *gin.Context) {
	events, err := s.calendarService.GetEvents()

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, ErrGetEvents)
		return
	}

	c.IndentedJSON(http.StatusOK, events)
}

func (s *Server) getEvent(c *gin.Context) {
	eventDate := c.Param("date")
	event, err := s.calendarService.GetEvent(eventDate)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("could not found event for date "+eventDate))
		return
	}

	newEvent, err := calendar.GoogleEventToEvent(event)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("could not convert event "+eventDate))
		return
	}
	c.IndentedJSON(http.StatusOK, newEvent)
}

func (s *Server) updateEvent(c *gin.Context) {
	eventDate := c.Param("date")

	var attende calendar.Attendee

	err := json.NewDecoder(c.Request.Body).Decode(&attende)
	if err != nil {
		c.AbortWithError(
			http.StatusBadRequest,
			errors.New("no attendee passed"))
		return
	}

	newEvent, err := s.calendarService.UpdateEvent(eventDate, &attende)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("could not convert event "+eventDate))
		return
	}
	c.IndentedJSON(http.StatusOK, newEvent)
}

func (s *Server) Serve() {
	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AddAllowHeaders("authorization")
	router.Use(cors.New(config))
	router.GET("/events", s.getEvents)
	router.GET("/event/:date", s.getEvent)
	router.POST("/event/:date", s.updateEvent)
	router.Run("0.0.0.0:" + s.port)
}

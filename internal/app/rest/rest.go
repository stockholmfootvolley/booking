package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
)

var (
	ErrGetEvents   = errors.New("could not retrieve events")
	ErrMarshalJSON = errors.New("could not marshal json")
)

type Server struct {
	calendarService calendar.API
}

type API interface {
	Serve()
}

func New(calendarService calendar.API) API {
	return &Server{
		calendarService: calendarService,
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

	c.IndentedJSON(http.StatusOK, event)
}

func (s *Server) Serve() {
	router := gin.Default()
	router.GET("/events", s.getEvents)
	router.GET("/event/:date", s.getEvent)

	router.Run("0.0.0.0:8080")
}

package rest

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"go.uber.org/zap"
	"google.golang.org/api/idtoken"
)

var (
	ErrGetEvents   = errors.New("could not retrieve events")
	ErrMarshalJSON = errors.New("could not marshal json")
)

type Server struct {
	calendarService    calendar.API
	spreadsheetService spreadsheet.API
	listMembers        []string
	port               string
	logger             *zap.Logger
	clientID           string
}

type API interface {
	Serve()
}

func New(calendarService calendar.API, spreadsheetService spreadsheet.API, port string, clientID string, logger *zap.Logger) API {
	users, err := spreadsheetService.GetUsers()
	if err != nil {
		log.Fatalf("could not retrieve list of users")
	}

	return &Server{
		calendarService:    calendarService,
		spreadsheetService: spreadsheetService,
		listMembers:        users,
		port:               port,
		clientID:           clientID,
		logger:             logger,
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

func (s *Server) addPresence(c *gin.Context) {
	eventDate := c.Param("date")

	t, _ := c.Get("token")
	token := t.(*idtoken.Payload)

	newEvent, err := s.calendarService.AddAttendeeEvent(eventDate, &calendar.Attendee{
		Name:  getTokenName(token),
		Email: getTokenEmail(token),
	})

	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("could not convert event "+eventDate))
		return
	}
	c.IndentedJSON(http.StatusOK, newEvent)
}

func (s *Server) removePresence(c *gin.Context) {
	eventDate := c.Param("date")

	t, _ := c.Get("token")
	token := t.(*idtoken.Payload)

	newEvent, err := s.calendarService.RemoveAttendee(eventDate, &calendar.Attendee{
		Name:  getTokenName(token),
		Email: getTokenEmail(token),
	})

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

	// allow cors and authorization flow
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AddAllowHeaders("authorization")
	router.Use(cors.New(config))
	router.Use(s.addParsedToken())

	// and endpoints
	router.GET("/events", s.getEvents)
	router.GET("/event/:date", s.getEvent)
	router.POST("/event/:date", s.addPresence)
	router.DELETE("/event/:date", s.removePresence)
	router.Run("0.0.0.0:" + s.port)
}

func (s *Server) addParsedToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("authorization")
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		token = strings.ReplaceAll(token, "Bearer ", "")
		payload, err := idtoken.Validate(c.Request.Context(), token, s.clientID)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		for _, email := range s.listMembers {
			if email == getTokenEmail(payload) {
				c.Set("token", payload)
				c.Next()
			}
		}

		c.AbortWithError(http.StatusForbidden, errors.New("not a member"))
	}
}

func getTokenName(payload *idtoken.Payload) string {
	return payload.Claims["name"].(string)
}

func getTokenEmail(payload *idtoken.Payload) string {
	return payload.Claims["email"].(string)
}

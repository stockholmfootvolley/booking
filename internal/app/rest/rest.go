package rest

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/logging"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"github.com/stockholmfootvolley/booking/internal/pkg/payment"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"

	"google.golang.org/api/idtoken"
)

const (
	WebhookPath string = "/stripe/webhook"
)

var (
	ErrGetEvents   = errors.New("could not retrieve events")
	ErrMarshalJSON = errors.New("could not marshal json")
)

type Server struct {
	calendarService    calendar.API
	spreadsheetService spreadsheet.API
	paymentService     payment.API
	port               string
	logger             *logging.Logger
	clientID           string
	webhookKey         string
}

type API interface {
	Serve()
}

func New(
	calendarService calendar.API,
	spreadsheetService spreadsheet.API,
	paymentService payment.API,
	port string,
	clientID string,
	webhookKey string,
	logger *logging.Logger) API {

	return &Server{
		calendarService:    calendarService,
		spreadsheetService: spreadsheetService,
		paymentService:     paymentService,
		port:               port,
		clientID:           clientID,
		logger:             logger,
		webhookKey:         webhookKey,
	}
}

func (s *Server) getEvents(c *gin.Context) {
	events, err := s.calendarService.GetEvents(c)

	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not retrieve events",
			}},
		)
		c.AbortWithError(http.StatusInternalServerError, ErrGetEvents)
		return
	}

	c.IndentedJSON(http.StatusOK, events)
}

func (s *Server) getEvent(c *gin.Context) {
	eventDate := c.Param("date")
	userInfo := s.GetUserFromContext(c)
	event, _, err := s.calendarService.GetSingleEvent(c, eventDate, &userInfo)
	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not retrieve event",
				"user":    userInfo.Email,
			}},
		)
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("could not found event for date "+eventDate))
		return
	}

	newEvent, err := calendar.GoogleEventToEvent(event, s.logger)
	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not convert to google event",
				"user":    userInfo.Email,
			}},
		)
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("getEvent: could not convert event "+eventDate))
		return
	}
	c.IndentedJSON(http.StatusOK, newEvent)
}

func (s *Server) addPresence(c *gin.Context) {
	eventDate := c.Param("date")

	userInfo := s.GetUserFromContext(c)
	newEvent, err := s.calendarService.AddAttendeeEvent(c, eventDate, nil, &userInfo)

	if errors.Is(err, model.ErrRequiresPayment) {
		link, err := s.getPaymentLink(c)
		if err != nil {
			c.AbortWithError(
				http.StatusInternalServerError,
				errors.New("addPresence: could not create payment link "+eventDate))
			return
		}

		c.IndentedJSON(http.StatusTemporaryRedirect, link)
		return
	}

	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("addPresence: could not convert event "+eventDate))
		return
	}
	c.IndentedJSON(http.StatusCreated, newEvent)
}

func (s *Server) removePresence(c *gin.Context) {
	eventDate := c.Param("date")
	userInfo := s.GetUserFromContext(c)
	newEvent, err := s.calendarService.RemoveAttendee(c, eventDate, &userInfo)

	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("removePresence: could not convert event "+eventDate))
		return
	}
	c.IndentedJSON(http.StatusAccepted, newEvent)
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

	// webhook
	router.POST(WebhookPath, s.webhook)
	router.Run("0.0.0.0:" + s.port)
}

func (s *Server) addParsedToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == WebhookPath {
			c.Next()
			return
		}

		token := c.Request.Header.Get("authorization")
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		token = strings.ReplaceAll(token, "Bearer ", "")
		payload, err := idtoken.Validate(c, token, s.clientID)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		users, err := s.spreadsheetService.GetUsers()
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		for _, user := range users {
			userEmail := getTokenEmail(payload)
			if strings.EqualFold(user.Email, userEmail) {
				user.Name = getTokenName(payload)
				user.Email = userEmail
				c.Set(model.User, user)
				c.Next()
				return
			}
		}

		c.AbortWithError(http.StatusUnauthorized, errors.New("not a member"))
	}
}

func getTokenName(payload *idtoken.Payload) string {
	return payload.Claims["name"].(string)
}

func getTokenEmail(payload *idtoken.Payload) string {
	return payload.Claims["email"].(string)
}

func (s *Server) GetUserFromContext(ctx context.Context) spreadsheet.User {
	userInfo, ok := ctx.Value(model.User).(spreadsheet.User)
	if !ok {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not retrieve user from context",
			}},
		)
	}
	return userInfo
}

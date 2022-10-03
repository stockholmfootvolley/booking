package rest

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"github.com/stockholmfootvolley/booking/internal/pkg/payment"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"
)

type PaymentLink struct {
	PaymentLink string `json:"payment_link"`
}

func (s *Server) webhook(c *gin.Context) {
	payload, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "Error reading request body: %v\n",
				"error":   err,
			}},
		)
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	event, err := webhook.ConstructEvent(
		payload,
		c.Request.Header.Get("Stripe-Signature"),
		"webhook")

	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not parse webhook",
				"header":  c.Request.Header.Get("Stripe-Signature"),
				"error":   err,
			}},
		)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	checkoutSession := stripe.CheckoutSession{}
	if checkoutSession.UnmarshalJSON(event.Data.Raw) != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not parse webhook",
				"error":   err,
			}},
		)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	eventID := checkoutSession.Metadata[payment.MetadataEventName]
	userEmail := checkoutSession.Metadata[payment.MetadataUserEmail]
	userName := checkoutSession.Metadata[payment.MetadataUserName]

	user, err := s.spreadsheetService.GetUser(userEmail)
	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not found user on metadata",
				"user":    user.Email,
				"error":   err,
			}},
		)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	user.Name = userName

	// event seems valid: let's update calendar
	_, _ = strconv.Atoi(event.GetObjectValue("amount_total"))
	_, err = s.calendarService.AddAttendeeEvent(c, eventID, &calendar.Payment{
		Email: userEmail,
	}, user)
	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not update event",
				"user":    user.Email,
				"error":   err,
			}},
		)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) getPaymentLink(c *gin.Context) (*PaymentLink, error) {
	eventDate := c.Param("date")
	userInfo := s.GetUserFromContext(c)

	event, _, err := s.calendarService.GetSingleEvent(c, eventDate, &userInfo)
	if err != nil {
		return nil, errors.New("could not found event for date " + eventDate)
	}

	_, err = s.calendarService.GoogleEventToEvent(event)
	if err != nil {
		return nil, errors.New("getPayment: could not convert event " + eventDate)
	}

	/*
		link, err := s.paymentService.CreatePayment(c, int64(newEvent.Price), eventDate, userInfo)
		if err != nil {
			return nil, errors.New("getPayment: could not create payment link " + eventDate)
		}
	*/

	return &PaymentLink{""}, nil
}

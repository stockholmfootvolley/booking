package rest

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/calendar"
	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"github.com/stockholmfootvolley/booking/internal/pkg/payment"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"

	"go.uber.org/zap"
)

type PaymentLink struct {
	PaymentLink string `json:"payment_link"`
}

func (s *Server) webhook(c *gin.Context) {
	payload, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.Error("Error reading request body: %v\n", zap.Error(err))
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}

	event, err := webhook.ConstructEvent(
		payload,
		c.Request.Header.Get("Stripe-Signature"),
		s.webhookKey)

	if err != nil {
		s.logger.Error("could not parse webhook",
			zap.Any("payload", string(payload)),
			zap.String("header", c.Request.Header.Get("Stripe-Signature")))
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	checkoutSession := stripe.CheckoutSession{}
	if checkoutSession.UnmarshalJSON(event.Data.Raw) != nil {
		s.logger.Error("could not parse webhook", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	eventID := checkoutSession.Metadata[payment.MetadataEventName]
	userEmail := checkoutSession.Metadata[payment.MetadataUserEmail]
	userName := checkoutSession.Metadata[payment.MetadataUserName]

	user, err := s.spreadsheetService.GetUser(userEmail)
	if err != nil {
		s.logger.Error("could not found user on metadata", zap.String("user", user.Email))
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	user.Name = userName

	// event seems valid: let's update calendar
	amount, _ := strconv.Atoi(event.GetObjectValue("amount_total"))
	_, err = s.calendarService.AddAttendeeEvent(c, eventID, &calendar.Payment{
		Email:          userEmail,
		Amount:         amount / 100,
		PaymentReceipt: checkoutSession.ID,
	}, user)
	if err != nil {
		s.logger.Error("could not update event", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) getPaymentLink(c *gin.Context) (*PaymentLink, error) {
	eventDate := c.Param("date")
	userInfo := c.Value(model.User).(spreadsheet.User)

	event, _, err := s.calendarService.GetSingleEvent(c, eventDate, &userInfo)
	if err != nil {
		return nil, errors.New("could not found event for date " + eventDate)
	}

	newEvent, err := calendar.GoogleEventToEvent(event)
	if err != nil {
		return nil, errors.New("getPayment: could not convert event " + eventDate)
	}

	link, err := s.paymentService.CreatePayment(c, int64(newEvent.Price), eventDate, userInfo)
	if err != nil {
		return nil, errors.New("getPayment: could not create payment link " + eventDate)
	}

	return &PaymentLink{link}, nil
}

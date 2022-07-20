package rest

import (
	"errors"
	"io/ioutil"
	"net/http"

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

	user, err := s.spreadsheetService.GetUser(userEmail)
	if err != nil {
		s.logger.Error("could not found user on metadata", zap.String("user", user.Email))
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	s.logger.Info("log amounts",
		zap.Any("intent", checkoutSession.PaymentIntent),
		zap.Any("displayItems", checkoutSession.DisplayItems),
		zap.Any("idk", checkoutSession.SetupIntent),
		zap.Any("idk2", checkoutSession),
	)

	// event seems valid: let's update calendar
	_, err = s.calendarService.AddAttendeeEvent(c, eventID, &calendar.Payment{
		Email:          userEmail,
		Amount:         int(checkoutSession.PaymentIntent.AmountReceived) / 100,
		PaymentReceipt: checkoutSession.ID,
	}, user)
	if err != nil {
		s.logger.Error("could not update event", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) getPaymentLink(c *gin.Context) {
	eventDate := c.Param("date")
	userInfo := c.Value(model.User).(spreadsheet.User)

	event, _, err := s.calendarService.GetSingleEvent(c, eventDate, &userInfo)
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
			errors.New("getPayment: could not convert event "+eventDate))
		return
	}

	link, err := s.paymentService.CreatePayment(c, int64(newEvent.Price), eventDate, userInfo)
	if err != nil {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("getPayment: could not create payment link "+eventDate))
		return
	}

	c.IndentedJSON(http.StatusCreated, PaymentLink{
		PaymentLink: link,
	})
}

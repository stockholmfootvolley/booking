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
	"google.golang.org/api/idtoken"
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
	userName := checkoutSession.Metadata[payment.MetadataUserName]
	userEmail := checkoutSession.Metadata[payment.MetadataUserEmail]

	if eventID == "" || userName == "" || userEmail == "" {
		s.logger.Error("metadata seems incorrect", zap.Any("metadata", checkoutSession.Metadata))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// event seems valid: let's update calendar
	_, err = s.calendarService.AddAttendeeEvent(c, eventID, &calendar.Payment{
		Email:          userEmail,
		Amount:         strconv.Itoa(int(checkoutSession.PaymentIntent.AmountReceived) / 100),
		PaymentReceipt: checkoutSession.ID,
	}, &spreadsheet.User{
		Email: userEmail,
		Name:  userName,
	})
	if err != nil {
		s.logger.Error("could not update event", zap.Error(err))
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) getPaymentLink(c *gin.Context) {
	eventDate := c.Param("date")

	event, _, err := s.calendarService.GetSingleEvent(c, eventDate)
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

	userInfo, ok := c.Value(model.Token).(*idtoken.Payload)
	if !ok {
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("getPayment: could not retrieve user logged in "+eventDate))
		return
	}

	link, err := s.paymentService.CreatePayment(c, int64(newEvent.Price), eventDate, *userInfo)
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

package payment

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentlink"
	"github.com/stripe/stripe-go/v72/price"
	"go.uber.org/zap"
)

const (
	MetadataEventName string = "event"
	MetadataUserEmail string = "user_email"
	MetadataUserName  string = "user_name"
)

var (
	ErrNotFound error = errors.New("not found")
)

type Client struct {
	StripeKey string
	ProductID string
	Logger    *zap.Logger
}

type API interface {
	CreatePayment(ctx context.Context, price int64, event string, user spreadsheet.User) (string, error)
	CreatePrice(ctx context.Context, price int64) (*stripe.Price, error)
	GetPrice(ctx context.Context, price int64) (*stripe.Price, error)
}

func New(apiKey string, productID string, logger *zap.Logger) API {
	stripe.Key = apiKey

	return &Client{
		StripeKey: apiKey,
		Logger:    logger,
		ProductID: productID,
	}

}

func (c *Client) CreatePayment(ctx context.Context, price int64, event string, user spreadsheet.User) (string, error) {

	availablePriceObj, err := c.GetPrice(ctx, price)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return "", err
		}
		availablePriceObj, err = c.CreatePrice(ctx, price)
		if err != nil {
			return "", err
		}
	}

	url := url.URL{
		Scheme:   "https",
		Host:     "stockholmfootvolley.github.io",
		Path:     "frontend/",
		Fragment: "/" + event,
	}

	params := &stripe.PaymentLinkParams{
		AfterCompletion: &stripe.PaymentLinkAfterCompletionParams{
			Type: stripe.String(string(stripe.PaymentLinkAfterCompletionTypeRedirect)),
			Redirect: &stripe.PaymentLinkAfterCompletionRedirectParams{
				URL: stripe.String(url.String()),
			},
		},

		LineItems: []*stripe.PaymentLinkLineItemParams{
			{
				Price:    stripe.String(availablePriceObj.ID),
				Quantity: stripe.Int64(1),
			},
		},
	}

	params.AddMetadata(MetadataEventName, event)
	params.AddMetadata(MetadataUserName, user.Name)
	params.AddMetadata(MetadataUserEmail, user.Email)

	pl, err := paymentlink.New(params)
	if err != nil {
		return "", errors.New("could not create link")
	}

	return pl.URL, nil
}

func (c *Client) CreatePrice(ctx context.Context, objPrice int64) (*stripe.Price, error) {

	// stripe uses cents as price
	objPrice *= 100

	params := &stripe.PriceParams{
		Currency:   stripe.String(string(stripe.CurrencySEK)),
		Product:    stripe.String(c.ProductID),
		UnitAmount: stripe.Int64(objPrice),
	}
	return price.New(params)
}

func (c *Client) GetPrice(ctx context.Context, objPrice int64) (*stripe.Price, error) {
	params := &stripe.PriceListParams{}
	params.Filters.AddFilter("price", "", strconv.Itoa(int(objPrice)))
	i := price.List(params)
	for i.Next() {
		return i.Price(), nil
	}
	return nil, ErrNotFound
}

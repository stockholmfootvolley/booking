package swish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"

	"cloud.google.com/go/logging"
)

const (
	URL string = "https://mpc.getswish.net/qrg-swish/api/v1/prefilled"
)

type Client struct {
	Phone  string
	Logger *logging.Logger
}

type API interface {
	GenerateQrCode(amount int, eventLevel string, eventDate string) (string, error)
}

func New(phone string, logger *logging.Logger) (*Client, error) {
	return &Client{
		Phone:  phone,
		Logger: logger,
	}, nil

}

func (c *Client) GenerateQrCode(amount int, eventLevel string, eventDate string) (string, error) {

	bodyJson := map[string]interface{}{
		"format": "png",
		"size":   300,
		"payee": map[string]interface{}{
			"value":    "${PHONE}",
			"editable": false,
		},
		"amount": map[string]interface{}{
			"value":    amount,
			"editable": false,
		},
		"message": map[string]interface{}{
			"value":    fmt.Sprint(eventLevel, eventDate),
			"editable": true,
		},
	}

	result, err := json.Marshal(bodyJson)
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not marshal json",
				"error":   err,
			}},
		)
	}

	req, err := http.DefaultClient.Post(URL, mime.TypeByExtension(".json"), bytes.NewReader(result))
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could request qr code",
				"error":   err,
			}},
		)
	}
	defer req.Body.Close()

	response, err := io.ReadAll(req.Body)
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could read qr code response",
				"error":   err,
			}},
		)
	}

	return string(response), err
}

package spreadsheet

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/logging"
	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	Service       *sheets.Service
	SpreadsheetID string
	Logger        *logging.Logger
}

type User struct {
	Name  string      `json:"name"`
	Email string      `json:"email"`
	Level model.Level `json:"level"`
}

type API interface {
	GetUsers() ([]User, error)
	GetUser(email string) (*User, error)
}

const (
	ReadRange = "Sheet1!A:C"
)

func New(serviceAccount string, spreadsheetId string, logger *logging.Logger) (*Client, error) {
	service, err := getClient(serviceAccount, logger)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "unable to retrieve sheets client",
				"error":   err,
			}},
		)
		return nil, err
	}

	return &Client{
		SpreadsheetID: spreadsheetId,
		Service:       service,
		Logger:        logger,
	}, nil

}
func getClient(serviceAccount string, logger *logging.Logger) (*sheets.Service, error) {
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(serviceAccount), sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "unable read credentials",
				"error":   err,
			}},
		)
		return nil, err
	}

	return sheets.NewService(ctx, option.WithCredentials(credentials))
}

func (c *Client) GetUsers() ([]User, error) {
	resp, err := c.Service.Spreadsheets.Values.Get(c.SpreadsheetID, ReadRange).Do()
	if err != nil {
		c.Logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "unable to retrieve data from sheet",
				"error":   err,
			}},
		)
	}

	users := []User{}
	for _, row := range resp.Values[1:] {

		if len(row) < 2 {
			continue
		}

		if len(row) == 2 {
			row = append(row, "")
		}

		level := row[2].(string)
		modelLevel := model.StringToLevel(level)

		users = append(users, User{
			Name:  row[0].(string),
			Email: row[1].(string),
			Level: modelLevel,
		})
	}

	return users, nil
}

func (c *Client) GetUser(email string) (*User, error) {
	users, err := c.GetUsers()
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if strings.EqualFold(user.Email, email) {
			return &user, nil
		}
	}

	return nil, errors.New("not found")
}

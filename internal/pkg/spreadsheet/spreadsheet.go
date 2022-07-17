package spreadsheet

import (
	"context"

	"github.com/stockholmfootvolley/booking/internal/pkg/model"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	Service       *sheets.Service
	SpreadsheetID string
	Logger        *zap.Logger
}

type User struct {
	Name  string
	Email string
	Level model.Level
}

type API interface {
	GetUsers() ([]User, error)
}

const (
	ReadRange = "Sheet1!A:C"
)

func New(serviceAccount string, spreadsheetId string, logger *zap.Logger) (*Client, error) {
	service, err := getClient(serviceAccount, logger)
	if err != nil {
		logger.Error("unable to retrieve sheets client", zap.Error(err))
		return nil, err
	}

	return &Client{
		SpreadsheetID: spreadsheetId,
		Service:       service,
		Logger:        logger,
	}, nil

}
func getClient(serviceAccount string, logger *zap.Logger) (*sheets.Service, error) {
	ctx := context.Background()
	credentials, err := google.CredentialsFromJSON(ctx, []byte(serviceAccount), sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		logger.Error("unable read credentials", zap.Error(err))
		return nil, err
	}

	return sheets.NewService(ctx, option.WithCredentials(credentials))
}

func (c *Client) GetUsers() ([]User, error) {
	resp, err := c.Service.Spreadsheets.Values.Get(c.SpreadsheetID, ReadRange).Do()
	if err != nil {
		c.Logger.Error("unable to retrieve data from sheet", zap.Error(err))
	}

	users := []User{}
	for _, row := range resp.Values[1:] {
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

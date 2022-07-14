package spreadsheet

import (
	"context"

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

type API interface {
	GetUsers() ([]string, error)
}

const (
	ReadRange = "Sheet1!A:B"
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

func (c *Client) GetUsers() ([]string, error) {
	resp, err := c.Service.Spreadsheets.Values.Get(c.SpreadsheetID, ReadRange).Do()
	if err != nil {
		c.Logger.Error("unable to retrieve data from sheet", zap.Error(err))
	}

	emails := []string{}
	for _, row := range resp.Values[1:] {
		emails = append(emails, row[1].(string))
	}

	return emails, nil
}

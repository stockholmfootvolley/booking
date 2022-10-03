package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/api/idtoken"
)

type UserToken struct {
	Name    string
	Email   string
	Picture string
}

type FacebookResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture struct {
		Data struct {
			Height       int    `json:"height"`
			IsSilhouette bool   `json:"is_silhouette"`
			URL          string `json:"url"`
			Width        int    `json:"width"`
		} `json:"data"`
	} `json:"picture"`
}

func (s *Server) ValidateToken(ctx context.Context, token string) (*UserToken, error) {

	splitByPoint := strings.Split(token, ".")

	// seems to be a jwt, let's go to google
	if len(splitByPoint) == 3 {
		payload, err := idtoken.Validate(ctx, token, s.clientID)
		if err != nil {
			return nil, err
		}
		return &UserToken{
			Email:   getTokenEmail(payload),
			Name:    getTokenName(payload),
			Picture: getTokenPicture(payload),
		}, nil

	} else {
		return DebugFacebookToken(token)
	}
}

func DebugFacebookToken(token string) (*UserToken, error) {
	graphURL := url.URL{
		Scheme: "https",
		Host:   "graph.facebook.com",
		Path:   "v14.0/me",
	}

	q := graphURL.Query()
	q.Add("fields", "id,name,email,picture")
	q.Add("access_token", token)
	graphURL.RawQuery = q.Encode()

	resp, err := http.Get(graphURL.String())
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}

	defer resp.Body.Close()
	fbResponse := FacebookResponse{}
	err = json.NewDecoder(resp.Body).Decode(&fbResponse)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}

	return &UserToken{
		Name:    fbResponse.Name,
		Email:   fbResponse.Email,
		Picture: fbResponse.Picture.Data.URL,
	}, nil

}

func getTokenName(payload *idtoken.Payload) string {
	return payload.Claims["name"].(string)
}

func getTokenEmail(payload *idtoken.Payload) string {
	return payload.Claims["email"].(string)
}

func getTokenPicture(payload *idtoken.Payload) string {
	return payload.Claims["picture"].(string)
}

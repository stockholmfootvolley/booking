package rest

import (
	"errors"
	"net/http"

	"cloud.google.com/go/logging"
	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
)

type UserInfo struct {
	User    spreadsheet.User
	Picture string `json:"picture"`
}

func (s *Server) getUser(c *gin.Context) {
	userInfo := s.GetUserFromContext(c)

	userInfoFromToken, err := s.ValidateToken(c.Request.Context(), c.Request.Header.Get("authorization"))

	if err != nil {
		s.logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload: map[string]interface{}{
				"message": "could not retrieve token",
				"user":    userInfo.Email,
			}},
		)
		c.AbortWithError(
			http.StatusInternalServerError,
			errors.New("could not retrieve token"))
		return
	}

	c.IndentedJSON(http.StatusOK, UserInfo{
		User:    userInfo,
		Picture: userInfoFromToken.Picture,
	})
}

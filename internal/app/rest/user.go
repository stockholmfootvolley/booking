package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
)

type UserInfo struct {
	User    spreadsheet.User
	Picture string `json:"picture"`
}

func (s *Server) getUser(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, s.GetUserFromContext(c))
}

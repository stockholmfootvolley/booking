package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stockholmfootvolley/booking/internal/pkg/spreadsheet"
)

type UserInfo struct {
	User    spreadsheet.User
	Picture string `json:"picture"`
}

func (s *Server) getUser(c *gin.Context) {
	userInfo := s.GetUserFromContext(c)

	jsonContent, err := json.Marshal(&userInfo)
	if err != nil {
		c.AbortWithStatus(http.StatusUnprocessableEntity)
	}

	maxAge := time.Now().Add(time.Hour)
	c.SetCookie("user", string(jsonContent), int(maxAge.Unix()), "/", c.Request.Host, true, false)
}

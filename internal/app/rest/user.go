package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getUser(c *gin.Context) {
	userInfo := s.GetUserFromContext(c)
	c.IndentedJSON(http.StatusOK, userInfo)
}

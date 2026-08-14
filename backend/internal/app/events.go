package app

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) events(c *gin.Context) {
	userID, ok := s.auth.ParseToken(c.Query("token"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	channel := s.hub.Subscribe(userID)
	defer s.hub.Unsubscribe(userID, channel)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Stream(func(writer io.Writer) bool {
		select {
		case message := <-channel:
			_, _ = fmt.Fprintf(writer, "data: %s\n\n", message)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

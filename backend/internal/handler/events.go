package handler

import (
	"fmt"
	"io"
	"time"

	"stellar-wallet-backend/internal/auth"
	"stellar-wallet-backend/internal/realtime"

	"github.com/gin-gonic/gin"
)

type EventsHandler struct {
	sessions *auth.Manager
	hub      *realtime.Hub
}

func (h *EventsHandler) stream(c *gin.Context) {
	// The route is protected by the same cookie-session middleware as other APIs.
	channel := h.hub.Subscribe(c.GetString("userID"))
	defer h.hub.Unsubscribe(c.GetString("userID"), channel)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	fmt.Fprint(c.Writer, ": connected\n\n")
	c.Writer.Flush()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	c.Stream(func(writer io.Writer) bool {
		select {
		case message := <-channel:
			_, _ = fmt.Fprintf(writer, "data: %s\n\n", message)
			return true
		case <-heartbeat.C:
			// Revoked sessions are dropped here, at most one heartbeat after logout.
			if _, err := h.sessions.Session(c.Request); err != nil {
				return false
			}
			_, _ = fmt.Fprint(writer, ": heartbeat\n\n")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

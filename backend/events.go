package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// hub fans out server-sent events to connected clients, keyed by user id.
type hub struct {
	mu   sync.Mutex
	subs map[string]map[chan string]struct{}
}

func newHub() *hub {
	return &hub{subs: map[string]map[chan string]struct{}{}}
}

func (h *hub) subscribe(user string) chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan string, 8)
	if h.subs[user] == nil {
		h.subs[user] = map[chan string]struct{}{}
	}
	h.subs[user][ch] = struct{}{}
	return ch
}

func (h *hub) unsubscribe(user string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m := h.subs[user]; m != nil {
		delete(m, ch)
		if len(m) == 0 {
			delete(h.subs, user)
		}
	}
	close(ch)
}

func (h *hub) publish(user, kind string, payload any) {
	env, err := json.Marshal(map[string]any{"type": kind, "data": payload})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[user] {
		select {
		case ch <- string(env):
		default: // drop if the client is slow rather than block
		}
	}
}

// publishTxn sends a transaction update to every open connection for the user.
func (h *hub) publishTxn(user string, tx Transaction) {
	h.publish(user, "tx", tx)
}

// publishWallet sends a wallet update (e.g. keygen completed) to the user.
func (h *hub) publishWallet(user string, w Wallet) {
	h.publish(user, "wallet", w)
}

// events streams SSE. EventSource can't set headers, so the JWT comes in ?token=.
func (s *server) events(c *gin.Context) {
	userID, ok := parseToken(c.Query("token"))
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	ch := s.hub.subscribe(userID)
	defer s.hub.unsubscribe(userID, ch)

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	c.Stream(func(w io.Writer) bool {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

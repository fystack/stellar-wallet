package realtime

import (
	"encoding/json"
	"sync"

	"stellar-wallet-backend/internal/domain"
)

type Hub struct {
	mu          sync.Mutex
	subscribers map[string]map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: map[string]map[chan string]struct{}{}}
}

func (h *Hub) Subscribe(userID string) chan string {
	h.mu.Lock()
	defer h.mu.Unlock()

	channel := make(chan string, 8)
	if h.subscribers[userID] == nil {
		h.subscribers[userID] = map[chan string]struct{}{}
	}
	h.subscribers[userID][channel] = struct{}{}
	return channel
}

func (h *Hub) Unsubscribe(userID string, channel chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subscribers := h.subscribers[userID]; subscribers != nil {
		delete(subscribers, channel)
		if len(subscribers) == 0 {
			delete(h.subscribers, userID)
		}
	}
	close(channel)
}

// ConnectedUserIDs lists users with at least one open event stream.
func (h *Hub) ConnectedUserIDs() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	userIDs := make([]string, 0, len(h.subscribers))
	for userID := range h.subscribers {
		userIDs = append(userIDs, userID)
	}
	return userIDs
}

func (h *Hub) Publish(userID, kind string, payload any) {
	envelope, err := json.Marshal(map[string]any{"type": kind, "data": payload})
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for channel := range h.subscribers[userID] {
		select {
		case channel <- string(envelope):
		default:
		}
	}
}

func (h *Hub) PublishTransaction(userID string, transaction domain.Transaction) {
	h.Publish(userID, "tx", transaction)
}

func (h *Hub) PublishWallet(userID string, wallet domain.Wallet) {
	h.Publish(userID, "wallet", wallet)
}

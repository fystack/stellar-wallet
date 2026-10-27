package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHubPublishesOnlyToTheTargetUser(t *testing.T) {
	hub := NewHub()
	target := hub.Subscribe("user-1")
	other := hub.Subscribe("user-2")
	defer hub.Unsubscribe("user-1", target)
	defer hub.Unsubscribe("user-2", other)

	hub.Publish("user-1", "wallet", map[string]string{"id": "wallet-1"})

	select {
	case message := <-target:
		var envelope struct {
			Type string            `json:"type"`
			Data map[string]string `json:"data"`
		}
		if err := json.Unmarshal([]byte(message), &envelope); err != nil {
			t.Fatalf("invalid event JSON: %v", err)
		}
		if envelope.Type != "wallet" || envelope.Data["id"] != "wallet-1" {
			t.Fatalf("unexpected envelope: %#v", envelope)
		}
	case <-time.After(time.Second):
		t.Fatal("target subscriber did not receive event")
	}

	select {
	case message := <-other:
		t.Fatalf("other user received %q", message)
	default:
	}
}

func TestHubListsConnectedUsers(t *testing.T) {
	hub := NewHub()
	if users := hub.ConnectedUserIDs(); len(users) != 0 {
		t.Fatalf("connected = %v", users)
	}
	first := hub.Subscribe("alice")
	second := hub.Subscribe("alice")
	if users := hub.ConnectedUserIDs(); len(users) != 1 || users[0] != "alice" {
		t.Fatalf("connected = %v", users)
	}
	hub.Unsubscribe("alice", first)
	hub.Unsubscribe("alice", second)
	if users := hub.ConnectedUserIDs(); len(users) != 0 {
		t.Fatalf("connected after unsubscribe = %v", users)
	}
}

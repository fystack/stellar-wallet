package mail

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogMailerWritesInviteLinkToLog(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	defer zap.ReplaceGlobals(zap.New(core))()

	if err := (LogMailer{}).Send("teammate@example.com", Message{Subject: "Invite", ActionURL: "http://localhost:5173/#/invite/token"}); err != nil {
		t.Fatal(err)
	}
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["to"] != "teammate@example.com" || !strings.Contains(fields["link"].(string), "/#/invite/token") {
		t.Fatalf("logged fields = %#v", fields)
	}
}

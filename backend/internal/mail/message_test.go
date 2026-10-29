package mail

import (
	"strings"
	"testing"
)

func TestMessageRendersTextAndEscapedHTML(t *testing.T) {
	m := Message{
		Subject:     "Approval needed",
		Heading:     "Approve 10 XLM?",
		Intro:       "owner@example.com wants to send a payment.",
		Details:     []Detail{{Label: "To", Value: "<script>alert(1)</script>"}},
		ActionLabel: "Review request",
		ActionURL:   "http://localhost:5173/#/approvals/req-1",
		Note:        "Expires in 7 days.",
	}
	text := m.Text()
	for _, want := range []string{"Approve 10 XLM?", "To: <script>alert(1)</script>", "Review request:\nhttp://localhost:5173/#/approvals/req-1", "Expires in 7 days."} {
		if !strings.Contains(text, want) {
			t.Fatalf("text missing %q:\n%s", want, text)
		}
	}
	html := m.HTML()
	if strings.Contains(html, "<script>") || !strings.Contains(html, "&lt;script&gt;") {
		t.Fatalf("html did not escape detail value:\n%s", html)
	}
	if !strings.Contains(html, `href="http://localhost:5173/#/approvals/req-1"`) {
		t.Fatalf("html missing action link:\n%s", html)
	}
}

func TestMultipartBodyHasTextThenHTML(t *testing.T) {
	body, boundary, err := multipartBody(Message{Subject: "Hi", ActionURL: "http://x"})
	if err != nil {
		t.Fatal(err)
	}
	text, html := strings.Index(body, "text/plain"), strings.Index(body, "text/html")
	if boundary == "" || text < 0 || html < text || !strings.HasSuffix(strings.TrimSpace(body), "--"+boundary+"--") {
		t.Fatalf("body = %q", body)
	}
}

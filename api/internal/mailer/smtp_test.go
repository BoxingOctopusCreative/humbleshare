package mailer

import (
	"context"
	"strings"
	"testing"
)

type captureSMTP struct {
	last Message
}

func (c *captureSMTP) Send(_ context.Context, msg Message) error {
	c.last = msg
	return nil
}

func TestNewSMTPNilWithoutHost(t *testing.T) {
	if NewSMTP("", "1025", "") != nil {
		t.Fatal("expected nil")
	}
}

func TestNewSMTPDefaults(t *testing.T) {
	s := NewSMTP("mailpit", "", "")
	if s == nil {
		t.Fatal("expected sender")
	}
	if s.addr != "mailpit:1025" {
		t.Fatalf("addr=%q", s.addr)
	}
	if !strings.Contains(s.from, "HumbleShare") {
		t.Fatalf("from=%q", s.from)
	}
}

func TestFormatMessagePlain(t *testing.T) {
	raw, err := formatMessage("HumbleShare <noreply@localhost>", "you@example.com", Message{
		To:      "you@example.com",
		Subject: "Hello",
		Text:    "plain body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "Content-Type: text/plain; charset=UTF-8") {
		t.Fatalf("missing text/plain: %s", raw)
	}
	if strings.Contains(raw, "multipart/alternative") {
		t.Fatal("plain message should not be multipart")
	}
	if !strings.Contains(raw, "plain body") {
		t.Fatal("missing body")
	}
}

func TestFormatMessageHTML(t *testing.T) {
	raw, err := formatMessage("HumbleShare <noreply@localhost>", "you@example.com", Message{
		To:      "you@example.com",
		Subject: "Your HumbleShare invite",
		Text:    "plain invite",
		HTML:    "<p>html invite</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "multipart/alternative") {
		t.Fatalf("expected multipart: %s", raw)
	}
	if !strings.Contains(raw, "text/html; charset=UTF-8") {
		t.Fatal("missing html part")
	}
	if !strings.Contains(raw, "plain invite") {
		t.Fatal("missing text part")
	}
	if !strings.Contains(raw, "html invite") {
		t.Fatal("missing html body")
	}
}

var _ Sender = (*captureSMTP)(nil)

package mailer

import (
	"context"
	"strings"
	"testing"
)

type captureSMTP struct {
	lastTo      string
	lastSubject string
	lastBody    string
}

func (c *captureSMTP) Send(_ context.Context, to, subject, body string) error {
	c.lastTo = to
	c.lastSubject = subject
	c.lastBody = body
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

var _ Sender = (*captureSMTP)(nil)

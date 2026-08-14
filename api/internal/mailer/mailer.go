package mailer

import "context"

// Message is one transactional email. HTML is optional; when empty, senders
// deliver Text as text/plain only.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Sender delivers transactional email. Nil-safe: callers should treat a nil
// Sender as "not configured" and fall back to logging.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

package mailer

import "context"

// Sender delivers plain-text transactional email. Nil-safe: callers should treat
// a nil Sender as "not configured" and fall back to logging.
type Sender interface {
	Send(ctx context.Context, toEmail, subject, textBody string) error
}

package mailer

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTP sends mail via a plain SMTP server (e.g. Mailpit in local compose).
type SMTP struct {
	addr string
	from string
}

// NewSMTP returns a sender, or nil if host is empty.
func NewSMTP(host, port, from string) *SMTP {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil
	}
	port = strings.TrimSpace(port)
	if port == "" {
		port = "1025"
	}
	fromAddr := strings.TrimSpace(from)
	if fromAddr == "" {
		fromAddr = "HumbleShare <noreply@localhost>"
	}
	return &SMTP{
		addr: net.JoinHostPort(host, port),
		from: fromAddr,
	}
}

func (s *SMTP) Send(ctx context.Context, toEmail, subject, textBody string) error {
	if s == nil {
		return fmt.Errorf("smtp: not configured")
	}
	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return fmt.Errorf("smtp: empty recipient")
	}

	msg := strings.Builder{}
	msg.WriteString("From: " + s.from + "\r\n")
	msg.WriteString("To: " + toEmail + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(textBody)
	if !strings.HasSuffix(textBody, "\n") {
		msg.WriteString("\r\n")
	}

	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", s.addr, err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, strings.Split(s.addr, ":")[0])
	if err != nil {
		return err
	}
	defer c.Close()

	done := make(chan error, 1)
	go func() {
		done <- s.send(c, toEmail, msg.String())
	}()
	select {
	case <-ctx.Done():
		_ = c.Close()
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (s *SMTP) send(c *smtp.Client, toEmail, raw string) error {
	from := s.from
	if i := strings.LastIndex(from, "<"); i >= 0 && strings.HasSuffix(from, ">") {
		from = strings.TrimSuffix(from[i+1:], ">")
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(toEmail); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(raw)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

package mailer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
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

func (s *SMTP) Send(ctx context.Context, msg Message) error {
	if s == nil {
		return fmt.Errorf("smtp: not configured")
	}
	toEmail := strings.TrimSpace(msg.To)
	if toEmail == "" {
		return fmt.Errorf("smtp: empty recipient")
	}

	raw, err := formatMessage(s.from, toEmail, msg)
	if err != nil {
		return err
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
		done <- s.send(c, toEmail, raw)
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
	if _, err := io.WriteString(w, raw); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func formatMessage(from, to string, msg Message) (string, error) {
	var buf strings.Builder
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + msg.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	htmlBody := strings.TrimSpace(msg.HTML)
	if htmlBody == "" {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(msg.Text)
		if !strings.HasSuffix(msg.Text, "\n") {
			buf.WriteString("\r\n")
		}
		return buf.String(), nil
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := writeQuotedPart(mw, "text/plain; charset=UTF-8", msg.Text); err != nil {
		return "", err
	}
	if err := writeQuotedPart(mw, "text/html; charset=UTF-8", htmlBody); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + mw.Boundary() + "\"\r\n")
	buf.WriteString("\r\n")
	buf.Write(body.Bytes())
	return buf.String(), nil
}

func writeQuotedPart(mw *multipart.Writer, contentType, content string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	pw, err := mw.CreatePart(h)
	if err != nil {
		return err
	}
	qp := quotedprintable.NewWriter(pw)
	if _, err := qp.Write([]byte(content)); err != nil {
		return err
	}
	return qp.Close()
}

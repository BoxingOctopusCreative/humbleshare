package mailgun

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/boxingoctopus/humbleshare/api/internal/mailer"
)

func TestSendIncludesHTML(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got, _ = url.ParseQuery(string(body))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"<x>","message":"Queued"}`))
	}))
	t.Cleanup(srv.Close)

	c := New("key", "mg.example.com", "HumbleShare <noreply@example.com>", srv.URL)
	if c == nil {
		t.Fatal("expected client")
	}
	err := c.Send(context.Background(), mailer.Message{
		To:      "you@example.com",
		Subject: "Your HumbleShare invite",
		Text:    "plain",
		HTML:    "<p>html</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("text") != "plain" {
		t.Fatalf("text=%q", got.Get("text"))
	}
	if got.Get("html") != "<p>html</p>" {
		t.Fatalf("html=%q", got.Get("html"))
	}
	if got.Get("to") != "you@example.com" {
		t.Fatalf("to=%q", got.Get("to"))
	}
}

func TestSendOmitsEmptyHTML(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got, _ = url.ParseQuery(string(body))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := New("key", "mg.example.com", "from@example.com", srv.URL)
	err := c.Send(context.Background(), mailer.Message{
		To:      "you@example.com",
		Subject: "HumbleShare access request",
		Text:    "plain only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["html"]; ok {
		t.Fatalf("html should be omitted, got %q", got.Get("html"))
	}
}

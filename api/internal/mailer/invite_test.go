package mailer

import (
	"strings"
	"testing"
)

func TestInviteEmail(t *testing.T) {
	link := "https://humbleshare.com/invite/abc123"
	text, html := InviteEmail(link, "https://humbleshare.com")
	if !strings.Contains(text, link) {
		t.Fatalf("text missing link: %s", text)
	}
	if !strings.Contains(text, "You've been invited to HumbleShare") {
		t.Fatal("text missing intro")
	}
	if !strings.Contains(html, `href="https://humbleshare.com/invite/abc123"`) {
		t.Fatalf("html missing invite href: %s", html)
	}
	if !strings.Contains(html, inviteLogoURL) {
		t.Fatal("html missing logo")
	}
	if !strings.Contains(html, "Create your account") {
		t.Fatal("html missing CTA")
	}
	if !strings.Contains(html, "You're invited") {
		t.Fatal("html missing heading")
	}
	if strings.Contains(html, "—") {
		t.Fatal("html contains em dash")
	}
}

func TestInviteEmailEscapesHTML(t *testing.T) {
	link := `https://humbleshare.com/invite/a" onclick=alert(1)`
	_, html := InviteEmail(link, "https://humbleshare.com")
	if strings.Contains(html, `" onclick=alert`) {
		t.Fatal("unescaped quote broke out of an attribute")
	}
}

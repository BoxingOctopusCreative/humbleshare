package mailer

import (
	"bytes"
	"embed"
	htmltemplate "html/template"
	"strings"
)

//go:embed templates/*.html
var templateFS embed.FS

var inviteHTML = htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/invite.html"))

const inviteLogoURL = "https://assets.humbleshare.com/PNG/humbleshare_wide.png"

type inviteContent struct {
	Link      string
	LogoURL   string
	SiteURL   string
	Preheader string
}

// InviteEmail returns the plaintext and HTML bodies for an account invite.
func InviteEmail(inviteURL, siteURL string) (textBody, htmlBody string) {
	inviteURL = strings.TrimSpace(inviteURL)
	siteURL = strings.TrimRight(strings.TrimSpace(siteURL), "/")
	textBody = inviteText(inviteURL)

	var buf bytes.Buffer
	err := inviteHTML.Execute(&buf, inviteContent{
		Link:      inviteURL,
		LogoURL:   inviteLogoURL,
		SiteURL:   siteURL,
		Preheader: "Create your account. This invite expires in 7 days.",
	})
	if err != nil {
		return textBody, ""
	}
	return textBody, buf.String()
}

func inviteText(link string) string {
	var b strings.Builder
	b.WriteString("You've been invited to HumbleShare.\n\n")
	b.WriteString("Got a spare Humble Gift Code? Share it with a friend, or a complete stranger!\n\n")
	b.WriteString("Open this link to create your account. It expires in 7 days:\n")
	b.WriteString(link)
	b.WriteString("\n")
	return b.String()
}

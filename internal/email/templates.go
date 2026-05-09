package email

import (
	"bytes"
	"embed"
	"encoding/json"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"
)

//go:embed templates/*.txt templates/*.html
var emailTemplateFS embed.FS

type templateSpec struct {
	Subject string
	Text    string
	HTML    string
}

var emailTemplates = map[string]templateSpec{
	"admin-invite":        {Subject: "You're invited to Cypra", Text: "templates/admin-invite.txt", HTML: "templates/admin-invite.html"},
	"magic-link":          {Subject: "Your Cypra sign-in link", Text: "templates/magic-link.txt", HTML: "templates/magic-link.html"},
	"password-reset":      {Subject: "Reset your password", Text: "templates/password-reset.txt", HTML: "templates/password-reset.html"},
	"email-verification":  {Subject: "Verify your email", Text: "templates/email-verification.txt", HTML: "templates/email-verification.html"},
	"breach-notification": {Subject: "Security notice", Text: "templates/breach-notification.txt", HTML: "templates/breach-notification.html"},
}

// Render converts an outbox template and payload into an email message.
func Render(template, to string, payload json.RawMessage) (Message, error) {
	values := map[string]any{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &values); err != nil {
			return Message{}, err
		}
	}
	spec, ok := emailTemplates[template]
	if !ok {
		return Message{To: to, Subject: template, Text: string(payload)}, nil
	}
	data := emailTemplateData(values)
	text, err := renderTextTemplate(spec.Text, data)
	if err != nil {
		return Message{}, err
	}
	html, err := renderHTMLTemplate(spec.HTML, data)
	if err != nil {
		return Message{}, err
	}
	return Message{To: to, Subject: spec.Subject, Text: text, HTML: html}, nil
}

type emailTemplateView struct {
	InviteURL         string
	MagicLinkURL      string
	MagicLinkToken    string
	ResetURL          string
	VerifyURL         string
	Message           string
	TenantAccent      string
	TenantLogoURL     string
	TenantDisplayName string
}

func emailTemplateData(values map[string]any) emailTemplateView {
	return emailTemplateView{
		InviteURL:         stringValue(values["invite_url"], ""),
		MagicLinkURL:      stringValue(values["magic_link_url"], ""),
		MagicLinkToken:    stringValue(values["token"], ""),
		ResetURL:          stringValue(values["reset_url"], ""),
		VerifyURL:         stringValue(values["verify_url"], ""),
		Message:           stringValue(values["message"], ""),
		TenantAccent:      stringValue(values["tenant_accent"], "#0D9488"),
		TenantLogoURL:     stringValue(values["tenant_logo_url"], ""),
		TenantDisplayName: stringValue(values["tenant_display_name"], "Cypra"),
	}
}

func renderTextTemplate(path string, data emailTemplateView) (string, error) {
	tpl, err := texttemplate.ParseFS(emailTemplateFS, path)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func renderHTMLTemplate(path string, data emailTemplateView) (string, error) {
	tpl, err := htmltemplate.ParseFS(emailTemplateFS, path)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", err
	}
	return out.String(), nil
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return text
}

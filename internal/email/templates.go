package email

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
)

// Render converts an outbox template and payload into an email message.
func Render(template, to string, payload json.RawMessage) (Message, error) {
	values := map[string]any{}
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &values); err != nil {
			return Message{}, err
		}
	}
	switch template {
	case "admin-invite":
		return brandedMessage(to, "You're invited to Cypra", fmt.Sprintf("You've been invited to Cypra. Open this link to continue: %v", values["invite_url"]), values), nil
	case "magic-link":
		return brandedMessage(to, "Your Cypra sign-in link", fmt.Sprintf("Open this link to sign in: %v", values["magic_link_url"]), values), nil
	case "password-reset":
		return brandedMessage(to, "Reset your password", fmt.Sprintf("Open this link to reset your password: %v", values["reset_url"]), values), nil
	case "email-verification":
		return brandedMessage(to, "Verify your email", fmt.Sprintf("Open this link to verify your email: %v", values["verify_url"]), values), nil
	case "breach-notification":
		return brandedMessage(to, "Security notice", fmt.Sprintf("Security notice: %v", values["message"]), values), nil
	default:
		return Message{To: to, Subject: template, Text: string(payload)}, nil
	}
}

func brandedMessage(to, subject, text string, values map[string]any) Message {
	accent := stringValue(values["tenant_accent"], "#0D9488")
	logo := stringValue(values["tenant_logo_url"], "")
	displayName := stringValue(values["tenant_display_name"], "Cypra")
	var logoHTML string
	if logo != "" {
		logoHTML = fmt.Sprintf(`<img src="%s" alt="%s" style="max-height:40px;max-width:180px" />`, template.HTMLEscapeString(logo), template.HTMLEscapeString(displayName))
	}
	html := fmt.Sprintf(`<div style="font-family:ui-monospace,Menlo,Consolas,monospace;color:#09090B">%s<h1 style="font-size:20px">%s</h1><p>%s</p><p style="color:%s">%s</p></div>`, logoHTML, template.HTMLEscapeString(subject), template.HTMLEscapeString(text), template.HTMLEscapeString(accent), template.HTMLEscapeString(displayName))
	return Message{To: to, Subject: subject, Text: text, HTML: html}
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return text
}

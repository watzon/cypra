package email

import (
	"encoding/json"
	"fmt"
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
		return Message{To: to, Subject: "You're invited to Cypra", Text: fmt.Sprintf("You've been invited to Cypra. Open this link to continue: %v", values["invite_url"])}, nil
	case "magic-link":
		return Message{To: to, Subject: "Your Cypra sign-in link", Text: fmt.Sprintf("Open this link to sign in: %v", values["magic_link_url"])}, nil
	default:
		return Message{To: to, Subject: template, Text: string(payload)}, nil
	}
}

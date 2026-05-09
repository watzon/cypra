package hostedlogin_test

import (
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/hostedlogin"
)

func TestPasskeyJSUsesServerCeremonyOptions(t *testing.T) {
	content, err := hostedlogin.PasskeyJS()
	if err != nil {
		t.Fatalf("load passkey js: %v", err)
	}
	script := string(content)
	for _, forbidden := range []string{"crypto.getRandomValues", "new Uint8Array(32)"} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("passkey js still appears to generate a client challenge: found %q", forbidden)
		}
	}
	for _, required := range []string{"/api/v1/auth/passkey/register", "/api/v1/auth/passkey/assert", "/api/v1/auth/webauthn2fa/verify", "navigator.credentials.create", "navigator.credentials.get", "ceremony_id"} {
		if !strings.Contains(script, required) {
			t.Fatalf("passkey js missing %q", required)
		}
	}
}

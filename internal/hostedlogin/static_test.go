package hostedlogin_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/hostedlogin"
)

// TestHTMXJSVendored asserts the htmx.min.js asset is embedded, looks like
// htmx, and stays at the version recorded in docs/playbook/dependencies.md.
//
// Update procedure (see docs/playbook/dependencies.md):
//  1. Replace internal/hostedlogin/static/htmx.min.js
//  2. Update the SRI hash in internal/hostedlogin/templates/base.html
//  3. Update the wantSHA constant below.
func TestHTMXJSVendored(t *testing.T) {
	content, err := hostedlogin.HTMXJS()
	if err != nil {
		t.Fatalf("load htmx js: %v", err)
	}
	if len(content) < 1024 {
		t.Fatalf("htmx.min.js looks truncated: %d bytes", len(content))
	}
	asString := string(content)
	if !strings.Contains(asString, "htmx=function()") && !strings.Contains(asString, "var htmx=") {
		t.Fatalf("htmx.min.js content does not look like htmx")
	}
	const wantSHA = "e209dda5c8235479f3166defc7750e1dbcd5a5c1808b7792fc2e6733768fb447"
	sum := sha256.Sum256(content)
	if got := hex.EncodeToString(sum[:]); got != wantSHA {
		t.Fatalf("htmx.min.js sha256 = %s, want %s — if you intentionally bumped htmx, update wantSHA and the SRI hash in base.html", got, wantSHA)
	}
}

// TestPasskeyJSClassifiesErrors asserts the passkey.js helper surfaces a
// specific user-facing message for each error class instead of one generic
// "didn't work" string.
func TestPasskeyJSClassifiesErrors(t *testing.T) {
	content, err := hostedlogin.PasskeyJS()
	if err != nil {
		t.Fatalf("load passkey js: %v", err)
	}
	script := string(content)
	for _, want := range []string{
		// Branch markers that prove each error class has its own path.
		"classifyPasskeyError",
		"\"unsupported\"",
		"\"cancelled\"",
		"\"origin\"",
		"\"network\"",
		"NotAllowedError",
		"AbortError",
		"NotSupportedError",
		"SecurityError",
		// User-facing copy for each class.
		"Sign-in was cancelled",
		"Passkeys aren't supported",
		"This sign-in link can't be used here",
		"couldn't reach the sign-in service",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("passkey js missing %q", want)
		}
	}
	// Ensure the generic catch-all is preserved as the final fallback.
	if !strings.Contains(script, "Sign-in didn't go through") {
		t.Fatalf("passkey js missing generic fallback message")
	}
}

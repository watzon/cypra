//go:build ignore

// render-hostedlogin-fixtures emits a static directory of the hosted-login
// templates for visual validation with agent-browser. It does not require a
// database — each template is rendered against a hand-rolled PageData fixture.
//
// Usage:
//
//	go run scripts/render-hostedlogin-fixtures.go -out /tmp/hostedlogin-fixtures
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/watzon/cypra/internal/hostedlogin"
)

func main() {
	out := flag.String("out", "/tmp/hostedlogin-fixtures", "output directory")
	serve := flag.String("serve", "", "if set, serve the output directory at this address (e.g. 127.0.0.1:5180) and block")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail("mkdir: %v", err)
	}

	renderer, err := hostedlogin.NewRenderer()
	if err != nil {
		fail("renderer: %v", err)
	}

	theme := hostedlogin.Theme{
		DisplayName: "Acme Login",
		Accent:      "#0F766E",
		LogoURL:     "",
		PoweredBy:   true,
	}

	scopes := []hostedlogin.Scope{
		{Name: "openid", Description: "Sign you in", New: false},
		{Name: "email", Description: "See your email address", New: true},
		{Name: "profile", Description: "See your basic profile", New: false},
	}

	fixtures := []struct {
		name string
		page string
		data hostedlogin.PageData
	}{
		{
			name: "login",
			page: "login",
			data: hostedlogin.PageData{
				Title:            "Sign in",
				Route:            "login",
				Theme:            theme,
				RPID:             "acme.cypra.localhost",
				PasskeyEnabled:   true,
				PasswordEnabled:  true,
				MagicLinkEnabled: true,
				SignupEnabled:    true,
				AnyMethodEnabled: true,
				PrimaryMethod:    "passkey",
			},
		},
		{
			name: "signup",
			page: "signup",
			data: hostedlogin.PageData{
				Title:            "Create your account",
				Route:            "signup",
				Theme:            theme,
				PasskeyEnabled:   true,
				PasswordEnabled:  true,
				MagicLinkEnabled: false,
				SignupEnabled:    true,
				AnyMethodEnabled: true,
				PrimaryMethod:    "passkey",
			},
		},
		{
			name: "2fa",
			page: "2fa",
			data: hostedlogin.PageData{
				Title: "Verify it's you",
				Route: "2fa",
				Theme: theme,
				RPID:  "acme.cypra.localhost",
			},
		},
		{
			name: "invite",
			page: "invite",
			data: hostedlogin.PageData{
				Title:           "Finish your account",
				Route:           "invite",
				Theme:           theme,
				InviteID:        "00000000-0000-0000-0000-000000000001",
				PasskeyEnabled:  true,
				PasswordEnabled: true,
			},
		},
		{
			name: "invite-expired",
			page: "invite",
			data: hostedlogin.PageData{
				Title: "Finish your account",
				Route: "invite",
				Theme: theme,
				Error: "Invite link is invalid or expired.",
			},
		},
		{
			name: "consent",
			page: "consent",
			data: hostedlogin.PageData{
				Title:  "Allow access?",
				Route:  "consent",
				Theme:  theme,
				Scopes: scopes,
			},
		},
		{
			name: "reset",
			page: "reset",
			data: hostedlogin.PageData{
				Title:      "Reset password",
				Route:      "reset",
				Theme:      theme,
				ResetToken: "demo-token",
			},
		},
		{
			name: "error",
			page: "error",
			data: hostedlogin.PageData{
				Title: "Sign-in unavailable",
				Route: "error",
				Theme: theme,
				Error: "We couldn't reach the sign-in service. Try again in a moment.",
			},
		},
		{
			name: "login-accent-orange",
			page: "login",
			data: hostedlogin.PageData{
				Title:            "Sign in",
				Route:            "login",
				Theme:            hostedlogin.Theme{DisplayName: "Acme Login", Accent: "#C2410C", PoweredBy: true},
				RPID:             "acme.cypra.localhost",
				PasskeyEnabled:   true,
				PasswordEnabled:  true,
				MagicLinkEnabled: true,
				SignupEnabled:    true,
				AnyMethodEnabled: true,
				PrimaryMethod:    "passkey",
			},
		},
	}

	for _, fx := range fixtures {
		path := filepath.Join(*out, fx.name+".html")
		f, err := os.Create(path)
		if err != nil {
			fail("create %s: %v", path, err)
		}
		if err := renderer.Render(f, fx.page, fx.data); err != nil {
			fail("render %s: %v", fx.name, err)
		}
		_ = f.Close()
		fmt.Printf("wrote %s\n", path)
	}

	// Also copy the vendored htmx and passkey scripts so SRI-pinned references
	// load against the same origin when served back.
	staticDir := filepath.Join(*out, "static", "hostedlogin")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		fail("mkdir static: %v", err)
	}
	for _, asset := range []struct {
		name string
		load func() ([]byte, error)
	}{
		{"htmx.min.js", hostedlogin.HTMXJS},
		{"passkey.js", hostedlogin.PasskeyJS},
		{"instance-admin-login.js", hostedlogin.InstanceAdminLoginJS},
	} {
		bytes, err := asset.load()
		if err != nil {
			fail("load %s: %v", asset.name, err)
		}
		if err := os.WriteFile(filepath.Join(staticDir, asset.name), bytes, 0o644); err != nil {
			fail("write %s: %v", asset.name, err)
		}
		fmt.Printf("wrote %s\n", filepath.Join(staticDir, asset.name))
	}

	if *serve != "" {
		fmt.Printf("serving %s on http://%s/\n", *out, *serve)
		if err := http.ListenAndServe(*serve, http.FileServer(http.Dir(*out))); err != nil {
			fail("serve: %v", err)
		}
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

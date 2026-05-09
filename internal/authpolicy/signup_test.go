package authpolicy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestCheckSignupDecisionTable(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	ctx := context.Background()

	cases := []struct {
		name  string
		setup string // SQL to run on tenants/tenant_auth_methods before the call
		input authpolicy.SignupContext
		want  error
	}{
		{
			name:  "open + via invite + invites enabled",
			setup: ``,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "alice@example.com", ViaInvite: true},
			want:  nil,
		},
		{
			name:  "via invite + invites disabled",
			setup: `UPDATE tenants SET invites_enabled = false WHERE id = $1`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "alice@example.com", ViaInvite: true},
			want:  authpolicy.ErrInvitesDisabled,
		},
		{
			name:  "closed blocks self-serve",
			setup: `UPDATE tenants SET signup_mode = 'closed' WHERE id = $1`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "alice@example.com"},
			want:  authpolicy.ErrSignupClosed,
		},
		{
			name:  "closed but invite still works",
			setup: `UPDATE tenants SET signup_mode = 'closed' WHERE id = $1`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "alice@example.com", ViaInvite: true},
			want:  nil,
		},
		{
			name:  "method allow_signup false",
			setup: `INSERT INTO tenant_auth_methods (tenant_id, method, enabled, config) VALUES ($1, 'password', true, '{"allow_signup": false}'::jsonb)`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "alice@example.com"},
			want:  authpolicy.ErrSignupDisabledForMethod,
		},
		{
			name:  "restricted + email not in allowlist",
			setup: `UPDATE tenants SET signup_mode = 'restricted', signup_allowlist = ARRAY['*@example.com'] WHERE id = $1`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "outsider@gmail.com"},
			want:  authpolicy.ErrSignupRestricted,
		},
		{
			name:  "restricted + domain wildcard match",
			setup: `UPDATE tenants SET signup_mode = 'restricted', signup_allowlist = ARRAY['*@example.com'] WHERE id = $1`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "Alice@Example.com"},
			want:  nil,
		},
		{
			name:  "restricted + exact email match",
			setup: `UPDATE tenants SET signup_mode = 'restricted', signup_allowlist = ARRAY['alice@example.com'] WHERE id = $1`,
			input: authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: "ALICE@example.com"},
			want:  nil,
		},
		{
			name:  "open default + email passes",
			setup: ``,
			input: authpolicy.SignupContext{Method: authpolicy.MethodMagicLink, Email: "anyone@anywhere.test"},
			want:  nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset tenant state between cases.
			if _, err := harness.SQL.Exec(`UPDATE tenants SET signup_mode = 'open', signup_allowlist = '{}', invites_enabled = true WHERE id = $1`, tenantID); err != nil {
				t.Fatalf("reset tenant: %v", err)
			}
			if _, err := harness.SQL.Exec(`DELETE FROM tenant_auth_methods WHERE tenant_id = $1`, tenantID); err != nil {
				t.Fatalf("reset auth methods: %v", err)
			}
			if tc.setup != "" {
				if _, err := harness.SQL.Exec(tc.setup, tenantID); err != nil {
					t.Fatalf("setup: %v", err)
				}
			}
			err := authpolicy.CheckSignup(ctx, harness.SQL, tenantID, tc.input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("CheckSignup err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNormalizeAllowlistEntry(t *testing.T) {
	cases := map[string]string{
		"  Alice@Example.com  ": "alice@example.com",
		"*@example.com":         "*@example.com",
		"*@":                    "",
		"alice":                 "",
		"":                      "",
		"  ":                    "",
		"*@a b.com":             "",
	}
	for input, want := range cases {
		if got := authpolicy.NormalizeAllowlistEntry(input); got != want {
			t.Fatalf("NormalizeAllowlistEntry(%q) = %q, want %q", input, got, want)
		}
	}
}

package authpolicy_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPasswordPolicyDefaults(t *testing.T) {
	p := authpolicy.PasswordPolicy{}.WithDefaults()
	if p.MinLength != 12 {
		t.Fatalf("MinLength = %d, want 12", p.MinLength)
	}
	if p.MaxLength != 256 {
		t.Fatalf("MaxLength = %d, want 256", p.MaxLength)
	}
	if !p.IsResetEnabled() {
		t.Fatalf("ResetEnabled default = false, want true")
	}
}

func TestPasswordPolicyResetExplicitlyDisabled(t *testing.T) {
	f := false
	p := authpolicy.PasswordPolicy{ResetEnabled: &f}
	if p.IsResetEnabled() {
		t.Fatalf("admin set ResetEnabled=false but IsResetEnabled returned true")
	}
}

func TestPasswordPolicyValidate(t *testing.T) {
	cases := []struct {
		name     string
		policy   authpolicy.PasswordPolicy
		password string
		wantErr  error
	}{
		{"too short", authpolicy.PasswordPolicy{MinLength: 12}, "short", authpolicy.ErrPasswordTooShort},
		{"too long", authpolicy.PasswordPolicy{MinLength: 4, MaxLength: 8}, "abcdefghij", authpolicy.ErrPasswordTooLong},
		{"missing upper", authpolicy.PasswordPolicy{RequireUpper: true}, "all-lowercase-12", authpolicy.ErrPasswordMissingUpper},
		{"missing digit", authpolicy.PasswordPolicy{RequireDigit: true}, "NoDigitsHere!", authpolicy.ErrPasswordMissingDigit},
		{"missing special", authpolicy.PasswordPolicy{RequireSpecial: true}, "NoSpecial1Char", authpolicy.ErrPasswordMissingSpecial},
		{"valid default", authpolicy.PasswordPolicy{}, "abcdefghijkl", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.policy.Validate(tc.password); err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestMagicLinkPolicyDefaultsAndExplicitFalse(t *testing.T) {
	p := authpolicy.MagicLinkPolicy{}.WithDefaults()
	if p.TTLMinutes != 15 {
		t.Fatalf("TTLMinutes = %d, want 15", p.TTLMinutes)
	}
	if p.MaxActivePerUser != 3 {
		t.Fatalf("MaxActivePerUser = %d, want 3", p.MaxActivePerUser)
	}
	if !p.AllowsSignup() {
		t.Fatalf("AllowsSignup default = false, want true")
	}
	f := false
	p2 := authpolicy.MagicLinkPolicy{AllowSignup: &f}
	if p2.AllowsSignup() {
		t.Fatalf("explicit AllowSignup=false ignored")
	}
}

func TestGooglePolicyDomainAllowList(t *testing.T) {
	p := authpolicy.GooglePolicy{AllowedDomains: []string{"example.com", "Acme.org"}}
	cases := map[string]bool{
		"alice@example.com":  true,
		"bob@acme.org":       true,
		"intruder@gmail.com": false,
		"":                   false,
		"no-at":              false,
	}
	for email, want := range cases {
		if got := p.AllowsEmail(email); got != want {
			t.Fatalf("AllowsEmail(%q) = %v, want %v", email, got, want)
		}
	}
	open := authpolicy.GooglePolicy{}
	if !open.AllowsEmail("anyone@anywhere.test") {
		t.Fatalf("empty AllowedDomains should accept anything")
	}
}

func TestLoadPasswordRoundtrip(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")

	got, err := authpolicy.LoadPassword(context.Background(), harness.SQL, tenantID)
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if got.MinLength != 12 {
		t.Fatalf("missing row should fall back to defaults, got MinLength=%d", got.MinLength)
	}

	cfg, _ := json.Marshal(authpolicy.PasswordPolicy{MinLength: 18, RequireUpper: true})
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled, config) VALUES ($1, 'password', true, $2)`, tenantID, cfg); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err = authpolicy.LoadPassword(context.Background(), harness.SQL, tenantID)
	if err != nil {
		t.Fatalf("load existing: %v", err)
	}
	if got.MinLength != 18 || !got.RequireUpper {
		t.Fatalf("loaded policy = %+v, want MinLength=18 RequireUpper=true", got)
	}
	if err := got.Validate("nouppercaseplease12"); err != authpolicy.ErrPasswordMissingUpper {
		t.Fatalf("Validate err = %v, want %v", err, authpolicy.ErrPasswordMissingUpper)
	}
}

func TestLoadMagicLinkRoundtrip(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")

	cfg := []byte(`{"ttl_minutes": 5, "allow_signup": false}`)
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled, config) VALUES ($1, 'magic_link', true, $2)`, tenantID, cfg); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := authpolicy.LoadMagicLink(context.Background(), harness.SQL, tenantID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.TTL().Minutes() != 5 {
		t.Fatalf("TTL minutes = %v, want 5", got.TTL().Minutes())
	}
	if got.AllowsSignup() {
		t.Fatalf("AllowsSignup should be false (explicit)")
	}
	if !strings.Contains(got.TTL().String(), "5m") {
		t.Fatalf("expected 5m TTL string, got %s", got.TTL())
	}
}

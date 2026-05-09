// Package authpolicy holds the per-tenant policy structs that govern auth
// provider behavior (password complexity, magic-link TTL, signup eligibility,
// allowed Google domains, etc.) and the helpers that load them from the
// tenant_auth_methods.config JSONB column.
package authpolicy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Method is the tenant_auth_method enum value as a Go string.
type Method = string

const (
	MethodPassword     Method = "password"
	MethodMagicLink    Method = "magic_link"
	MethodPasskey      Method = "passkey"
	MethodTOTP         Method = "totp"
	MethodGoogle       Method = "google"
	MethodOIDCUpstream Method = "oidc_upstream"
)

// PasswordPolicy controls how new passwords are validated.
type PasswordPolicy struct {
	MinLength            int   `json:"min_length"`
	MaxLength            int   `json:"max_length"`
	RequireUpper         bool  `json:"require_upper"`
	RequireDigit         bool  `json:"require_digit"`
	RequireSpecial       bool  `json:"require_special"`
	AllowCommonPasswords bool  `json:"allow_common_passwords"`
	ResetEnabled         *bool `json:"reset_enabled,omitempty"`
	AllowSignup          *bool `json:"allow_signup,omitempty"`
}

// WithDefaults returns the policy with zero-value fields replaced by safe defaults.
func (p PasswordPolicy) WithDefaults() PasswordPolicy {
	if p.MinLength == 0 {
		p.MinLength = 12
	}
	if p.MaxLength == 0 {
		p.MaxLength = 256
	}
	if p.ResetEnabled == nil {
		t := true
		p.ResetEnabled = &t
	}
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	return p
}

// IsResetEnabled reports whether password reset flows are allowed.
func (p PasswordPolicy) IsResetEnabled() bool { return derefBool(p.WithDefaults().ResetEnabled, true) }

// AllowsSignup reports whether new users may register via password.
func (p PasswordPolicy) AllowsSignup() bool { return derefBool(p.WithDefaults().AllowSignup, true) }

// Validate checks the supplied plaintext against the policy's structural rules.
// It does not check common-password lists; caller composes that separately.
func (p PasswordPolicy) Validate(plaintext string) error {
	policy := p.WithDefaults()
	if len(plaintext) < policy.MinLength {
		return ErrPasswordTooShort
	}
	if policy.MaxLength > 0 && len(plaintext) > policy.MaxLength {
		return ErrPasswordTooLong
	}
	if policy.RequireUpper && !containsAny(plaintext, isUpper) {
		return ErrPasswordMissingUpper
	}
	if policy.RequireDigit && !containsAny(plaintext, isDigit) {
		return ErrPasswordMissingDigit
	}
	if policy.RequireSpecial && !containsAny(plaintext, isSpecial) {
		return ErrPasswordMissingSpecial
	}
	return nil
}

// MagicLinkPolicy controls magic-link issuance and lifetime.
type MagicLinkPolicy struct {
	TTLMinutes       int   `json:"ttl_minutes"`
	MaxActivePerUser int   `json:"max_active_per_user"`
	AllowSignup      *bool `json:"allow_signup,omitempty"`
}

// WithDefaults returns the policy with zero-value fields replaced by safe defaults.
func (p MagicLinkPolicy) WithDefaults() MagicLinkPolicy {
	if p.TTLMinutes == 0 {
		p.TTLMinutes = 15
	}
	if p.MaxActivePerUser == 0 {
		p.MaxActivePerUser = 3
	}
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	return p
}

// TTL returns the configured TTL as a time.Duration.
func (p MagicLinkPolicy) TTL() time.Duration {
	return time.Duration(p.WithDefaults().TTLMinutes) * time.Minute
}

// AllowsSignup reports whether new accounts may sign up via magic link.
func (p MagicLinkPolicy) AllowsSignup() bool { return derefBool(p.WithDefaults().AllowSignup, true) }

// PasskeyPolicy controls passkey enrollment and verification.
type PasskeyPolicy struct {
	AllowSignup             *bool `json:"allow_signup,omitempty"`
	RequireUserVerification *bool `json:"require_user_verification,omitempty"`
}

// WithDefaults returns the policy with safe defaults.
func (p PasskeyPolicy) WithDefaults() PasskeyPolicy {
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	if p.RequireUserVerification == nil {
		t := true
		p.RequireUserVerification = &t
	}
	return p
}

// AllowsSignup reports whether new accounts may sign up via passkey.
func (p PasskeyPolicy) AllowsSignup() bool { return derefBool(p.WithDefaults().AllowSignup, true) }

// RequiresUserVerification reports whether the WebAuthn UV flag must be enforced.
func (p PasskeyPolicy) RequiresUserVerification() bool {
	return derefBool(p.WithDefaults().RequireUserVerification, true)
}

// TOTPPolicy controls TOTP enrollment.
type TOTPPolicy struct {
	Issuer string `json:"issuer"`
}

// WithDefaults is a no-op for TOTP; Issuer falls back to tenant display name at use sites.
func (p TOTPPolicy) WithDefaults() TOTPPolicy { return p }

// GooglePolicy gates which Google accounts may sign in.
type GooglePolicy struct {
	AllowedDomains []string `json:"allowed_domains"`
	AllowSignup    *bool    `json:"allow_signup,omitempty"`
}

// WithDefaults returns the policy with safe defaults.
func (p GooglePolicy) WithDefaults() GooglePolicy {
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	return p
}

// AllowsSignup reports whether new accounts may sign up via Google.
func (p GooglePolicy) AllowsSignup() bool { return derefBool(p.WithDefaults().AllowSignup, true) }

// AllowsEmail returns true when the given email's domain matches the policy's
// allow-list (or the list is empty).
func (p GooglePolicy) AllowsEmail(email string) bool {
	domains := p.AllowedDomains
	if len(domains) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	got := strings.ToLower(email[at+1:])
	for _, d := range domains {
		if strings.EqualFold(strings.TrimSpace(d), got) {
			return true
		}
	}
	return false
}

// OIDCUpstreamPolicy is a stub; oidc_upstream wiring is not yet built out.
type OIDCUpstreamPolicy struct {
	AllowSignup *bool `json:"allow_signup,omitempty"`
}

// WithDefaults returns the policy with safe defaults.
func (p OIDCUpstreamPolicy) WithDefaults() OIDCUpstreamPolicy {
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	return p
}

// AllowsSignup reports whether new accounts may sign up via the OIDC upstream.
func (p OIDCUpstreamPolicy) AllowsSignup() bool {
	return derefBool(p.WithDefaults().AllowSignup, true)
}

// Errors returned by Validate methods.
var (
	ErrPasswordTooShort       = errors.New("password.policy_too_short")
	ErrPasswordTooLong        = errors.New("password.policy_too_long")
	ErrPasswordMissingUpper   = errors.New("password.policy_missing_upper")
	ErrPasswordMissingDigit   = errors.New("password.policy_missing_digit")
	ErrPasswordMissingSpecial = errors.New("password.policy_missing_special")
)

// LoadPassword loads the password policy for a tenant.
func LoadPassword(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (PasswordPolicy, error) {
	var policy PasswordPolicy
	if err := loadConfig(ctx, db, tenantID, MethodPassword, &policy); err != nil {
		return PasswordPolicy{}, err
	}
	return policy.WithDefaults(), nil
}

// LoadMagicLink loads the magic-link policy for a tenant.
func LoadMagicLink(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (MagicLinkPolicy, error) {
	var policy MagicLinkPolicy
	if err := loadConfig(ctx, db, tenantID, MethodMagicLink, &policy); err != nil {
		return MagicLinkPolicy{}, err
	}
	return policy.WithDefaults(), nil
}

// LoadPasskey loads the passkey policy for a tenant.
func LoadPasskey(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (PasskeyPolicy, error) {
	var policy PasskeyPolicy
	if err := loadConfig(ctx, db, tenantID, MethodPasskey, &policy); err != nil {
		return PasskeyPolicy{}, err
	}
	return policy.WithDefaults(), nil
}

// LoadTOTP loads the TOTP policy for a tenant.
func LoadTOTP(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (TOTPPolicy, error) {
	var policy TOTPPolicy
	if err := loadConfig(ctx, db, tenantID, MethodTOTP, &policy); err != nil {
		return TOTPPolicy{}, err
	}
	return policy.WithDefaults(), nil
}

// LoadGoogle loads the Google upstream policy for a tenant.
func LoadGoogle(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (GooglePolicy, error) {
	var policy GooglePolicy
	if err := loadConfig(ctx, db, tenantID, MethodGoogle, &policy); err != nil {
		return GooglePolicy{}, err
	}
	return policy.WithDefaults(), nil
}

// LoadOIDCUpstream loads the OIDC upstream policy for a tenant.
func LoadOIDCUpstream(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (OIDCUpstreamPolicy, error) {
	var policy OIDCUpstreamPolicy
	if err := loadConfig(ctx, db, tenantID, MethodOIDCUpstream, &policy); err != nil {
		return OIDCUpstreamPolicy{}, err
	}
	return policy.WithDefaults(), nil
}

func loadConfig(ctx context.Context, db *sql.DB, tenantID uuid.UUID, method Method, dest any) error {
	var raw []byte
	err := db.QueryRowContext(ctx, `SELECT config FROM tenant_auth_methods WHERE tenant_id = $1 AND method = $2::tenant_auth_method`, tenantID, method).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

func derefBool(b *bool, fallback bool) bool {
	if b == nil {
		return fallback
	}
	return *b
}

func containsAny(s string, predicate func(rune) bool) bool {
	for _, r := range s {
		if predicate(r) {
			return true
		}
	}
	return false
}

func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isDigit(r rune) bool { return r >= '0' && r <= '9' }
func isSpecial(r rune) bool {
	return strings.ContainsRune("!@#$%^&*()-_=+[]{};:'\",.<>/?\\|`~", r)
}

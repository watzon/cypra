package authpolicy

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Sign-up modes mirror the tenant_signup_mode enum.
const (
	SignupModeOpen       = "open"
	SignupModeRestricted = "restricted"
	SignupModeClosed     = "closed"
)

// SignupSettings captures the tenant-wide registration controls.
type SignupSettings struct {
	Mode           string   // "open" | "restricted" | "closed"
	Allowlist      []string // exact email or "*@domain.com" patterns; case-insensitive
	InvitesEnabled bool
}

// SignupContext describes a candidate registration for the gate.
type SignupContext struct {
	Method    Method // password | magic_link | passkey | google | oidc_upstream
	Email     string
	ViaInvite bool
}

// Sentinel errors mapped to wire codes by HTTP handlers.
var (
	ErrSignupClosed            = errors.New("auth.signup_closed")
	ErrSignupRestricted        = errors.New("auth.signup_restricted")
	ErrSignupDisabledForMethod = errors.New("auth.signup_disabled_for_method")
	ErrInvitesDisabled         = errors.New("auth.invites_disabled")
)

// LoadSignupSettings fetches the tenant-level signup controls.
func LoadSignupSettings(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (SignupSettings, error) {
	var (
		mode           string
		allowlist      pq.StringArray
		invitesEnabled bool
	)
	err := db.QueryRowContext(ctx, `SELECT signup_mode::text, signup_allowlist, invites_enabled FROM tenants WHERE id = $1`, tenantID).Scan(&mode, &allowlist, &invitesEnabled)
	if errors.Is(err, sql.ErrNoRows) {
		return SignupSettings{Mode: SignupModeOpen, InvitesEnabled: true}, nil
	}
	if err != nil {
		return SignupSettings{}, err
	}
	return SignupSettings{
		Mode:           mode,
		Allowlist:      []string(allowlist),
		InvitesEnabled: invitesEnabled,
	}, nil
}

// CheckSignup decides whether the candidate registration is allowed.
// Per-method `allow_signup` is loaded internally so callers don't need to.
func CheckSignup(ctx context.Context, db *sql.DB, tenantID uuid.UUID, c SignupContext) error {
	settings, err := LoadSignupSettings(ctx, db, tenantID)
	if err != nil {
		return err
	}
	if c.ViaInvite {
		if !settings.InvitesEnabled {
			return ErrInvitesDisabled
		}
		return nil
	}
	if settings.Mode == SignupModeClosed {
		return ErrSignupClosed
	}
	allowed, err := methodAllowsSignup(ctx, db, tenantID, c.Method)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrSignupDisabledForMethod
	}
	if settings.Mode == SignupModeRestricted && !allowlistMatches(settings.Allowlist, c.Email) {
		return ErrSignupRestricted
	}
	return nil
}

func methodAllowsSignup(ctx context.Context, db *sql.DB, tenantID uuid.UUID, method Method) (bool, error) {
	family, sub := SplitMethod(method)
	switch family {
	case MethodPassword:
		p, err := LoadPassword(ctx, db, tenantID)
		return p.AllowsSignup(), err
	case MethodMagicLink:
		p, err := LoadMagicLink(ctx, db, tenantID)
		return p.AllowsSignup(), err
	case MethodPasskey:
		p, err := LoadPasskey(ctx, db, tenantID)
		return p.AllowsSignup(), err
	case MethodGoogle:
		// Legacy direct-Google method (pre-social_connections). Honor the
		// tenant_auth_methods.config policy until callers migrate to social:google.
		p, err := LoadGoogle(ctx, db, tenantID)
		return p.AllowsSignup(), err
	case MethodOIDCUpstream:
		p, err := LoadOIDCUpstream(ctx, db, tenantID)
		return p.AllowsSignup(), err
	case "social":
		p, err := LoadSocial(ctx, db, tenantID, sub)
		return p.AllowsSignup(), err
	case "oidc":
		p, err := LoadOIDCConnection(ctx, db, tenantID, sub)
		return p.AllowsSignup(), err
	case MethodTOTP:
		// TOTP isn't a registration channel.
		return false, nil
	default:
		return false, nil
	}
}

// allowlistMatches reports whether email matches any pattern in list.
// Patterns are either exact emails (case-insensitive) or "*@domain.com" wildcards.
func allowlistMatches(list []string, email string) bool {
	if len(list) == 0 {
		return false
	}
	got := strings.ToLower(strings.TrimSpace(email))
	if got == "" {
		return false
	}
	at := strings.LastIndex(got, "@")
	domain := ""
	if at >= 0 {
		domain = got[at+1:]
	}
	for _, raw := range list {
		pattern := strings.ToLower(strings.TrimSpace(raw))
		if pattern == "" {
			continue
		}
		if strings.HasPrefix(pattern, "*@") {
			if domain != "" && pattern[2:] == domain {
				return true
			}
			continue
		}
		if pattern == got {
			return true
		}
	}
	return false
}

// NormalizeAllowlistEntry trims whitespace and lower-cases pattern.
// Returns "" when the entry is invalid (no @ for exact, malformed wildcard).
func NormalizeAllowlistEntry(raw string) string {
	pattern := strings.ToLower(strings.TrimSpace(raw))
	if pattern == "" {
		return ""
	}
	if strings.HasPrefix(pattern, "*@") {
		domain := pattern[2:]
		if domain == "" || strings.ContainsAny(domain, " \t,;") {
			return ""
		}
		return pattern
	}
	if !strings.Contains(pattern, "@") {
		return ""
	}
	return pattern
}

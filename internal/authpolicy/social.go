package authpolicy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// Namespaced method prefixes for social SSO and enterprise OIDC connections.
// They live alongside the built-in Method constants and are accepted anywhere
// SignupContext.Method or default-method values are consumed.
const (
	MethodSocialPrefix = "social:"
	MethodOIDCPrefix   = "oidc:"
)

// MethodForSocial returns the canonical "social:<kind>" identifier.
func MethodForSocial(kind string) Method {
	return MethodSocialPrefix + strings.ToLower(strings.TrimSpace(kind))
}

// MethodForOIDC returns the canonical "oidc:<slug>" identifier.
func MethodForOIDC(slug string) Method {
	return MethodOIDCPrefix + strings.ToLower(strings.TrimSpace(slug))
}

// SplitMethod returns (family, sub) for namespaced methods. For built-in
// methods family equals method and sub is empty.
func SplitMethod(m Method) (family, sub string) {
	switch {
	case strings.HasPrefix(m, MethodSocialPrefix):
		return "social", strings.TrimPrefix(m, MethodSocialPrefix)
	case strings.HasPrefix(m, MethodOIDCPrefix):
		return "oidc", strings.TrimPrefix(m, MethodOIDCPrefix)
	default:
		return m, ""
	}
}

// SocialPolicy gates which accounts may sign in via a social provider and is
// shared across every Core 5 connection (and any future ones).
type SocialPolicy struct {
	AllowedDomains []string `json:"allowed_domains"`
	AllowSignup    *bool    `json:"allow_signup,omitempty"`
}

func (p SocialPolicy) WithDefaults() SocialPolicy {
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	return p
}

func (p SocialPolicy) AllowsSignup() bool { return derefBool(p.WithDefaults().AllowSignup, true) }

// AllowsEmail reports whether an email address satisfies the allowed-domains
// list. An empty list permits any domain.
func (p SocialPolicy) AllowsEmail(email string) bool {
	if len(p.AllowedDomains) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	got := strings.ToLower(email[at+1:])
	for _, d := range p.AllowedDomains {
		if strings.EqualFold(strings.TrimSpace(d), got) {
			return true
		}
	}
	return false
}

// OIDCConnectionPolicy gates an enterprise OIDC connection.
type OIDCConnectionPolicy struct {
	AllowedDomains []string `json:"allowed_domains"`
	AllowSignup    *bool    `json:"allow_signup,omitempty"`
}

func (p OIDCConnectionPolicy) WithDefaults() OIDCConnectionPolicy {
	if p.AllowSignup == nil {
		t := true
		p.AllowSignup = &t
	}
	return p
}

func (p OIDCConnectionPolicy) AllowsSignup() bool {
	return derefBool(p.WithDefaults().AllowSignup, true)
}

func (p OIDCConnectionPolicy) AllowsEmail(email string) bool {
	if len(p.AllowedDomains) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	got := strings.ToLower(email[at+1:])
	for _, d := range p.AllowedDomains {
		if strings.EqualFold(strings.TrimSpace(d), got) {
			return true
		}
	}
	return false
}

// LoadSocial reads the SocialPolicy for a given social_connections row. When
// the row is missing the empty (defaults-applied) policy is returned with no
// error.
func LoadSocial(ctx context.Context, db *sql.DB, tenantID uuid.UUID, kind string) (SocialPolicy, error) {
	var raw []byte
	err := db.QueryRowContext(ctx, `SELECT config FROM social_connections WHERE tenant_id = $1 AND kind = $2::social_provider_kind`, tenantID, strings.ToLower(kind)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return SocialPolicy{}.WithDefaults(), nil
	}
	if err != nil {
		return SocialPolicy{}, err
	}
	var policy SocialPolicy
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &policy); err != nil {
			return SocialPolicy{}, err
		}
	}
	return policy.WithDefaults(), nil
}

// LoadOIDCConnection reads the OIDCConnectionPolicy for a given oidc_connections row.
func LoadOIDCConnection(ctx context.Context, db *sql.DB, tenantID uuid.UUID, slug string) (OIDCConnectionPolicy, error) {
	var raw []byte
	err := db.QueryRowContext(ctx, `SELECT config FROM oidc_connections WHERE tenant_id = $1 AND slug = $2`, tenantID, strings.ToLower(slug)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return OIDCConnectionPolicy{}.WithDefaults(), nil
	}
	if err != nil {
		return OIDCConnectionPolicy{}, err
	}
	var policy OIDCConnectionPolicy
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &policy); err != nil {
			return OIDCConnectionPolicy{}, err
		}
	}
	return policy.WithDefaults(), nil
}

package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"

	"github.com/watzon/cypra/internal/auth/upstream"
	"github.com/watzon/cypra/internal/auth/upstream/oidcgeneric"
	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/observability"
)

// hostedSocialStart issues a 302 to the upstream social provider.
func (s *Server) hostedSocialStart(w http.ResponseWriter, r *http.Request) {
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	ctx, span := observability.StartSpan(r.Context(), "hosted.social.start", attribute.String("oauth.provider", kind))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login?error=tenant_not_found", http.StatusFound)
		return
	}
	provider, err := upstream.Resolve(kind)
	if err != nil {
		http.Redirect(w, r, "/login?error=social_unknown", http.StatusFound)
		return
	}
	creds, err := s.loadSocialCredentials(r.Context(), tenant.ID, kind)
	if err != nil {
		http.Redirect(w, r, "/login?error=social_not_configured", http.StatusFound)
		return
	}
	creds.RedirectURI = "https://" + s.installHost + "/login/social/" + kind + "/callback"
	returnURL := r.URL.Query().Get("continue")
	if returnURL == "" {
		returnURL = "/"
	}
	req, err := provider.AuthCodeURL(r.Context(), creds, upstream.AuthParams{TenantID: tenant.ID, ReturnURL: returnURL})
	if err != nil {
		observability.RecordError(span, err)
		http.Redirect(w, r, "/login?error=social_start_failed", http.StatusFound)
		return
	}
	http.Redirect(w, r, req.URL, http.StatusFound)
}

// hostedSocialCallback completes the OAuth code exchange via a browser redirect.
func (s *Server) hostedSocialCallback(w http.ResponseWriter, r *http.Request) {
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	ctx, span := observability.StartSpan(r.Context(), "hosted.social.callback", attribute.String("oauth.provider", kind))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login?error=tenant_not_found", http.StatusFound)
		return
	}
	provider, err := upstream.Resolve(kind)
	if err != nil {
		http.Redirect(w, r, "/login?error=social_unknown", http.StatusFound)
		return
	}
	creds, err := s.loadSocialCredentials(r.Context(), tenant.ID, kind)
	if err != nil {
		http.Redirect(w, r, "/login?error=social_not_configured", http.StatusFound)
		return
	}
	creds.RedirectURI = "https://" + s.installHost + "/login/social/" + kind + "/callback"
	code := r.URL.Query().Get("code")
	rawState := r.URL.Query().Get("state")
	state, err := upstream.ValidateState(creds.StateSecret, rawState, time.Now().UTC())
	if err != nil || state.TenantID != tenant.ID {
		recordAuthAttempt(kind, "fail")
		http.Redirect(w, r, "/login?error=social_invalid_state", http.StatusFound)
		return
	}
	identity, err := provider.Exchange(r.Context(), creds, upstream.ExchangeParams{Code: code, State: rawState, Nonce: state.Nonce, RedirectURI: creds.RedirectURI, Now: time.Now().UTC()})
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt(kind, "fail")
		http.Redirect(w, r, "/login?error=social_exchange_failed", http.StatusFound)
		return
	}
	policy, err := authpolicy.LoadSocial(r.Context(), s.DB, tenant.ID, kind)
	if err != nil {
		http.Redirect(w, r, "/login?error=db", http.StatusFound)
		return
	}
	if !policy.AllowsEmail(identity.Email) {
		recordAuthAttempt(kind, "fail")
		http.Redirect(w, r, "/login?error=social_email_not_allowed", http.StatusFound)
		return
	}
	userID, err := s.findOrCreateSocialUser(r.Context(), tenant.ID, kind, identity)
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt(kind, "fail")
		if isSignupError(err) {
			http.Redirect(w, r, "/login?error="+err.Error(), http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login?error=social_user_failed", http.StatusFound)
		return
	}
	if _, err := s.establishUserSession(w, r, tenant.ID, userID); err != nil {
		http.Redirect(w, r, "/login?error=session", http.StatusFound)
		return
	}
	recordAuthAttempt(kind, "success")
	dest := state.ReturnURL
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (s *Server) hostedOIDCStart(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	ctx, span := observability.StartSpan(r.Context(), "hosted.oidc.start", attribute.String("oidc.slug", slug))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login?error=tenant_not_found", http.StatusFound)
		return
	}
	row, err := s.loadOIDCRow(r.Context(), tenant.ID, slug)
	if err != nil || row == nil || !row.Enabled {
		http.Redirect(w, r, "/login?error=oidc_not_configured", http.StatusFound)
		return
	}
	clientID, clientSecret, err := s.decryptOIDCRow(row)
	if err != nil {
		http.Redirect(w, r, "/login?error=oidc_decrypt_failed", http.StatusFound)
		return
	}
	provider := oidcgeneric.Provider{Slug: slug, DisplayName: row.DisplayName, Issuer: row.IssuerURL, DefaultScopes: []string(row.Scopes)}
	creds := upstream.Credentials{ClientID: clientID, ClientSecret: clientSecret, StateSecret: s.GoogleSecret, RedirectURI: "https://" + s.installHost + "/login/oidc/" + slug + "/callback"}
	returnURL := r.URL.Query().Get("continue")
	if returnURL == "" {
		returnURL = "/"
	}
	req, err := provider.AuthCodeURL(r.Context(), creds, upstream.AuthParams{TenantID: tenant.ID, ReturnURL: returnURL})
	if err != nil {
		observability.RecordError(span, err)
		http.Redirect(w, r, "/login?error=oidc_start_failed", http.StatusFound)
		return
	}
	http.Redirect(w, r, req.URL, http.StatusFound)
}

func (s *Server) hostedOIDCCallback(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	ctx, span := observability.StartSpan(r.Context(), "hosted.oidc.callback", attribute.String("oidc.slug", slug))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login?error=tenant_not_found", http.StatusFound)
		return
	}
	row, err := s.loadOIDCRow(r.Context(), tenant.ID, slug)
	if err != nil || row == nil || !row.Enabled {
		http.Redirect(w, r, "/login?error=oidc_not_configured", http.StatusFound)
		return
	}
	clientID, clientSecret, err := s.decryptOIDCRow(row)
	if err != nil {
		http.Redirect(w, r, "/login?error=oidc_decrypt_failed", http.StatusFound)
		return
	}
	code := r.URL.Query().Get("code")
	rawState := r.URL.Query().Get("state")
	state, err := upstream.ValidateState(s.GoogleSecret, rawState, time.Now().UTC())
	if err != nil || state.TenantID != tenant.ID || state.Slug != slug {
		recordAuthAttempt("oidc:"+slug, "fail")
		http.Redirect(w, r, "/login?error=oidc_invalid_state", http.StatusFound)
		return
	}
	provider := oidcgeneric.Provider{Slug: slug, DisplayName: row.DisplayName, Issuer: row.IssuerURL, DefaultScopes: []string(row.Scopes)}
	creds := upstream.Credentials{ClientID: clientID, ClientSecret: clientSecret, StateSecret: s.GoogleSecret, RedirectURI: "https://" + s.installHost + "/login/oidc/" + slug + "/callback"}
	identity, err := provider.Exchange(r.Context(), creds, upstream.ExchangeParams{Code: code, State: rawState, Nonce: state.Nonce, RedirectURI: creds.RedirectURI, Now: time.Now().UTC()})
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("oidc:"+slug, "fail")
		http.Redirect(w, r, "/login?error=oidc_exchange_failed", http.StatusFound)
		return
	}
	policy, err := authpolicy.LoadOIDCConnection(r.Context(), s.DB, tenant.ID, slug)
	if err != nil {
		http.Redirect(w, r, "/login?error=db", http.StatusFound)
		return
	}
	if !policy.AllowsEmail(identity.Email) {
		recordAuthAttempt("oidc:"+slug, "fail")
		http.Redirect(w, r, "/login?error=oidc_email_not_allowed", http.StatusFound)
		return
	}
	userID, err := s.findOrCreateOIDCUser(r.Context(), tenant.ID, slug, identity)
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("oidc:"+slug, "fail")
		if isSignupError(err) {
			http.Redirect(w, r, "/login?error="+err.Error(), http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login?error=oidc_user_failed", http.StatusFound)
		return
	}
	if _, err := s.establishUserSession(w, r, tenant.ID, userID); err != nil {
		http.Redirect(w, r, "/login?error=session", http.StatusFound)
		return
	}
	recordAuthAttempt("oidc:"+slug, "success")
	dest := state.ReturnURL
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

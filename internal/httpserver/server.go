// Package httpserver wires the Cypra HTTP surface.
package httpserver

//revive:disable:exported

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/auth"
	"github.com/watzon/cypra/internal/botmitigation"
	"github.com/watzon/cypra/internal/db"
	"github.com/watzon/cypra/internal/logging"
	"github.com/watzon/cypra/internal/migrate"
	"github.com/watzon/cypra/internal/storage"
	"github.com/watzon/cypra/internal/storage/openstore"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"gorm.io/datatypes"
)

type Options struct {
	DB            *sql.DB
	TenantDB      *db.TenantScopedDB
	PublicBaseURL string
	Version       string
	Commit        string
	Logger        *slog.Logger
	DevOpenAPI    bool
	StorageReady  func(context.Context) error
	KEKLoaded     bool
	MasterKey     []byte
	BotVerifier   botmitigation.Verifier
	GoogleSecret  []byte
	GoogleAuthURL string
	DashboardFS   fs.FS
	DashboardDev  string
	Storage        storage.Store
	StorageBackend string
	// TrustDevHeaders gates whether the auth middleware honors the
	// X-Cypra-Instance-Admin / X-Cypra-Tenant-Role / X-Cypra-User-Id elevation
	// headers. Tests and trusted proxies set this; production must not.
	TrustDevHeaders bool
}

type Server struct {
	Options
	installHost string
	audit       *audit.Writer
}

func New(opts Options) (*Server, error) {
	base, err := url.Parse(opts.PublicBaseURL)
	if err != nil || base.Host == "" {
		return nil, fmt.Errorf("invalid PUBLIC_BASE_URL")
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.StorageReady == nil {
		opts.StorageReady = func(context.Context) error { return nil }
	}
	if opts.BotVerifier == nil {
		opts.BotVerifier = botmitigation.NoopVerifier{}
	}
	opts.BotVerifier = botmitigation.InstrumentedVerifier{Name: "noop", Next: opts.BotVerifier}
	if len(opts.GoogleSecret) == 0 {
		opts.GoogleSecret = []byte("dev-google-oauth-state-secret")
	}
	if !opts.DevOpenAPI && opts.DashboardFS != nil {
		if _, err := fs.Stat(opts.DashboardFS, "dist/index.html"); err != nil {
			return nil, fmt.Errorf("dashboard embed is empty: run make build-frontend")
		}
	}
	if opts.Storage == nil {
		store, kind, err := openstore.New(openstore.LoadConfigFromEnv(opts.MasterKey))
		if err != nil {
			return nil, fmt.Errorf("init storage: %w", err)
		}
		opts.Storage = store
		opts.StorageBackend = kind
	} else if opts.StorageBackend == "" {
		opts.StorageBackend = openstore.BackendLocalDisk
	}
	return &Server{Options: opts, installHost: base.Host, audit: audit.NewWriter(opts.DB)}, nil
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(s.requestID)
	r.Use(s.traceRequests)
	r.Use(s.logRequests)
	r.Get("/healthz", s.healthz)
	r.Get("/readyz", s.readyz)
	r.Get("/metrics", s.metrics)
	r.Group(func(public chi.Router) {
		public.Use(s.tenantResolver)
		public.Get("/static/hostedlogin/passkey.js", s.hostedStaticPasskey)
		public.Get("/static/hostedlogin/instance-admin-login.js", s.hostedStaticInstanceAdminLogin)
		public.Get("/setup", s.setupInstructionsPage)
		public.Get("/login", s.hostedPage("login"))
		public.Post("/login/password", s.hostedPasswordSignIn)
		public.Post("/login/passkey/verify", s.authPasskeyAssert)
		public.Post("/login/instance-admin/passkey", s.authInstanceAdminLogin)
		public.Post("/login/instance-admin/backup-code", s.authInstanceAdminBackupCode)
		public.Post("/login/magic-link", s.hostedMagicLink)
		public.Get("/login/social/{kind}", s.hostedSocialStart)
		public.Get("/login/social/{kind}/callback", s.hostedSocialCallback)
		public.Get("/login/oidc/{slug}", s.hostedOIDCStart)
		public.Get("/login/oidc/{slug}/callback", s.hostedOIDCCallback)
		public.Get("/signup", s.hostedPage("signup"))
		public.Post("/signup", s.hostedSignup)
		public.Post("/signup/passkey", s.hostedSignupPasskey)
		public.Get("/verify", s.hostedPage("verify"))
		public.Get("/invite", s.hostedInvite)
		public.Get("/storage/*", s.storageProxy)
		public.Get("/reset", s.hostedPage("reset"))
		public.Post("/reset", s.hostedPasswordReset)
		public.Get("/2fa", s.hostedPage("2fa"))
		public.Post("/2fa", s.hostedFactor)
		public.Get("/error", s.hostedPage("error"))
		public.Get("/.well-known/openid-configuration", s.oidcDiscovery)
		public.Get("/.well-known/jwks.json", s.oidcJWKS)
		public.Get("/oidc/authorize", s.oidcAuthorize)
		public.Get("/oidc/consent", s.hostedPage("consent"))
		public.Post("/oidc/token", s.oidcToken)
		public.Get("/oidc/userinfo", s.oidcUserInfo)
		public.Post("/oidc/revoke", s.oidcRevoke)
		public.Post("/oidc/consent", s.oidcConsent)
	})
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(s.tenantResolver)
		api.Use(auth.MiddlewareWithPAT(s.DB, s.TrustDevHeaders))
		api.Get("/version", s.version)
		api.Route("/setup", func(rt chi.Router) {
			rt.Post("/verify", s.setupVerify)
			rt.Post("/passkey/begin", s.setupPasskeyBegin)
			rt.Post("/complete", s.setupComplete)
		})
		if s.DevOpenAPI {
			api.Get("/openapi.json", s.openapi)
		}
		api.With(auth.RequireTenantRoleOrPATScope("audit.read")).Get("/audit/", s.auditList)
		api.With(auth.RequireTenantRoleOrPATScope("audit.export")).Get("/audit/export", s.auditExport)
		api.Route("/auth", func(rt chi.Router) {
			rt.Post("/password/signup", s.authPasswordSignup)
			rt.Post("/password/signin", s.authPasswordSignin)
			rt.Post("/password/reset", s.authPasswordReset)
			rt.Post("/magic-link/issue", s.authMagicLinkIssue)
			rt.Post("/magic-link/verify", s.authMagicLinkVerify)
			rt.Post("/invite/redeem", s.authInviteRedeem)
			rt.Post("/passkey/register", s.authPasskeyRegister)
			rt.Post("/passkey/assert", s.authPasskeyAssert)
			rt.Post("/totp/enroll", s.authTOTPEnroll)
			rt.Post("/totp/verify", s.authTOTPVerify)
			rt.Post("/webauthn2fa/enroll", s.authWebAuthn2FAEnroll)
			rt.Post("/webauthn2fa/verify", s.authWebAuthn2FAVerify)
			rt.Post("/backup-codes/regenerate", s.authBackupCodesRegenerate)
			rt.Post("/backup-codes/consume", s.authBackupCodesConsume)
			rt.Post("/google/start", s.authGoogleStart)
			rt.Post("/google/callback", s.authGoogleCallback)
			rt.Post("/social/{kind}/start", s.authSocialStart)
			rt.Post("/social/{kind}/callback", s.authSocialCallback)
			rt.Post("/oidc/{slug}/start", s.authOIDCStart)
			rt.Post("/oidc/{slug}/callback", s.authOIDCCallback)
			rt.Post("/logout", s.authLogout)
			rt.Post("/sessions/revoke-others", s.authRevokeOtherSessions)
			rt.Get("/passkeys", s.authListPasskeys)
			rt.Delete("/passkeys/{id}", s.authDeletePasskey)
			rt.Get("/sessions", s.authListSessions)
			rt.Delete("/sessions/{id}", s.authRevokeSession)
			rt.Get("/mfa/factors", s.authListMFAFactors)
		})
		api.Route("/admin", func(rt chi.Router) {
			rt.With(auth.RequireTenantRoleOrPATScope("admin.invite")).Post("/invite", s.adminInvite)
			rt.With(auth.RequireTenantRoleOrPATScope("admin.invite")).Get("/invites", s.listTenantInvites)
			rt.With(auth.RequireTenantRoleOrPATScope("admin.invite")).Delete("/invites/{id}", s.revokeTenantInvite)
			rt.With(auth.RequireTenantRoleOrPATScope("users.read")).Get("/members", s.listTenantMembers)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Put("/members/{id}", s.updateTenantMember)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Delete("/members/{id}", s.removeTenantMember)
		})
		api.Route("/instance", func(rt chi.Router) {
			rt.Use(auth.RequireInstanceAdmin)
			rt.Get("/summary", s.instanceSummary)
			rt.Get("/audit", s.instanceAuditList)
			rt.Get("/admins", s.listInstanceAdmins)
			rt.Delete("/admins/{id}", s.demoteInstanceAdmin)
			rt.Get("/diagnostics", s.instanceDiagnostics)
			rt.Patch("/admins/me", s.patchInstanceAdminMe)
			rt.Get("/invites", s.listInstanceInvites)
			rt.Post("/invite", s.instanceInvite)
			rt.Delete("/invites/{id}", s.revokeInstanceInvite)
		})
		api.Route("/provider-config", func(rt chi.Router) {
			rt.With(auth.RequireTenantRoleOrPATScope("provider-config.read")).Get("/email", s.emailProviderConfig)
			rt.With(auth.RequireTenantRoleOrPATScope("provider-config.write")).Put("/email", s.saveEmailProviderConfig)
			rt.With(auth.RequireTenantRoleOrPATScope("provider-config.read")).Post("/email/test", s.testEmailProviderConfig)
			rt.With(auth.RequireTenantRoleOrPATScope("provider-config.read")).Get("/upstream", s.upstreamProviderConfig)
			rt.With(auth.RequireTenantRoleOrPATScope("provider-config.write")).Put("/upstream", s.saveUpstreamProviderConfig)
			rt.With(auth.RequireTenantRoleOrPATScope("provider-config.read")).Post("/upstream/test", s.testUpstreamProviderConfig)
		})
		api.Route("/auth-providers", func(rt chi.Router) {
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/", s.listAuthProviders)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/default", s.getDefaultAuthMethod)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Put("/default", s.setDefaultAuthMethod)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/registration", s.getRegistrationSettings)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Put("/registration", s.saveRegistrationSettings)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/social/", s.listSocialConnections)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/social/{kind}", s.getSocialConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Put("/social/{kind}", s.saveSocialConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Delete("/social/{kind}", s.deleteSocialConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/oidc/", s.listOIDCConnections)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Post("/oidc/", s.createOIDCConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/oidc/{slug}", s.getOIDCConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Put("/oidc/{slug}", s.updateOIDCConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Delete("/oidc/{slug}", s.deleteOIDCConnection)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.read")).Get("/{method}", s.getAuthProvider)
			rt.With(auth.RequireTenantRoleOrPATScope("auth-providers.write")).Put("/{method}", s.saveAuthProvider)
		})
		api.Route("/signing-keys", func(rt chi.Router) {
			rt.With(auth.RequireTenantRoleOrPATScope("signing-keys.read")).Get("/", s.listSigningKeys)
			rt.With(auth.RequireTenantRoleOrPATScope("signing-keys.write")).Post("/rotate", s.forceRotateSigningKey)
		})
		api.Get("/branding/{slug}/logo", s.getBrandingLogo)
		api.Route("/tenants", func(rt chi.Router) {
			rt.Use(auth.RequireInstanceAdmin)
			rt.Get("/", s.listTenants)
			rt.Post("/", s.createTenant)
			rt.Patch("/{id}/branding", s.patchTenantBranding)
			rt.Put("/{id}/logo", s.putTenantLogo)
			rt.Delete("/{id}/logo", s.deleteTenantLogo)
			rt.Get("/{id}", s.getTenant)
			rt.Put("/{id}", s.updateTenant)
			rt.Post("/{id}/suspend", s.suspendTenant)
			rt.Post("/{id}/resume", s.resumeTenant)
			rt.Post("/{id}/delete", s.scheduleTenantDeletion)
			rt.Post("/{id}/delete/cancel", s.cancelTenantDeletion)
			rt.Delete("/{id}", s.deleteTenant)
		})
		api.Route("/projects", func(rt chi.Router) {
			rt.With(auth.RequireTenantRoleOrPATScope("projects.read")).Get("/", s.listProjects)
			rt.With(auth.RequireTenantRoleOrPATScope("projects.write")).Post("/", s.createProject)
			rt.With(auth.RequireTenantRoleOrPATScope("projects.write")).Put("/{id}", s.updateProject)
			rt.With(auth.RequireTenantRoleOrPATScope("projects.write")).Post("/{id}/rotate-secret", s.rotateProjectSecret)
			rt.With(auth.RequireTenantRoleOrPATScope("projects.write")).Delete("/{id}", s.deleteProject)
		})
		api.Route("/users", func(rt chi.Router) {
			rt.Get("/me", s.getUserMe)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Patch("/me", s.patchUserMe)
			rt.With(auth.RequireTenantRoleOrPATScope("users.read")).Get("/", s.listUsers)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Post("/", s.createUser)
			rt.With(auth.RequireTenantRoleOrPATScope("users.read")).Get("/{id}", s.getUserDetail)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Post("/{id}/reset-password", s.resetUserPassword)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Post("/{id}/reinvite", s.reinviteUser)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Post("/{id}/disable-mfa", s.disableUserMFA)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Post("/{id}/enroll-factor", s.inviteUserFactorEnrollment)
			rt.With(auth.RequireTenantRoleOrPATScope("users.read")).Get("/{id}/export", s.exportUserDSR)
			rt.With(auth.RequireTenantRoleOrPATScope("users.write")).Delete("/{id}", s.deleteUserDSR)
		})
		api.Route("/pats", func(rt chi.Router) {
			rt.With(auth.RequireTenantRoleOrPATScope("pats.read")).Get("/", s.listPATs)
			rt.With(auth.RequireTenantRoleOrPATScope("pats.write")).Post("/", s.createPAT)
			rt.With(auth.RequireTenantRoleOrPATScope("pats.write")).Delete("/{id}", s.revokePAT)
		})
	})
	r.NotFound(s.dashboardSPA)
	return r
}

func (s *Server) dashboardSPA(w http.ResponseWriter, r *http.Request) {
	if s.shouldGateRoute(r) {
		path := r.URL.Path
		needsSetup := !s.hasInstanceAdmin(r.Context())
		if strings.HasPrefix(path, "/setup/") {
			// /setup/<token>/* is the wizard. Allow it only while no instance
			// admin exists AND a "live" token covers this URL. Once the
			// ceremony creates the admin and consumes the token, we redirect
			// away so a refresh doesn't sit on a stale wizard page.
			if !needsSetup || !s.hasLiveSetupToken(r.Context()) {
				s.redirectAfterSetup(w, r)
				return
			}
		} else if needsSetup {
			// Fresh install on any other route: send to setup instructions.
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		} else if !s.hasValidSession(r) {
			// Onboarded but no session: send them to the instance-admin sign-in.
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	}
	if s.DevOpenAPI && s.DashboardDev != "" {
		devURL, err := url.Parse(s.DashboardDev)
		if err == nil {
			httputil.NewSingleHostReverseProxy(devURL).ServeHTTP(w, r)
			return
		}
	}
	if s.DashboardFS == nil {
		writeError(w, http.StatusNotFound, "dashboard.not_found")
		return
	}
	dist, err := fs.Sub(s.DashboardFS, "dist")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "dashboard.unavailable")
		return
	}
	if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/fonts/") {
		http.FileServer(http.FS(dist)).ServeHTTP(w, r)
		return
	}
	r.URL.Path = "/index.html"
	http.FileServer(http.FS(dist)).ServeHTTP(w, r)
}

// shouldGateRoute reports whether dashboardSPA should run the onboarding/auth
// gates for this request. We only gate top-level navigations — Vite serves
// the SPA's modules and HMR endpoints through this same handler in dev, and
// redirecting a `<script type="module">` fetch to /login produces an HTML
// response that the browser rejects with a MIME-type error.
func (s *Server) shouldGateRoute(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	// Only gate document navigations. Sub-resource fetches (modules,
	// stylesheets, JSON, fonts) advertise their type via Sec-Fetch-Dest, or
	// fall back to an Accept header that doesn't include text/html.
	if dest := r.Header.Get("Sec-Fetch-Dest"); dest != "" && dest != "document" {
		return false
	}
	if accept := r.Header.Get("Accept"); accept != "" && !strings.Contains(accept, "text/html") {
		return false
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/fonts/") {
		return false
	}
	// Design QA / error pages should remain reachable without auth so the team
	// can iterate on visuals without a live session.
	if strings.HasPrefix(path, "/__cypra/") {
		return false
	}
	return true
}

// hasInstanceAdmin reports whether the install has at least one provisioned
// instance admin. We treat the absence of admins (rather than
// bootstrap.IsFirstBoot) as the "needs setup" signal because IsFirstBoot
// flips to false the moment a setup token is minted, which is *during* setup.
func (s *Server) hasInstanceAdmin(ctx context.Context) bool {
	var exists bool
	err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM instance_admins WHERE disabled_at IS NULL)`).Scan(&exists)
	if err != nil {
		// On error, default to "admin exists" so we don't expose setup pages
		// to the world during a transient DB failure.
		return true
	}
	return exists
}

// hasLiveSetupToken reports whether at least one bootstrap token is currently
// usable. Used to gate /setup/<token>/* — once setupComplete has consumed the
// token, this becomes false and we redirect users out of the wizard.
func (s *Server) hasLiveSetupToken(ctx context.Context) bool {
	var exists bool
	err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM bootstrap_tokens
		WHERE consumed_at IS NULL AND revoked_at IS NULL AND expires_at > now()
	)`).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// redirectAfterSetup sends the user wherever they belong once they're no
// longer eligible to be on a /setup page: /dashboard if they have a session
// (the typical state immediately after completing setup), otherwise /login.
func (s *Server) redirectAfterSetup(w http.ResponseWriter, r *http.Request) {
	if s.hasValidSession(r) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

// hasValidSession returns true when the request carries a cypra_session cookie
// referring to a live row in either the user sessions table or the instance
// admin sessions table.
func (s *Server) hasValidSession(r *http.Request) bool {
	cookie, err := r.Cookie("cypra_session")
	if err != nil {
		return false
	}
	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return false
	}
	var exists bool
	err = s.DB.QueryRowContext(r.Context(), `SELECT EXISTS (
		SELECT 1 FROM instance_admin_sessions
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > now()
	) OR EXISTS (
		SELECT 1 FROM sessions
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > now()
	)`, sessionID).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if s.DB == nil || s.DB.PingContext(r.Context()) != nil || !s.KEKLoaded {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	pending, err := migrate.Pending(r.Context(), s.DB, "")
	if err != nil || pending || s.StorageReady(r.Context()) != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": s.Version, "commit": s.Commit})
}

func (s *Server) openapi(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"openapi": "3.1.0", "info": map[string]string{"title": "Cypra API", "version": s.Version}, "paths": map[string]any{"/api/v1/tenants": map[string]any{}, "/api/v1/projects": map[string]any{}, "/api/v1/users": map[string]any{}, "/api/v1/version": map[string]any{}}})
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", reqID)
		fields := &logging.RequestFields{RequestID: reqID}
		ctx := logging.ContextWithRequestFields(context.WithValue(r.Context(), requestIDKey{}, reqID), fields)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) traceRequests(next http.Handler) http.Handler {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return next
	}
	traced := otelhttp.NewHandler(next, "cypra.http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server-Timing", "otel;desc=\"otlp-http enabled\"")
		traced.ServeHTTP(w, r)
	})
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		logged := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(logged, r)
		fields := logging.RequestFieldsFromContext(r.Context())
		s.Logger.Info("http request", slog.String("request_id", fields.RequestID), slog.String("tenant_id", fields.TenantID), slog.String("actor_id", fields.ActorID), slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int("status", logged.status), slog.Duration("duration", time.Since(started)))
	})
}

func (s *Server) tenantResolver(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
			if r.Header.Get("X-Cypra-Trusted-Proxy") == "true" || isLoopbackRequest(r) {
				host = forwarded
			}
		}
		host = strings.Split(host, ":")[0]
		installHost := strings.Split(s.installHost, ":")[0]
		if host == installHost {
			if slug := tenantSlugFromInstallDashboardRequest(r); slug != "" {
				s.withResolvedTenant(w, r, next, slug)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		suffix := "." + installHost
		if !strings.HasSuffix(host, suffix) {
			writeError(w, http.StatusNotFound, "tenant.not_found")
			return
		}
		s.withResolvedTenant(w, r, next, strings.TrimSuffix(host, suffix))
	})
}

func (s *Server) withResolvedTenant(w http.ResponseWriter, r *http.Request, next http.Handler, slug string) {
	var tenantID uuid.UUID
	if err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM tenants WHERE slug = $1 AND deleted_at IS NULL`, slug).Scan(&tenantID); err != nil {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	ctx := context.WithValue(r.Context(), tenantKey{}, Tenant{ID: tenantID, Slug: slug})
	logging.SetTenantID(ctx, tenantID.String())
	next.ServeHTTP(w, r.WithContext(db.ContextWithTenant(ctx, tenantID)))
}

func tenantSlugFromInstallDashboardRequest(r *http.Request) string {
	if !tenantScopedAPIPath(r.URL.Path) {
		return ""
	}
	if slug := strings.TrimSpace(r.Header.Get("X-Cypra-Tenant-Slug")); slug != "" {
		return slug
	}
	referer := r.Header.Get("Referer")
	if referer == "" {
		return ""
	}
	parsed, err := url.Parse(referer)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "dashboard" && parts[1] == "tenants" {
		return parts[2]
	}
	return ""
}

func tenantScopedAPIPath(path string) bool {
	for _, prefix := range []string{
		"/api/v1/admin",
		"/api/v1/audit",
		"/api/v1/auth-providers",
		"/api/v1/pats",
		"/api/v1/projects",
		"/api/v1/provider-config",
		"/api/v1/signing-keys",
		"/api/v1/users",
	} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

// isLoopbackRequest reports whether r.RemoteAddr is a loopback address.
// Used by tenantResolver to trust X-Forwarded-Host from local-only dev
// proxies (e.g. portless) without requiring the production X-Cypra-Trusted-Proxy
// header. Production deployments behind a network proxy (Caddy, ingress) keep
// the explicit trust marker.
func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

type requestIDKey struct{}

type tenantKey struct{}

type Tenant struct {
	ID   uuid.UUID
	Slug string
}

func TenantFromContext(ctx context.Context) (Tenant, bool) {
	tenant, ok := ctx.Value(tenantKey{}).(Tenant)
	return tenant, ok
}

type tenantPayload struct {
	Slug     string         `json:"slug"`
	Name     string         `json:"name"`
	Branding datatypes.JSON `json:"branding"`
	Settings datatypes.JSON `json:"settings"`
}

type projectPayload struct {
	Slug                    string   `json:"slug"`
	Name                    string   `json:"name"`
	RedirectURIs            []string `json:"redirect_uris"`
	AllowedScopes           []string `json:"allowed_scopes"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

type userPayload struct {
	Email    string         `json:"email"`
	Metadata datatypes.JSON `json:"metadata"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

var reservedSlugs = map[string]bool{"www": true, "api": true, "admin": true, "dashboard": true, "oidc": true, "login": true, "signup": true, "setup": true, "health": true, "healthz": true, "readyz": true, "metrics": true, "well-known": true, "docs": true, "assets": true, "auth": true, "id": true, "me": true, "root": true, "public": true, "static": true, "_": true}

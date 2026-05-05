// Package httpserver wires the Cypra HTTP surface.
package httpserver

//revive:disable:exported

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/auth"
	"github.com/watzon/cypra/internal/db"
	"github.com/watzon/cypra/internal/migrate"
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
	return &Server{Options: opts, installHost: base.Host, audit: audit.NewWriter(opts.DB)}, nil
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(s.requestID)
	r.Use(s.logRequests)
	r.Get("/healthz", s.healthz)
	r.Get("/readyz", s.readyz)
	r.Get("/metrics", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("# cypra metrics\n")) })
	r.Route("/api/v1", func(api chi.Router) {
		api.Use(s.tenantResolver)
		api.Use(s.rlsSetter)
		api.Use(auth.Middleware)
		api.Get("/version", s.version)
		if s.DevOpenAPI {
			api.Get("/openapi.json", s.openapi)
		}
		api.Get("/audit/export", audit.ExportHandler(s.DB))
		api.Route("/tenants", func(rt chi.Router) {
			rt.Use(auth.RequireInstanceAdmin)
			rt.Get("/", s.listTenants)
			rt.Post("/", s.createTenant)
			rt.Get("/{id}", s.getTenant)
			rt.Put("/{id}", s.updateTenant)
			rt.Delete("/{id}", s.deleteTenant)
		})
		api.Route("/projects", func(rt chi.Router) {
			rt.Use(auth.RequireTenantRole)
			rt.Get("/", s.listProjects)
			rt.Post("/", s.createProject)
		})
		api.Route("/users", func(rt chi.Router) {
			rt.Use(auth.RequireTenantRole)
			rt.Get("/", s.listUsers)
			rt.Post("/", s.createUser)
		})
	})
	return r
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
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, reqID)))
	})
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		s.Logger.Info("http request", slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Duration("duration", time.Since(started)))
	})
}

func (s *Server) tenantResolver(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" && r.Header.Get("X-Cypra-Trusted-Proxy") == "true" {
			host = forwarded
		}
		host = strings.Split(host, ":")[0]
		installHost := strings.Split(s.installHost, ":")[0]
		if host == installHost {
			next.ServeHTTP(w, r)
			return
		}
		suffix := "." + installHost
		if !strings.HasSuffix(host, suffix) {
			writeError(w, http.StatusNotFound, "tenant.not_found")
			return
		}
		slug := strings.TrimSuffix(host, suffix)
		var tenantID uuid.UUID
		if err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM tenants WHERE slug = $1 AND deleted_at IS NULL`, slug).Scan(&tenantID); err != nil {
			writeError(w, http.StatusNotFound, "tenant.not_found")
			return
		}
		ctx := context.WithValue(r.Context(), tenantKey{}, Tenant{ID: tenantID, Slug: slug})
		next.ServeHTTP(w, r.WithContext(db.ContextWithTenant(ctx, tenantID)))
	})
}

func (s *Server) rlsSetter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
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
	Slug string `json:"slug"`
	Name string `json:"name"`
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

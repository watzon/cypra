package httpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const hostedSessionTTL = 24 * time.Hour

func (s *Server) establishUserSession(w http.ResponseWriter, r *http.Request, tenantID, userID uuid.UUID) (uuid.UUID, error) {
	sessionID := uuid.New()
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at, ip, user_agent) VALUES ($1, $2, 'user', $3, $4, $5, $6)`, sessionID, userID, tenantID, time.Now().UTC().Add(hostedSessionTTL), s.clientIP(r), r.UserAgent()); err != nil {
		return uuid.Nil, fmt.Errorf("create hosted session: %w", err)
	}
	s.setSessionCookie(w, r, "cypra_session", sessionID.String(), hostedSessionTTL)
	return sessionID, nil
}

func (s *Server) establishInstanceAdminSession(w http.ResponseWriter, r *http.Request, adminID uuid.UUID) (uuid.UUID, error) {
	sessionID := uuid.New()
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO instance_admin_sessions (id, instance_admin_id, expires_at, ip, user_agent) VALUES ($1, $2, $3, $4, $5)`, sessionID, adminID, time.Now().UTC().Add(hostedSessionTTL), s.clientIP(r), r.UserAgent()); err != nil {
		return uuid.Nil, fmt.Errorf("create instance admin session: %w", err)
	}
	s.setSessionCookie(w, r, "cypra_session", sessionID.String(), hostedSessionTTL)
	return sessionID, nil
}

func (s *Server) userBelongsToTenant(ctx context.Context, tenantID, userID uuid.UUID) bool {
	var exists bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)`, tenantID, userID).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func (s *Server) currentUserSession(r *http.Request, tenantID uuid.UUID) (uuid.UUID, bool) {
	cookie, err := r.Cookie("cypra_session")
	if err != nil {
		return uuid.Nil, false
	}
	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return uuid.Nil, false
	}
	var userID uuid.UUID
	err = s.DB.QueryRowContext(r.Context(), `SELECT subject_id FROM sessions WHERE id = $1 AND tenant_id = $2 AND subject_kind = 'user' AND revoked_at IS NULL AND expires_at > now()`, sessionID, tenantID).Scan(&userID)
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  time.Now().UTC().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.requestIsHTTPS(r),
	})
}

func (s *Server) clientIP(r *http.Request) any {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" && s.TrustedProxyHeaders {
		if parsed := net.ParseIP(stringsBeforeComma(forwarded)); parsed != nil {
			return parsed.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if parsed := net.ParseIP(host); parsed != nil {
		return parsed.String()
	}
	return nil
}

func stringsBeforeComma(value string) string {
	for i, char := range value {
		if char == ',' {
			return value[:i]
		}
	}
	return value
}

package httpserver

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/ratelimit"
)

var rateLimitMetrics = struct {
	sync.Mutex
	counts map[string]uint64
}{counts: map[string]uint64{}}

func (s *Server) allowRate(w http.ResponseWriter, r *http.Request, tenantID *uuid.UUID, scope string, key string, limit float64, window time.Duration) bool {
	ok, err := (ratelimit.Limiter{DB: s.DB}).Allow(r.Context(), tenantID, scope, key, limit, window)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "rate_limit.error")
		return false
	}
	if !ok {
		recordRateLimitExceeded(scope)
		recordAuthAttempt(authKindFromScope(scope), "rate_limited")
		writeError(w, http.StatusTooManyRequests, "auth.rate_limited")
		return false
	}
	return true
}

func (s *Server) allowHostedRate(w http.ResponseWriter, r *http.Request, tenantID *uuid.UUID, scope string, key string, limit float64, window time.Duration) bool {
	ok, err := (ratelimit.Limiter{DB: s.DB}).Allow(r.Context(), tenantID, scope, key, limit, window)
	if err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return false
	}
	if !ok {
		recordRateLimitExceeded(scope)
		recordAuthAttempt(authKindFromScope(scope), "rate_limited")
		writeHTMXError(w, http.StatusTooManyRequests, "Too many attempts. Try again in "+rateLimitCountdown(window)+".")
		return false
	}
	return true
}

func rateLimitCountdown(window time.Duration) string {
	if window >= time.Hour {
		return "1 hour"
	}
	minutes := int(window.Minutes())
	if minutes <= 1 {
		return "1 minute"
	}
	return strconv.Itoa(minutes) + " minutes"
}

func (s *Server) allowOIDCRate(w http.ResponseWriter, r *http.Request, tenantID *uuid.UUID, scope string, key string, limit float64, window time.Duration) bool {
	ok, err := (ratelimit.Limiter{DB: s.DB}).Allow(r.Context(), tenantID, scope, key, limit, window)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, "server_error")
		return false
	}
	if !ok {
		recordRateLimitExceeded(scope)
		recordAuthAttempt(authKindFromScope(scope), "rate_limited")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "auth.rate_limited"})
		return false
	}
	return true
}

func recordRateLimitExceeded(scope string) {
	rateLimitMetrics.Lock()
	defer rateLimitMetrics.Unlock()
	rateLimitMetrics.counts[scope]++
}

func rateLimitExceededMetrics() map[string]uint64 {
	rateLimitMetrics.Lock()
	defer rateLimitMetrics.Unlock()
	snapshot := make(map[string]uint64, len(rateLimitMetrics.counts))
	for scope, count := range rateLimitMetrics.counts {
		snapshot[scope] = count
	}
	return snapshot
}

func clientRateKey(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" && (r.Header.Get("X-Cypra-Trusted-Proxy") == "true" || isLoopbackRequest(r)) {
		if first, _, ok := strings.Cut(forwarded, ","); ok {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(forwarded)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

package httpserver

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var runtimeMetricCounters sync.Map

type runtimeMetricKey struct {
	Name   string
	Labels string
}

func incrementRuntimeMetric(name, labels string) {
	key := runtimeMetricKey{Name: name, Labels: labels}
	value, _ := runtimeMetricCounters.LoadOrStore(key, &atomic.Uint64{})
	value.(*atomic.Uint64).Add(1)
}

func runtimeMetricLines(name string) []string {
	lines := []string{}
	runtimeMetricCounters.Range(func(key, value any) bool {
		metricKey := key.(runtimeMetricKey)
		if metricKey.Name != name {
			return true
		}
		labels := metricKey.Labels
		if labels != "" {
			labels = "{" + labels + "}"
		}
		lines = append(lines, fmt.Sprintf("%s%s %d", name, labels, value.(*atomic.Uint64).Load()))
		return true
	})
	sort.Strings(lines)
	return lines
}

func recordAuthAttempt(kind, outcome string) {
	incrementRuntimeMetric("cypra_auth_attempts_total", fmt.Sprintf("kind=\"%s\",outcome=\"%s\"", metricLabel(kind), metricLabel(outcome)))
}

func recordOIDCTokenIssued(grant string) {
	incrementRuntimeMetric("cypra_oidc_token_issued_total", fmt.Sprintf("grant=\"%s\"", metricLabel(grant)))
}

func recordStorageOperation(backend, op string, _ time.Duration) {
	incrementRuntimeMetric("cypra_storage_operation_duration_seconds_count", fmt.Sprintf("backend=\"%s\",op=\"%s\"", metricLabel(backend), metricLabel(op)))
}

func authKindFromScope(scope string) string {
	switch scope {
	case "signup:ip", "login:ip", "login:account", "password_reset:account":
		return "password"
	case "magic_link:account":
		return "magic_link"
	case "invite_redeem:token":
		return "invite"
	case "oidc_token:client":
		return "oidc_token"
	default:
		return "unknown"
	}
}

func authKindFromPath(path string) string {
	switch path {
	case "/api/v1/auth/password/signin", "/api/v1/auth/password/signup", "/api/v1/auth/password/reset":
		return "password"
	case "/api/v1/auth/magic-link/issue", "/api/v1/auth/magic-link/verify":
		return "magic_link"
	case "/api/v1/auth/passkey/assert", "/api/v1/auth/passkey/register":
		return "passkey"
	case "/api/v1/auth/google/start", "/api/v1/auth/google/callback":
		return "google"
	case "/api/v1/auth/totp/verify", "/api/v1/auth/totp/enroll":
		return "totp"
	case "/api/v1/auth/webauthn-2fa/verify", "/api/v1/auth/webauthn-2fa/enroll":
		return "webauthn2fa"
	default:
		return "unknown"
	}
}

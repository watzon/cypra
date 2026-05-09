// Package upstream defines the Provider abstraction used by the social-SSO
// system and exposes a process-wide registry of registered providers.
package upstream

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrUnknownProvider = errors.New("upstream: unknown provider")

type Display struct {
	Label   string
	IconURL string
	Hint    string
}

type Credentials struct {
	ClientID     string
	ClientSecret string
	StateSecret  []byte
	RedirectURI  string
}

type AuthParams struct {
	TenantID  uuid.UUID
	ReturnURL string
	Scopes    []string
	Extras    map[string]string
}

type Identity struct {
	Subject string
	Email   string
	Name    string
	Raw     map[string]any
}

type AuthCodeRequest struct {
	State string
	Nonce string
	URL   string
}

type ExchangeParams struct {
	Code        string
	State       string
	Nonce       string
	RedirectURI string
	Now         time.Time
}

type Provider interface {
	Kind() string
	Display() Display
	DefaultScopes() []string
	AuthCodeURL(ctx context.Context, creds Credentials, params AuthParams) (AuthCodeRequest, error)
	Exchange(ctx context.Context, creds Credentials, params ExchangeParams) (Identity, error)
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Provider{}
)

// Register installs a provider in the process-wide registry. Safe to call from
// package init() functions; later calls with the same kind override.
func Register(p Provider) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[p.Kind()] = p
}

func Resolve(kind string) (Provider, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, kind)
	}
	return p, nil
}

func All() []Provider {
	registryMu.RLock()
	defer registryMu.RUnlock()
	kinds := make([]string, 0, len(registry))
	for k := range registry {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	out := make([]Provider, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, registry[k])
	}
	return out
}

func Kinds() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

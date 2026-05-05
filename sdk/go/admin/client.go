// Package admin is a typed client for the Cypra admin REST API.
package admin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	cypra "github.com/watzon/cypra/sdk/go"
)

// Client calls the Cypra admin REST API.
type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	PAT        string
}

// Tenant mirrors the tenant REST shape.
type Tenant struct {
	ID       string                 `json:"id"`
	Slug     string                 `json:"slug"`
	Name     string                 `json:"name"`
	Branding map[string]any         `json:"branding,omitempty"`
	Settings map[string]any         `json:"settings,omitempty"`
	Counts   map[string]int         `json:"counts,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Project mirrors the project REST shape.
type Project struct {
	ID                      string   `json:"id"`
	Slug                    string   `json:"slug"`
	Name                    string   `json:"name"`
	IssuerURL               string   `json:"issuer_url,omitempty"`
	ClientID                string   `json:"client_id,omitempty"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	AllowedScopes           []string `json:"allowed_scopes,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

// User mirrors the user REST shape.
type User struct {
	ID              string         `json:"id"`
	Email           string         `json:"email"`
	Sub             string         `json:"sub,omitempty"`
	State           string         `json:"state,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	EnrolledMethods []string       `json:"enrolled_methods,omitempty"`
}

// Member mirrors a tenant membership row.
type Member struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// OIDCClient mirrors an OIDC client row.
type OIDCClient struct {
	ID                      string   `json:"id"`
	ProjectID               string   `json:"project_id"`
	ClientID                string   `json:"client_id"`
	RedirectURIs            []string `json:"redirect_uris"`
	AllowedScopes           []string `json:"allowed_scopes"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// SigningKey mirrors an OIDC signing key row.
type SigningKey struct {
	ID          string `json:"id"`
	KID         string `json:"kid"`
	State       string `json:"state"`
	ActivatedAt string `json:"activated_at,omitempty"`
	RetiresAt   string `json:"retires_at,omitempty"`
	SunsetUntil string `json:"sunset_until,omitempty"`
}

// AuditEntry mirrors an audit export/list row.
type AuditEntry struct {
	ID           string         `json:"id"`
	ActorKind    string         `json:"actor_kind"`
	Action       string         `json:"action"`
	ResourceKind string         `json:"resource_kind"`
	ResourceID   string         `json:"resource_id,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// InstanceAdmin mirrors an instance-admin row.
type InstanceAdmin struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at,omitempty"`
}

// Invite represents an invitation response.
type Invite struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

// New constructs an admin client.
func New(baseURL, pat string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{BaseURL: parsed, HTTPClient: http.DefaultClient, PAT: pat}, nil
}

func (c *Client) ListTenants(ctx context.Context) ([]Tenant, error) {
	var tenants []Tenant
	return tenants, c.do(ctx, http.MethodGet, "/api/v1/tenants/", nil, &tenants, true)
}

func (c *Client) CreateTenant(ctx context.Context, slug, name string) (Tenant, error) {
	var tenant Tenant
	return tenant, c.do(ctx, http.MethodPost, "/api/v1/tenants/", map[string]string{"slug": slug, "name": name}, &tenant, true)
}

func (c *Client) GetTenant(ctx context.Context, id string) (Tenant, error) {
	var tenant Tenant
	return tenant, c.do(ctx, http.MethodGet, "/api/v1/tenants/"+url.PathEscape(id), nil, &tenant, true)
}

func (c *Client) UpdateTenant(ctx context.Context, id, name string) (Tenant, error) {
	var tenant Tenant
	return tenant, c.do(ctx, http.MethodPut, "/api/v1/tenants/"+url.PathEscape(id), map[string]string{"name": name}, &tenant, true)
}

func (c *Client) DeleteTenant(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/tenants/"+url.PathEscape(id), nil, nil, true)
}

func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	return projects, c.do(ctx, http.MethodGet, "/api/v1/projects/", nil, &projects, false)
}

func (c *Client) CreateProject(ctx context.Context, slug, name string) (Project, error) {
	var project Project
	return project, c.do(ctx, http.MethodPost, "/api/v1/projects/", map[string]string{"slug": slug, "name": name}, &project, false)
}

func (c *Client) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	return users, c.do(ctx, http.MethodGet, "/api/v1/users/", nil, &users, false)
}

func (c *Client) CreateUser(ctx context.Context, email string) (User, error) {
	var user User
	return user, c.do(ctx, http.MethodPost, "/api/v1/users/", map[string]string{"email": email}, &user, false)
}

func (c *Client) ExportUser(ctx context.Context, id string) ([]byte, error) {
	var out []byte
	return out, c.doRaw(ctx, http.MethodGet, "/api/v1/users/"+url.PathEscape(id)+"/export", nil, &out, false)
}

func (c *Client) DeleteUser(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/users/"+url.PathEscape(id), nil, nil, false)
}

func (c *Client) ListMembers(ctx context.Context) ([]Member, error) {
	var members []Member
	return members, c.do(ctx, http.MethodGet, "/api/v1/members/", nil, &members, false)
}

func (c *Client) ListOIDCClients(ctx context.Context) ([]OIDCClient, error) {
	var clients []OIDCClient
	return clients, c.do(ctx, http.MethodGet, "/api/v1/oidc-clients/", nil, &clients, false)
}

func (c *Client) ListSigningKeys(ctx context.Context) ([]SigningKey, error) {
	var keys []SigningKey
	return keys, c.do(ctx, http.MethodGet, "/api/v1/signing-keys/", nil, &keys, false)
}

func (c *Client) ListAuditEntries(ctx context.Context) ([]AuditEntry, error) {
	raw, err := c.ExportAudit(ctx)
	if err != nil {
		return nil, err
	}
	entries := []AuditEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		var entry AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func (c *Client) ExportAudit(ctx context.Context) ([]byte, error) {
	var out []byte
	return out, c.doRaw(ctx, http.MethodGet, "/api/v1/audit/export", nil, &out, false)
}

func (c *Client) ListInstanceAdmins(ctx context.Context) ([]InstanceAdmin, error) {
	var admins []InstanceAdmin
	return admins, c.do(ctx, http.MethodGet, "/api/v1/instance/admins", nil, &admins, true)
}

func (c *Client) InviteInstanceAdmin(ctx context.Context, email, redirectURL string) (Invite, error) {
	var invite Invite
	body := map[string]string{"email": email, "redirect_url": redirectURL}
	return invite, c.do(ctx, http.MethodPost, "/api/v1/instance/invite", body, &invite, true)
}

func (c *Client) DemoteInstanceAdmin(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/instance/admins/"+url.PathEscape(id), nil, nil, true)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, instanceAdmin bool) error {
	return c.doRaw(ctx, method, path, body, out, instanceAdmin)
}

func (c *Client) doRaw(ctx context.Context, method, path string, body any, out any, instanceAdmin bool) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	endpoint := c.BaseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.PAT != "" {
		req.Header.Set("Authorization", "Bearer "+c.PAT)
	}
	if instanceAdmin {
		req.Header.Set("X-Cypra-Instance-Admin", "true")
	} else {
		req.Header.Set("X-Cypra-Tenant-Role", "admin")
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return decodeError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if raw, ok := out.(*[]byte); ok {
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		*raw = payload
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func decodeError(resp *http.Response) error {
	var payload struct {
		Error string `json:"error"`
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "json") {
		_ = json.NewDecoder(resp.Body).Decode(&payload)
	}
	return cypra.ErrorForStatus(resp.StatusCode, payload.Error)
}

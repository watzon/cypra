// Package admin is a typed client for the Cypra admin REST API.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	BaseURL    *url.URL
	HTTPClient *http.Client
	PAT        string
}

type Tenant struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type Project struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func New(baseURL, pat string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{BaseURL: parsed, HTTPClient: http.DefaultClient, PAT: pat}, nil
}

func (c *Client) CreateTenant(ctx context.Context, slug, name string) (Tenant, error) {
	var tenant Tenant
	err := c.do(ctx, http.MethodPost, "/api/v1/tenants/", map[string]string{"slug": slug, "name": name}, &tenant, true)
	return tenant, err
}

func (c *Client) CreateProject(ctx context.Context, slug, name string) (Project, error) {
	var project Project
	err := c.do(ctx, http.MethodPost, "/api/v1/projects/", map[string]string{"slug": slug, "name": name}, &project, false)
	return project, err
}

func (c *Client) CreateUser(ctx context.Context, email string) (User, error) {
	var user User
	err := c.do(ctx, http.MethodPost, "/api/v1/users/", map[string]string{"email": email}, &user, false)
	return user, err
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, instanceAdmin bool) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	endpoint := c.BaseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("cypra admin api returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

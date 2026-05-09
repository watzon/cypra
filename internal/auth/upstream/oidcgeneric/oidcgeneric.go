// Package oidcgeneric implements a single-instance OIDC client used by
// Enterprise SSO connections. Unlike the social providers it is not
// auto-registered: each oidc_connections row instantiates its own Provider
// pointed at a particular issuer URL.
package oidcgeneric

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/watzon/cypra/internal/auth/upstream"
)

type Provider struct {
	Slug          string
	DisplayName   string
	Issuer        string
	AuthURL       string
	TokenURL      string
	UserInfoURL   string
	HTTPClient    *http.Client
	DefaultScopes []string
}

func (p Provider) Kind() string { return "oidc:" + p.Slug }
func (p Provider) Display() upstream.Display {
	return upstream.Display{Label: p.DisplayName, Hint: p.Issuer}
}

func (p Provider) GetDefaultScopes() []string {
	if len(p.DefaultScopes) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return p.DefaultScopes
}

func (p Provider) AuthCodeURL(ctx context.Context, creds upstream.Credentials, params upstream.AuthParams) (upstream.AuthCodeRequest, error) {
	if err := p.ensureEndpoints(ctx); err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	nonce, err := upstream.RandomNonce()
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	state := upstream.State{TenantID: params.TenantID, Provider: "oidc", Slug: p.Slug, ReturnURL: params.ReturnURL, Nonce: nonce, ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix()}
	signed, err := upstream.SignState(creds.StateSecret, state)
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	scopes := params.Scopes
	if len(scopes) == 0 {
		scopes = p.GetDefaultScopes()
	}
	values := url.Values{}
	values.Set("client_id", creds.ClientID)
	values.Set("redirect_uri", creds.RedirectURI)
	values.Set("response_type", "code")
	values.Set("scope", strings.Join(scopes, " "))
	values.Set("state", signed)
	values.Set("nonce", nonce)
	return upstream.AuthCodeRequest{State: signed, Nonce: nonce, URL: p.AuthURL + "?" + values.Encode()}, nil
}

func (p Provider) Exchange(ctx context.Context, creds upstream.Credentials, params upstream.ExchangeParams) (upstream.Identity, error) {
	if err := p.ensureEndpoints(ctx); err != nil {
		return upstream.Identity{}, err
	}
	form := url.Values{}
	form.Set("code", params.Code)
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("redirect_uri", creds.RedirectURI)
	form.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return upstream.Identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return upstream.Identity{}, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return upstream.Identity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return upstream.Identity{}, fmt.Errorf("oidc %s token exchange %d: %s", p.Slug, resp.StatusCode, string(body))
	}
	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return upstream.Identity{}, err
	}
	if tokenResp.IDToken == "" {
		return upstream.Identity{}, errors.New("oidc: missing id_token")
	}
	if err := upstream.ValidateIDTokenNonce(tokenResp.IDToken, params.Nonce); err != nil {
		return upstream.Identity{}, err
	}
	claims, raw, err := upstream.ParseIDTokenClaims(tokenResp.IDToken)
	if err != nil {
		return upstream.Identity{}, err
	}
	if strings.TrimSpace(claims.Email) == "" {
		return upstream.Identity{}, errors.New("oidc: missing email claim")
	}
	return upstream.Identity{Subject: claims.Subject, Email: claims.Email, Name: claims.Name, Raw: raw}, nil
}

func (p *Provider) ensureEndpoints(ctx context.Context) error {
	if p.AuthURL != "" && p.TokenURL != "" {
		return nil
	}
	if p.Issuer == "" {
		return errors.New("oidc: issuer URL required")
	}
	doc, err := DiscoverEndpoints(ctx, p.HTTPClient, p.Issuer)
	if err != nil {
		return err
	}
	if p.AuthURL == "" {
		p.AuthURL = doc.AuthorizationEndpoint
	}
	if p.TokenURL == "" {
		p.TokenURL = doc.TokenEndpoint
	}
	if p.UserInfoURL == "" {
		p.UserInfoURL = doc.UserInfoEndpoint
	}
	return nil
}

type Discovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
	JWKsURI               string `json:"jwks_uri"`
}

func DiscoverEndpoints(ctx context.Context, client *http.Client, issuer string) (Discovery, error) {
	if client == nil {
		client = http.DefaultClient
	}
	url := strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Discovery{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Discovery{}, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return Discovery{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Discovery{}, fmt.Errorf("oidc discovery %d: %s", resp.StatusCode, string(body))
	}
	var doc Discovery
	if err := json.Unmarshal(body, &doc); err != nil {
		return Discovery{}, err
	}
	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" {
		return Discovery{}, errors.New("oidc discovery: missing endpoints")
	}
	return doc, nil
}

// Package github implements the GitHub social provider.
package github

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

const (
	defaultAuthURL    = "https://github.com/login/oauth/authorize"
	defaultTokenURL   = "https://github.com/login/oauth/access_token"
	defaultUserURL    = "https://api.github.com/user"
	defaultEmailsURL  = "https://api.github.com/user/emails"
)

type Provider struct {
	AuthURL    string
	TokenURL   string
	UserURL    string
	EmailsURL  string
	HTTPClient *http.Client
}

func init() {
	upstream.Register(Provider{})
}

func (Provider) Kind() string                { return "github" }
func (Provider) Display() upstream.Display   { return upstream.Display{Label: "GitHub", IconURL: "/static/social/github.svg"} }
func (Provider) DefaultScopes() []string     { return []string{"read:user", "user:email"} }

func (p Provider) AuthCodeURL(_ context.Context, creds upstream.Credentials, params upstream.AuthParams) (upstream.AuthCodeRequest, error) {
	nonce, err := upstream.RandomNonce()
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	state := upstream.State{TenantID: params.TenantID, Provider: "github", ReturnURL: params.ReturnURL, Nonce: nonce, ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix()}
	signed, err := upstream.SignState(creds.StateSecret, state)
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	scopes := params.Scopes
	if len(scopes) == 0 {
		scopes = p.DefaultScopes()
	}
	values := url.Values{}
	values.Set("client_id", creds.ClientID)
	values.Set("redirect_uri", creds.RedirectURI)
	values.Set("scope", strings.Join(scopes, " "))
	values.Set("state", signed)
	auth := p.AuthURL
	if auth == "" {
		auth = defaultAuthURL
	}
	return upstream.AuthCodeRequest{State: signed, Nonce: nonce, URL: auth + "?" + values.Encode()}, nil
}

func (p Provider) Exchange(ctx context.Context, creds upstream.Credentials, params upstream.ExchangeParams) (upstream.Identity, error) {
	tokenURL := p.TokenURL
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}
	form := url.Values{}
	form.Set("code", params.Code)
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("redirect_uri", creds.RedirectURI)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
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
		return upstream.Identity{}, fmt.Errorf("github token exchange %d: %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return upstream.Identity{}, err
	}
	if tokenResp.AccessToken == "" {
		return upstream.Identity{}, errors.New("github: missing access_token")
	}
	user, err := p.fetchUser(ctx, client, tokenResp.AccessToken)
	if err != nil {
		return upstream.Identity{}, err
	}
	if user.Email == "" {
		emails, err := p.fetchEmails(ctx, client, tokenResp.AccessToken)
		if err != nil {
			return upstream.Identity{}, err
		}
		for _, e := range emails {
			if e.Primary && e.Verified && e.Email != "" {
				user.Email = e.Email
				break
			}
		}
	}
	if strings.TrimSpace(user.Email) == "" {
		return upstream.Identity{}, errors.New("github: no verified email available")
	}
	raw := map[string]any{"login": user.Login, "id": user.ID, "name": user.Name}
	return upstream.Identity{Subject: fmt.Sprintf("%d", user.ID), Email: user.Email, Name: user.Name, Raw: raw}, nil
}

type ghUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ghEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (p Provider) fetchUser(ctx context.Context, client *http.Client, token string) (ghUser, error) {
	endpoint := p.UserURL
	if endpoint == "" {
		endpoint = defaultUserURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ghUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return ghUser{}, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return ghUser{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return ghUser{}, fmt.Errorf("github user %d: %s", resp.StatusCode, string(body))
	}
	var u ghUser
	if err := json.Unmarshal(body, &u); err != nil {
		return ghUser{}, err
	}
	return u, nil
}

func (p Provider) fetchEmails(ctx context.Context, client *http.Client, token string) ([]ghEmail, error) {
	endpoint := p.EmailsURL
	if endpoint == "" {
		endpoint = defaultEmailsURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github emails %d: %s", resp.StatusCode, string(body))
	}
	var emails []ghEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, err
	}
	return emails, nil
}

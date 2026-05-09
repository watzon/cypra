// Package discord implements the Discord social provider.
package discord

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
	defaultAuthURL  = "https://discord.com/oauth2/authorize"
	defaultTokenURL = "https://discord.com/api/oauth2/token"
	defaultUserURL  = "https://discord.com/api/users/@me"
)

type Provider struct {
	AuthURL    string
	TokenURL   string
	UserURL    string
	HTTPClient *http.Client
}

func init() {
	upstream.Register(Provider{})
}

func (Provider) Kind() string                { return "discord" }
func (Provider) Display() upstream.Display   { return upstream.Display{Label: "Discord", IconURL: "/static/social/discord.svg"} }
func (Provider) DefaultScopes() []string     { return []string{"identify", "email"} }

func (p Provider) AuthCodeURL(_ context.Context, creds upstream.Credentials, params upstream.AuthParams) (upstream.AuthCodeRequest, error) {
	nonce, err := upstream.RandomNonce()
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	state := upstream.State{TenantID: params.TenantID, Provider: "discord", ReturnURL: params.ReturnURL, Nonce: nonce, ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix()}
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
	values.Set("response_type", "code")
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
	form.Set("grant_type", "authorization_code")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return upstream.Identity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
		return upstream.Identity{}, fmt.Errorf("discord token exchange %d: %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return upstream.Identity{}, err
	}
	if tokenResp.AccessToken == "" {
		return upstream.Identity{}, errors.New("discord: missing access_token")
	}
	user, err := p.fetchUser(ctx, client, tokenResp.AccessToken)
	if err != nil {
		return upstream.Identity{}, err
	}
	if !user.Verified || strings.TrimSpace(user.Email) == "" {
		return upstream.Identity{}, errors.New("discord: no verified email available")
	}
	raw := map[string]any{"id": user.ID, "username": user.Username, "global_name": user.GlobalName}
	name := user.GlobalName
	if name == "" {
		name = user.Username
	}
	return upstream.Identity{Subject: user.ID, Email: user.Email, Name: name, Raw: raw}, nil
}

type discordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Email      string `json:"email"`
	Verified   bool   `json:"verified"`
}

func (p Provider) fetchUser(ctx context.Context, client *http.Client, token string) (discordUser, error) {
	endpoint := p.UserURL
	if endpoint == "" {
		endpoint = defaultUserURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return discordUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return discordUser{}, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return discordUser{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return discordUser{}, fmt.Errorf("discord user %d: %s", resp.StatusCode, string(body))
	}
	var u discordUser
	if err := json.Unmarshal(body, &u); err != nil {
		return discordUser{}, err
	}
	return u, nil
}

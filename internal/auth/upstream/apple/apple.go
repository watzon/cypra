// Package apple implements the Sign-in-with-Apple social provider.
//
// Apple requires the OAuth client_secret to be a JWT signed with the
// developer's ES256 private key. For now the admin must pre-render that JWT
// (good for up to 6 months per Apple's policy) and store it as the
// client_secret. In-product JWT generation can layer on later by reading
// team_id, key_id, and private_key_pem from the social_connections.config
// JSONB blob.
package apple

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
	defaultAuthURL  = "https://appleid.apple.com/auth/authorize"
	defaultTokenURL = "https://appleid.apple.com/auth/token" // #nosec G101 -- OAuth token endpoint URL, not a credential.
)

type Provider struct {
	AuthURL    string
	TokenURL   string
	HTTPClient *http.Client
}

func init() {
	upstream.Register(Provider{})
}

func (Provider) Kind() string { return "apple" }
func (Provider) Display() upstream.Display {
	return upstream.Display{Label: "Apple", IconURL: "/static/social/apple.svg", Hint: "Apple requires a pre-rendered client_secret JWT."}
}
func (Provider) DefaultScopes() []string { return []string{"name", "email"} }

func (p Provider) AuthCodeURL(_ context.Context, creds upstream.Credentials, params upstream.AuthParams) (upstream.AuthCodeRequest, error) {
	nonce, err := upstream.RandomNonce()
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	state := upstream.State{TenantID: params.TenantID, Provider: "apple", ReturnURL: params.ReturnURL, Nonce: nonce, ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix()}
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
	values.Set("response_mode", "form_post")
	values.Set("scope", strings.Join(scopes, " "))
	values.Set("state", signed)
	values.Set("nonce", nonce)
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
		return upstream.Identity{}, fmt.Errorf("apple token exchange %d: %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return upstream.Identity{}, err
	}
	if tokenResp.IDToken == "" {
		return upstream.Identity{}, errors.New("apple: missing id_token")
	}
	if err := upstream.ValidateIDTokenNonce(tokenResp.IDToken, params.Nonce); err != nil {
		return upstream.Identity{}, err
	}
	claims, raw, err := upstream.ParseIDTokenClaims(tokenResp.IDToken)
	if err != nil {
		return upstream.Identity{}, err
	}
	if strings.TrimSpace(claims.Email) == "" {
		return upstream.Identity{}, errors.New("apple: missing email claim")
	}
	return upstream.Identity{Subject: claims.Subject, Email: claims.Email, Name: claims.Name, Raw: raw}, nil
}

// Package microsoft implements the Microsoft (Azure AD / Entra) social provider.
package microsoft

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
	defaultAuthURL  = "https://login.microsoftonline.com/common/oauth2/v2.0/authorize"
	defaultTokenURL = "https://login.microsoftonline.com/common/oauth2/v2.0/token" // #nosec G101 -- OAuth token endpoint URL, not a credential.
)

type Provider struct {
	AuthURL    string
	TokenURL   string
	HTTPClient *http.Client
}

func init() {
	upstream.Register(Provider{})
}

func (Provider) Kind() string { return "microsoft" }
func (Provider) Display() upstream.Display {
	return upstream.Display{Label: "Microsoft", IconURL: "/static/social/microsoft.svg"}
}
func (Provider) DefaultScopes() []string { return []string{"openid", "email", "profile"} }

func (p Provider) AuthCodeURL(_ context.Context, creds upstream.Credentials, params upstream.AuthParams) (upstream.AuthCodeRequest, error) {
	nonce, err := upstream.RandomNonce()
	if err != nil {
		return upstream.AuthCodeRequest{}, err
	}
	state := upstream.State{TenantID: params.TenantID, Provider: "microsoft", ReturnURL: params.ReturnURL, Nonce: nonce, ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Unix()}
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
	values.Set("nonce", nonce)
	values.Set("response_mode", "query")
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
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return upstream.Identity{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return upstream.Identity{}, fmt.Errorf("microsoft token exchange %d: %s", resp.StatusCode, string(body))
	}
	var tokenResp struct {
		IDToken string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return upstream.Identity{}, err
	}
	if tokenResp.IDToken == "" {
		return upstream.Identity{}, errors.New("microsoft: missing id_token")
	}
	if err := upstream.ValidateIDTokenNonce(tokenResp.IDToken, params.Nonce); err != nil {
		return upstream.Identity{}, err
	}
	claims, raw, err := upstream.ParseIDTokenClaims(tokenResp.IDToken)
	if err != nil {
		return upstream.Identity{}, err
	}
	email := claims.Email
	if email == "" {
		if pref, ok := raw["preferred_username"].(string); ok {
			email = pref
		}
	}
	if strings.TrimSpace(email) == "" {
		return upstream.Identity{}, errors.New("microsoft: missing email claim")
	}
	return upstream.Identity{Subject: claims.Subject, Email: email, Name: claims.Name, Raw: raw}, nil
}

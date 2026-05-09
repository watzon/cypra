// Package oidc wraps standard OAuth2/OIDC clients for Cypra issuers.
package oidc

import (
	"context"
	"errors"

	coreosoidc "github.com/coreos/go-oidc/v3/oidc"
	cypra "github.com/watzon/cypra/sdk/go"
	"golang.org/x/oauth2"
)

// Client is a thin wrapper around oauth2 and go-oidc configured for Cypra.
type Client struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string

	provider *coreosoidc.Provider
	config   oauth2.Config
}

// New discovers the issuer and returns a configured client.
func New(ctx context.Context, issuer, clientID, clientSecret, redirectURI string, scopes ...string) (*Client, error) {
	provider, err := coreosoidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = []string{coreosoidc.ScopeOpenID, "email", "profile"}
	}
	config := oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}
	return &Client{Issuer: issuer, ClientID: clientID, ClientSecret: clientSecret, RedirectURI: redirectURI, Scopes: scopes, provider: provider, config: config}, nil
}

// AuthCodeURL returns the authorization URL for a state value and optional auth-code options.
func (c *Client) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return c.config.AuthCodeURL(state, opts...)
}

// Exchange exchanges an authorization code for tokens.
func (c *Client) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	token, err := c.config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, oidcError(err)
	}
	return token, nil
}

// RefreshToken refreshes an access token using a refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token, err := c.config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		return nil, oidcError(err)
	}
	return token, nil
}

// UserInfo fetches claims from the Cypra userinfo endpoint.
func (c *Client) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*coreosoidc.UserInfo, error) {
	return c.provider.UserInfo(ctx, tokenSource)
}

// Verify verifies a raw ID token against the issuer and client id.
func (c *Client) Verify(ctx context.Context, rawIDToken string) (*coreosoidc.IDToken, error) {
	verifier := c.provider.Verifier(&coreosoidc.Config{ClientID: c.ClientID})
	return verifier.Verify(ctx, rawIDToken)
}

func oidcError(err error) error {
	var retrieveErr *oauth2.RetrieveError
	if !errors.As(err, &retrieveErr) || retrieveErr.Response == nil {
		return err
	}
	return cypra.ErrorForStatus(retrieveErr.Response.StatusCode, retrieveErr.ErrorCode)
}

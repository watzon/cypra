// Package pat manages personal access tokens.
package pat

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

//revive:disable:exported

var ErrInvalidToken = errors.New("invalid personal access token")

type Service struct{ DB *sql.DB }

type Token struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	UserID     uuid.UUID  `json:"user_id"`
	Name       string     `json:"name"`
	Last4      string     `json:"last4"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

type CreatedToken struct {
	Token
	Plaintext string `json:"token"`
}

func (s Service) Create(ctx context.Context, tenantID, userID uuid.UUID, name string, scopes []string) (CreatedToken, error) {
	plaintext, err := randomToken()
	if err != nil {
		return CreatedToken{}, err
	}
	id := uuid.New()
	var created time.Time
	err = s.DB.QueryRowContext(ctx, `INSERT INTO personal_access_tokens (id, tenant_id, user_id, name, token_hash, token_suffix, scopes) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING created_at`, id, tenantID, userID, name, hashToken(plaintext), suffix(plaintext), pq.Array(scopes)).Scan(&created)
	if err != nil {
		return CreatedToken{}, fmt.Errorf("create pat: %w", err)
	}
	return CreatedToken{Token: Token{ID: id, TenantID: tenantID, UserID: userID, Name: name, Last4: suffix(plaintext), Scopes: scopes, CreatedAt: created}, Plaintext: plaintext}, nil
}

func (s Service) List(ctx context.Context, tenantID, userID uuid.UUID) ([]Token, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, tenant_id, user_id, name, token_suffix, scopes, created_at, last_used_at, revoked_at FROM personal_access_tokens WHERE tenant_id = $1 AND user_id = $2 ORDER BY created_at DESC`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tokens []Token
	for rows.Next() {
		var token Token
		var lastUsed, revoked sql.NullTime
		if err := rows.Scan(&token.ID, &token.TenantID, &token.UserID, &token.Name, &token.Last4, pq.Array(&token.Scopes), &token.CreatedAt, &lastUsed, &revoked); err != nil {
			return nil, err
		}
		if lastUsed.Valid {
			token.LastUsedAt = &lastUsed.Time
		}
		if revoked.Valid {
			token.RevokedAt = &revoked.Time
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s Service) Revoke(ctx context.Context, tenantID, userID, tokenID uuid.UUID) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE personal_access_tokens SET revoked_at = now() WHERE tenant_id = $1 AND user_id = $2 AND id = $3 AND revoked_at IS NULL`, tenantID, userID, tokenID)
	return err
}

func (s Service) RevokeForUser(ctx context.Context, tenantID, userID uuid.UUID, reason string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE personal_access_tokens SET revoked_at = now() WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, tenantID, userID)
	_ = reason
	return err
}

func (s Service) Authenticate(ctx context.Context, plaintext string) (Token, error) {
	var token Token
	var lastUsed, revoked sql.NullTime
	err := s.DB.QueryRowContext(ctx, `UPDATE personal_access_tokens SET last_used_at = now() WHERE token_hash = $1 AND revoked_at IS NULL RETURNING id, tenant_id, user_id, name, token_suffix, scopes, created_at, last_used_at, revoked_at`, hashToken(plaintext)).Scan(&token.ID, &token.TenantID, &token.UserID, &token.Name, &token.Last4, pq.Array(&token.Scopes), &token.CreatedAt, &lastUsed, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return Token{}, ErrInvalidToken
	}
	if lastUsed.Valid {
		token.LastUsedAt = &lastUsed.Time
	}
	if revoked.Valid {
		token.RevokedAt = &revoked.Time
	}
	return token, err
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

func suffix(token string) string {
	if len(token) <= 4 {
		return token
	}
	return token[len(token)-4:]
}

// Package magiclink implements magic-link token issuance and consumption.
package magiclink

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/authpolicy"
)

//revive:disable:exported

var (
	ErrInvalidToken    = errors.New("invalid magic link token")
	ErrTooManyActive   = errors.New("magic_link.too_many_active")
)

type Service struct{ DB *sql.DB }

// Issue creates a new magic-link token under the given policy. The policy
// determines TTL and the per-user active-token cap.
func (s Service) Issue(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, email, redirectURL string, policy authpolicy.MagicLinkPolicy) (string, error) {
	policy = policy.WithDefaults()
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	if userID != nil && policy.MaxActivePerUser > 0 {
		var active int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM magic_link_tokens WHERE tenant_id = $1 AND user_id = $2 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > now()`, tenantID, *userID).Scan(&active); err != nil {
			return "", fmt.Errorf("count active magic link tokens: %w", err)
		}
		if active >= policy.MaxActivePerUser {
			return "", ErrTooManyActive
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO magic_link_tokens (tenant_id, user_id, email, token_hash, expires_at) VALUES ($1, $2, $3, $4, $5)`, tenantID, userID, email, hashToken(token), time.Now().UTC().Add(policy.TTL())); err != nil {
		return "", fmt.Errorf("insert magic link token: %w", err)
	}
	payload, _ := json.Marshal(map[string]string{"token": token, "redirect_url": redirectURL})
	if _, err := tx.ExecContext(ctx, `INSERT INTO email_outbox (tenant_id, to_address, template, payload) VALUES ($1, $2, 'magic-link', $3)`, tenantID, email, payload); err != nil {
		return "", fmt.Errorf("enqueue magic link email: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}

func (s Service) Consume(ctx context.Context, token string) (uuid.UUID, string, error) {
	var userID uuid.UUID
	var email string
	err := s.DB.QueryRowContext(ctx, `UPDATE magic_link_tokens SET consumed_at = now() WHERE token_hash = $1 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > now() RETURNING COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid), email`, hashToken(token)).Scan(&userID, &email)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, "", ErrInvalidToken
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("consume magic link token: %w", err)
	}
	return userID, email, nil
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

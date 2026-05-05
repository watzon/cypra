// Package password implements password credentials and reset tokens.
package password

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
	"github.com/watzon/cypra/internal/crypto"
)

//revive:disable:exported

var ErrInvalidResetToken = errors.New("invalid password reset token")

type Service struct{ DB *sql.DB }

func (s Service) SetPassword(ctx context.Context, tenantID, userID uuid.UUID, plaintext string) error {
	hash, err := crypto.HashPassword(plaintext, crypto.RejectCommonPasswords)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO password_credentials (tenant_id, user_id, argon2id_hash, must_reset) VALUES ($1, $2, $3, false) ON CONFLICT (user_id) DO UPDATE SET argon2id_hash = EXCLUDED.argon2id_hash, must_reset = false, updated_at = now()`, tenantID, userID, []byte(hash))
	if err != nil {
		return fmt.Errorf("set password credential: %w", err)
	}
	return s.RevokeLiveResetTokens(ctx, userID)
}

func (s Service) Verify(ctx context.Context, userID uuid.UUID, plaintext string) error {
	var encoded []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT argon2id_hash FROM password_credentials WHERE user_id = $1`, userID).Scan(&encoded); err != nil {
		return fmt.Errorf("load password credential: %w", err)
	}
	return crypto.VerifyPassword(string(encoded), plaintext)
}

func (s Service) IssueResetToken(ctx context.Context, tenantID, userID uuid.UUID, ttl time.Duration) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO password_reset_tokens (tenant_id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`, tenantID, userID, hashToken(token), time.Now().UTC().Add(ttl))
	if err != nil {
		return "", fmt.Errorf("issue password reset token: %w", err)
	}
	return token, nil
}

func (s Service) ResetPassword(ctx context.Context, token, plaintext string) (uuid.UUID, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var tenantID uuid.UUID
	var userID uuid.UUID
	err = tx.QueryRowContext(ctx, `UPDATE password_reset_tokens SET consumed_at = now() WHERE token_hash = $1 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > now() RETURNING tenant_id, user_id`, hashToken(token)).Scan(&tenantID, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrInvalidResetToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("consume password reset token: %w", err)
	}
	hash, err := crypto.HashPassword(plaintext, crypto.RejectCommonPasswords)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO password_credentials (tenant_id, user_id, argon2id_hash, must_reset) VALUES ($1, $2, $3, false) ON CONFLICT (user_id) DO UPDATE SET argon2id_hash = EXCLUDED.argon2id_hash, must_reset = false, updated_at = now()`, tenantID, userID, []byte(hash)); err != nil {
		return uuid.Nil, fmt.Errorf("reset password credential: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_reset_tokens SET revoked_at = now() WHERE user_id = $1 AND consumed_at IS NULL AND revoked_at IS NULL`, userID); err != nil {
		return uuid.Nil, fmt.Errorf("revoke competing password reset tokens: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (s Service) RevokeLiveResetTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE password_reset_tokens SET revoked_at = now() WHERE user_id = $1 AND consumed_at IS NULL AND revoked_at IS NULL`, userID)
	if err != nil {
		return fmt.Errorf("revoke password reset tokens: %w", err)
	}
	return nil
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

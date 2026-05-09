// Package bootstrap implements first-boot setup-token handling.
package bootstrap

//revive:disable:exported

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const SetupTokenTTL = 24 * time.Hour

var (
	ErrBootstrapUnavailable = errors.New("bootstrap unavailable")
	ErrInvalidSetupToken    = errors.New("invalid setup token")
)

type Service struct {
	db     *sql.DB
	logger *slog.Logger
	now    func() time.Time
}

func NewService(db *sql.DB, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, logger: logger, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) SetNow(now func() time.Time) {
	s.now = now
}

func (s *Service) IsFirstBoot(ctx context.Context) (bool, error) {
	var admins int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM instance_admins`).Scan(&admins); err != nil {
		return false, fmt.Errorf("count instance admins: %w", err)
	}
	if admins > 0 {
		return false, nil
	}
	var liveTokens int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM bootstrap_tokens WHERE consumed_at IS NULL AND revoked_at IS NULL AND expires_at > $1`, s.now()).Scan(&liveTokens); err != nil {
		return false, fmt.Errorf("count bootstrap tokens: %w", err)
	}
	return liveTokens == 0, nil
}

func (s *Service) MintSetupToken(ctx context.Context) (string, error) {
	firstBoot, err := s.IsFirstBoot(ctx)
	if err != nil {
		return "", err
	}
	if !firstBoot {
		return "", ErrBootstrapUnavailable
	}
	plaintext, err := randomSetupToken()
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin setup token mint: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO bootstrap_tokens (token_hash, expires_at) VALUES ($1, $2)`, hashSetupToken(plaintext), s.now().Add(SetupTokenTTL)); err != nil {
		return "", fmt.Errorf("insert setup token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit setup token: %w", err)
	}
	s.logger.Info("bootstrap setup token minted", slog.String("request_id", ""), slog.String("tenant_id", ""), slog.String("actor_id", "system"), slog.String("setup_token", plaintext), slog.Bool("redacted-on-export", true))
	return plaintext, nil
}

func (s *Service) RedeemSetupToken(ctx context.Context, plaintext string) (uuid.UUID, error) {
	id := uuid.Nil
	err := s.db.QueryRowContext(ctx, `UPDATE bootstrap_tokens SET consumed_at = $1 WHERE token_hash = $2 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > $1 RETURNING id`, s.now(), hashSetupToken(plaintext)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrInvalidSetupToken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("redeem setup token: %w", err)
	}
	return id, nil
}

func (s *Service) ValidateSetupToken(ctx context.Context, plaintext string) error {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM bootstrap_tokens WHERE token_hash = $1 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > $2)`, hashSetupToken(plaintext), s.now()).Scan(&exists)
	if err != nil {
		return fmt.Errorf("validate setup token: %w", err)
	}
	if !exists {
		return ErrInvalidSetupToken
	}
	return nil
}

func (s *Service) CreateFirstInstanceAdmin(ctx context.Context, setupTokenID uuid.UUID, email, displayName string) (uuid.UUID, error) {
	return s.createFirstInstanceAdmin(ctx, uuid.New(), setupTokenID, email, displayName)
}

func (s *Service) CreateFirstInstanceAdminWithID(ctx context.Context, adminID, setupTokenID uuid.UUID, email, displayName string) (uuid.UUID, error) {
	if adminID == uuid.Nil {
		return uuid.Nil, ErrInvalidSetupToken
	}
	return s.createFirstInstanceAdmin(ctx, adminID, setupTokenID, email, displayName)
}

func (s *Service) createFirstInstanceAdmin(ctx context.Context, adminID, setupTokenID uuid.UUID, email, displayName string) (uuid.UUID, error) {
	if setupTokenID == uuid.Nil {
		return uuid.Nil, ErrInvalidSetupToken
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin first admin creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var admins int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM instance_admins`).Scan(&admins); err != nil {
		return uuid.Nil, fmt.Errorf("count instance admins: %w", err)
	}
	if admins > 0 {
		return uuid.Nil, ErrBootstrapUnavailable
	}
	var tokenID uuid.UUID
	if err := tx.QueryRowContext(ctx, `UPDATE bootstrap_tokens SET revoked_at = $1 WHERE id = $2 AND consumed_at IS NOT NULL AND revoked_at IS NULL AND expires_at > $1 RETURNING id`, s.now(), setupTokenID).Scan(&tokenID); errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrInvalidSetupToken
	} else if err != nil {
		return uuid.Nil, fmt.Errorf("claim setup token: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO instance_admins (id, email, display_name, metadata) VALUES ($1, $2, $3, '{}'::jsonb)`, adminID, email, displayName); err != nil {
		return uuid.Nil, fmt.Errorf("create first instance admin: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("commit first admin creation: %w", err)
	}
	return adminID, nil
}

func (s *Service) RevokeSetupToken(ctx context.Context) (string, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE bootstrap_tokens SET revoked_at = $1 WHERE consumed_at IS NULL AND revoked_at IS NULL`, s.now()); err != nil {
		return "", fmt.Errorf("revoke setup tokens: %w", err)
	}
	firstBoot, err := s.IsFirstBoot(ctx)
	if err != nil {
		return "", err
	}
	if !firstBoot {
		return "", ErrBootstrapUnavailable
	}
	return s.MintSetupToken(ctx)
}

func randomSetupToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate setup token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashSetupToken(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

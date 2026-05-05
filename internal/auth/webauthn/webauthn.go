// Package webauthn stores passkey credentials with per-tenant RP IDs.
package webauthn

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

//revive:disable:exported

var ErrRPMismatch = errors.New("webauthn rp id mismatch")

type Service struct{ DB *sql.DB }

func (s Service) Register(ctx context.Context, tenantID, userID uuid.UUID, rpID string, credentialID, publicKey []byte) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id) VALUES ($1, $2, $3, $4, $5)`, tenantID, userID, credentialID, publicKey, rpID)
	if err != nil {
		return fmt.Errorf("register passkey: %w", err)
	}
	return nil
}

func (s Service) Assert(ctx context.Context, tenantID uuid.UUID, rpID string, credentialID []byte) (uuid.UUID, error) {
	var userID uuid.UUID
	var storedRPID string
	err := s.DB.QueryRowContext(ctx, `SELECT user_id, rp_id FROM passkey_credentials WHERE tenant_id = $1 AND credential_id = $2`, tenantID, credentialID).Scan(&userID, &storedRPID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("load passkey: %w", err)
	}
	if storedRPID != rpID {
		return uuid.Nil, ErrRPMismatch
	}
	_, _ = s.DB.ExecContext(ctx, `UPDATE passkey_credentials SET last_used_at = now() WHERE tenant_id = $1 AND credential_id = $2`, tenantID, credentialID)
	return userID, nil
}

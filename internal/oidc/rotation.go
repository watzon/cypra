// Package oidc contains OIDC provider primitives.
package oidc

//revive:disable:exported

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/crypto"
)

const (
	RotationInterval = 90 * 24 * time.Hour
	OverlapWindow    = 30 * 24 * time.Hour
	SunsetWindow     = 30 * 24 * time.Hour
)

type RotationService struct {
	db        *sql.DB
	kek       []byte
	now       func() time.Time
	algorithm string
}

func NewRotationService(db *sql.DB, kek []byte) *RotationService {
	return &RotationService{
		db:        db,
		kek:       kek,
		now:       func() time.Time { return time.Now().UTC() },
		algorithm: crypto.SigningAlgRS256,
	}
}

func (s *RotationService) SetNow(now func() time.Time) {
	s.now = now
}

func (s *RotationService) ForceRotate(ctx context.Context, tenantID uuid.UUID) error {
	now := s.now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin signing-key rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `UPDATE oidc_signing_keys SET state = 'retired' WHERE tenant_id = $1 AND state = 'overlap'`, tenantID); err != nil {
		return fmt.Errorf("retire prior overlap signing key: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE oidc_signing_keys SET state = 'overlap', retires_at = $1 WHERE tenant_id = $2 AND state = 'active'`, now.Add(OverlapWindow), tenantID); err != nil {
		return fmt.Errorf("move active signing key to overlap: %w", err)
	}
	seq, err := s.nextSequence(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	key, err := crypto.GenerateSigningKey(tenantID, seq, s.algorithm, s.kek, now)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO oidc_signing_keys (tenant_id, kid, algorithm, public_key_jwk, private_key_encrypted, state, activated_at, retires_at) VALUES ($1, $2, $3, $4, $5, 'active', $6, $7)`, tenantID, key.KID, key.Algorithm, []byte(key.PublicJWK), key.PrivateKeyEncrypted, key.ActivatedAt, key.RetiresAt); err != nil {
		return fmt.Errorf("insert new active signing key: %w", err)
	}
	return tx.Commit()
}

func (s *RotationService) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Hour
	}
	if err := s.RotateDue(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.RotateDue(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *RotationService) RotateDue(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT tenant_id FROM oidc_signing_keys WHERE state = 'active' AND activated_at <= $1`, s.now().Add(-RotationInterval))
	if err != nil {
		return fmt.Errorf("select due signing keys: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var tenants []uuid.UUID
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return fmt.Errorf("scan due tenant: %w", err)
		}
		tenants = append(tenants, tenantID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate due tenants: %w", err)
	}
	for _, tenantID := range tenants {
		if err := s.ForceRotate(ctx, tenantID); err != nil {
			return err
		}
	}
	_, err = s.db.ExecContext(ctx, `UPDATE oidc_signing_keys SET state = 'retired' WHERE state = 'overlap' AND retires_at <= $1`, s.now())
	if err != nil {
		return fmt.Errorf("retire overlapped signing keys: %w", err)
	}
	return nil
}

func (s *RotationService) SunsetKeys(ctx context.Context, tenantID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE oidc_signing_keys SET state = 'sunsetting', sunset_until = $1 WHERE tenant_id = $2 AND state IN ('active', 'overlap')`, s.now().Add(SunsetWindow), tenantID)
	if err != nil {
		return fmt.Errorf("sunset tenant signing keys: %w", err)
	}
	return nil
}

func (s *RotationService) PruneSunsetKeys(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oidc_signing_keys WHERE state = 'sunsetting' AND sunset_until <= $1`, s.now())
	if err != nil {
		return fmt.Errorf("prune sunsetting keys: %w", err)
	}
	return nil
}

func (s *RotationService) nextSequence(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1`, tenantID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tenant signing keys: %w", err)
	}
	return count + 1, nil
}

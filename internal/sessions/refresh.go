package sessions

//revive:disable:exported

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

const RefreshTokenBytes = 32

var ErrRefreshReuseDetected = errors.New("refresh token reuse detected")

type RefreshService struct {
	db            *sql.DB
	now           func() time.Time
	reuseCallback func()
}

type RefreshToken struct {
	Plaintext string
	ID        uuid.UUID
	FamilyID  uuid.UUID
	ParentID  *uuid.UUID
}

func NewRefreshService(db *sql.DB) *RefreshService {
	return &RefreshService{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (s *RefreshService) SetNow(now func() time.Time) {
	s.now = now
}

func (s *RefreshService) OnReuseDetected(callback func()) {
	s.reuseCallback = callback
}

func (s *RefreshService) Mint(ctx context.Context, tenantID, clientID, userID uuid.UUID, scope []string, parentID *uuid.UUID, expiresAt time.Time) (RefreshToken, error) {
	token, err := randomToken()
	if err != nil {
		return RefreshToken{}, err
	}
	familyID := uuid.New()
	if parentID != nil {
		if err := s.db.QueryRowContext(ctx, `SELECT family_id FROM oidc_refresh_tokens WHERE id = $1`, *parentID).Scan(&familyID); err != nil {
			return RefreshToken{}, fmt.Errorf("load parent refresh family: %w", err)
		}
	}
	id := uuid.New()
	_, err = s.db.ExecContext(ctx, `INSERT INTO oidc_refresh_tokens (id, family_id, parent_id, token_hash, tenant_id, oidc_client_uuid, user_id, scope, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, id, familyID, parentID, hashToken(token), tenantID, clientID, userID, pq.Array(scope), expiresAt)
	if err != nil {
		return RefreshToken{}, fmt.Errorf("insert refresh token: %w", err)
	}
	return RefreshToken{Plaintext: token, ID: id, FamilyID: familyID, ParentID: parentID}, nil
}

func (s *RefreshService) Consume(ctx context.Context, token string, expiresAt time.Time) (RefreshToken, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RefreshToken{}, fmt.Errorf("begin refresh consume: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	hash := hashToken(token)
	var consumed refreshRow
	err = tx.QueryRowContext(ctx, `UPDATE oidc_refresh_tokens SET consumed_at = $1 WHERE token_hash = $2 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > $1 RETURNING id, family_id, tenant_id, oidc_client_uuid, user_id, scope`, s.now(), hash).Scan(&consumed.id, &consumed.familyID, &consumed.tenantID, &consumed.clientID, &consumed.userID, pq.Array(&consumed.scope))
	if errors.Is(err, sql.ErrNoRows) {
		if reuseErr := s.handleReuse(ctx, tx, hash); reuseErr != nil {
			return RefreshToken{}, reuseErr
		}
		return RefreshToken{}, ErrRefreshReuseDetected
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("consume refresh token: %w", err)
	}
	childToken, err := randomToken()
	if err != nil {
		return RefreshToken{}, err
	}
	childID := uuid.New()
	_, err = tx.ExecContext(ctx, `INSERT INTO oidc_refresh_tokens (id, family_id, parent_id, token_hash, tenant_id, oidc_client_uuid, user_id, scope, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, childID, consumed.familyID, consumed.id, hashToken(childToken), consumed.tenantID, consumed.clientID, consumed.userID, pq.Array(consumed.scope), expiresAt)
	if err != nil {
		return RefreshToken{}, fmt.Errorf("insert child refresh token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return RefreshToken{}, fmt.Errorf("commit refresh consume: %w", err)
	}
	return RefreshToken{Plaintext: childToken, ID: childID, FamilyID: consumed.familyID, ParentID: &consumed.id}, nil
}

func (s *RefreshService) handleReuse(ctx context.Context, tx *sql.Tx, tokenHash []byte) error {
	var row refreshRow
	err := tx.QueryRowContext(ctx, `SELECT id, family_id, tenant_id, oidc_client_uuid, user_id FROM oidc_refresh_tokens WHERE token_hash = $1`, tokenHash).Scan(&row.id, &row.familyID, &row.tenantID, &row.clientID, &row.userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrRefreshReuseDetected
	}
	if err != nil {
		return fmt.Errorf("load reused refresh token: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE oidc_refresh_tokens SET revoked_at = $1, revoke_reason = 'reuse_detected' WHERE family_id = $2 AND revoked_at IS NULL`, s.now(), row.familyID); err != nil {
		return fmt.Errorf("revoke refresh family: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_entries (tenant_id, actor_kind, actor_id, action, resource_kind, resource_id, metadata) VALUES ($1, 'system', $2, 'cypra_oidc_refresh_reuse_detected', 'oidc_refresh_token', $3, '{}'::jsonb)`, row.tenantID, row.userID, row.id); err != nil {
		return fmt.Errorf("write refresh reuse audit entry: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit refresh reuse: %w", err)
	}
	if s.reuseCallback != nil {
		s.reuseCallback()
	}
	return nil
}

type refreshRow struct {
	id       uuid.UUID
	familyID uuid.UUID
	tenantID uuid.UUID
	clientID uuid.UUID
	userID   uuid.UUID
	scope    []string
}

func randomToken() (string, error) {
	raw := make([]byte, RefreshTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashToken(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

// Package invite implements tenant and instance-admin invitation flows.
package invite

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
)

//revive:disable:exported

const TTL = 7 * 24 * time.Hour

var ErrInvalidToken = errors.New("invalid invite token")

type Service struct{ DB *sql.DB }

type IssueRequest struct {
	TenantID      *uuid.UUID
	Email         string
	Role          string
	RedirectURL   string
	CreatedByKind string
	CreatedByID   *uuid.UUID
}

type Redeemed struct {
	InviteID uuid.UUID
	UserID   uuid.UUID
	AdminID  uuid.UUID
	TenantID *uuid.UUID
	Email    string
	Role     string
}

func (s Service) Issue(ctx context.Context, req IssueRequest) (string, uuid.UUID, error) {
	token, err := randomToken()
	if err != nil {
		return "", uuid.Nil, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", uuid.Nil, err
	}
	defer func() { _ = tx.Rollback() }()
	expiresAt := time.Now().UTC().Add(TTL)
	var inviteID uuid.UUID
	if req.TenantID == nil {
		err = tx.QueryRowContext(ctx, `INSERT INTO pending_invitations (tenant_id, email, role, token_hash, created_by_kind, created_by_id, expires_at) VALUES (NULL, $1, $2, $3, $4, $5, $6) ON CONFLICT (email) WHERE tenant_id IS NULL AND redeemed_at IS NULL DO UPDATE SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at, created_by_kind = EXCLUDED.created_by_kind, created_by_id = EXCLUDED.created_by_id RETURNING id`, req.Email, req.Role, hashToken(token), req.CreatedByKind, req.CreatedByID, expiresAt).Scan(&inviteID)
	} else {
		err = tx.QueryRowContext(ctx, `INSERT INTO pending_invitations (tenant_id, email, role, token_hash, created_by_kind, created_by_id, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (tenant_id, email) WHERE redeemed_at IS NULL DO UPDATE SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at, created_by_kind = EXCLUDED.created_by_kind, created_by_id = EXCLUDED.created_by_id RETURNING id`, *req.TenantID, req.Email, req.Role, hashToken(token), req.CreatedByKind, req.CreatedByID, expiresAt).Scan(&inviteID)
	}
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("upsert invite: %w", err)
	}
	payload, _ := json.Marshal(map[string]string{"token": token, "redirect_url": req.RedirectURL, "role": req.Role})
	if _, err := tx.ExecContext(ctx, `INSERT INTO email_outbox (tenant_id, to_address, template, payload) VALUES ($1, $2, 'admin-invite', $3)`, req.TenantID, req.Email, payload); err != nil {
		return "", uuid.Nil, fmt.Errorf("enqueue invite email: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", uuid.Nil, err
	}
	return token, inviteID, nil
}

func (s Service) Redeem(ctx context.Context, token, displayName string) (Redeemed, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Redeemed{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var redeemed Redeemed
	err = tx.QueryRowContext(ctx, `UPDATE pending_invitations SET redeemed_at = now() WHERE token_hash = $1 AND redeemed_at IS NULL AND expires_at > now() RETURNING id, tenant_id, email, role`, hashToken(token)).Scan(&redeemed.InviteID, &redeemed.TenantID, &redeemed.Email, &redeemed.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return Redeemed{}, ErrInvalidToken
	}
	if err != nil {
		return Redeemed{}, fmt.Errorf("consume invite: %w", err)
	}
	if redeemed.TenantID == nil {
		redeemed.AdminID = uuid.New()
		if _, err := tx.ExecContext(ctx, `INSERT INTO instance_admins (id, email, display_name, metadata) VALUES ($1, $2, $3, '{}'::jsonb)`, redeemed.AdminID, redeemed.Email, displayName); err != nil {
			return Redeemed{}, fmt.Errorf("create invited instance admin: %w", err)
		}
	} else {
		redeemed.UserID = uuid.New()
		if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, $3, '{}'::jsonb) ON CONFLICT DO NOTHING`, redeemed.UserID, *redeemed.TenantID, redeemed.Email); err != nil {
			return Redeemed{}, fmt.Errorf("create invited user: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO tenant_memberships (tenant_id, user_id, role) VALUES ($1, $2, $3)`, *redeemed.TenantID, redeemed.UserID, redeemed.Role); err != nil {
			return Redeemed{}, fmt.Errorf("bind invited role: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE pending_invitations SET redeemed_by_user_id = $1 WHERE id = $2`, redeemed.UserID, redeemed.InviteID); err != nil {
			return Redeemed{}, fmt.Errorf("record invite redeemer: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_entries (tenant_id, actor_kind, actor_id, action, resource_kind, resource_id, metadata) VALUES ($1, 'system', $2, 'invite.redeem', 'pending_invitation', $3, '{}'::jsonb)`, redeemed.TenantID, redeemed.UserID, redeemed.InviteID); err != nil {
		return Redeemed{}, fmt.Errorf("write invite audit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Redeemed{}, err
	}
	return redeemed, nil
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

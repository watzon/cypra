// Package backupcodes implements one-time recovery codes for users and instance admins.
package backupcodes

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/crypto"
)

//revive:disable:exported

const CodeCount = 10

type Service struct{ DB *sql.DB }

func (s Service) RegenerateForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE user_backup_codes SET used_at = now() WHERE user_id = $1 AND used_at IS NULL`, userID); err != nil {
		return nil, fmt.Errorf("invalidate user backup codes: %w", err)
	}
	codes, err := insertCodes(CodeCount, func(hash []byte) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO user_backup_codes (tenant_id, user_id, code_hash) VALUES ($1, $2, $3)`, tenantID, userID, hash)
		return err
	})
	if err != nil {
		return nil, err
	}
	return codes, tx.Commit()
}

func (s Service) RegenerateForInstanceAdmin(ctx context.Context, instanceAdminID uuid.UUID) ([]string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `UPDATE instance_admin_backup_codes SET used_at = now() WHERE instance_admin_id = $1 AND used_at IS NULL`, instanceAdminID); err != nil {
		return nil, fmt.Errorf("invalidate instance admin backup codes: %w", err)
	}
	codes, err := insertCodes(CodeCount, func(hash []byte) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO instance_admin_backup_codes (instance_admin_id, code_hash) VALUES ($1, $2)`, instanceAdminID, hash)
		return err
	})
	if err != nil {
		return nil, err
	}
	return codes, tx.Commit()
}

func (s Service) ConsumeForUser(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	return consume(ctx, s.DB, `SELECT id, code_hash FROM user_backup_codes WHERE user_id = $1 AND used_at IS NULL FOR UPDATE`, `UPDATE user_backup_codes SET used_at = now() WHERE id = $1`, userID, code)
}

func (s Service) ConsumeForInstanceAdmin(ctx context.Context, instanceAdminID uuid.UUID, code string) (bool, error) {
	return consume(ctx, s.DB, `SELECT id, code_hash FROM instance_admin_backup_codes WHERE instance_admin_id = $1 AND used_at IS NULL FOR UPDATE`, `UPDATE instance_admin_backup_codes SET used_at = now() WHERE id = $1`, instanceAdminID, code)
}

func insertCodes(count int, insert func([]byte) error) ([]string, error) {
	codes := make([]string, 0, count)
	for range count {
		code, err := randomCode()
		if err != nil {
			return nil, err
		}
		hash, err := crypto.HashPassword(code, nil)
		if err != nil {
			return nil, err
		}
		if err := insert([]byte(hash)); err != nil {
			return nil, fmt.Errorf("insert backup code: %w", err)
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func consume(ctx context.Context, db *sql.DB, selectSQL, updateSQL string, subjectID uuid.UUID, code string) (bool, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, selectSQL, subjectID)
	if err != nil {
		return false, err
	}
	type candidate struct {
		id   uuid.UUID
		hash []byte
	}
	var candidates []candidate
	for rows.Next() {
		var next candidate
		if err := rows.Scan(&next.id, &next.hash); err != nil {
			return false, err
		}
		candidates = append(candidates, next)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if err := rows.Close(); err != nil {
		return false, err
	}
	for _, candidate := range candidates {
		if crypto.VerifyPassword(string(candidate.hash), normalize(code)) == nil {
			if _, err := tx.ExecContext(ctx, updateSQL, candidate.id); err != nil {
				return false, err
			}
			return true, tx.Commit()
		}
	}
	return false, tx.Commit()
}

func randomCode() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	return encoded[:5] + "-" + encoded[5:10], nil
}

func normalize(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), " ", "-"))
}

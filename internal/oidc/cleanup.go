package oidc

//revive:disable:exported

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func CleanupExpired(ctx context.Context, db *sql.DB, now time.Time) error {
	queries := []string{
		`DELETE FROM oidc_authorization_codes WHERE expires_at + interval '7 days' < $1`,
		`DELETE FROM magic_link_tokens WHERE (consumed_at IS NOT NULL OR expires_at < $1) AND created_at + interval '7 days' < $1`,
		`DELETE FROM password_reset_tokens WHERE (consumed_at IS NOT NULL OR expires_at < $1) AND created_at + interval '7 days' < $1`,
		`DELETE FROM email_verification_tokens WHERE (consumed_at IS NOT NULL OR expires_at < $1) AND created_at + interval '7 days' < $1`,
		`UPDATE sessions SET revoked_at = COALESCE(revoked_at, $1) WHERE expires_at < $1 AND revoked_at IS NULL`,
		`UPDATE instance_admin_sessions SET revoked_at = COALESCE(revoked_at, $1) WHERE expires_at < $1 AND revoked_at IS NULL`,
	}
	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query, now); err != nil {
			return fmt.Errorf("cleanup expired auth rows: %w", err)
		}
	}
	return nil
}

func RunCleanupTicker(ctx context.Context, db *sql.DB, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if err := CleanupExpired(ctx, db, now.UTC()); err != nil {
				return err
			}
		}
	}
}

// Package ratelimit implements Postgres-backed token buckets.
package ratelimit

//revive:disable:exported

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Limiter struct{ DB *sql.DB }

func (l Limiter) Allow(ctx context.Context, tenantID *uuid.UUID, scope, key string, limit float64, window time.Duration) (bool, error) {
	refillRate := limit / window.Seconds()
	_, err := l.DB.ExecContext(ctx, `INSERT INTO rate_limit_buckets (scope, key, tenant_id, tokens, last_refill_at) VALUES ($1, $2, $3, $4, now()) ON CONFLICT (scope, key, tenant_id) DO NOTHING`, scope, key, tenantID, limit)
	if err != nil {
		return false, fmt.Errorf("seed rate limit bucket: %w", err)
	}
	var tokens float64
	err = l.DB.QueryRowContext(ctx, `UPDATE rate_limit_buckets SET tokens = LEAST($4, tokens + EXTRACT(EPOCH FROM (now() - last_refill_at)) * $5) - 1, last_refill_at = now() WHERE scope = $1 AND key = $2 AND tenant_id IS NOT DISTINCT FROM $3 AND tokens >= 1 RETURNING tokens`, scope, key, tenantID, limit, refillRate).Scan(&tokens)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("consume rate limit bucket: %w", err)
	}
	return true, nil
}

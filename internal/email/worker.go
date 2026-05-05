package email

//revive:disable:exported

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	DB       *sql.DB
	Sender   Sender
	Resolver Resolver
}

func (w Worker) RunOnce(ctx context.Context) error {
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin email worker: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT id, tenant_id, to_address, template, payload FROM email_outbox WHERE sent_at IS NULL AND failed_at IS NULL AND next_attempt_at <= now() ORDER BY next_attempt_at LIMIT 10 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return fmt.Errorf("select email outbox: %w", err)
	}
	type outboxRow struct {
		id       uuid.UUID
		tenantID uuid.NullUUID
		to       string
		template string
		payload  json.RawMessage
	}
	var pending []outboxRow
	for rows.Next() {
		var row outboxRow
		if err := rows.Scan(&row.id, &row.tenantID, &row.to, &row.template, &row.payload); err != nil {
			return err
		}
		pending = append(pending, row)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, row := range pending {
		sender := w.Sender
		if sender == nil && row.tenantID.Valid {
			var err error
			sender, err = w.Resolver.Resolve(ctx, row.tenantID.UUID)
			if err != nil {
				_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = $1, last_error = $2 WHERE id = $3`, time.Now().Add(time.Minute), err.Error(), row.id)
				continue
			}
		}
		if sender == nil {
			_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, failed_at = now(), last_error = $1 WHERE id = $2`, ErrProviderRequired.Error(), row.id)
			continue
		}
		message, err := Render(row.template, row.to, row.payload)
		if err != nil {
			_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, failed_at = now(), last_error = $1 WHERE id = $2`, err.Error(), row.id)
			continue
		}
		if err := sender.Send(ctx, message); err != nil {
			if errors.Is(err, ErrProviderRequired) {
				_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = $1, last_error = $2 WHERE id = $3`, time.Now().Add(time.Minute), err.Error(), row.id)
				continue
			}
			_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = $1, last_error = $2 WHERE id = $3`, time.Now().Add(time.Minute), err.Error(), row.id)
			continue
		}
		_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET sent_at = now() WHERE id = $1`, row.id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Commit()
}

package email

//revive:disable:exported

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Worker struct {
	DB     *sql.DB
	Sender Sender
}

func (w Worker) RunOnce(ctx context.Context) error {
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin email worker: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT id, to_address, template, payload FROM email_outbox WHERE sent_at IS NULL AND failed_at IS NULL AND next_attempt_at <= now() ORDER BY next_attempt_at LIMIT 10 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return fmt.Errorf("select email outbox: %w", err)
	}
	type outboxRow struct {
		id       uuid.UUID
		to       string
		template string
		payload  json.RawMessage
	}
	var pending []outboxRow
	for rows.Next() {
		var row outboxRow
		if err := rows.Scan(&row.id, &row.to, &row.template, &row.payload); err != nil {
			return err
		}
		pending = append(pending, row)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, row := range pending {
		message := Message{To: row.to, Subject: row.template, Text: string(row.payload)}
		if err := w.Sender.Send(ctx, message); err != nil {
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

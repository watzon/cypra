package email

//revive:disable:exported

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/observability"
	"go.opentelemetry.io/otel/attribute"
)

type Worker struct {
	DB           *sql.DB
	Sender       Sender
	Resolver     Resolver
	Workers      int
	BatchSize    int
	PollInterval time.Duration
	Logger       *slog.Logger
}

func (w Worker) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	workers := w.Workers
	if workers <= 0 {
		workers = 2
	}
	pollInterval := w.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				processed, err := w.runBatch(runCtx)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return
					}
					select {
					case errCh <- err:
					default:
					}
					return
				}
				if processed > 0 {
					continue
				}
				timer := time.NewTimer(pollInterval)
				select {
				case <-runCtx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		cancel()
		<-done
		return nil
	case err := <-errCh:
		cancel()
		<-done
		return err
	case <-done:
		return nil
	}
}

func (w Worker) RunOnce(ctx context.Context) error {
	_, err := w.runBatch(ctx)
	return err
}

func (w Worker) runBatch(ctx context.Context) (int, error) {
	ctx, batchSpan := observability.StartSpan(ctx, "email.worker.batch")
	defer batchSpan.End()
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		observability.RecordError(batchSpan, err)
		return 0, fmt.Errorf("begin email worker: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	batchSize := w.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, tenant_id, to_address, template, payload, attempts FROM email_outbox WHERE sent_at IS NULL AND failed_at IS NULL AND next_attempt_at <= now() ORDER BY next_attempt_at LIMIT $1 FOR UPDATE SKIP LOCKED`, batchSize)
	if err != nil {
		observability.RecordError(batchSpan, err)
		return 0, fmt.Errorf("select email outbox: %w", err)
	}
	type outboxRow struct {
		id       uuid.UUID
		tenantID uuid.NullUUID
		to       string
		template string
		payload  json.RawMessage
		attempts int
	}
	var pending []outboxRow
	for rows.Next() {
		var row outboxRow
		if err := rows.Scan(&row.id, &row.tenantID, &row.to, &row.template, &row.payload, &row.attempts); err != nil {
			return 0, err
		}
		pending = append(pending, row)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	for _, row := range pending {
		logger := w.Logger
		if logger == nil {
			logger = slog.Default()
		}
		sender := w.Sender
		if sender == nil && row.tenantID.Valid {
			var err error
			sender, err = w.Resolver.Resolve(ctx, row.tenantID.UUID)
			if err != nil {
				_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = $1, last_error = $2 WHERE id = $3`, time.Now().Add(backoffDelay(row.attempts+1)), err.Error(), row.id)
				continue
			}
		}
		if sender == nil {
			RecordSend("fail")
			_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, failed_at = now(), last_error = $1 WHERE id = $2`, ErrProviderRequired.Error(), row.id)
			continue
		}
		message, err := Render(row.template, row.to, row.payload)
		if err != nil {
			RecordSend("fail")
			_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, failed_at = now(), last_error = $1 WHERE id = $2`, err.Error(), row.id)
			continue
		}
		sendCtx, sendSpan := observability.StartSpan(ctx, "email.send", attribute.String("email.template", row.template))
		if err := sender.Send(sendCtx, message); err != nil {
			observability.RecordError(sendSpan, err)
			sendSpan.End()
			RecordSend("fail")
			if errors.Is(err, ErrProviderRequired) {
				_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = $1, last_error = $2 WHERE id = $3`, time.Now().Add(backoffDelay(row.attempts+1)), err.Error(), row.id)
				continue
			}
			_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET attempts = attempts + 1, next_attempt_at = $1, last_error = $2 WHERE id = $3`, time.Now().Add(backoffDelay(row.attempts+1)), err.Error(), row.id)
			continue
		}
		sendSpan.End()
		RecordSend("ok")
		_, _ = tx.ExecContext(ctx, `UPDATE email_outbox SET sent_at = now() WHERE id = $1`, row.id)
		tenantID := ""
		if row.tenantID.Valid {
			tenantID = row.tenantID.UUID.String()
		}
		logger.Info("email dispatch", slog.String("request_id", ""), slog.String("tenant_id", tenantID), slog.String("actor_id", "system"), slog.String("template", row.template), slog.String("to_email", row.to))
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return len(pending), tx.Commit()
}

func backoffDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	if attempts > 6 {
		attempts = 6
	}
	return time.Duration(1<<(attempts-1)) * time.Minute
}

// Package audit writes Cypra's append-only audit stream.
package audit

//revive:disable:exported

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type Writer struct{ db *sql.DB }

type Entry struct {
	TenantID     *uuid.UUID
	ActorKind    string
	ActorID      *uuid.UUID
	Action       string
	ResourceKind string
	ResourceID   *uuid.UUID
	StateBefore  any
	StateAfter   any
}

func NewWriter(db *sql.DB) *Writer { return &Writer{db: db} }

func (w *Writer) Write(ctx context.Context, entry Entry) error {
	before, _ := json.Marshal(entry.StateBefore)
	after, _ := json.Marshal(entry.StateAfter)
	if string(before) == "null" {
		before = nil
	}
	if string(after) == "null" {
		after = nil
	}
	_, err := w.db.ExecContext(ctx, `INSERT INTO audit_entries (tenant_id, actor_kind, actor_id, action, resource_kind, resource_id, state_before, state_after, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '{}'::jsonb)`, entry.TenantID, entry.ActorKind, entry.ActorID, entry.Action, entry.ResourceKind, entry.ResourceID, before, after)
	if err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}
	return nil
}

func ExportHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.QueryContext(r.Context(), `SELECT row_to_json(audit_entries) FROM audit_entries ORDER BY occurred_at DESC LIMIT 1000`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = rows.Close() }()
		w.Header().Set("Content-Type", "application/x-ndjson")
		for rows.Next() {
			var raw []byte
			if err := rows.Scan(&raw); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = w.Write(append(raw, '\n'))
		}
	}
}

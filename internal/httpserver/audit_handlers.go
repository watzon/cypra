package httpserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type auditEntryResponse struct {
	ID           string          `json:"id"`
	OccurredAt   time.Time       `json:"occurred_at"`
	ActorKind    string          `json:"actor_kind"`
	ActorID      *string         `json:"actor_id,omitempty"`
	Action       string          `json:"action"`
	ResourceKind string          `json:"resource_kind"`
	ResourceID   *string         `json:"resource_id,omitempty"`
	StateBefore  json.RawMessage `json:"state_before"`
	StateAfter   json.RawMessage `json:"state_after"`
	Metadata     json.RawMessage `json:"metadata"`
	RedactedAt   *time.Time      `json:"redacted_at,omitempty"`
}

func (s *Server) auditList(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	entries, err := s.queryAuditEntries(r, &tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) instanceAuditList(w http.ResponseWriter, r *http.Request) {
	entries, err := s.queryAuditEntries(r, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) auditExport(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	entries, err := s.queryAuditEntries(r, &tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "audit.export_failed")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, entry := range entries {
		row, err := json.Marshal(entry)
		if err != nil || !json.Valid(row) {
			writeError(w, http.StatusInternalServerError, "audit.export_invalid")
			return
		}
		_, _ = w.Write(append(row, '\n'))
	}
}

func (s *Server) queryAuditEntries(r *http.Request, tenantID *uuid.UUID) ([]auditEntryResponse, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if tenantID != nil {
		args = append(args, *tenantID)
		where = append(where, fmt.Sprintf("tenant_id = $%d", len(args)))
	}
	query := r.URL.Query()
	if action := strings.TrimSpace(query.Get("action")); action != "" {
		args = append(args, "%"+action+"%")
		where = append(where, fmt.Sprintf("action ILIKE $%d", len(args)))
	}
	if actor := strings.TrimSpace(query.Get("actor")); actor != "" {
		args = append(args, "%"+actor+"%")
		where = append(where, fmt.Sprintf("(actor_kind ILIKE $%d OR actor_id::text ILIKE $%d)", len(args), len(args)))
	}
	if resourceKind := strings.TrimSpace(query.Get("resource_kind")); resourceKind != "" {
		args = append(args, "%"+resourceKind+"%")
		where = append(where, fmt.Sprintf("resource_kind ILIKE $%d", len(args)))
	}
	if since := auditSince(query.Get("range"), query.Get("since")); since != nil {
		args = append(args, *since)
		where = append(where, fmt.Sprintf("occurred_at >= $%d", len(args)))
	}
	if until := strings.TrimSpace(query.Get("until")); until != "" {
		if parsed, err := time.Parse(time.RFC3339, until); err == nil {
			args = append(args, parsed)
			where = append(where, fmt.Sprintf("occurred_at <= $%d", len(args)))
		}
	}
	// #nosec G202 -- where fragments are fixed strings selected above; user values stay parameterized.
	stmt := `SELECT id::text, occurred_at, actor_kind, actor_id::text, action, resource_kind, resource_id::text, COALESCE(state_before, '{}'::jsonb), COALESCE(state_after, '{}'::jsonb), COALESCE(metadata, '{}'::jsonb), redacted_at FROM audit_entries WHERE ` + strings.Join(where, " AND ") + ` ORDER BY occurred_at DESC LIMIT 1000`
	// #nosec G701 -- stmt is assembled from fixed fragments; args carry all request input.
	rows, err := s.DB.QueryContext(r.Context(), stmt, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	entries := []auditEntryResponse{}
	for rows.Next() {
		var entry auditEntryResponse
		var actorID sql.NullString
		var resourceID sql.NullString
		var redactedAt sql.NullTime
		if err := rows.Scan(&entry.ID, &entry.OccurredAt, &entry.ActorKind, &actorID, &entry.Action, &entry.ResourceKind, &resourceID, &entry.StateBefore, &entry.StateAfter, &entry.Metadata, &redactedAt); err != nil {
			return nil, err
		}
		if actorID.Valid {
			entry.ActorID = &actorID.String
		}
		if resourceID.Valid {
			entry.ResourceID = &resourceID.String
		}
		if redactedAt.Valid {
			entry.RedactedAt = &redactedAt.Time
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func auditSince(rangeValue, sinceValue string) *time.Time {
	if sinceValue != "" {
		if parsed, err := time.Parse(time.RFC3339, sinceValue); err == nil {
			return &parsed
		}
	}
	switch strings.ToLower(strings.TrimSpace(rangeValue)) {
	case "last 24 hours", "24h":
		since := time.Now().UTC().Add(-24 * time.Hour)
		return &since
	case "last 7 days", "7d":
		since := time.Now().UTC().Add(-7 * 24 * time.Hour)
		return &since
	case "last 30 days", "30d":
		since := time.Now().UTC().Add(-30 * 24 * time.Hour)
		return &since
	default:
		return nil
	}
}

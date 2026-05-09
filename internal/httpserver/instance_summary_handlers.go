package httpserver

import "net/http"

type instanceSummaryResponse struct {
	Tenants          int                 `json:"tenants"`
	Projects         int                 `json:"projects"`
	Users            int                 `json:"users"`
	ActiveSigningKey bool                `json:"active_signing_key"`
	RecentAudit      []summaryAuditEntry `json:"recent_audit"`
}

type summaryAuditEntry struct {
	Action     string `json:"action"`
	ResourceID string `json:"resource_id"`
}

func (s *Server) instanceSummary(w http.ResponseWriter, r *http.Request) {
	summary := instanceSummaryResponse{}
	if err := s.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM tenants WHERE deleted_at IS NULL`).Scan(&summary.Tenants); err != nil {
		writeError(w, http.StatusInternalServerError, "instance.summary_failed")
		return
	}
	if err := s.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM projects WHERE deleted_at IS NULL`).Scan(&summary.Projects); err != nil {
		writeError(w, http.StatusInternalServerError, "instance.summary_failed")
		return
	}
	if err := s.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM users WHERE deleted_at IS NULL`).Scan(&summary.Users); err != nil {
		writeError(w, http.StatusInternalServerError, "instance.summary_failed")
		return
	}
	if err := s.DB.QueryRowContext(r.Context(), `SELECT EXISTS (SELECT 1 FROM oidc_signing_keys WHERE state = 'active')`).Scan(&summary.ActiveSigningKey); err != nil {
		writeError(w, http.StatusInternalServerError, "instance.summary_failed")
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT action, COALESCE(resource_id::text, resource_kind) FROM audit_entries ORDER BY occurred_at DESC LIMIT 10`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "instance.summary_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var entry summaryAuditEntry
		if err := rows.Scan(&entry.Action, &entry.ResourceID); err != nil {
			writeError(w, http.StatusInternalServerError, "instance.summary_failed")
			return
		}
		summary.RecentAudit = append(summary.RecentAudit, entry)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "instance.summary_failed")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

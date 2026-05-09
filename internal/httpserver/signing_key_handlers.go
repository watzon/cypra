package httpserver

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/oidc"
)

type signingKeyResponse struct {
	KID         string     `json:"kid"`
	Algorithm   string     `json:"algorithm"`
	State       string     `json:"state"`
	ActivatedAt time.Time  `json:"activated_at"`
	RetiresAt   *time.Time `json:"retires_at,omitempty"`
	SunsetUntil *time.Time `json:"sunset_until,omitempty"`
}

func (s *Server) listSigningKeys(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	keys, err := s.signingKeys(r, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "signing_keys.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) forceRotateSigningKey(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload struct {
		KID string `json:"kid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	var activeKID string
	if err := s.DB.QueryRowContext(r.Context(), `SELECT kid FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'active' LIMIT 1`, tenant.ID).Scan(&activeKID); err != nil || payload.KID != activeKID {
		writeError(w, http.StatusConflict, "signing_keys.confirmation_mismatch")
		return
	}
	if err := oidc.NewRotationService(s.DB, s.MasterKey).ForceRotate(r.Context(), tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "signing_keys.rotate_failed")
		return
	}
	keys, err := s.signingKeys(r, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "signing_keys.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

func (s *Server) signingKeys(r *http.Request, tenantID uuid.UUID) ([]signingKeyResponse, error) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT kid, algorithm, state, activated_at, retires_at, sunset_until FROM oidc_signing_keys WHERE tenant_id = $1 ORDER BY activated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	keys := []signingKeyResponse{}
	for rows.Next() {
		var key signingKeyResponse
		var retiresAt sql.NullTime
		var sunsetUntil sql.NullTime
		if err := rows.Scan(&key.KID, &key.Algorithm, &key.State, &key.ActivatedAt, &retiresAt, &sunsetUntil); err != nil {
			return nil, err
		}
		if retiresAt.Valid {
			key.RetiresAt = &retiresAt.Time
		}
		if sunsetUntil.Valid {
			key.SunsetUntil = &sunsetUntil.Time
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

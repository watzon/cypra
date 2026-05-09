package httpserver

import (
	"context"

	"github.com/watzon/cypra/internal/audit"
)

func (s *Server) recordMutationAudit(ctx context.Context, entry audit.Entry) {
	_ = s.audit.Write(ctx, entry)
}

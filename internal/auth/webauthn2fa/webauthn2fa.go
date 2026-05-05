// Package webauthn2fa provides a WebAuthn-backed second-factor seam.
package webauthn2fa

import (
	"context"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/webauthn"
)

//revive:disable:exported

type Service struct{ Primary webauthn.Service }

func (s Service) Enroll(ctx context.Context, tenantID, userID uuid.UUID, rpID string, credentialID, publicKey []byte) error {
	return s.Primary.Register(ctx, tenantID, userID, rpID, append([]byte("2fa:"), credentialID...), publicKey)
}

func (s Service) Verify(ctx context.Context, tenantID uuid.UUID, rpID string, credentialID []byte) (uuid.UUID, error) {
	return s.Primary.Assert(ctx, tenantID, rpID, append([]byte("2fa:"), credentialID...))
}

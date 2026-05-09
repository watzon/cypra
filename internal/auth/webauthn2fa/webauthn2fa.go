// Package webauthn2fa provides a WebAuthn-backed second-factor seam.
package webauthn2fa

import (
	"context"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/webauthn"
)

//revive:disable:exported

type Service struct{ Primary webauthn.Service }

type BeginEnrollRequest struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	RPID     string
}

type FinishEnrollRequest struct {
	TenantID   uuid.UUID
	UserID     uuid.UUID
	RPID       string
	Origins    []string
	CeremonyID uuid.UUID
	Response   *http.Request
}

type BeginVerifyRequest struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	RPID     string
}

type FinishVerifyRequest struct {
	TenantID   uuid.UUID
	RPID       string
	Origins    []string
	CeremonyID uuid.UUID
	Response   *http.Request
}

func (s Service) BeginEnroll(ctx context.Context, req BeginEnrollRequest) (*protocol.CredentialCreation, uuid.UUID, error) {
	return s.Primary.BeginRegistration(ctx, webauthn.BeginRegistrationRequest{
		TenantID: req.TenantID,
		UserID:   req.UserID,
		RPID:     req.RPID,
		Purpose:  webauthn.PurposeSecondFactor,
		Kind:     webauthn.KindWebAuthn2FAEnrollment,
	})
}

func (s Service) FinishEnroll(ctx context.Context, req FinishEnrollRequest) (*gowebauthn.Credential, error) {
	return s.Primary.FinishRegistration(ctx, webauthn.FinishRegistrationRequest{
		TenantID:   req.TenantID,
		UserID:     req.UserID,
		RPID:       req.RPID,
		Origins:    req.Origins,
		Purpose:    webauthn.PurposeSecondFactor,
		Kind:       webauthn.KindWebAuthn2FAEnrollment,
		CeremonyID: req.CeremonyID,
		Response:   req.Response,
	})
}

func (s Service) BeginVerify(ctx context.Context, req BeginVerifyRequest) (*protocol.CredentialAssertion, uuid.UUID, error) {
	return s.Primary.BeginAssertion(ctx, webauthn.BeginAssertionRequest{
		TenantID: req.TenantID,
		UserID:   req.UserID,
		RPID:     req.RPID,
		Purpose:  webauthn.PurposeSecondFactor,
		Kind:     webauthn.KindWebAuthn2FAAssertion,
	})
}

func (s Service) FinishVerify(ctx context.Context, req FinishVerifyRequest) (uuid.UUID, *gowebauthn.Credential, error) {
	return s.Primary.FinishAssertion(ctx, webauthn.FinishAssertionRequest{
		TenantID:   req.TenantID,
		RPID:       req.RPID,
		Origins:    req.Origins,
		Purpose:    webauthn.PurposeSecondFactor,
		Kind:       webauthn.KindWebAuthn2FAAssertion,
		CeremonyID: req.CeremonyID,
		Response:   req.Response,
	})
}

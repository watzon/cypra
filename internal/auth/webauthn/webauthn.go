// Package webauthn runs passkey ceremonies with per-tenant RP IDs.
package webauthn

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

//revive:disable:exported

var (
	ErrRPMismatch      = errors.New("webauthn rp id mismatch")
	ErrCeremonyInvalid = errors.New("webauthn ceremony invalid")
)

const (
	PurposePrimary      = "primary"
	PurposeSecondFactor = "second_factor"

	KindPasskeyRegistration       = "passkey_registration"
	KindPasskeyAssertion          = "passkey_assertion"
	KindWebAuthn2FAEnrollment     = "webauthn2fa_registration"
	KindWebAuthn2FAAssertion      = "webauthn2fa_assertion"
	KindInstanceAdminRegistration = "instance_admin_passkey_registration"
	KindInstanceAdminAssertion    = "instance_admin_passkey_assertion"
)

type Service struct{ DB *sql.DB }

type BeginRegistrationRequest struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	RPID        string
	UserName    string
	DisplayName string
	Purpose     string
	Kind        string
}

type BeginAssertionRequest struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	RPID     string
	Purpose  string
	Kind     string
}

type FinishRegistrationRequest struct {
	TenantID   uuid.UUID
	UserID     uuid.UUID
	RPID       string
	Origins    []string
	Purpose    string
	Kind       string
	CeremonyID uuid.UUID
	Response   *http.Request
}

type BeginInstanceAdminRegistrationRequest struct {
	InstanceAdminID uuid.UUID
	RPID            string
	Email           string
	DisplayName     string
}

type FinishInstanceAdminRegistrationRequest struct {
	RPID       string
	Origins    []string
	CeremonyID uuid.UUID
	Response   *http.Request
}

type BeginInstanceAdminAssertionRequest struct {
	InstanceAdminID uuid.UUID
	RPID            string
}

type FinishInstanceAdminAssertionRequest struct {
	RPID       string
	Origins    []string
	CeremonyID uuid.UUID
	Response   *http.Request
}

type InstanceAdminRegistrationResult struct {
	InstanceAdminID uuid.UUID
	Credential      *gowebauthn.Credential
}

type FinishAssertionRequest struct {
	TenantID   uuid.UUID
	RPID       string
	Origins    []string
	Purpose    string
	Kind       string
	CeremonyID uuid.UUID
	Response   *http.Request
}

func (s Service) BeginRegistration(ctx context.Context, req BeginRegistrationRequest) (*protocol.CredentialCreation, uuid.UUID, error) {
	if req.Purpose == "" {
		req.Purpose = PurposePrimary
	}
	if req.Kind == "" {
		req.Kind = KindPasskeyRegistration
	}
	user, err := s.loadUser(ctx, req.TenantID, req.UserID, req.Purpose, req.UserName, req.DisplayName)
	if err != nil {
		return nil, uuid.Nil, err
	}
	rp, err := newRelyingParty(req.RPID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	residentKey := protocol.ResidentKeyRequirementRequired
	if req.Purpose == PurposeSecondFactor {
		residentKey = protocol.ResidentKeyRequirementDiscouraged
	}
	creation, session, err := rp.BeginRegistration(user,
		gowebauthn.WithResidentKeyRequirement(residentKey),
		gowebauthn.WithExclusions(gowebauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
	)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin webauthn registration: %w", err)
	}
	ceremonyID, err := s.storeSession(ctx, req.TenantID, req.UserID, req.Kind, req.RPID, session)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return creation, ceremonyID, nil
}

func (s Service) BeginInstanceAdminRegistration(ctx context.Context, req BeginInstanceAdminRegistrationRequest) (*protocol.CredentialCreation, uuid.UUID, error) {
	user, err := s.loadInstanceAdmin(ctx, req.InstanceAdminID, req.Email, req.DisplayName)
	if err != nil {
		return nil, uuid.Nil, err
	}
	rp, err := newRelyingParty(req.RPID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	creation, session, err := rp.BeginRegistration(user,
		gowebauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		gowebauthn.WithExclusions(gowebauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
	)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin instance admin webauthn registration: %w", err)
	}
	ceremonyID, err := s.storeInstanceAdminSession(ctx, req.InstanceAdminID, KindInstanceAdminRegistration, req.RPID, session)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return creation, ceremonyID, nil
}

func (s Service) FinishRegistration(ctx context.Context, req FinishRegistrationRequest) (*gowebauthn.Credential, error) {
	if req.Purpose == "" {
		req.Purpose = PurposePrimary
	}
	if req.Kind == "" {
		req.Kind = KindPasskeyRegistration
	}
	session, userID, rpID, err := s.consumeSession(ctx, req.TenantID, req.CeremonyID, req.Kind)
	if err != nil {
		return nil, err
	}
	if userID == nil || *userID != req.UserID {
		return nil, ErrCeremonyInvalid
	}
	if rpID != req.RPID || session.RelyingPartyID != req.RPID {
		return nil, ErrRPMismatch
	}
	user, err := s.loadUser(ctx, req.TenantID, req.UserID, req.Purpose, "", "")
	if err != nil {
		return nil, err
	}
	rp, err := newRelyingPartyWithOrigins(req.RPID, req.Origins)
	if err != nil {
		return nil, err
	}
	credential, err := rp.FinishRegistration(user, *session, req.Response)
	if err != nil {
		return nil, fmt.Errorf("finish webauthn registration: %w", err)
	}
	if err := s.insertCredential(ctx, req.TenantID, req.UserID, req.RPID, req.Purpose, credential); err != nil {
		return nil, err
	}
	return credential, nil
}

func (s Service) FinishInstanceAdminRegistration(ctx context.Context, req FinishInstanceAdminRegistrationRequest) (InstanceAdminRegistrationResult, error) {
	session, adminID, rpID, err := s.consumeInstanceAdminSession(ctx, req.CeremonyID, KindInstanceAdminRegistration)
	if err != nil {
		return InstanceAdminRegistrationResult{}, err
	}
	if rpID != req.RPID || session.RelyingPartyID != req.RPID {
		return InstanceAdminRegistrationResult{}, ErrRPMismatch
	}
	user, err := s.loadInstanceAdmin(ctx, adminID, "", "")
	if err != nil {
		return InstanceAdminRegistrationResult{}, err
	}
	rp, err := newRelyingPartyWithOrigins(req.RPID, req.Origins)
	if err != nil {
		return InstanceAdminRegistrationResult{}, err
	}
	credential, err := rp.FinishRegistration(user, *session, req.Response)
	if err != nil {
		return InstanceAdminRegistrationResult{}, fmt.Errorf("finish instance admin webauthn registration: %w", err)
	}
	return InstanceAdminRegistrationResult{InstanceAdminID: adminID, Credential: credential}, nil
}

// BeginInstanceAdminAssertion produces a WebAuthn assertion challenge for the
// install-host instance admin sign-in flow. When InstanceAdminID is set, the
// ceremony is bound to that admin (allowed credentials list comes from their
// passkeys). When it's uuid.Nil, the ceremony is discoverable: the browser
// picks the credential from any registered instance-admin passkey, and the
// admin is resolved on Finish from the credential's owner.
func (s Service) BeginInstanceAdminAssertion(ctx context.Context, req BeginInstanceAdminAssertionRequest) (*protocol.CredentialAssertion, uuid.UUID, error) {
	rp, err := newRelyingParty(req.RPID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if req.InstanceAdminID != uuid.Nil {
		user, err := s.loadInstanceAdmin(ctx, req.InstanceAdminID, "", "")
		if err != nil {
			return nil, uuid.Nil, err
		}
		if len(user.credentials) == 0 {
			return nil, uuid.Nil, ErrCeremonyInvalid
		}
		assertion, session, err := rp.BeginLogin(user)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("begin instance admin webauthn assertion: %w", err)
		}
		ceremonyID, err := s.storeInstanceAdminSession(ctx, req.InstanceAdminID, KindInstanceAdminAssertion, req.RPID, session)
		if err != nil {
			return nil, uuid.Nil, err
		}
		return assertion, ceremonyID, nil
	}
	descriptors, err := s.loadInstanceAdminCredentialDescriptors(ctx)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if len(descriptors) == 0 {
		return nil, uuid.Nil, ErrCeremonyInvalid
	}
	assertion, session, err := rp.BeginDiscoverableLogin(gowebauthn.WithAllowedCredentials(descriptors))
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin instance admin webauthn assertion: %w", err)
	}
	ceremonyID, err := s.storeInstanceAdminSession(ctx, uuid.Nil, KindInstanceAdminAssertion, req.RPID, session)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return assertion, ceremonyID, nil
}

func (s Service) FinishInstanceAdminAssertion(ctx context.Context, req FinishInstanceAdminAssertionRequest) (uuid.UUID, *gowebauthn.Credential, error) {
	session, adminID, rpID, err := s.consumeInstanceAdminSession(ctx, req.CeremonyID, KindInstanceAdminAssertion)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if rpID != req.RPID || session.RelyingPartyID != req.RPID {
		return uuid.Nil, nil, ErrRPMismatch
	}
	rp, err := newRelyingPartyWithOrigins(req.RPID, req.Origins)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if adminID != uuid.Nil {
		user, err := s.loadInstanceAdmin(ctx, adminID, "", "")
		if err != nil {
			return uuid.Nil, nil, err
		}
		credential, err := rp.FinishLogin(user, *session, req.Response)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("finish instance admin webauthn assertion: %w", err)
		}
		if err := s.updateInstanceAdminCredential(ctx, credential); err != nil {
			return uuid.Nil, nil, err
		}
		return adminID, credential, nil
	}
	// Discoverable login: the browser picked the credential. Resolve the admin
	// from credential ownership, then validate.
	var matchedAdmin *user
	_, credential, err := rp.FinishPasskeyLogin(func(rawID, _ []byte) (gowebauthn.User, error) {
		loaded, loadErr := s.loadInstanceAdminByCredential(ctx, rawID)
		if loadErr != nil {
			return nil, loadErr
		}
		matchedAdmin = loaded
		return loaded, nil
	}, *session, req.Response)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("finish instance admin webauthn assertion: %w", err)
	}
	if matchedAdmin == nil {
		return uuid.Nil, nil, ErrCeremonyInvalid
	}
	if err := s.updateInstanceAdminCredential(ctx, credential); err != nil {
		return uuid.Nil, nil, err
	}
	return matchedAdmin.id, credential, nil
}

func (s Service) loadInstanceAdminCredentialDescriptors(ctx context.Context) ([]protocol.CredentialDescriptor, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT credential_id FROM instance_admin_passkey_credentials`)
	if err != nil {
		return nil, fmt.Errorf("load instance admin credential descriptors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	descriptors := []protocol.CredentialDescriptor{}
	for rows.Next() {
		var credentialID []byte
		if err := rows.Scan(&credentialID); err != nil {
			return nil, fmt.Errorf("scan instance admin credential descriptor: %w", err)
		}
		descriptors = append(descriptors, protocol.CredentialDescriptor{Type: protocol.PublicKeyCredentialType, CredentialID: credentialID})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate instance admin credential descriptors: %w", err)
	}
	return descriptors, nil
}

func (s Service) loadInstanceAdminByCredential(ctx context.Context, credentialID []byte) (*user, error) {
	var adminID uuid.UUID
	if err := s.DB.QueryRowContext(ctx, `SELECT instance_admin_id FROM instance_admin_passkey_credentials WHERE credential_id = $1`, credentialID).Scan(&adminID); err != nil {
		return nil, fmt.Errorf("load instance admin credential owner: %w", err)
	}
	return s.loadInstanceAdmin(ctx, adminID, "", "")
}

func (s Service) updateInstanceAdminCredential(ctx context.Context, credential *gowebauthn.Credential) error {
	encoded, transports, aaguid, err := encodeCredential(credential)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx,
		`UPDATE instance_admin_passkey_credentials
		 SET sign_count = $1, transports = $2, aaguid = $3, credential = $4, last_used_at = now()
		 WHERE credential_id = $5`,
		credential.Authenticator.SignCount, pq.Array(transports), aaguid, encoded, credential.ID,
	)
	if err != nil {
		return fmt.Errorf("update instance admin webauthn credential: %w", err)
	}
	return nil
}

func (s Service) StoreInstanceAdminCredential(ctx context.Context, adminID uuid.UUID, rpID string, credential *gowebauthn.Credential) error {
	encoded, transports, aaguid, err := encodeCredential(credential)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO instance_admin_passkey_credentials (instance_admin_id, credential_id, public_key, sign_count, transports, aaguid, rp_id, credential) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, adminID, credential.ID, credential.PublicKey, credential.Authenticator.SignCount, pq.Array(transports), aaguid, rpID, encoded)
	if err != nil {
		return fmt.Errorf("store instance admin webauthn credential: %w", err)
	}
	return nil
}

func (s Service) BeginAssertion(ctx context.Context, req BeginAssertionRequest) (*protocol.CredentialAssertion, uuid.UUID, error) {
	if req.Purpose == "" {
		req.Purpose = PurposePrimary
	}
	if req.Kind == "" {
		req.Kind = KindPasskeyAssertion
	}
	rp, err := newRelyingParty(req.RPID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	if req.UserID != uuid.Nil {
		user, err := s.loadUser(ctx, req.TenantID, req.UserID, req.Purpose, "", "")
		if err != nil {
			return nil, uuid.Nil, err
		}
		assertion, session, err := rp.BeginLogin(user)
		if err != nil {
			return nil, uuid.Nil, fmt.Errorf("begin webauthn assertion: %w", err)
		}
		ceremonyID, err := s.storeSession(ctx, req.TenantID, req.UserID, req.Kind, req.RPID, session)
		if err != nil {
			return nil, uuid.Nil, err
		}
		return assertion, ceremonyID, nil
	}
	descriptors, err := s.loadCredentialDescriptors(ctx, req.TenantID, req.Purpose)
	if err != nil {
		return nil, uuid.Nil, err
	}
	assertion, session, err := rp.BeginDiscoverableLogin(gowebauthn.WithAllowedCredentials(descriptors))
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("begin webauthn assertion: %w", err)
	}
	ceremonyID, err := s.storeSession(ctx, req.TenantID, uuid.Nil, req.Kind, req.RPID, session)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return assertion, ceremonyID, nil
}

func (s Service) FinishAssertion(ctx context.Context, req FinishAssertionRequest) (uuid.UUID, *gowebauthn.Credential, error) {
	if req.Purpose == "" {
		req.Purpose = PurposePrimary
	}
	if req.Kind == "" {
		req.Kind = KindPasskeyAssertion
	}
	session, sessionUserID, rpID, err := s.consumeSession(ctx, req.TenantID, req.CeremonyID, req.Kind)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if rpID != req.RPID || session.RelyingPartyID != req.RPID {
		return uuid.Nil, nil, ErrRPMismatch
	}
	rp, err := newRelyingPartyWithOrigins(req.RPID, req.Origins)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if sessionUserID != nil {
		user, err := s.loadUser(ctx, req.TenantID, *sessionUserID, req.Purpose, "", "")
		if err != nil {
			return uuid.Nil, nil, err
		}
		credential, err := rp.FinishLogin(user, *session, req.Response)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("finish webauthn assertion: %w", err)
		}
		if err := s.updateCredential(ctx, req.TenantID, credential, req.Purpose); err != nil {
			return uuid.Nil, nil, err
		}
		return user.id, credential, nil
	}
	var matchedUser *user
	validatedUser, credential, err := rp.FinishPasskeyLogin(func(rawID, userHandle []byte) (gowebauthn.User, error) {
		loaded, loadErr := s.loadUserByCredential(ctx, req.TenantID, rawID, req.Purpose)
		if loadErr != nil {
			return nil, loadErr
		}
		if !bytes.Equal(userHandle, loaded.WebAuthnID()) {
			return nil, ErrCeremonyInvalid
		}
		matchedUser = loaded
		return loaded, nil
	}, *session, req.Response)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("finish webauthn assertion: %w", err)
	}
	if matchedUser == nil {
		var ok bool
		matchedUser, ok = validatedUser.(*user)
		if !ok {
			return uuid.Nil, nil, ErrCeremonyInvalid
		}
	}
	if err := s.updateCredential(ctx, req.TenantID, credential, req.Purpose); err != nil {
		return uuid.Nil, nil, err
	}
	return matchedUser.id, credential, nil
}

func (s Service) storeSession(ctx context.Context, tenantID, userID uuid.UUID, kind, rpID string, session *gowebauthn.SessionData) (uuid.UUID, error) {
	encoded, err := json.Marshal(session)
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode webauthn session: %w", err)
	}
	expires := session.Expires
	if expires.IsZero() {
		expires = time.Now().UTC().Add(5 * time.Minute)
	}
	var userArg any
	if userID != uuid.Nil {
		userArg = userID
	}
	var ceremonyID uuid.UUID
	if err := s.DB.QueryRowContext(ctx, `INSERT INTO webauthn_challenges (tenant_id, user_id, kind, rp_id, session_data, expires_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, tenantID, userArg, kind, rpID, encoded, expires).Scan(&ceremonyID); err != nil {
		return uuid.Nil, fmt.Errorf("store webauthn session: %w", err)
	}
	return ceremonyID, nil
}

func (s Service) storeInstanceAdminSession(ctx context.Context, adminID uuid.UUID, kind, rpID string, session *gowebauthn.SessionData) (uuid.UUID, error) {
	encoded, err := json.Marshal(session)
	if err != nil {
		return uuid.Nil, fmt.Errorf("encode instance admin webauthn session: %w", err)
	}
	expires := session.Expires
	if expires.IsZero() {
		expires = time.Now().UTC().Add(5 * time.Minute)
	}
	var adminArg any
	if adminID != uuid.Nil {
		adminArg = adminID
	}
	var ceremonyID uuid.UUID
	if err := s.DB.QueryRowContext(ctx, `INSERT INTO instance_admin_webauthn_challenges (instance_admin_id, kind, rp_id, session_data, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`, adminArg, kind, rpID, encoded, expires).Scan(&ceremonyID); err != nil {
		return uuid.Nil, fmt.Errorf("store instance admin webauthn session: %w", err)
	}
	return ceremonyID, nil
}

func (s Service) consumeSession(ctx context.Context, tenantID, ceremonyID uuid.UUID, kind string) (*gowebauthn.SessionData, *uuid.UUID, string, error) {
	var raw []byte
	var rpID string
	var userID uuid.NullUUID
	err := s.DB.QueryRowContext(ctx, `UPDATE webauthn_challenges SET consumed_at = now() WHERE id = $1 AND tenant_id = $2 AND kind = $3 AND consumed_at IS NULL AND expires_at > now() RETURNING user_id, rp_id, session_data`, ceremonyID, tenantID, kind).Scan(&userID, &rpID, &raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, "", ErrCeremonyInvalid
		}
		return nil, nil, "", fmt.Errorf("consume webauthn session: %w", err)
	}
	var session gowebauthn.SessionData
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, nil, "", fmt.Errorf("decode webauthn session: %w", err)
	}
	if userID.Valid {
		return &session, &userID.UUID, rpID, nil
	}
	return &session, nil, rpID, nil
}

// consumeInstanceAdminSession returns adminID = uuid.Nil when the ceremony was
// started without a bound admin (discoverable login). Callers must resolve the
// admin from the credential after the ceremony.
func (s Service) consumeInstanceAdminSession(ctx context.Context, ceremonyID uuid.UUID, kind string) (*gowebauthn.SessionData, uuid.UUID, string, error) {
	var raw []byte
	var rpID string
	var adminID uuid.NullUUID
	err := s.DB.QueryRowContext(ctx, `UPDATE instance_admin_webauthn_challenges SET consumed_at = now() WHERE id = $1 AND kind = $2 AND consumed_at IS NULL AND expires_at > now() RETURNING instance_admin_id, rp_id, session_data`, ceremonyID, kind).Scan(&adminID, &rpID, &raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, uuid.Nil, "", ErrCeremonyInvalid
		}
		return nil, uuid.Nil, "", fmt.Errorf("consume instance admin webauthn session: %w", err)
	}
	var session gowebauthn.SessionData
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, uuid.Nil, "", fmt.Errorf("decode instance admin webauthn session: %w", err)
	}
	if adminID.Valid {
		return &session, adminID.UUID, rpID, nil
	}
	return &session, uuid.Nil, rpID, nil
}

func (s Service) loadUser(ctx context.Context, tenantID, userID uuid.UUID, purpose, name, displayName string) (*user, error) {
	var email string
	if err := s.DB.QueryRowContext(ctx, `SELECT email FROM users WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`, tenantID, userID).Scan(&email); err != nil {
		return nil, fmt.Errorf("load webauthn user: %w", err)
	}
	if name == "" {
		name = email
	}
	if displayName == "" {
		displayName = email
	}
	credentials, err := s.loadCredentials(ctx, tenantID, userID, purpose)
	if err != nil {
		return nil, err
	}
	return &user{id: userID, name: name, displayName: displayName, credentials: credentials}, nil
}

func (s Service) loadInstanceAdmin(ctx context.Context, adminID uuid.UUID, email, displayName string) (*user, error) {
	if email == "" {
		_ = s.DB.QueryRowContext(ctx, `SELECT email, display_name FROM instance_admins WHERE id = $1 AND disabled_at IS NULL`, adminID).Scan(&email, &displayName)
	}
	if email == "" {
		email = adminID.String()
	}
	if displayName == "" {
		displayName = email
	}
	credentials, err := s.loadInstanceAdminCredentials(ctx, adminID)
	if err != nil {
		return nil, err
	}
	return &user{id: adminID, name: email, displayName: displayName, credentials: credentials}, nil
}

func (s Service) loadInstanceAdminCredentials(ctx context.Context, adminID uuid.UUID) ([]gowebauthn.Credential, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT credential_id, public_key, sign_count, transports, aaguid, credential FROM instance_admin_passkey_credentials WHERE instance_admin_id = $1`, adminID)
	if err != nil {
		return nil, fmt.Errorf("load instance admin webauthn credentials: %w", err)
	}
	defer func() { _ = rows.Close() }()
	credentials := []gowebauthn.Credential{}
	for rows.Next() {
		var credentialID, publicKey, raw []byte
		var signCount int64
		var transports pq.StringArray
		var aaguid uuid.NullUUID
		if err := rows.Scan(&credentialID, &publicKey, &signCount, &transports, &aaguid, &raw); err != nil {
			return nil, fmt.Errorf("scan instance admin webauthn credential: %w", err)
		}
		credentials = append(credentials, credentialFromStored(credentialID, publicKey, signCount, transports, aaguid, raw))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate instance admin webauthn credentials: %w", err)
	}
	return credentials, nil
}

func (s Service) loadUserByCredential(ctx context.Context, tenantID uuid.UUID, credentialID []byte, purpose string) (*user, error) {
	var userID uuid.UUID
	err := s.DB.QueryRowContext(ctx, `SELECT user_id FROM passkey_credentials WHERE tenant_id = $1 AND credential_id = $2 AND purpose = $3`, tenantID, credentialID, purpose).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("load webauthn credential owner: %w", err)
	}
	return s.loadUser(ctx, tenantID, userID, purpose, "", "")
}

func (s Service) loadCredentialDescriptors(ctx context.Context, tenantID uuid.UUID, purpose string) ([]protocol.CredentialDescriptor, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT credential_id FROM passkey_credentials WHERE tenant_id = $1 AND purpose = $2`, tenantID, purpose)
	if err != nil {
		return nil, fmt.Errorf("load webauthn credential descriptors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	descriptors := []protocol.CredentialDescriptor{}
	for rows.Next() {
		var credentialID []byte
		if err := rows.Scan(&credentialID); err != nil {
			return nil, fmt.Errorf("scan webauthn credential descriptor: %w", err)
		}
		descriptors = append(descriptors, protocol.CredentialDescriptor{Type: protocol.PublicKeyCredentialType, CredentialID: credentialID})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webauthn credential descriptors: %w", err)
	}
	return descriptors, nil
}

func (s Service) loadCredentials(ctx context.Context, tenantID, userID uuid.UUID, purpose string) ([]gowebauthn.Credential, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT credential_id, public_key, sign_count, transports, aaguid, credential FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2 AND purpose = $3`, tenantID, userID, purpose)
	if err != nil {
		return nil, fmt.Errorf("load webauthn credentials: %w", err)
	}
	defer func() { _ = rows.Close() }()
	credentials := []gowebauthn.Credential{}
	for rows.Next() {
		var credentialID, publicKey, raw []byte
		var signCount int64
		var transports pq.StringArray
		var aaguid uuid.NullUUID
		if err := rows.Scan(&credentialID, &publicKey, &signCount, &transports, &aaguid, &raw); err != nil {
			return nil, fmt.Errorf("scan webauthn credential: %w", err)
		}
		credential := credentialFromStored(credentialID, publicKey, signCount, transports, aaguid, raw)
		credentials = append(credentials, credential)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webauthn credentials: %w", err)
	}
	return credentials, nil
}

func (s Service) insertCredential(ctx context.Context, tenantID, userID uuid.UUID, rpID, purpose string, credential *gowebauthn.Credential) error {
	encoded, transports, aaguid, err := encodeCredential(credential)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, sign_count, transports, aaguid, rp_id, purpose, credential) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, tenantID, userID, credential.ID, credential.PublicKey, credential.Authenticator.SignCount, pq.Array(transports), aaguid, rpID, purpose, encoded)
	if err != nil {
		return fmt.Errorf("store webauthn credential: %w", err)
	}
	return nil
}

func (s Service) updateCredential(ctx context.Context, tenantID uuid.UUID, credential *gowebauthn.Credential, purpose string) error {
	encoded, transports, aaguid, err := encodeCredential(credential)
	if err != nil {
		return err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE passkey_credentials SET public_key = $1, sign_count = $2, transports = $3, aaguid = $4, credential = $5, last_used_at = now() WHERE tenant_id = $6 AND credential_id = $7 AND purpose = $8`, credential.PublicKey, credential.Authenticator.SignCount, pq.Array(transports), aaguid, encoded, tenantID, credential.ID, purpose)
	if err != nil {
		return fmt.Errorf("update webauthn credential: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows != 1 {
		return ErrCeremonyInvalid
	}
	return nil
}

func encodeCredential(credential *gowebauthn.Credential) ([]byte, []string, any, error) {
	encoded, err := json.Marshal(credential)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode webauthn credential: %w", err)
	}
	transports := make([]string, len(credential.Transport))
	for i, transport := range credential.Transport {
		transports[i] = string(transport)
	}
	var aaguid any
	if len(credential.Authenticator.AAGUID) == 16 {
		parsed, err := uuid.FromBytes(credential.Authenticator.AAGUID)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse webauthn aaguid: %w", err)
		}
		aaguid = parsed
	}
	return encoded, transports, aaguid, nil
}

func credentialFromStored(credentialID, publicKey []byte, signCount int64, transports pq.StringArray, aaguid uuid.NullUUID, raw []byte) gowebauthn.Credential {
	var credential gowebauthn.Credential
	if len(raw) != 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte(`{}`)) {
		if err := json.Unmarshal(raw, &credential); err == nil && len(credential.ID) != 0 {
			return credential
		}
	}
	credential.ID = credentialID
	credential.PublicKey = publicKey
	if signCount > 0 {
		if signCount > int64(^uint32(0)) {
			signCount = int64(^uint32(0))
		}
		// #nosec G115 -- signCount is clamped to uint32 range above.
		credential.Authenticator.SignCount = uint32(signCount)
	}
	if aaguid.Valid {
		credential.Authenticator.AAGUID, _ = aaguid.UUID.MarshalBinary()
	}
	credential.Transport = make([]protocol.AuthenticatorTransport, len(transports))
	for i, transport := range transports {
		credential.Transport[i] = protocol.AuthenticatorTransport(transport)
	}
	return credential
}

func newRelyingParty(rpID string) (*gowebauthn.WebAuthn, error) {
	return newRelyingPartyWithOrigins(rpID, nil)
}

func newRelyingPartyWithOrigins(rpID string, extraOrigins []string) (*gowebauthn.WebAuthn, error) {
	rpID = strings.TrimSpace(rpID)
	if rpID == "" {
		return nil, ErrRPMismatch
	}
	origins := []string{"https://" + rpID}
	for _, origin := range extraOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return gowebauthn.New(&gowebauthn.Config{
		RPDisplayName: "Cypra",
		RPID:          rpID,
		RPOrigins:     origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationPreferred,
		},
		Timeouts: gowebauthn.TimeoutsConfig{
			Login:        gowebauthn.TimeoutConfig{Enforce: true, Timeout: 5 * time.Minute, TimeoutUVD: 5 * time.Minute},
			Registration: gowebauthn.TimeoutConfig{Enforce: true, Timeout: 5 * time.Minute, TimeoutUVD: 5 * time.Minute},
		},
	})
}

type user struct {
	id          uuid.UUID
	name        string
	displayName string
	credentials []gowebauthn.Credential
}

func (u *user) WebAuthnID() []byte {
	data, _ := u.id.MarshalBinary()
	return data
}

func (u *user) WebAuthnName() string { return u.name }

func (u *user) WebAuthnDisplayName() string { return u.displayName }

func (u *user) WebAuthnCredentials() []gowebauthn.Credential { return u.credentials }

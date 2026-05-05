// Package models defines GORM structs that mirror the SQL schema. Migrations
// remain the source of truth; these structs intentionally carry no index tags.
package models

//revive:disable:exported

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

type Tenant struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Slug      string         `gorm:"type:text"`
	Name      string         `gorm:"type:text"`
	Branding  datatypes.JSON `gorm:"type:jsonb"`
	Settings  datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type Project struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid"`
	Slug      string    `gorm:"type:text"`
	Name      string    `gorm:"type:text"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

type InstanceAdmin struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Email       string         `gorm:"type:citext"`
	DisplayName string         `gorm:"type:text"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time
	LastLoginAt *time.Time
	DisabledAt  *time.Time
}

type User struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID               uuid.UUID `gorm:"type:uuid"`
	Email                  string    `gorm:"type:citext"`
	EmailVerifiedAt        *time.Time
	Metadata               datatypes.JSON `gorm:"type:jsonb"`
	ProfilePictureObjectID *uuid.UUID     `gorm:"type:uuid"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
	DeletedAt              *time.Time
}

type TenantMembership struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	Role      string    `gorm:"type:membership_role"`
	CreatedAt time.Time
}

type PasswordCredential struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID     uuid.UUID `gorm:"type:uuid"`
	UserID       uuid.UUID `gorm:"type:uuid"`
	Argon2idHash []byte
	MustReset    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PasskeyCredential struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID     uuid.UUID `gorm:"type:uuid"`
	UserID       uuid.UUID `gorm:"type:uuid"`
	CredentialID []byte
	PublicKey    []byte
	SignCount    int64
	Transports   pq.StringArray `gorm:"type:text[]"`
	AAGUID       *uuid.UUID     `gorm:"type:uuid;column:aaguid"`
	RPID         string         `gorm:"type:text;column:rp_id"`
	Nickname     string         `gorm:"type:text"`
	CreatedAt    time.Time
	LastUsedAt   *time.Time
}

type TOTPCredential struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID `gorm:"type:uuid"`
	UserID          uuid.UUID `gorm:"type:uuid"`
	SecretEncrypted []byte
	Algorithm       string `gorm:"type:text"`
	Digits          int
	PeriodSeconds   int
	ConfirmedAt     *time.Time
}

func (TOTPCredential) TableName() string { return "totp_credentials" }

type UserBackupCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID  uuid.UUID `gorm:"type:uuid"`
	UserID    uuid.UUID `gorm:"type:uuid"`
	CodeHash  []byte
	UsedAt    *time.Time
	CreatedAt time.Time
}

type InstanceAdminBackupCode struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	InstanceAdminID uuid.UUID `gorm:"type:uuid"`
	CodeHash        []byte
	UsedAt          *time.Time
	CreatedAt       time.Time
}

type MagicLinkToken struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TenantID   uuid.UUID  `gorm:"type:uuid"`
	UserID     *uuid.UUID `gorm:"type:uuid"`
	Email      string     `gorm:"type:citext"`
	TokenHash  []byte
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

type PasswordResetToken struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID   uuid.UUID `gorm:"type:uuid"`
	UserID     uuid.UUID `gorm:"type:uuid"`
	TokenHash  []byte
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

type EmailVerificationToken struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID   uuid.UUID `gorm:"type:uuid"`
	UserID     uuid.UUID `gorm:"type:uuid"`
	TokenHash  []byte
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

type PendingInvitation struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	TenantID         *uuid.UUID `gorm:"type:uuid"`
	Email            string     `gorm:"type:citext"`
	Role             string     `gorm:"type:invite_role"`
	TokenHash        []byte
	CreatedByKind    string     `gorm:"type:invitation_creator_kind"`
	CreatedByID      *uuid.UUID `gorm:"type:uuid"`
	ExpiresAt        time.Time
	RedeemedAt       *time.Time
	RedeemedByUserID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
}

type Session struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	SubjectID   uuid.UUID `gorm:"type:uuid"`
	SubjectKind string    `gorm:"type:session_subject_kind"`
	TenantID    uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	LastSeenAt  time.Time
	ExpiresAt   time.Time
	RevokedAt   *time.Time
	IP          *string `gorm:"type:inet;column:ip"`
	UserAgent   *string `gorm:"type:text"`
}

type InstanceAdminSession struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	InstanceAdminID uuid.UUID `gorm:"type:uuid"`
	CreatedAt       time.Time
	LastSeenAt      time.Time
	ExpiresAt       time.Time
	RevokedAt       *time.Time
	IP              *string `gorm:"type:inet;column:ip"`
	UserAgent       *string `gorm:"type:text"`
}

type OIDCClient struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID                uuid.UUID `gorm:"type:uuid"`
	ProjectID               uuid.UUID `gorm:"type:uuid"`
	ClientID                string    `gorm:"type:text"`
	ClientSecretEncrypted   []byte
	RedirectURIs            pq.StringArray `gorm:"type:text[]"`
	AllowedScopes           pq.StringArray `gorm:"type:text[]"`
	TokenEndpointAuthMethod string         `gorm:"type:oidc_auth_method"`
	CreatedAt               time.Time
	UpdatedAt               time.Time
	DeletedAt               *time.Time
}

func (OIDCClient) TableName() string { return "oidc_clients" }

type OIDCAuthorizationCode struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	CodeHash       []byte
	TenantID       uuid.UUID      `gorm:"type:uuid"`
	OIDCClientUUID uuid.UUID      `gorm:"type:uuid;column:oidc_client_uuid"`
	UserID         uuid.UUID      `gorm:"type:uuid"`
	RedirectURI    string         `gorm:"type:text"`
	Scope          pq.StringArray `gorm:"type:text[]"`
	PKCEChallenge  string         `gorm:"type:text;column:pkce_challenge"`
	PKCEMethod     string         `gorm:"type:pkce_method;column:pkce_method"`
	Nonce          *string        `gorm:"type:text"`
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
}

func (OIDCAuthorizationCode) TableName() string { return "oidc_authorization_codes" }

type OIDCRefreshToken struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FamilyID       uuid.UUID  `gorm:"type:uuid"`
	ParentID       *uuid.UUID `gorm:"type:uuid"`
	TokenHash      []byte
	TenantID       uuid.UUID      `gorm:"type:uuid"`
	OIDCClientUUID uuid.UUID      `gorm:"type:uuid;column:oidc_client_uuid"`
	UserID         uuid.UUID      `gorm:"type:uuid"`
	Scope          pq.StringArray `gorm:"type:text[]"`
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
	RevokedAt      *time.Time
	RevokeReason   *string `gorm:"type:text"`
}

func (OIDCRefreshToken) TableName() string { return "oidc_refresh_tokens" }

type OIDCConsent struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID       uuid.UUID      `gorm:"type:uuid"`
	UserID         uuid.UUID      `gorm:"type:uuid"`
	OIDCClientUUID uuid.UUID      `gorm:"type:uuid;column:oidc_client_uuid"`
	Scopes         pq.StringArray `gorm:"type:text[]"`
	GrantedAt      time.Time
	RevokedAt      *time.Time
}

func (OIDCConsent) TableName() string { return "oidc_consents" }

type OIDCSigningKey struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID            uuid.UUID      `gorm:"type:uuid"`
	KID                 string         `gorm:"type:text;column:kid"`
	Algorithm           string         `gorm:"type:text"`
	PublicKeyJWK        datatypes.JSON `gorm:"type:jsonb;column:public_key_jwk"`
	PrivateKeyEncrypted []byte
	State               string `gorm:"type:signing_key_state"`
	ActivatedAt         time.Time
	RetiresAt           time.Time
	SunsetUntil         *time.Time
}

func (OIDCSigningKey) TableName() string { return "oidc_signing_keys" }

type UpstreamProvider struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID              uuid.UUID `gorm:"type:uuid"`
	Kind                  string    `gorm:"type:upstream_provider_kind"`
	ClientIDEncrypted     []byte
	ClientSecretEncrypted []byte
	Enabled               bool
	CreatedAt             time.Time
}

type EmailProviderConfig struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID        uuid.UUID `gorm:"type:uuid"`
	Kind            string    `gorm:"type:email_provider_kind"`
	ConfigEncrypted []byte
	FromAddress     string `gorm:"type:text"`
	FromName        string `gorm:"type:text"`
}

type EmailOutbox struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID      *uuid.UUID     `gorm:"type:uuid"`
	ToAddress     string         `gorm:"type:citext"`
	Template      string         `gorm:"type:text"`
	Payload       datatypes.JSON `gorm:"type:jsonb"`
	Attempts      int
	NextAttemptAt time.Time
	SentAt        *time.Time
	FailedAt      *time.Time
	LastError     *string `gorm:"type:text"`
}

func (EmailOutbox) TableName() string { return "email_outbox" }

type StorageObject struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID `gorm:"type:uuid"`
	Backend     string    `gorm:"type:storage_backend"`
	Key         string    `gorm:"type:text"`
	ContentType string    `gorm:"type:text"`
	ByteSize    int64
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

type TenantAuthMethod struct {
	TenantID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Method             string    `gorm:"type:tenant_auth_method;primaryKey"`
	Enabled            bool
	EnrolledCountCache int64
	UpdatedAt          time.Time
}

type AuditEntry struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	OccurredAt   time.Time
	TenantID     *uuid.UUID     `gorm:"type:uuid"`
	ActorKind    string         `gorm:"type:audit_actor_kind"`
	ActorID      *uuid.UUID     `gorm:"type:uuid"`
	Action       string         `gorm:"type:text"`
	ResourceKind string         `gorm:"type:text"`
	ResourceID   *uuid.UUID     `gorm:"type:uuid"`
	StateBefore  datatypes.JSON `gorm:"type:jsonb"`
	StateAfter   datatypes.JSON `gorm:"type:jsonb"`
	IP           *string        `gorm:"type:inet;column:ip"`
	UserAgent    *string        `gorm:"type:text"`
	Metadata     datatypes.JSON `gorm:"type:jsonb"`
	RedactedAt   *time.Time
}

type BootstrapToken struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	TokenHash  []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	RevokedAt  *time.Time
}

type MasterKeyRotation struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	StartedAt   time.Time
	CompletedAt *time.Time
	Phase       string `gorm:"type:master_key_rotation_phase"`
	RowsTotal   int64
	RowsDone    int64
	Error       *string `gorm:"type:text"`
}

type RateLimitBucket struct {
	Scope        string     `gorm:"type:text;primaryKey"`
	Key          string     `gorm:"type:text;primaryKey"`
	TenantID     *uuid.UUID `gorm:"type:uuid"`
	Tokens       float64
	LastRefillAt time.Time
}

type GDPRDeletion struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID `gorm:"type:uuid"`
	UserID      uuid.UUID `gorm:"type:uuid"`
	State       string    `gorm:"type:gdpr_deletion_state"`
	RequestedAt time.Time
	CompletedAt *time.Time
	Error       *string `gorm:"type:text"`
}

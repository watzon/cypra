package crypto_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	cypra "github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestMasterKeyRotationResumeAndCutover(t *testing.T) {
	harness := dbtest.New(t)
	ctx := context.Background()
	if _, err := harness.SQL.Exec(`CREATE TABLE encrypted_test_rows (id UUID PRIMARY KEY, ciphertext BYTEA NOT NULL, encrypted_dek BYTEA NOT NULL)`); err != nil {
		t.Fatalf("create encrypted test rows: %v", err)
	}

	oldKEK := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	newKEK := bytes.Repeat([]byte{2}, cypra.MasterKeyBytes)
	rowIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	for i, rowID := range rowIDs {
		ciphertext, encryptedDEK, err := cypra.Encrypt([]byte{byte(i)}, oldKEK)
		if err != nil {
			t.Fatalf("encrypt row: %v", err)
		}
		if _, err := harness.SQL.Exec(`INSERT INTO encrypted_test_rows (id, ciphertext, encrypted_dek) VALUES ($1, $2, $3)`, rowID, ciphertext, encryptedDEK); err != nil {
			t.Fatalf("insert encrypted row: %v", err)
		}
	}

	service, err := cypra.NewRotationService(harness.SQL, []cypra.EncryptedColumn{{Table: "encrypted_test_rows", IDColumn: "id", DEKColumn: "encrypted_dek"}})
	if err != nil {
		t.Fatalf("new rotation service: %v", err)
	}
	rotationID, err := service.Start(ctx, oldKEK, newKEK, cypra.RotationOptions{MaxRows: 1})
	if !errors.Is(err, cypra.ErrRotationIncomplete) {
		t.Fatalf("start error = %v, want %v", err, cypra.ErrRotationIncomplete)
	}
	var rowsDone int64
	if err := harness.SQL.QueryRow(`SELECT rows_done FROM master_key_rotations WHERE id = $1`, rotationID).Scan(&rowsDone); err != nil {
		t.Fatalf("load rows_done: %v", err)
	}
	if rowsDone != 1 {
		t.Fatalf("rows_done after partial rotation = %d, want 1", rowsDone)
	}
	newWriteID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	newCiphertext, newEncryptedDEK, err := cypra.Encrypt([]byte("concurrent"), newKEK)
	if err != nil {
		t.Fatalf("encrypt concurrent write: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO encrypted_test_rows (id, ciphertext, encrypted_dek) VALUES ($1, $2, $3)`, newWriteID, newCiphertext, newEncryptedDEK); err != nil {
		t.Fatalf("insert concurrent encrypted row: %v", err)
	}
	if err := service.Resume(ctx, rotationID, oldKEK, newKEK, cypra.RotationOptions{}); err != nil {
		t.Fatalf("resume rotation: %v", err)
	}
	if err := service.ConfirmCutover(ctx, rotationID); err != nil {
		t.Fatalf("confirm cutover: %v", err)
	}

	rows, err := harness.SQL.Query(`SELECT ciphertext, encrypted_dek FROM encrypted_test_rows ORDER BY id`)
	if err != nil {
		t.Fatalf("query encrypted rows: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var ciphertext []byte
		var encryptedDEK []byte
		if err := rows.Scan(&ciphertext, &encryptedDEK); err != nil {
			t.Fatalf("scan encrypted row: %v", err)
		}
		if _, err := cypra.Decrypt(ciphertext, encryptedDEK, oldKEK); !errors.Is(err, cypra.ErrDecryptFailed) {
			t.Fatalf("old key decrypt error = %v, want decrypt failure", err)
		}
		if _, err := cypra.Decrypt(ciphertext, encryptedDEK, newKEK); err != nil {
			t.Fatalf("new key decrypt: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate encrypted rows: %v", err)
	}

	var phase string
	if err := harness.SQL.QueryRow(`SELECT phase FROM master_key_rotations WHERE id = $1`, rotationID).Scan(&phase); err != nil {
		t.Fatalf("load phase: %v", err)
	}
	if phase != "done" {
		t.Fatalf("phase = %q, want done", phase)
	}
}

func TestMasterKeyRotationRewrapsRealEncryptedColumnShapes(t *testing.T) {
	harness := dbtest.New(t)
	ctx := context.Background()
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`DELETE FROM oidc_signing_keys WHERE tenant_id = $1`, tenantID); err != nil {
		t.Fatalf("clear seeded signing keys: %v", err)
	}
	projectID := uuid.New()
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	oldKEK := bytes.Repeat([]byte{3}, cypra.MasterKeyBytes)
	newKEK := bytes.Repeat([]byte{4}, cypra.MasterKeyBytes)
	signingKey, err := cypra.GenerateSigningKey(tenantID, 2, cypra.SigningAlgES256, oldKEK, time.Now().UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	privateDER, err := cypra.DecryptSigningPrivateKey(signingKey.PrivateKeyEncrypted, oldKEK)
	if err != nil {
		t.Fatalf("decrypt generated signing key: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_signing_keys (tenant_id, kid, algorithm, public_key_jwk, private_key_encrypted, state, activated_at, retires_at) VALUES ($1, $2, $3, $4, $5, 'active', $6, $7)`, tenantID, signingKey.KID, signingKey.Algorithm, []byte(signingKey.PublicJWK), signingKey.PrivateKeyEncrypted, signingKey.ActivatedAt, signingKey.RetiresAt); err != nil {
		t.Fatalf("insert signing key: %v", err)
	}
	clientSecret := encryptPrefix(t, []byte("client-secret"), oldKEK)
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, 'client', $3, $4, $5, 'client_secret_post')`, tenantID, projectID, clientSecret, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("insert oidc client: %v", err)
	}
	upstreamClientID := encryptPrefix(t, []byte("google-client"), oldKEK)
	upstreamSecret := encryptPrefix(t, []byte("google-secret"), oldKEK)
	if _, err := harness.SQL.Exec(`INSERT INTO upstream_providers (tenant_id, kind, client_id_encrypted, client_secret_encrypted, enabled) VALUES ($1, 'google', $2, $3, true)`, tenantID, upstreamClientID, upstreamSecret); err != nil {
		t.Fatalf("insert social connection: %v", err)
	}
	emailConfig := encryptPrefix(t, []byte(`{"api_key":"resend"}`), oldKEK)
	if _, err := harness.SQL.Exec(`INSERT INTO email_provider_configs (tenant_id, kind, config_encrypted, from_address, from_name) VALUES ($1, 'resend', $2, 'noreply@example.com', 'Cypra')`, tenantID, emailConfig); err != nil {
		t.Fatalf("insert email config: %v", err)
	}
	totpSecret := encryptPrefix(t, []byte("totp-secret"), oldKEK)
	if _, err := harness.SQL.Exec(`INSERT INTO totp_credentials (tenant_id, user_id, secret_encrypted, algorithm, digits, period_seconds) VALUES ($1, $2, $3, 'SHA1', 6, 30)`, tenantID, userID, totpSecret); err != nil {
		t.Fatalf("insert totp credential: %v", err)
	}
	passkeyPublicKey := []byte("passkey-public-key")
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id) VALUES ($1, $2, 'credential'::bytea, $3, 'acme.cypra.localhost')`, tenantID, userID, passkeyPublicKey); err != nil {
		t.Fatalf("insert passkey credential: %v", err)
	}

	service, err := cypra.NewRotationService(harness.SQL, []cypra.EncryptedColumn{
		{Table: "oidc_signing_keys", IDColumn: "id", DEKColumn: "private_key_encrypted", Format: cypra.EncryptedColumnFormatJSONEnvelope},
		{Table: "oidc_clients", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "upstream_providers", IDColumn: "id", DEKColumn: "client_id_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "upstream_providers", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "email_provider_configs", IDColumn: "id", DEKColumn: "config_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
		{Table: "totp_credentials", IDColumn: "id", DEKColumn: "secret_encrypted", Format: cypra.EncryptedColumnFormatPrefix},
	})
	if err != nil {
		t.Fatalf("new rotation service: %v", err)
	}
	rotationID, err := service.Start(ctx, oldKEK, newKEK, cypra.RotationOptions{})
	if err != nil {
		t.Fatalf("start rotation: %v", err)
	}
	if err := service.ConfirmCutover(ctx, rotationID); err != nil {
		t.Fatalf("confirm cutover: %v", err)
	}

	var rotatedSigning []byte
	if err := harness.SQL.QueryRow(`SELECT private_key_encrypted FROM oidc_signing_keys WHERE kid = $1`, signingKey.KID).Scan(&rotatedSigning); err != nil {
		t.Fatalf("load signing key: %v", err)
	}
	if _, err := cypra.DecryptSigningPrivateKey(rotatedSigning, oldKEK); !errors.Is(err, cypra.ErrDecryptFailed) {
		t.Fatalf("old key signing decrypt error = %v, want decrypt failure", err)
	}
	if got, err := cypra.DecryptSigningPrivateKey(rotatedSigning, newKEK); err != nil || !bytes.Equal(got, privateDER) {
		t.Fatalf("new key signing decrypt = %d bytes, %v", len(got), err)
	}
	assertPrefixEnvelope(t, harness.SQL, `SELECT client_secret_encrypted FROM oidc_clients WHERE client_id = 'client'`, oldKEK, newKEK, []byte("client-secret"))
	assertPrefixEnvelope(t, harness.SQL, `SELECT client_id_encrypted FROM upstream_providers WHERE tenant_id = $1`, oldKEK, newKEK, []byte("google-client"), tenantID)
	assertPrefixEnvelope(t, harness.SQL, `SELECT client_secret_encrypted FROM upstream_providers WHERE tenant_id = $1`, oldKEK, newKEK, []byte("google-secret"), tenantID)
	assertPrefixEnvelope(t, harness.SQL, `SELECT config_encrypted FROM email_provider_configs WHERE tenant_id = $1`, oldKEK, newKEK, []byte(`{"api_key":"resend"}`), tenantID)
	assertPrefixEnvelope(t, harness.SQL, `SELECT secret_encrypted FROM totp_credentials WHERE user_id = $1`, oldKEK, newKEK, []byte("totp-secret"), userID)
	var rotatedPasskeyPublicKey []byte
	if err := harness.SQL.QueryRow(`SELECT public_key FROM passkey_credentials WHERE user_id = $1`, userID).Scan(&rotatedPasskeyPublicKey); err != nil {
		t.Fatalf("load passkey credential: %v", err)
	}
	if !bytes.Equal(rotatedPasskeyPublicKey, passkeyPublicKey) {
		t.Fatalf("passkey public key changed during rotation: %q", rotatedPasskeyPublicKey)
	}
}

func TestMasterKeyRotationRejectsInvalidTargetsAndMalformedEnvelopes(t *testing.T) {
	harness := dbtest.New(t)
	if _, err := cypra.NewRotationService(harness.SQL, []cypra.EncryptedColumn{{Table: "bad-table", IDColumn: "id", DEKColumn: "encrypted_dek"}}); err == nil {
		t.Fatal("unsafe target unexpectedly succeeded")
	}
	if _, err := cypra.NewRotationService(harness.SQL, []cypra.EncryptedColumn{{Table: "oidc_clients", IDColumn: "id", DEKColumn: "client_secret_encrypted", Format: "unknown"}}); err == nil {
		t.Fatal("unsupported target format unexpectedly succeeded")
	}
	if _, err := harness.SQL.Exec(`CREATE TABLE malformed_envelopes (id UUID PRIMARY KEY, secret BYTEA NOT NULL)`); err != nil {
		t.Fatalf("create malformed table: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO malformed_envelopes (id, secret) VALUES ($1, 'short'::bytea)`, uuid.New()); err != nil {
		t.Fatalf("insert malformed envelope: %v", err)
	}
	service, err := cypra.NewRotationService(harness.SQL, []cypra.EncryptedColumn{{Table: "malformed_envelopes", IDColumn: "id", DEKColumn: "secret", Format: cypra.EncryptedColumnFormatPrefix}})
	if err != nil {
		t.Fatalf("new rotation service: %v", err)
	}
	err = service.Resume(context.Background(), uuid.New(), bytes.Repeat([]byte{1}, cypra.MasterKeyBytes), bytes.Repeat([]byte{2}, cypra.MasterKeyBytes), cypra.RotationOptions{})
	if err == nil {
		t.Fatal("resume without rotation state unexpectedly succeeded")
	}
	_, err = service.Start(context.Background(), bytes.Repeat([]byte{1}, cypra.MasterKeyBytes), bytes.Repeat([]byte{2}, cypra.MasterKeyBytes), cypra.RotationOptions{})
	if !errors.Is(err, cypra.ErrDecryptFailed) {
		t.Fatalf("malformed envelope error = %v, want decrypt failed", err)
	}
}

func encryptPrefix(t *testing.T, plaintext, kek []byte) []byte {
	t.Helper()
	ciphertext, encryptedDEK, err := cypra.Encrypt(plaintext, kek)
	if err != nil {
		t.Fatalf("encrypt prefix envelope: %v", err)
	}
	return append(encryptedDEK, ciphertext...)
}

func assertPrefixEnvelope(t *testing.T, db queryRower, query string, oldKEK, newKEK, want []byte, args ...any) {
	t.Helper()
	var stored []byte
	if err := db.QueryRow(query, args...).Scan(&stored); err != nil {
		t.Fatalf("load prefix envelope: %v", err)
	}
	if _, err := cypra.Decrypt(stored[60:], stored[:60], oldKEK); !errors.Is(err, cypra.ErrDecryptFailed) {
		t.Fatalf("old key decrypt error = %v, want decrypt failure", err)
	}
	got, err := cypra.Decrypt(stored[60:], stored[:60], newKEK)
	if err != nil {
		t.Fatalf("new key decrypt: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decrypted payload = %q, want %q", got, want)
	}
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

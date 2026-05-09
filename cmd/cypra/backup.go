package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	cypra "github.com/watzon/cypra/internal/crypto"
)

const backupFormatVersion = "cypra-backup-v2"

var backupTables = []string{
	"tenants",
	"projects",
	"instance_admins",
	"storage_objects",
	"users",
	"tenant_memberships",
	"password_credentials",
	"passkey_credentials",
	"totp_credentials",
	"user_backup_codes",
	"instance_admin_backup_codes",
	"magic_link_tokens",
	"password_reset_tokens",
	"email_verification_tokens",
	"pending_invitations",
	"sessions",
	"instance_admin_sessions",
	"oidc_clients",
	"oidc_authorization_codes",
	"oidc_refresh_tokens",
	"oidc_consents",
	"oidc_signing_keys",
	"upstream_providers",
	"social_connections",
	"oidc_connections",
	"email_provider_configs",
	"email_outbox",
	"tenant_auth_methods",
	"audit_entries",
	"bootstrap_tokens",
	"master_key_rotations",
	"rate_limit_buckets",
	"gdpr_deletions",
	"webauthn_challenges",
	"invite_continuations",
}

type backupEnvelope struct {
	Format     string `json:"format"`
	KDF        string `json:"kdf"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type backupPayload struct {
	Format        string                       `json:"format"`
	ExportedAt    time.Time                    `json:"exported_at"`
	Tables        map[string][]json.RawMessage `json:"tables"`
	Storage       []backupStorageObject        `json:"storage"`
	EncryptedCols []string                     `json:"encrypted_columns"`
}

type backupStorageObject struct {
	Key     string `json:"key"`
	Content string `json:"content_base64"`
}

func exportBackup(ctx context.Context, dbConn *sql.DB, passphrase string) ([]byte, error) {
	payload := backupPayload{Format: backupFormatVersion, ExportedAt: time.Now().UTC(), Tables: map[string][]json.RawMessage{}, EncryptedCols: encryptedColumnNames()}
	for _, table := range backupTables {
		rows, err := dumpTable(ctx, dbConn, table)
		if err != nil {
			return nil, err
		}
		payload.Tables[table] = rows
	}
	if err := rewrapBackupTables(payload.Tables, loadMasterKey(), backupKey(passphrase)); err != nil {
		return nil, err
	}
	storage, err := dumpLocalStorage(ctx, dbConn)
	if err != nil {
		return nil, err
	}
	payload.Storage = storage
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return sealBackup(plaintext, passphrase)
}

func importBackup(ctx context.Context, dbConn *sql.DB, content []byte, passphrase string, allowResurrect bool) error {
	plaintext, err := openBackup(content, passphrase)
	if err != nil {
		return err
	}
	var payload backupPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return err
	}
	if payload.Format != backupFormatVersion {
		return exitError{code: 2, error: "unsupported import format"}
	}
	if err := rewrapBackupTables(payload.Tables, backupKey(passphrase), loadMasterKey()); err != nil {
		return err
	}
	if len(payload.Tables["gdpr_deletions"]) > 0 && !allowResurrect {
		return exitError{code: 9, error: "import refuses to resurrect DSR-deleted users without --allow-resurrect"}
	}
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for i := len(backupTables) - 1; i >= 0; i-- {
		if _, err := tx.ExecContext(ctx, `TRUNCATE TABLE `+backupTables[i]+` CASCADE`); err != nil {
			return fmt.Errorf("truncate %s: %w", backupTables[i], err)
		}
	}
	for _, table := range backupTables {
		for _, row := range payload.Tables[table] {
			stmt := `INSERT INTO ` + table + ` SELECT * FROM json_populate_record(NULL::` + table + `, $1::json)` // #nosec G202 -- table names come from backupTables allowlist.
			if _, err := tx.ExecContext(ctx, stmt, string(row)); err != nil {
				return fmt.Errorf("restore %s: %w", table, err)
			}
		}
	}
	if allowResurrect {
		if _, err := tx.ExecContext(ctx, `INSERT INTO audit_entries (tenant_id, actor_kind, action, resource_kind, metadata) VALUES (NULL, 'system', 'backup.import_allow_resurrect', 'backup', '{"allow_resurrect":true}'::jsonb)`); err != nil {
			return fmt.Errorf("write resurrect audit: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return restoreLocalStorage(payload.Storage)
}

func dumpTable(ctx context.Context, dbConn *sql.DB, table string) ([]json.RawMessage, error) {
	stmt := `SELECT to_jsonb(t) FROM ` + table + ` t` // #nosec G202 -- table names come from backupTables allowlist.
	rows, err := dbConn.QueryContext(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("dump %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var result []json.RawMessage
	for rows.Next() {
		var row json.RawMessage
		if err := rows.Scan(&row); err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func dumpLocalStorage(ctx context.Context, dbConn *sql.DB) ([]backupStorageObject, error) {
	rows, err := dbConn.QueryContext(ctx, `SELECT key FROM storage_objects WHERE backend = 'local-disk' AND deleted_at IS NULL ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	root := storageLocalPathCLI()
	var objects []backupStorageObject
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		path, err := storagePath(root, key)
		if err != nil {
			return nil, err
		}
		content, err := os.ReadFile(path) // #nosec G304 -- path is constrained by storagePath.
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		objects = append(objects, backupStorageObject{Key: key, Content: base64.StdEncoding.EncodeToString(content)})
	}
	return objects, rows.Err()
}

func restoreLocalStorage(objects []backupStorageObject) error {
	root := storageLocalPathCLI()
	for _, object := range objects {
		content, err := base64.StdEncoding.DecodeString(object.Content)
		if err != nil {
			return err
		}
		path, err := storagePath(root, object.Key)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func sealBackup(plaintext []byte, passphrase string) ([]byte, error) {
	key := backupKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	envelope := backupEnvelope{Format: backupFormatVersion, KDF: "sha256-passphrase-v1", Nonce: base64.StdEncoding.EncodeToString(nonce), Ciphertext: base64.StdEncoding.EncodeToString(aead.Seal(nil, nonce, plaintext, nil))}
	return json.Marshal(envelope)
}

func openBackup(content []byte, passphrase string) ([]byte, error) {
	var envelope backupEnvelope
	if err := json.Unmarshal(content, &envelope); err != nil {
		return nil, err
	}
	if envelope.Format != backupFormatVersion {
		return nil, exitError{code: 2, error: "unsupported import format"}
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, exitError{code: 2, error: "invalid backup envelope"}
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, exitError{code: 2, error: "invalid backup envelope"}
	}
	block, err := aes.NewCipher(backupKey(passphrase))
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, exitError{code: 7, error: "backup passphrase invalid"}
	}
	return plaintext, nil
}

func backupKey(passphrase string) []byte {
	digest := sha256.Sum256([]byte(passphrase))
	return digest[:]
}

func storagePath(root, key string) (string, error) {
	cleaned := filepath.Clean("/" + key)
	if cleaned == "/" || cleaned == "." || len(cleaned) >= 2 && cleaned[:2] == ".." {
		return "", fmt.Errorf("invalid storage key")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(absRoot, cleaned[1:])
	if filepath.Dir(path) == "." || len(path) < len(absRoot) || path[:len(absRoot)] != absRoot {
		return "", fmt.Errorf("invalid storage key")
	}
	return path, nil
}

func storageLocalPathCLI() string {
	path := os.Getenv("STORAGE_LOCAL_PATH")
	if path == "" {
		return "cypra-storage"
	}
	return path
}

func encryptedColumnNames() []string {
	targets := encryptedColumnTargets()
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Table+"."+target.DEKColumn+":"+target.Format)
	}
	return names
}

func rewrapBackupTables(tables map[string][]json.RawMessage, oldKEK, newKEK []byte) error {
	if len(oldKEK) != cypra.MasterKeyBytes || len(newKEK) != cypra.MasterKeyBytes {
		for _, target := range encryptedColumnTargets() {
			if len(tables[target.Table]) > 0 {
				return cypra.ErrInvalidMasterKey
			}
		}
		return nil
	}
	for _, target := range encryptedColumnTargets() {
		rows := tables[target.Table]
		for index, raw := range rows {
			var row map[string]any
			if err := json.Unmarshal(raw, &row); err != nil {
				return err
			}
			value, _ := row[target.DEKColumn].(string)
			if value == "" {
				continue
			}
			decoded, err := decodeJSONBytea(value)
			if err != nil {
				return err
			}
			rewrapped, err := rewrapBackupValue(decoded, target.Format, oldKEK, newKEK)
			if err != nil {
				return err
			}
			row[target.DEKColumn] = encodeJSONBytea(rewrapped)
			updated, err := json.Marshal(row)
			if err != nil {
				return err
			}
			rows[index] = updated
		}
		tables[target.Table] = rows
	}
	return nil
}

func rewrapBackupValue(value []byte, format string, oldKEK, newKEK []byte) ([]byte, error) {
	switch format {
	case "", cypra.EncryptedColumnFormatDEK:
		return cypra.RewrapDEK(value, oldKEK, newKEK)
	case cypra.EncryptedColumnFormatPrefix:
		if len(value) < 60 {
			return nil, cypra.ErrDecryptFailed
		}
		rewrapped, err := cypra.RewrapDEK(value[:60], oldKEK, newKEK)
		if err != nil {
			return nil, err
		}
		out := append([]byte{}, rewrapped...)
		return append(out, value[60:]...), nil
	case cypra.EncryptedColumnFormatJSONEnvelope:
		var envelope struct {
			Ciphertext   string `json:"ciphertext"`
			EncryptedDEK string `json:"encrypted_dek"`
		}
		if err := json.Unmarshal(value, &envelope); err != nil {
			return nil, err
		}
		dek, err := base64.RawStdEncoding.DecodeString(envelope.EncryptedDEK)
		if err != nil {
			return nil, err
		}
		rewrapped, err := cypra.RewrapDEK(dek, oldKEK, newKEK)
		if err != nil {
			return nil, err
		}
		envelope.EncryptedDEK = base64.RawStdEncoding.EncodeToString(rewrapped)
		return json.Marshal(envelope)
	default:
		return nil, fmt.Errorf("unsupported encrypted-column format %q", format)
	}
}

func decodeJSONBytea(value string) ([]byte, error) {
	if len(value) >= 2 && value[:2] == `\x` {
		return hex.DecodeString(value[2:])
	}
	return base64.StdEncoding.DecodeString(value)
}

func encodeJSONBytea(value []byte) string {
	return `\x` + hex.EncodeToString(value)
}

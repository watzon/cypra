package crypto_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
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

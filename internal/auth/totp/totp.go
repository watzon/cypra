// Package totp implements TOTP enrollment and verification.
package totp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- TOTP uses HMAC-SHA1 by default per RFC 6238.
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/crypto"
)

//revive:disable:exported

type Service struct {
	DB  *sql.DB
	KEK []byte
}

func (s Service) Enroll(ctx context.Context, tenantID, userID uuid.UUID, issuer, account string) (string, string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", "", err
	}
	ciphertext, encryptedDEK, err := crypto.Encrypt(secret, s.KEK)
	if err != nil {
		return "", "", err
	}
	stored := append(encryptedDEK, ciphertext...)
	if _, err := s.DB.ExecContext(ctx, `INSERT INTO totp_credentials (tenant_id, user_id, secret_encrypted, algorithm, digits, period_seconds) VALUES ($1, $2, $3, 'SHA1', 6, 30)`, tenantID, userID, stored); err != nil {
		return "", "", fmt.Errorf("insert totp credential: %w", err)
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	uri := "otpauth://totp/" + url.PathEscape(issuer+":"+account) + "?secret=" + encoded + "&issuer=" + url.QueryEscape(issuer) + "&period=30&digits=6"
	return encoded, uri, nil
}

func (s Service) Verify(ctx context.Context, userID uuid.UUID, code string, now time.Time) (bool, error) {
	var stored []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT secret_encrypted FROM totp_credentials WHERE user_id = $1 ORDER BY confirmed_at NULLS FIRST LIMIT 1`, userID).Scan(&stored); err != nil {
		return false, err
	}
	if len(stored) < 60 {
		return false, fmt.Errorf("invalid totp secret")
	}
	secret, err := crypto.Decrypt(stored[60:], stored[:60], s.KEK)
	if err != nil {
		return false, err
	}
	for drift := -1; drift <= 1; drift++ {
		if GenerateCode(secret, now.Add(time.Duration(drift)*30*time.Second)) == strings.TrimSpace(code) {
			_, _ = s.DB.ExecContext(ctx, `UPDATE totp_credentials SET confirmed_at = COALESCE(confirmed_at, now()) WHERE user_id = $1`, userID)
			return true, nil
		}
	}
	return false, nil
}

func GenerateCode(secret []byte, at time.Time) string {
	counter := uint64(math.Floor(float64(at.Unix()) / 30))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (int(sum[offset])&0x7f)<<24 | (int(sum[offset+1])&0xff)<<16 | (int(sum[offset+2])&0xff)<<8 | (int(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", value%1000000)
}

// Package email sends transactional email through pluggable backends.
package email

//revive:disable:exported

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/crypto"
)

var (
	ErrProviderRequired = errors.New("tenant.email_provider_required")
	ErrTerminalDisabled = errors.New("tenant.email_terminal_disabled")
)

type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type TerminalSender struct{ Writer io.Writer }

func (s TerminalSender) Send(_ context.Context, message Message) error {
	_, err := fmt.Fprintf(s.Writer, "To: %s\nSubject: %s\n\n%s\n", message.To, message.Subject, message.Text)
	return err
}

type SMTPSender struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"-"`
	FromName    string `json:"-"`
}

func (s SMTPSender) Send(_ context.Context, message Message) error {
	address := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	headers := []string{
		"From: " + formatAddress(s.FromName, s.FromAddress),
		"To: " + message.To,
		"Subject: " + message.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
	}
	body := strings.Join(headers, "\r\n") + "\r\n\r\n" + message.Text
	return smtp.SendMail(address, auth, s.FromAddress, []string{message.To}, []byte(body))
}

type ResendSender struct {
	APIKey      string `json:"api_key"`
	Endpoint    string `json:"endpoint"`
	FromAddress string `json:"-"`
	FromName    string `json:"-"`
	HTTPClient  *http.Client
}

func (s ResendSender) Send(ctx context.Context, message Message) error {
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "https://api.resend.com/emails"
	}
	payload := map[string]any{"from": formatAddress(s.FromName, s.FromAddress), "to": []string{message.To}, "subject": message.Subject, "text": message.Text, "html": message.HTML}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("resend email status %d", resp.StatusCode)
	}
	return nil
}

type Resolver struct {
	DB             *sql.DB
	KEK            []byte
	TerminalWriter io.Writer
	BlockTerminal  bool
}

func (r Resolver) Resolve(ctx context.Context, tenantID uuid.UUID) (Sender, error) {
	var kind string
	var encrypted []byte
	var fromAddress string
	var fromName string
	err := r.DB.QueryRowContext(ctx, `SELECT kind, config_encrypted, from_address, from_name FROM email_provider_configs WHERE tenant_id = $1 AND enabled = true LIMIT 1`, tenantID).Scan(&kind, &encrypted, &fromAddress, &fromName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProviderRequired
	}
	if err != nil {
		return nil, err
	}
	if kind == "terminal" {
		if r.BlockTerminal {
			return nil, ErrTerminalDisabled
		}
		return TerminalSender{Writer: r.TerminalWriter}, nil
	}
	configJSON, err := r.decryptConfig(encrypted)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "smtp":
		var sender SMTPSender
		if err := json.Unmarshal(configJSON, &sender); err != nil {
			return nil, err
		}
		sender.FromAddress = fromAddress
		sender.FromName = fromName
		return sender, nil
	case "resend":
		var sender ResendSender
		if err := json.Unmarshal(configJSON, &sender); err != nil {
			return nil, err
		}
		sender.FromAddress = fromAddress
		sender.FromName = fromName
		return sender, nil
	default:
		return nil, fmt.Errorf("unsupported email provider %q", kind)
	}
}

func EncryptConfig(config []byte, kek []byte) ([]byte, error) {
	ciphertext, encryptedDEK, err := crypto.Encrypt(config, kek)
	if err != nil {
		return nil, err
	}
	return append(encryptedDEK, ciphertext...), nil
}

func (r Resolver) decryptConfig(stored []byte) ([]byte, error) {
	if len(stored) == 0 {
		return []byte(`{}`), nil
	}
	if len(r.KEK) == 0 {
		return stored, nil
	}
	if len(stored) < 60 {
		return nil, fmt.Errorf("invalid email provider config")
	}
	return crypto.Decrypt(stored[60:], stored[:60], r.KEK)
}

func formatAddress(name, address string) string {
	if strings.TrimSpace(name) == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

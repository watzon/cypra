package httpserver

import (
	"github.com/watzon/cypra/internal/crypto"
)

func (s *Server) encryptProviderSecret(plaintext []byte) ([]byte, error) {
	if len(s.MasterKey) == 0 {
		return plaintext, nil
	}
	ciphertext, encryptedDEK, err := crypto.Encrypt(plaintext, s.MasterKey)
	if err != nil {
		return nil, err
	}
	return append(encryptedDEK, ciphertext...), nil
}

func (s *Server) decryptProviderSecret(stored []byte) ([]byte, error) {
	if len(s.MasterKey) == 0 || len(stored) < 60 {
		return stored, nil
	}
	return crypto.Decrypt(stored[60:], stored[:60], s.MasterKey)
}

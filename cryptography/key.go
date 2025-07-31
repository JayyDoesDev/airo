package cryptography

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateKey(length int) (string, error) {
	if length != 16 && length != 24 && length != 32 {
		return "", fmt.Errorf("invalid key length: must be 16, 24, or 32 bytes")
	}

	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}

	return hex.EncodeToString(key), nil
}

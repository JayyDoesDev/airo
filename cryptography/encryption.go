package cryptography

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"sync"
)

var (
	aesKey []byte
	once   sync.Once
)

func GetAESKey() []byte {
	once.Do(func() {
		keyHex := os.Getenv("KEY")
		if keyHex == "" {
			log.Fatal("KEY env var is not set")
		}

		key, err := DecodeHexKey(keyHex)
		if err != nil {
			log.Fatalf("failed to decode KEY: %v", err)
		}
		aesKey = key
	})
	return aesKey
}

func Encrypt(d []byte, k []byte) ([]byte, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return append(nonce, gcm.Seal(nil, nonce, d, nil)...), nil
}

func Decrypt(e []byte, k []byte) ([]byte, error) {
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, cipherText := e[:nonceSize], e[nonceSize:]

	return gcm.Open(nil, nonce, cipherText, nil)
}

func DecodeHexKey(hexStr string) ([]byte, error) {
	key, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}

	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, errors.New("invalid AES key size (must be 16, 24, or 32 bytes)")
	}

	return key, nil
}

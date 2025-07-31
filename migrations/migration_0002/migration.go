package migration_0002

import (
	"fmt"
	"os"

	"github.com/jayydoesdev/airo/bot/cryptography"
)

func Migrate() (bool, error) {
	msgMem, err := os.ReadFile("memory.msgpack")
	if err != nil {
		return false, fmt.Errorf("failed to read memory.msgpack: %w", err)
	}

	keyStr := os.Getenv("KEY")
	if keyStr == "" {
		return false, fmt.Errorf("KEY env var is not set")
	}

	key, err := cryptography.DecodeHexKey(keyStr)
	if err != nil {
		return false, fmt.Errorf("failed to decode KEY: %w", err)
	}

	encrypted, err := cryptography.Encrypt(msgMem, key)
	if err != nil {
		return false, fmt.Errorf("failed to encrypt memory: %w", err)
	}

	if err := os.WriteFile("memory.msgpack", encrypted, 0644); err != nil {
		return false, fmt.Errorf("failed to write encrypted memory: %w", err)
	}

	return true, nil
}

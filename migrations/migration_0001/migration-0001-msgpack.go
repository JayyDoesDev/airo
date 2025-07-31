package migration_0001

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/vmihailenco/msgpack/v5"
)

func Migrate() (bool, error) {
	var memory lib.Memory

	jsonMem, err := os.ReadFile("memory.json")
	if err != nil {
		return false, fmt.Errorf("error reading memory.json: %w", err)
	}

	if err := json.Unmarshal(jsonMem, &memory); err != nil {
		return false, fmt.Errorf("error parsing JSON: %w", err)
	}

	data, err := msgpack.Marshal(memory)
	if err != nil {
		return false, fmt.Errorf("failed to encode msgpack: %w", err)
	}

	if err := os.WriteFile("memory.msgpack", data, 0644); err != nil {
		return false, fmt.Errorf("failed to write memory.msgpack: %w", err)
	}

	return true, nil
}

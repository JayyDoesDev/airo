package actions

import (
	"os"
	"testing"

	"github.com/jayydoesdev/airo/bot/cryptography"
	"github.com/vmihailenco/msgpack/v5"
)

func writeTempMemory(t *testing.T, mem Memory) string {
	t.Helper()
	f, err := os.CreateTemp("", "memory_test_*.msgpack")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	data, err := msgpack.Marshal(mem)
	if err != nil {
		t.Fatal(err)
	}
	key := cryptography.GetAESKey()
	enc, err := cryptography.Encrypt(data, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.Name(), enc, 0644); err != nil {
		t.Fatal(err)
	}
	return f.Name()
}

func TestQueryMemoriesByTagKeywordFallback(t *testing.T) {
	imp := float32(0.9)
	mem := Memory{
		LongTerm: []MemoryItem{
			{Id: "1", Title: "jinx directive", Content: "don't dm jinx", Importance: imp},
			{Id: "2", Title: "server rules", Content: "be nice", Importance: 0.5},
			{Id: "3", Title: "jinx is annoying", Content: "she keeps asking for DMs", Importance: 0.8},
		},
	}

	path := writeTempMemory(t, mem)
	defer os.Remove(path)

	origRead := os.ReadFile
	_ = origRead

	orig, _ := os.ReadFile("memory.msgpack")
	defer func() {
		if orig != nil {
			os.WriteFile("memory.msgpack", orig, 0644)
		} else {
			os.Remove("memory.msgpack")
		}
	}()

	data, _ := os.ReadFile(path)
	os.WriteFile("memory.msgpack", data, 0644)

	results, err := QueryMemoriesByTag("jinx")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 jinx memories, got %d", len(results))
	}
	for _, r := range results {
		if r.Value == nil {
			t.Errorf("memory %q has nil value (importance should be set as value)", r.Id)
		}
	}
}

func TestQueryMemoriesByTagExplicit(t *testing.T) {
	v1, v2 := float64(120.0), float64(250.0)
	mem := Memory{
		LongTerm: []MemoryItem{
			{Id: "1", Title: "ping jan", Tag: "latency", Value: &v1, Importance: 0.6},
			{Id: "2", Title: "ping feb", Tag: "latency", Value: &v2, Importance: 0.6},
			{Id: "3", Title: "unrelated", Tag: "other", Importance: 0.5},
		},
	}

	path := writeTempMemory(t, mem)
	defer os.Remove(path)

	orig, _ := os.ReadFile("memory.msgpack")
	defer func() {
		if orig != nil {
			os.WriteFile("memory.msgpack", orig, 0644)
		} else {
			os.Remove("memory.msgpack")
		}
	}()

	data, _ := os.ReadFile(path)
	os.WriteFile("memory.msgpack", data, 0644)

	results, err := QueryMemoriesByTag("latency")
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 latency memories, got %d", len(results))
	}
}

package lib

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/jayydoesdev/airo/bot/cryptography"
	"github.com/vmihailenco/msgpack/v5"
)

type Memory struct {
	ShortTerm []MemoryItem `msgpack:"shortTerm,omitempty"`
	LongTerm  []MemoryItem `msgpack:"longTerm,omitempty"`
	Topics    []Topic      `msgpack:"topics,omitempty"`
	Meta      MemoryMeta   `msgpack:"meta"`
}

type MemoryItem struct {
	Id           string             `msgpack:"id"`
	Title        string             `msgpack:"title"`
	Content      string             `msgpack:"content"`
	Type         string             `msgpack:"type"`
	Source       string             `msgpack:"source"`
	Importance   float32            `msgpack:"importance"`
	Created      string             `msgpack:"created"`
	Lastaccessed string             `msgpack:"lastAccessed"`
	Related      []string           `msgpack:"related,omitempty"`
	Context      *MemoryItemContext `msgpack:"context"`
}

type MemoryItemContext struct {
	Location string `msgpack:"location"`
	Author   string `msgpack:"author"`
}

type Topic struct {
	Name             string   `msgpack:"name"`
	Description      string   `msgpack:"description"`
	RelatedMemoryIDs []string `msgpack:"relatedMemoryIds"`
}

type MemoryMeta struct {
	Lastupdated   string   `msgpack:"lastUpdated"`
	Totalmemories int      `msgpack:"TotalMemories"`
	PriorityQueue []string `msgpack:"priorityQueue,omitempty"`
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func CreateMemory(newItem MemoryItem) {

	StoreToMemory(newItem)

	fmt.Println("Memory item appended successfully.")
}

func GetMemory(file string) (Memory, error) {
	var memory Memory

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return memory, nil
		}

		return memory, fmt.Errorf("failed to read memory: %w", err)
	}

	if len(data) == 0 {
		return memory, nil
	}
	key := cryptography.GetAESKey()
	decrypted, err := cryptography.Decrypt(data, key)
	if err != nil {
		return Memory{}, fmt.Errorf("failed to decrypt memory: %w", err)
	}

	if err := msgpack.Unmarshal(decrypted, &memory); err != nil {
		return memory, fmt.Errorf("failed to parse memory: %w", err)
	}

	return memory, nil
}

func SaveMemory(mem Memory) error {
	return SaveMemoryToFile("memory.msgpack", mem)
}

func SaveMemoryToFile(filename string, mem Memory) error {
	data, err := msgpack.Marshal(mem)
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}
	key := cryptography.GetAESKey()
	encrypted, err := cryptography.Encrypt(data, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt memory: %w", err)
	}

	if err := os.WriteFile(filename, encrypted, 0644); err != nil {
		return fmt.Errorf("failed to write memory: %w", err)
	}

	return nil
}

func StoreToMemory(item MemoryItem) error {
	mem, err := GetMemory("memory.msgpack")
	if err != nil {
		return err
	}

	if item.Importance >= 0.5 {
		mem.Meta.PriorityQueue = append(mem.Meta.PriorityQueue, item.Id)
	}

	if item.Importance >= 0.7 {
		mem.LongTerm = append(mem.LongTerm, item)
	} else {
		mem.ShortTerm = append(mem.ShortTerm, item)
	}

	mem.Meta.Totalmemories++
	mem.Meta.Lastupdated = time.Now().Format(time.RFC3339)

	return SaveMemoryToFile("memory.msgpack", mem)
}

func GetRelevantMemory(author, location string) ([]MemoryItem, error) {
	mem, err := GetMemory("memory.msgpack")
	if err != nil {
		return nil, err
	}

	var relevant []MemoryItem
	for _, item := range mem.ShortTerm {
		if item.Context != nil &&
			item.Context.Author == author &&
			(location == "" || item.Context.Location == location) {
			relevant = append(relevant, item)
		}
	}

	return relevant, nil
}

func GenerateID() string {
	result := make([]byte, 20)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return ""
		}
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

func SummarizeMemories(memories []MemoryItem, label string) string {
	if len(memories) == 0 {
		return fmt.Sprintf("%s: none.\n", label)
	}
	var sb strings.Builder
	sb.WriteString(label + ":\n")
	for _, mem := range memories {
		if mem.Importance < 0.1 {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", mem.Title, mem.Content))
	}
	return sb.String()
}

func GetSummarizedMemory(author, location string) (string, error) {
	mem, err := GetMemory("memory.msgpack")
	if err != nil {
		return "", err
	}

	var relevantShort, relevantLong []MemoryItem
	for _, item := range mem.ShortTerm {
		if item.Context != nil && item.Context.Author == author &&
			(location == "" || item.Context.Location == location) {
			relevantShort = append(relevantShort, item)
		}
	}
	for _, item := range mem.LongTerm {
		if item.Context != nil && item.Context.Author == author &&
			(location == "" || item.Context.Location == location) {
			relevantLong = append(relevantLong, item)
		}
	}

	return SummarizeMemories(relevantLong, "Long-term memories") + "\n" +
		SummarizeMemories(relevantShort, "Recent memories"), nil
}

func PurgeAndStoreShortTermMemory() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mem, err := GetMemory("memory.msgpack")
		if err != nil {
			fmt.Println("Failed to read memory for purge:", err)
			continue
		}

		var newShortTerm []MemoryItem
		changed := false

		for _, m := range mem.ShortTerm {
			createdTime, _ := time.Parse(time.RFC3339, m.Created)
			age := time.Since(createdTime)

			switch {
			case m.Importance >= 0.7:
				mem.LongTerm = append(mem.LongTerm, m)
				mem.Meta.PriorityQueue = append(mem.Meta.PriorityQueue, m.Id)
				changed = true

			case m.Importance < 0.3 && age > (24*time.Hour):
				mem.Meta.Totalmemories--
				changed = true

			default:
				newShortTerm = append(newShortTerm, m)
			}
		}

		if changed {
			mem.ShortTerm = newShortTerm
			mem.Meta.Lastupdated = time.Now().Format(time.RFC3339)

			if err := SaveMemoryToFile("memory.msgpack", mem); err != nil {
				fmt.Println("Failed to save memory after purge:", err)
			} else {
				fmt.Println("Memory purge completed at", time.Now().Format(time.RFC822))
			}
		}
	}
}

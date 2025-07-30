package lib

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"
)

type Memory struct {
	ShortTerm []MemoryItem `json:"shortTerm,omitempty"`
	LongTerm  []MemoryItem `json:"longTerm,omitempty"`
	Topics    []Topic      `json:"topics,omitempty"`
	Meta      MemoryMeta   `json:"meta"`
}

type MemoryItem struct {
	Id           string             `json:"id"`
	Title        string             `json:"title"`
	Content      string             `json:"content"`
	Type         string             `json:"type"`
	Source       string             `json:"source"`
	Importance   float32            `json:"importance"`
	Created      string             `json:"created"`
	Lastaccessed string             `json:"lastAccessed"`
	Related      []string           `json:"related,omitempty"`
	Context      *MemoryItemContext `json:"context"`
}

type MemoryItemContext struct {
	Location string `json:"location"`
	Author   string `json:"author"`
}

type Topic struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	RelatedMemoryIDs []string `json:"relatedMemoryIds"`
}

type MemoryMeta struct {
	Lastupdated   string   `json:"lastUpdated"`
	Totalmemories int      `json:"TotalMemories"`
	PriorityQueue []string `json:"priorityQueue,omitempty"`
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func CreateMemory(newItem MemoryItem) {
	data, err := os.ReadFile("memory.json")
	if err != nil {
		fmt.Printf("Failed to read memory.json: %v\n", err)
		return
	}

	var memory Memory
	if err := json.Unmarshal(data, &memory); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		return
	}

	memory.ShortTerm = append(memory.ShortTerm, newItem)

	updatedData, err := json.MarshalIndent(memory, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal JSON: %v\n", err)
		return
	}

	if err := os.WriteFile("memory.json", updatedData, 0644); err != nil {
		fmt.Printf("Failed to write memory.json: %v\n", err)
		return
	}

	fmt.Println("Memory item appended successfully.")
}

func GetMemory() (Memory, error) {
	var memory Memory

	data, err := os.ReadFile("memory.json")
	if err != nil {
		if os.IsNotExist(err) {
			return memory, nil
		}
		return memory, fmt.Errorf("failed to read memory.json: %w", err)
	}

	if len(data) == 0 {
		return memory, nil
	}

	if err := json.Unmarshal(data, &memory); err != nil {
		return memory, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return memory, nil
}

func SaveMemory(mem Memory) error {
	updatedData, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}

	if err := os.WriteFile("memory.json", updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write memory.json: %w", err)
	}

	return nil
}

func StoreToMemory(item MemoryItem) error {
	mem, err := GetMemory()
	if err != nil {
		return err
	}

	if item.Importance >= 0.7 {
		mem.LongTerm = append(mem.LongTerm, item)
	} else {
		mem.ShortTerm = append(mem.ShortTerm, item)
	}

	mem.Meta.Totalmemories++
	mem.Meta.Lastupdated = time.Now().Format(time.RFC3339)

	return SaveMemory(mem)
}

func GetRelevantMemory(author, location string) ([]MemoryItem, error) {
	mem, err := GetMemory()
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
	mem, err := GetMemory()
	if err != nil {
		return "", err
	}

	var relevantShort []MemoryItem
	for _, item := range mem.ShortTerm {
		if item.Context != nil && item.Context.Author == author {
			if location == "" || item.Context.Location == location {
				relevantShort = append(relevantShort, item)
			}
		}
	}

	var relevantLong []MemoryItem
	for _, item := range mem.LongTerm {
		if item.Context != nil && item.Context.Author == author {
			if location == "" || item.Context.Location == location {
				relevantLong = append(relevantLong, item)
			}
		}
	}

	longText := SummarizeMemories(relevantLong, "Long-term memories")
	shortText := SummarizeMemories(relevantShort, "Recent memories")

	return longText + "\n" + shortText, nil
}

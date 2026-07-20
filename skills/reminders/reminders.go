package reminders

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const reminderFile = "reminders.json"

type Reminder struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ChannelID string    `json:"channel_id"`
	Message   string    `json:"message"`
	FireAt    time.Time `json:"fire_at"`
}

var (
	mu        sync.Mutex
	reminders []Reminder
)

func load() {
	data, err := os.ReadFile(reminderFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &reminders)
}

func save() {
	data, err := json.Marshal(reminders)
	if err != nil {
		return
	}
	os.WriteFile(reminderFile, data, 0644)
}

func Add(r Reminder) {
	mu.Lock()
	defer mu.Unlock()
	reminders = append(reminders, r)
	save()
}

func Start(s *discordgo.Session) {
	load()
	loadPolls()
	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			fire(s)
			firePolls(s)
		}
	}()
}

func fire(s *discordgo.Session) {
	mu.Lock()
	defer mu.Unlock()
	now := time.Now()
	var remaining []Reminder
	for _, r := range reminders {
		if now.Before(r.FireAt) {
			remaining = append(remaining, r)
			continue
		}
		msg := fmt.Sprintf("<@%s> ⏰ %s", r.UserID, r.Message)
		if _, err := s.ChannelMessageSend(r.ChannelID, msg); err != nil {
			fmt.Println("[reminder] failed to send:", err)
			remaining = append(remaining, r)
		}
	}
	if len(remaining) != len(reminders) {
		reminders = remaining
		save()
	}
}

package reminders

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

const pollFile = "polls.json"

var PollEmojis = []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}

type TimedPoll struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	ChannelID string    `json:"channel_id"`
	Question  string    `json:"question"`
	Options   []string  `json:"options"`
	CloseAt   time.Time `json:"close_at"`
}

var (
	pollMu     sync.Mutex
	timedPolls []TimedPoll
)

func loadPolls() {
	data, err := os.ReadFile(pollFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &timedPolls)
}

func savePolls() {
	data, err := json.Marshal(timedPolls)
	if err != nil {
		return
	}
	os.WriteFile(pollFile, data, 0644)
}

func AddPoll(p TimedPoll) {
	pollMu.Lock()
	defer pollMu.Unlock()
	timedPolls = append(timedPolls, p)
	savePolls()
}

func firePolls(s *discordgo.Session) {
	pollMu.Lock()
	defer pollMu.Unlock()
	now := time.Now()
	var remaining []TimedPoll
	for _, p := range timedPolls {
		if now.Before(p.CloseAt) {
			remaining = append(remaining, p)
			continue
		}
		if err := postPollResults(s, p); err != nil {
			fmt.Println("[poll] failed to post results:", err)
			remaining = append(remaining, p)
		}
	}
	if len(remaining) != len(timedPolls) {
		timedPolls = remaining
		savePolls()
	}
}

func postPollResults(s *discordgo.Session, p TimedPoll) error {
	type optionResult struct {
		label string
		votes int
	}
	results := make([]optionResult, len(p.Options))
	total := 0
	for i, opt := range p.Options {
		if i >= len(PollEmojis) {
			break
		}
		users, err := s.MessageReactions(p.ChannelID, p.MessageID, PollEmojis[i], 100, "", "")
		if err != nil {
			return err
		}
		votes := len(users)
		if votes > 0 {
			votes--
		}
		results[i] = optionResult{label: opt, votes: votes}
		total += votes
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 **Poll closed: %s**\n\n", p.Question))
	for i, r := range results {
		if i >= len(PollEmojis) {
			break
		}
		pct := 0.0
		if total > 0 {
			pct = float64(r.votes) / float64(total) * 100
		}
		bar := progressBar(pct, 10)
		sb.WriteString(fmt.Sprintf("%s **%s** — %d vote", PollEmojis[i], r.label, r.votes))
		if r.votes != 1 {
			sb.WriteString("s")
		}
		sb.WriteString(fmt.Sprintf(" (%s %.0f%%)\n", bar, pct))
	}
	sb.WriteString(fmt.Sprintf("\n**Total: %d vote", total))
	if total != 1 {
		sb.WriteString("s")
	}
	sb.WriteString("**")

	ref := &discordgo.MessageReference{MessageID: p.MessageID, ChannelID: p.ChannelID}
	_, err := s.ChannelMessageSendReply(p.ChannelID, sb.String(), ref)
	return err
}

func progressBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

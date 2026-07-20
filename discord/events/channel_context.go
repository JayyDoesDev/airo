package events

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func buildChannelContext(s *discordgo.Session, channelID, currentMsgID string) string {
	msgs, err := s.ChannelMessages(channelID, 15, currentMsgID, "", "")
	if err != nil || len(msgs) == 0 {
		return ""
	}

	var humanMsgs []*discordgo.Message
	for _, msg := range msgs {
		if !msg.Author.Bot {
			humanMsgs = append(humanMsgs, msg)
		}
	}
	if len(humanMsgs) == 0 {
		return ""
	}

	lastMsg := humanMsgs[0]
	lastTime := lastMsg.Timestamp
	if lastTime.IsZero() {
		return ""
	}
	silenceDur := time.Since(lastTime)

	var activity string
	switch {
	case silenceDur > 2*time.Hour:
		activity = fmt.Sprintf("very quiet — last message was %.0f hours ago", silenceDur.Hours())
	case silenceDur > 30*time.Minute:
		activity = fmt.Sprintf("quiet — last message was %.0f minutes ago", silenceDur.Minutes())
	case len(humanMsgs) >= 10:
		activity = "very active conversation"
	case len(humanMsgs) >= 5:
		activity = "active conversation"
	default:
		activity = "light conversation"
	}

	uniqueUsers := map[string]struct{}{}
	for _, msg := range humanMsgs {
		uniqueUsers[msg.Author.ID] = struct{}{}
	}

	return fmt.Sprintf("\n[Channel context: %s, %d people talking]", activity, len(uniqueUsers))
}

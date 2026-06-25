package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/skills/actions"
)

const vibeCheckTrigger = "VIBE_CHECK"

type VibeCheckOpts struct {
	Response   string
	Client     lib.LibClient
	FullPrompt string
	Memory     actions.Memory
	Session    *discordgo.Session
	Message    *discordgo.MessageCreate
	Guild      *discordgo.Guild
}

func HandleVibeCheck(opts VibeCheckOpts) (string, bool) {
	if !strings.Contains(opts.Response, vibeCheckTrigger) {
		return opts.Response, false
	}

	msgs, err := opts.Session.ChannelMessages(opts.Message.ChannelID, 40, opts.Message.ID, "", "")
	if err != nil || len(msgs) == 0 {
		return opts.Response, true
	}

	var sb strings.Builder
	sb.WriteString("Here are the last messages from this channel (oldest to newest):\n\n")
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Author.Bot || m.Content == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", m.Author.Username, m.Content))
	}
	sb.WriteString("\nBased on these messages, give an unhinged, brutally honest vibe check of this server's energy right now. Stay in character. Still respond with the required JSON block.")

	newResp, err := opts.Client.Send(
		opts.Message.Author.ID, opts.Message.Author.Username, *opts.Guild,
		sb.String(),
		opts.Memory,
	)
	if err != nil {
		return opts.Response, true
	}

	return newResp, true
}

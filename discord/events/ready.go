package events

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/discord/commands"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(r.User.Username + " is now online!")
	taskqueue.BotQueue.Start(r.User.Username)
	commands.RegisterCommands(s, os.Getenv("GUILD_ID"))
	seedVoiceStateCache(s, r.Guilds)
}

func seedVoiceStateCache(s *discordgo.Session, guilds []*discordgo.Guild) {
	for _, partial := range guilds {
		g, err := s.State.Guild(partial.ID)
		if err != nil {
			continue
		}
		voiceStateCacheMu.Lock()
		for _, vs := range g.VoiceStates {
			if vs.ChannelID != "" {
				voiceStateCache[vs.GuildID+":"+vs.UserID] = vs.ChannelID
			}
		}
		voiceStateCacheMu.Unlock()
	}
}

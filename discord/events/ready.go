package events

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/discord/commands"
	"github.com/jayydoesdev/airo/bot/skills/reminders"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	taskqueue.StartTime = time.Now()
	fmt.Println(r.User.Username + " is now online!")
	taskqueue.BotQueue.Start(r.User.Username)
	if err := commands.RegisterCommands(s, os.Getenv("GUILD_ID")); err != nil {
		fmt.Println("[commands] registration failed:", err)
	} else {
		fmt.Println("[commands] registered successfully, guild:", os.Getenv("GUILD_ID"))
	}
	seedVoiceStateCache(s, r.Guilds)
	LoadTiers()
	reminders.Start(s)
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

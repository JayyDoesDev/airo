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
}

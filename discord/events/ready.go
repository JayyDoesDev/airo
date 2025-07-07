package events

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/discord/commands"
)

func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Println(r.User.Username + " is now online!")
	commands.RegisterCommands(s, os.Getenv("GUILD_ID"))
}

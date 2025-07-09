package events

import (
	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/discord/commands"
)

func OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	commands.FireCommands(s, i)
}

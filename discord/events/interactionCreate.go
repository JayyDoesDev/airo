package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/discord/commands"
)

func OnInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	fmt.Println("[OnInteractionCreate] received, type:", i.Type)
	commands.TriggerCommands(s, i)
}

package commands

import (
	"github.com/bwmarrin/discordgo"
)

type Command struct {
	*discordgo.ApplicationCommand
	Execute func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var bot_commands = [][]*Command{generic_commands, ai_commands}

func SendAnError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func RegisterCommands(s *discordgo.Session, guildID string) error {
	post_commands := []*discordgo.ApplicationCommand{}
	for _, cmd := range bot_commands {
		for _, command := range cmd {
			post_commands = append(post_commands, command.ApplicationCommand)
		}
	}

	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, post_commands)
	if err != nil {
		return err
	}

	return nil
}

func TriggerCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	for _, cmd := range bot_commands {
		for _, command := range cmd {
			if command.Name == i.ApplicationCommandData().Name {
				command.Execute(s, i)
			}
		}
	}
}

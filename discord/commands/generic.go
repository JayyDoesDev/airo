package commands

import "github.com/bwmarrin/discordgo"

var (
	HelpCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Sends the github link",
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "https://github.com/jayydoesdev/airo",
				},
			})
		},
	}
	PingCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "ping",
			Description: "Pong!",
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "pong",
				},
			})
		},
	}
)

var generic_commands = []*Command{
	&HelpCommand,
	&PingCommand,
}

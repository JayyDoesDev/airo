package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "prompt",
			Description: "Ask Airo a question!",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "provider",
					Description: "Choose the AI provider you would like to use!",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "OpenAI",
							Value: "openai",
						},
						{
							Name:  "Anthropic",
							Value: "anthropic",
						},
					},
				},
				{
					Name:        "question",
					Description: "The prompt to ask the AI",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
	}

	commandHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"prompt": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			data := i.ApplicationCommandData()
			var provider, question string

			for _, option := range data.Options {
				switch option.Name {
				case "provider":
					provider = option.StringValue()
				case "question":
					question = option.StringValue()
				}
			}

			content := fmt.Sprintf("You chose **%s** and asked:\n> %s", provider, question)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: content,
				},
			})
		},
	}
)

func RegisterCommands(s *discordgo.Session, guildID string) error {
	for _, cmd := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
		if err != nil {
			return fmt.Errorf("cannot create '%s' command: %w", cmd.Name, err)
		}
		fmt.Printf("Registered command: /%s\n", cmd.Name)
	}
	return nil
}

func FireCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if h, ok := commandHandler[i.ApplicationCommandData().Name]; ok {
		h(s, i)
	}
}

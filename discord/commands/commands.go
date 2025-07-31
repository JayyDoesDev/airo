package commands

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
)

var commands = []*discordgo.ApplicationCommand{
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
					{Name: "OpenAI", Value: "openai"},
					{Name: "Anthropic", Value: "anthropic"},
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

var commandHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
	"prompt": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		data := i.ApplicationCommandData()
		var provider, question string
		for _, opt := range data.Options {
			switch opt.Name {
			case "provider":
				provider = opt.StringValue()
			case "question":
				question = opt.StringValue()
			}
		}

		client, err := lib.NewClient(provider, os.Getenv("OPENAPI_API_KEY"))
		if err != nil {
			SendAnError(s, i, fmt.Sprintf("Error initializing AI client: %v", err))
			return
		}
		mem, err := lib.GetMemory("memory.msgpack")
		if err != nil {
			SendAnError(s, i, fmt.Sprintf("Error sending prompt to %s: %v", provider, err))
			return
		}

		promptMem := "Here are your memories:\n"
		for _, item := range mem.ShortTerm {
			promptMem += fmt.Sprintf("- [Short] %s: %s\n", item.Title, item.Content)
		}
		for _, item := range mem.LongTerm {
			promptMem += fmt.Sprintf("- [Long] %s: %s\n", item.Title, item.Content)
		}

		fullPrompt := promptMem + "\nUser says: " + question

		guild, err := s.Guild(i.GuildID)
		if err != nil {
			SendAnError(s, i, "Error: "+err.Error())
			return
		}
		resp, err := client.Send(i.Member.User.ID, i.Member.User.Username, *guild, fullPrompt, mem)
		if err != nil {
			SendAnError(s, i, fmt.Sprintf("Error sending prompt to %s: %v", provider, err))
			return
		}

		content := fmt.Sprintf("You chose **%s** and asked:\n> %s\n\n**Response:**\n%s", provider, question, resp)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	},
}

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

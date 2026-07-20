package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/skills/actions"
	"github.com/jayydoesdev/airo/bot/trust"
)

func extractResponse(raw string) string {
	_, data, err := actions.ParseAIResponse(raw)
	if err != nil {
		return raw
	}
	if data.ResponseMsg != "" {
		return data.ResponseMsg
	}
	return raw
}

func optionValue(data discordgo.ApplicationCommandInteractionData, name string) string {
	for _, opt := range data.Options {
		if opt.Name == name {
			return opt.StringValue()
		}
	}
	return ""
}

var (
	ExplainCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "explain",
			Description: "Explain a concept, code snippet, or anything",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "text",
					Description: "What do you want explained?",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if rejectBlocked(s, i) {
				return
			}
			text := lib.SanitizeInjection(optionValue(i.ApplicationCommandData(), "text"))
			prompt := fmt.Sprintf("Explain the following in a clear, engaging way. Use examples where helpful. Be concise but thorough:%s\n\n%s", tierContext(i), text)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			client, err := lib.NewClient("deepseek", os.Getenv("DEEPSEEK_API_KEY"))
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("ai error: " + err.Error()),
				})
				return
			}

			resp, err := client.Send(i.Member.User.ID, i.Member.User.Username, discordgo.Guild{}, prompt, lib.EmptyMemory())
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("error: " + err.Error()),
				})
				return
			}

			content := fmt.Sprintf("**%s**\n\n%s", text, extractResponse(resp))
			if len(content) > 1900 {
				content = content[:1900] + "...\n*(truncated)*"
			}
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
		},
	}

	CompareCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "compare",
			Description: "Compare two things — products, ideas, code, anything",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "thing1",
					Description: "First thing to compare",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
				{
					Name:        "thing2",
					Description: "Second thing to compare",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if rejectBlocked(s, i) {
				return
			}
			data := i.ApplicationCommandData()
			t1 := lib.SanitizeInjection(optionValue(data, "thing1"))
			t2 := lib.SanitizeInjection(optionValue(data, "thing2"))
			prompt := fmt.Sprintf("Compare and contrast the following two things. Highlight key similarities, differences, pros and cons, and which is better for different use cases:%s\n\nThing 1: %s\n\nThing 2: %s", tierContext(i), t1, t2)

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			client, err := lib.NewClient("deepseek", os.Getenv("DEEPSEEK_API_KEY"))
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("ai error: " + err.Error()),
				})
				return
			}

			resp, err := client.Send(i.Member.User.ID, i.Member.User.Username, discordgo.Guild{}, prompt, lib.EmptyMemory())
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("error: " + err.Error()),
				})
				return
			}

			content := fmt.Sprintf("**%s** vs **%s**\n\n%s", t1, t2, extractResponse(resp))
			if len(content) > 1900 {
				content = content[:1900] + "...\n*(truncated)*"
			}
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
		},
	}

	ChimeCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "chime",
			Description: "Ask Aira to chime in on recent channel activity",
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if rejectBlocked(s, i) {
				return
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			msgs, err := s.ChannelMessages(i.ChannelID, 20, "", "", "")
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("couldn't fetch messages: " + err.Error()),
				})
				return
			}

			var chat strings.Builder
			for j := len(msgs) - 1; j >= 0; j-- {
				m := msgs[j]
				if m.Author.Bot {
					continue
				}
				name := m.Author.Username
				t := ""
				if !m.Timestamp.IsZero() {
					t = m.Timestamp.Format(time.Kitchen)
				}
				content := m.Content
				if content == "" && len(m.Embeds) > 0 {
					content = "[sent an embed]"
				}
				if content == "" {
					content = "[attachment]"
				}
				chat.WriteString(fmt.Sprintf("[%s] %s: %s\n", t, name, content))
			}

			prompt := fmt.Sprintf("Here are the recent messages in this channel. Chime in naturally if you have something relevant, interesting, or funny to say. If you have nothing to add, just say you've got nothing:%s\n\n%s", tierContext(i), chat.String())

			client, err := lib.NewClient("deepseek", os.Getenv("DEEPSEEK_API_KEY"))
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("ai error: " + err.Error()),
				})
				return
			}

			resp, err := client.Send(i.Member.User.ID, i.Member.User.Username, discordgo.Guild{}, prompt, lib.EmptyMemory())
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("error: " + err.Error()),
				})
				return
			}

			content := extractResponse(resp)
			if len(content) > 1900 {
				content = content[:1900] + "...\n*(truncated)*"
			}
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
		},
	}
)

func strPtr(s string) *string { return &s }

func isBlocked(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	return trust.GetTier(i.Member.User.ID) == trust.TierBlocked
}

func rejectBlocked(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if !isBlocked(i) {
		return false
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You don't have permission to use this command.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	return true
}

func tierContext(i *discordgo.InteractionCreate) string {
	if i.Member == nil {
		return ""
	}
	tier := trust.GetTier(i.Member.User.ID)
	return fmt.Sprintf("\n[User trust tier: %s]", trust.Label(tier))
}

var tools_commands = []*Command{
	&ExplainCommand,
	&CompareCommand,
	&ChimeCommand,
}

package events

import (
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	botID := s.State.User.ID
	mention1 := "<@" + botID + ">"
	mention2 := "<@!" + botID + ">"

	if strings.HasPrefix(m.Content, mention1) || strings.HasPrefix(m.Content, mention2) {
		client, err := lib.NewClient("anthropic", os.Getenv("OPENAPI_API_KEY"))
		if err != nil {
			panic(err)
		}

		content := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(m.Content, mention1), mention2))

		resp, err := client.Send(m.Author.ID, m.Author.Username, content)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		naturalMsg, actionData, err := lib.ParseAIResponse(resp)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error parsing AI response: "+err.Error())
			return
		}

		var msg *discordgo.Message
		switch strings.ToLower(actionData.ResponseType) {
		case "embed":
			embed := &discordgo.MessageEmbed{
				Title:       actionData.EmbedTitle,
				Description: actionData.EmbedDescription,
				Color:       0xFF69B4,
			}
			msg, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		default:
			msg, err = s.ChannelMessageSend(m.ChannelID, actionData.ResponseMsg)
		}

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to send response: "+err.Error())
			return
		}

		if strings.ToLower(actionData.ResponseType) != "embed" {
			embed := &discordgo.MessageEmbed{
				Description: naturalMsg,
				Color:       0xFF69B4,
			}
			_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel: m.ChannelID,
				ID:      msg.ID,
				Embeds:  &[]*discordgo.MessageEmbed{embed},
			})
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Failed to attach embed: "+err.Error())
			}
		}

		if len(actionData.Tasks) > 0 {
			for _, t := range actionData.Tasks {
				task := t
				taskqueue.BotQueue.Add(taskqueue.Task{
					Name:        task.Action,
					GuildID:     m.GuildID,
					UserID:      task.TargetUser,
					Action:      task.Action,
					Reason:      task.Reason,
					Role:        task.Role,
					DMContent:   task.DMContent,
					ResponseMsg: task.ResponseMsg,
					Execute: func() error {
						switch task.Action {
						case "kick_user":
							return s.GuildMemberDeleteWithReason(m.GuildID, task.TargetUser, task.Reason)

						case "ban_user":
							err := s.GuildBanCreateWithReason(m.GuildID, task.TargetUser, task.Reason, 0)
							if err != nil {
								s.ChannelMessageSend(m.ChannelID, "Fail to ban user: "+err.Error())
								return err
							}
							return nil

						case "assign_role":
							return s.GuildMemberRoleAdd(m.GuildID, task.TargetUser, task.Role)

						case "remove_role":
							return s.GuildMemberRoleRemove(m.GuildID, task.TargetUser, task.Role)

						case "dm_user":
							dmChannel, err := s.UserChannelCreate(task.TargetUser)
							if err != nil {
								s.ChannelMessageSend(m.ChannelID, "Failed to open DM channel: "+err.Error())
								return err
							}
							_, err = s.ChannelMessageSend(dmChannel.ID, task.DMContent)
							if err != nil {
								s.ChannelMessageSend(m.ChannelID, "Failed to send DM: "+err.Error())
								return err
							}
							return nil

						default:
							return nil
						}
					},
				})
			}
		}
	}
}

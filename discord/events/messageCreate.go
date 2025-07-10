package events

import (
	"encoding/json"
	"fmt"
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

		if jsonOut, _ := json.MarshalIndent(actionData, "", "  "); true {
			fmt.Println("=== ACTION DATA ===")
			fmt.Println(string(jsonOut))
		}

		var msg *discordgo.Message
		if actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed" {
			embed := &discordgo.MessageEmbed{
				Title:       actionData.EmbedTitle,
				Description: actionData.EmbedDescription,
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: actionData.EmbedThumbnailUrl,
				},
				Image: &discordgo.MessageEmbedImage{
					URL: actionData.EmbedImageUrl,
				},
				Color: 0xFF69B4,
			}
			msg, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		} else {
			msg, err = s.ChannelMessageSend(m.ChannelID, actionData.ResponseMsg)
		}

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to send response: "+err.Error())
			return
		}

		if !actionData.UseEmbed && strings.ToLower(actionData.ResponseType) != "embed" {
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

		allTasks := actionData.Tasks
		if len(allTasks) == 0 && actionData.Action != "" && actionData.TargetUser != "" {
			allTasks = append(allTasks, lib.Action{
				Action:            actionData.Action,
				TargetUser:        actionData.TargetUser,
				Reason:            actionData.Reason,
				Role:              actionData.Role,
				DMContent:         actionData.DMContent,
				ResponseMsg:       actionData.ResponseMsg,
				EmbedTitle:        actionData.EmbedTitle,
				EmbedDescription:  actionData.EmbedDescription,
				EmbedThumbnailUrl: actionData.EmbedThumbnailUrl,
				UseEmbed:          actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed",
			})
		}

		for _, t := range allTasks {
			task := t

			taskqueue.BotQueue.Add(taskqueue.Task{
				Name:              task.Action,
				GuildID:           m.GuildID,
				UserID:            task.TargetUser,
				Action:            task.Action,
				Reason:            task.Reason,
				Role:              task.Role,
				DMContent:         task.DMContent,
				ResponseMsg:       task.ResponseMsg,
				EmbedTitle:        task.EmbedTitle,
				EmbedDescription:  task.EmbedDescription,
				EmbedThumbnailUrl: actionData.EmbedThumbnailUrl,
				UseEmbed:          task.UseEmbed,
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

					case "list_user_roles":
						member, err := s.GuildMember(m.GuildID, task.TargetUser)
						if err != nil {
							s.ChannelMessageSend(m.ChannelID, "Failed to fetch member roles: "+err.Error())
							return err
						}
						guildRoles, err := s.GuildRoles(m.GuildID)
						if err != nil {
							s.ChannelMessageSend(m.ChannelID, "Failed to fetch guild roles: "+err.Error())
							return err
						}
						roleMap := make(map[string]string)
						for _, role := range guildRoles {
							roleMap[role.ID] = role.Name
						}
						var roleNames []string
						for _, roleID := range member.Roles {
							if name, ok := roleMap[roleID]; ok {
								roleNames = append(roleNames, "• "+name)
							}
						}
						roleList := "No roles found, huh? That's suspicious 🤨"
						if len(roleNames) > 0 {
							roleList = strings.Join(roleNames, "\n")
						}
						if task.UseEmbed {
							embed := &discordgo.MessageEmbed{
								Title:       task.EmbedTitle,
								Description: roleList,
								Color:       0x1ABC9C,
							}
							if task.ResponseMsg != "" {
								s.ChannelMessageSend(m.ChannelID, task.ResponseMsg)
							}
							s.ChannelMessageSendEmbed(m.ChannelID, embed)
						} else {
							if task.ResponseMsg != "" {
								s.ChannelMessageSend(m.ChannelID, task.ResponseMsg)
							} else {
								s.ChannelMessageSend(m.ChannelID, roleList)
							}
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

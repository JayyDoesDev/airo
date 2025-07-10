package events

import (
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
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

		switch actionData.Action {
		case "kick_user":
			s.GuildMemberDeleteWithReason(m.GuildID, actionData.TargetUser, actionData.Reason)
		case "ban_user":
			err := s.GuildBanCreateWithReason(m.GuildID, actionData.TargetUser, actionData.Reason, 0)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Fail to ban user: "+err.Error())
				return
			}
		case "assign_role":
			s.GuildMemberRoleAdd(m.GuildID, actionData.TargetUser, actionData.Role)
		case "remove_role":
			s.GuildMemberRoleRemove(m.GuildID, actionData.TargetUser, actionData.Role)
		case "dm_user":
			dmChannel, err := s.UserChannelCreate(actionData.TargetUser)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Failed to open DM channel: "+err.Error())
				return
			}
			_, err = s.ChannelMessageSend(dmChannel.ID, actionData.DMContent)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Failed to send DM: "+err.Error())
				return
			}
		case "list_user_roles":
			member, err := s.GuildMember(m.GuildID, actionData.TargetUser)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Failed to fetch member roles: "+err.Error())
				return
			}

			guildRoles, err := s.GuildRoles(m.GuildID)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Failed to fetch guild roles: "+err.Error())
				return
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

			if strings.ToLower(actionData.ResponseType) == "embed" {
				embed := &discordgo.MessageEmbed{
					Title:       actionData.EmbedTitle,
					Description: roleList,
					Color:       0x1ABC9C,
				}
				if actionData.ResponseMsg != "" {
					s.ChannelMessageSend(m.ChannelID, actionData.ResponseMsg)
				}
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
			} else {
				if actionData.ResponseMsg != "" {
					s.ChannelMessageSend(m.ChannelID, actionData.ResponseMsg)
				} else {
					s.ChannelMessageSend(m.ChannelID, roleList)
				}
			}
		}
	}
}

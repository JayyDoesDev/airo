package events

import (
	"fmt"
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

		s.ChannelMessageSend(m.ChannelID, naturalMsg)
		fmt.Println(actionData)

		switch actionData.Action {
		case "kick_user":
			s.GuildMemberDeleteWithReason(m.GuildID, actionData.TargetUser, actionData.Reason)

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
		}
	}
}

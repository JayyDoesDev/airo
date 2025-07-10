package lib

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

func HandleActions(task Action, s *discordgo.Session, m *discordgo.MessageCreate) error {
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
}

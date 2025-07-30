package lib

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func HandleActions(task Action, s *discordgo.Session, m *discordgo.MessageCreate) error {
	switch task.Action {
	case "kick_user":
		err := s.GuildMemberDeleteWithReason(m.GuildID, task.TargetUser, task.Reason)
		if err == nil {
			StoreToMemory(MemoryItem{
				Id:         GenerateID(),
				Title:      "User kicked",
				Content:    "Kicked user " + task.TargetUser + " for: " + task.Reason,
				Type:       "moderation",
				Source:     "action",
				Importance: 0.6,
				Created:    time.Now().Format(time.RFC3339),
				Context: &MemoryItemContext{
					Location: m.ChannelID,
					Author:   m.Author.ID,
				},
			})
		}
		return err

	case "ban_user":
		err := s.GuildBanCreateWithReason(m.GuildID, task.TargetUser, task.Reason, 0)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Fail to ban user: "+err.Error())
			return err
		}

		StoreToMemory(MemoryItem{
			Id:         GenerateID(),
			Title:      "User banned",
			Content:    "Banned user " + task.TargetUser + " for: " + task.Reason,
			Type:       "moderation",
			Source:     "action",
			Importance: 0.7,
			Created:    time.Now().Format(time.RFC3339),
			Context: &MemoryItemContext{
				Location: m.ChannelID,
				Author:   m.Author.ID,
			},
		})

		return nil

	case "assign_role":
		err := s.GuildMemberRoleAdd(m.GuildID, task.TargetUser, task.Role)
		if err == nil {
			StoreToMemory(MemoryItem{
				Id:         GenerateID(),
				Title:      "Assigned role",
				Content:    "Assigned role " + task.Role + " to user " + task.TargetUser,
				Type:       "role",
				Source:     "action",
				Importance: 0.3,
				Created:    time.Now().Format(time.RFC3339),
				Context: &MemoryItemContext{
					Location: m.ChannelID,
					Author:   m.Author.ID,
				},
			})
		}
		return err

	case "remove_role":
		err := s.GuildMemberRoleRemove(m.GuildID, task.TargetUser, task.Role)
		if err == nil {
			StoreToMemory(MemoryItem{
				Id:         GenerateID(),
				Title:      "Removed role",
				Content:    "Removed role " + task.Role + " from user " + task.TargetUser,
				Type:       "role",
				Source:     "action",
				Importance: 0.3,
				Created:    time.Now().Format(time.RFC3339),
				Context: &MemoryItemContext{
					Location: m.ChannelID,
					Author:   m.Author.ID,
				},
			})
		}
		return err

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

		StoreToMemory(MemoryItem{
			Id:         GenerateID(),
			Title:      "DM sent",
			Content:    "Sent DM to " + task.TargetUser + ": " + task.DMContent,
			Type:       "message",
			Source:     "DM",
			Importance: 0.4,
			Created:    time.Now().Format(time.RFC3339),
			Context: &MemoryItemContext{
				Location: m.ChannelID,
				Author:   m.Author.ID,
			},
		})

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

		StoreToMemory(MemoryItem{
			Id:         GenerateID(),
			Title:      "Listed user roles",
			Content:    "User " + task.TargetUser + " has roles:\n" + roleList,
			Type:       "role",
			Source:     "action",
			Importance: 0.2,
			Created:    time.Now().Format(time.RFC3339),
			Context: &MemoryItemContext{
				Location: m.ChannelID,
				Author:   m.Author.ID,
			},
		})

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
			}
			s.ChannelMessageSend(m.ChannelID, roleList)
		}

		return nil

	default:
		return nil
	}
}

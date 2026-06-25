package events

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

var (
	botMemberCache   = map[string]*discordgo.Member{}
	botMemberCacheMu sync.Mutex

	voiceStateCache   = map[string]string{} // "guildID:userID" -> channelID
	voiceStateCacheMu sync.RWMutex
)

func OnVoiceStateUpdate(_ *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	key := v.GuildID + ":" + v.UserID
	voiceStateCacheMu.Lock()
	if v.ChannelID == "" {
		delete(voiceStateCache, key)
	} else {
		voiceStateCache[key] = v.ChannelID
	}
	voiceStateCacheMu.Unlock()
	fmt.Printf("[voice] state update: userID=%s channelID=%q\n", v.UserID, v.ChannelID)
}

func getBotMember(s *discordgo.Session, guildID string) (*discordgo.Member, error) {
	botMemberCacheMu.Lock()
	defer botMemberCacheMu.Unlock()

	if m, ok := botMemberCache[guildID]; ok {
		return m, nil
	}

	m, err := s.GuildMember(guildID, s.State.User.ID)
	if err != nil {
		return nil, err
	}
	botMemberCache[guildID] = m
	return m, nil
}

func getBotPermissions(s *discordgo.Session, guild *discordgo.Guild, channelID string) int64 {
	botID := s.State.User.ID

	member, err := getBotMember(s, guild.ID)
	if err != nil {
		return 0
	}

	var perms int64
	for _, role := range guild.Roles {
		if role.ID == guild.ID {
			perms |= role.Permissions
			break
		}
	}

	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				perms |= role.Permissions
				break
			}
		}
	}

	if perms&discordgo.PermissionAdministrator != 0 {
		return discordgo.PermissionAll
	}

	channel, err := s.State.Channel(channelID)
	if err != nil {
		return perms
	}

	for _, ow := range channel.PermissionOverwrites {
		if ow.ID == guild.ID {
			perms &^= ow.Deny
			perms |= ow.Allow
			break
		}
	}

	var allow, deny int64
	for _, roleID := range member.Roles {
		for _, ow := range channel.PermissionOverwrites {
			if ow.ID == roleID {
				allow |= ow.Allow
				deny |= ow.Deny
			}
		}
	}
	perms &^= deny
	perms |= allow

	for _, ow := range channel.PermissionOverwrites {
		if ow.ID == botID {
			perms &^= ow.Deny
			perms |= ow.Allow
			break
		}
	}

	return perms
}

func resolveUserVoiceChannel(_ *discordgo.Session, guildID, userID string) (string, error) {
	key := guildID + ":" + userID
	voiceStateCacheMu.RLock()
	ch, ok := voiceStateCache[key]
	voiceStateCacheMu.RUnlock()
	if ok && ch != "" {
		return ch, nil
	}
	return "", fmt.Errorf("user not in voice channel")
}

func formatPermissions(perms int64) string {
	type perm struct {
		bit  int64
		name string
	}
	all := []perm{
		{discordgo.PermissionAdministrator, "Administrator"},
		{discordgo.PermissionManageServer, "Manage Server"},
		{discordgo.PermissionManageRoles, "Manage Roles"},
		{discordgo.PermissionManageChannels, "Manage Channels"},
		{discordgo.PermissionKickMembers, "Kick Members"},
		{discordgo.PermissionBanMembers, "Ban Members"},
		{discordgo.PermissionManageMessages, "Manage Messages"},
		{discordgo.PermissionMentionEveryone, "Mention Everyone"},
		{discordgo.PermissionModerateMembers, "Timeout Members"},
		{discordgo.PermissionSendMessages, "Send Messages"},
		{discordgo.PermissionReadMessageHistory, "Read Message History"},
		{discordgo.PermissionViewChannel, "View Channels"},
		{discordgo.PermissionEmbedLinks, "Embed Links"},
		{discordgo.PermissionAttachFiles, "Attach Files"},
		{discordgo.PermissionAddReactions, "Add Reactions"},
		{discordgo.PermissionManageNicknames, "Manage Nicknames"},
		{discordgo.PermissionChangeNickname, "Change Nickname"},
	}

	var granted []string
	for _, p := range all {
		if perms&p.bit != 0 {
			granted = append(granted, "- "+p.name)
		}
	}
	if len(granted) == 0 {
		return "- None"
	}
	return strings.Join(granted, "\n")
}

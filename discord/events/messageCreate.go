package events

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

const userCooldown = 5 * time.Second
const maxDMsPerRequest = 1

var (
	cooldowns   = map[string]time.Time{}
	cooldownsMu sync.Mutex
)

func isOnCooldown(userID string) bool {
	cooldownsMu.Lock()
	defer cooldownsMu.Unlock()
	if t, ok := cooldowns[userID]; ok && time.Since(t) < userCooldown {
		return true
	}
	cooldowns[userID] = time.Now()
	return false
}

func HandleMentions(Id string) (string, string) {
	return "<@" + Id + ">", "<@!" + Id + ">"
}

type HandleGoogleSearchOpts struct {
	Response   string
	Client     lib.LibClient
	FullPrompt string
	Memory     actions.Memory
	Session    *discordgo.Session
	Message    *discordgo.MessageCreate
	Guild      *discordgo.Guild
}

func HandleGoogleSearch(opts HandleGoogleSearchOpts) (string, bool, []skills.References) {
	isGoogleSearch := strings.Contains(opts.Response, `SEARCH("`)
	if !isGoogleSearch {
		return opts.Response, false, []skills.References{}
	}

	limit, _ := strconv.Atoi(os.Getenv("GOOGLE_RESULT_LIMIT"))
	google := skills.GoogelClient(skills.Google{
		APIKey:     os.Getenv("GOOGLE_API_KEY"),
		CXEngineID: os.Getenv("GOOGLE_CX_ENGINE_ID"),
		Limit:      limit,
	})

	start := strings.Index(opts.Response, "SEARCH(\"")
	opts.Response = opts.Response[start:]
	q := strings.TrimSuffix(strings.TrimPrefix(opts.Response, `SEARCH("`), `")`)

	results, err := google.Search(q)
	if err != nil {
		opts.Session.ChannelMessageSend(opts.Message.ChannelID, "Search failed: "+err.Error())
		return opts.Response, true, []skills.References{}
	}

	items := google.LimitItems(results.Items)
	refs := google.GetReferences(items)

	var newResp string

	if len(items) == 0 {
		newResp, err = opts.Client.Send(
			opts.Message.Author.ID, opts.Message.Author.Username, *opts.Guild,
			opts.FullPrompt+lib.SecondPromptResultsNotFound,
			opts.Memory,
		)
	} else {
		var sb strings.Builder
		sb.WriteString(lib.SecondPromptTitle)
		for i, item := range items {
			sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n\n", i+1, item.Title, item.Link, item.Snippet))
		}

		secondPrompt := strings.Builder{}
		secondPrompt.WriteString(lib.SecondPromptHeader)
		secondPrompt.WriteString(sb.String())
		secondPrompt.WriteString(lib.SecondPromptRules)

		newResp, err = opts.Client.Send(
			opts.Message.Author.ID, opts.Message.Author.Username, *opts.Guild,
			secondPrompt.String(),
			actions.Memory{ShortTerm: nil, LongTerm: nil},
		)
	}

	if err != nil {
		opts.Session.ChannelMessageSend(opts.Message.ChannelID, "Error: "+err.Error())
		return opts.Response, true, refs
	}

	return newResp, true, refs
}

type GoogleEmbedOptions struct {
	Title     string
	Thumbnail string
	Image     string
	Color     int
}

func GoogleEmbed(description string, opts *GoogleEmbedOptions) *discordgo.MessageEmbed {
	color := 0x4285F4
	if opts != nil && opts.Color != 0 {
		color = opts.Color
	}

	embed := &discordgo.MessageEmbed{
		Description: description,
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Search results provided by Google",
			IconURL: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQ2sSeQqjaUTuZ3gRgkKjidpaipF_l6s72lBw&s",
		},
	}

	if opts != nil {
		if opts.Title != "" {
			embed.Title = opts.Title
		}
		if opts.Thumbnail != "" {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: opts.Thumbnail}
		}
		if opts.Image != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: opts.Image}
		}
	}

	return embed
}

func GoogleReferencesEmbed(refs []skills.References) *discordgo.MessageEmbed {
	if len(refs) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "No References Found",
			Description: "Google returned no usable search results.",
			Color:       0x4285F4,
		}
	}

	var desc strings.Builder
	for idx, r := range refs {
		if idx >= 3 {
			desc.WriteString("**…and more results were trimmed**")
			break
		}
		desc.WriteString(
			fmt.Sprintf("**[%d] %s**\n%s\n*%s*\n\n",
				r.Index, r.Title, r.Url, r.Snippet),
		)
	}

	return &discordgo.MessageEmbed{
		Title:       "🔎 References",
		Description: desc.String(),
		Color:       0x4285F4,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Search results provided by Google",
			IconURL: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQ2sSeQqjaUTuZ3gRgkKjidpaipF_l6s72lBw&s",
		},
	}
}

func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	mention1, mention2 := HandleMentions(s.State.User.ID)
	var references []skills.References
	if strings.HasPrefix(m.Content, mention1) || strings.HasPrefix(m.Content, mention2) {
		if isOnCooldown(m.Author.ID) {
			return
		}

		err := s.ChannelTyping(m.ChannelID)
		if err != nil {
			return
		}

		client, err := lib.NewClient("deepseek", os.Getenv("DEEPSEEK_API_KEY"))
		if err != nil {
			panic(err)
		}

		content := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(m.Content, mention1), mention2))
		content = lib.SanitizeInjection(content)
		mem, err := actions.GetMemory("memory.msgpack")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		guild, err := s.Guild(m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		botPerms := getBotPermissions(s, guild, m.ChannelID)
		fullPrompt := "Your permissions in this server:\n" + formatPermissions(botPerms) + "\n\nUser says: " + content

		resp, err := client.Send(m.Author.ID, m.Author.Username, *guild, fullPrompt, mem)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		isGoogleSearch := strings.Contains(resp, `SEARCH("`)

		var didSearch bool
		var refs []skills.References

		resp, didSearch, refs = HandleGoogleSearch(HandleGoogleSearchOpts{
			Response:   resp,
			Client:     client,
			FullPrompt: fullPrompt,
			Memory:     mem,
			Session:    s,
			Message:    m,
			Guild:      guild,
		})

		references = refs
		isGoogleSearch = didSearch

		fmt.Println("RAW RESPONSE:")
		fmt.Println(resp)

		naturalMsg, actionData, err := actions.ParseAIResponse(resp)

		actionData.ResponseMsg = strings.ReplaceAll(actionData.ResponseMsg, "@everyone", "everyone")
		actionData.ResponseMsg = strings.ReplaceAll(actionData.ResponseMsg, "@here", "here")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error parsing AI response: "+err.Error())
			return
		}

		if jsonOut, _ := json.MarshalIndent(actionData, "", "  "); true {
			fmt.Println("=== ACTION DATA ===")
			fmt.Println(string(jsonOut))
		}

		if len(actionData.Memories) > 0 {
			for _, mem := range actionData.Memories {
				location := m.ChannelID
				if mem.Context != nil {
					location = mem.Context.Location
				}
				actions.CreateMemory(actions.MemoryItem{
					Id:           actions.GenerateID(),
					Title:        mem.Title,
					Content:      mem.Content,
					Type:         mem.Type,
					Source:       mem.Source,
					Importance:   mem.Importance,
					Created:      time.Now().Format(time.RFC3339),
					Lastaccessed: time.Now().Format(time.RFC3339),
					Related:      mem.Related,
					Context: &actions.MemoryItemContext{
						Location: location,
						Author:   m.Author.ID,
					},
				})
			}
		}

		var embeds []*discordgo.MessageEmbed

		if actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed" {
			var mainEmbed *discordgo.MessageEmbed
			if isGoogleSearch {
				mainEmbed = GoogleEmbed(actionData.EmbedDescription, &GoogleEmbedOptions{
					Title:     actionData.EmbedTitle,
					Thumbnail: actionData.EmbedThumbnailUrl,
					Image:     actionData.EmbedImageUrl,
				})
			} else {
				mainEmbed = &discordgo.MessageEmbed{
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
			}

			embeds = append(embeds, mainEmbed)

		} else {
			embeds = append(embeds, &discordgo.MessageEmbed{
				Description: actionData.ResponseMsg,
				Color:       0xFF69B4,
			})
		}

		if isGoogleSearch {
			embeds = append(embeds, GoogleEmbed(naturalMsg, &GoogleEmbedOptions{}))
		}

		if len(references) > 0 {
			embeds = append(embeds, GoogleReferencesEmbed(references))
		}

		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds: embeds,
		})

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to send response: "+err.Error())
			return
		}

		allTasks := actionData.Tasks
		if len(allTasks) == 0 && actionData.Action != "" && actionData.TargetUser != "" {
			allTasks = append(allTasks, actions.Action{
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

		dmCount := 0
		for _, t := range allTasks {
			task := t
			if task.Action == "dm_user" {
				if dmCount >= maxDMsPerRequest {
					continue
				}
				dmCount++
			}

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
				Execute:           MakeExecute(task, s, m),
			})
		}
	}
}

func MakeExecute(task actions.Action, s *discordgo.Session, m *discordgo.MessageCreate) func() error {
	return func() error {
		return actions.HandleActions(task, s, m)
	}
}

var (
	botMemberCache   = map[string]*discordgo.Member{}
	botMemberCacheMu sync.Mutex
)

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

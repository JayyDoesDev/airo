package events

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

func HandleMentions(id string) (string, string) {
	return "<@" + id + ">", "<@!" + id + ">"
}

func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	mention1, mention2 := HandleMentions(s.State.User.ID)
	if !strings.HasPrefix(m.Content, mention1) && !strings.HasPrefix(m.Content, mention2) {
		return
	}

	if isOnCooldown(m.Author.ID) {
		return
	}

	if err := s.ChannelTyping(m.ChannelID); err != nil {
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

	resp, didSearch, refs := HandleSearch(SearchOpts{
		Response:   resp,
		Client:     client,
		FullPrompt: fullPrompt,
		Memory:     mem,
		Session:    s,
		Message:    m,
		Guild:      guild,
	})

	fmt.Println("RAW RESPONSE:")
	fmt.Println(resp)

	naturalMsg, actionData, err := actions.ParseAIResponse(resp)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error parsing AI response: "+err.Error())
		return
	}

	actionData.ResponseMsg = strings.ReplaceAll(actionData.ResponseMsg, "@everyone", "everyone")
	actionData.ResponseMsg = strings.ReplaceAll(actionData.ResponseMsg, "@here", "here")

	if jsonOut, _ := json.MarshalIndent(actionData, "", "  "); true {
		fmt.Println("=== ACTION DATA ===")
		fmt.Println(string(jsonOut))
	}

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

	embeds := buildEmbeds(actionData, naturalMsg, didSearch, refs)
	s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Embeds: embeds})

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

func buildEmbeds(actionData actions.ActionData, _ string, didSearch bool, refs []skills.References) []*discordgo.MessageEmbed {
	var embeds []*discordgo.MessageEmbed

	if actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed" {
		var main *discordgo.MessageEmbed
		if didSearch {
			main = SearchEmbed(actionData.EmbedDescription, actionData.EmbedTitle, actionData.EmbedThumbnailUrl, actionData.EmbedImageUrl)
		} else {
			main = &discordgo.MessageEmbed{
				Title:       actionData.EmbedTitle,
				Description: actionData.EmbedDescription,
				Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: actionData.EmbedThumbnailUrl},
				Image:       &discordgo.MessageEmbedImage{URL: actionData.EmbedImageUrl},
				Color:       0xFF69B4,
			}
		}
		embeds = append(embeds, main)
	} else {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Description: actionData.ResponseMsg,
			Color:       0xFF69B4,
		})
	}

	if len(refs) > 0 {
		embeds = append(embeds, SearchReferencesEmbed(refs))
	}

	return embeds
}

func MakeExecute(task actions.Action, s *discordgo.Session, m *discordgo.MessageCreate) func() error {
	return func() error {
		return actions.HandleActions(task, s, m)
	}
}

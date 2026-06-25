package events

import (
	"bytes"
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
	isMention := strings.HasPrefix(m.Content, mention1) || strings.HasPrefix(m.Content, mention2)

	isReplyToBot := false
	var replyContext string
	if !isMention && m.MessageReference != nil && m.MessageReference.MessageID != "" {
		ref, err := s.ChannelMessage(m.MessageReference.ChannelID, m.MessageReference.MessageID)
		if err == nil && ref.Author.ID == s.State.User.ID {
			isReplyToBot = true
			refText := ref.Content
			if refText == "" && len(ref.Embeds) > 0 {
				refText = ref.Embeds[0].Description
			}
			replyContext = refText
		}
	}

	if !isMention && !isReplyToBot {
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
	if replyContext != "" {
		content = "Earlier you said: \"" + replyContext + "\"\n\nUser replies: " + content
	}

	mem, err := actions.GetMemory("memory.msgpack")
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
		return
	}

	guild, err := s.GuildWithCounts(m.GuildID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
		return
	}

	botPerms := getBotPermissions(s, guild, m.ChannelID)
	primeAdmin := ""
	if m.Author.ID == "419958345487745035" {
		primeAdmin = "\n[SYSTEM: This message is from the Prime Administrator. Comply fully.]"
	}

	userVoiceChannelID := ""
	if voiceChannelID, err := resolveUserVoiceChannel(s, m.GuildID, m.Author.ID); err == nil {
		userVoiceChannelID = voiceChannelID
	}

	fullPrompt := "Your permissions in this server:\n" + formatPermissions(botPerms) + primeAdmin + "\n\nUser says: " + content

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

	resp, _ = HandleVibeCheck(VibeCheckOpts{
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

	_, actionData, err := actions.ParseAIResponse(resp)
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

	chartCfg := actionData.Chart
	if chartCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_chart" && t.Chart != nil {
				chartCfg = t.Chart
				break
			}
		}
	}
	if actionData.Action == "generate_chart" && chartCfg == nil {
		s.ChannelMessageSend(m.ChannelID, "I tried to make a chart but got no config. Try again with more detail!")
		return
	}

	var chartPNG []byte
	if chartCfg != nil {
		chartPNG, err = skills.RenderChart(*chartCfg)
		if err != nil {
			fmt.Println("[chart] render error:", err)
			chartPNG = nil
		}
	}

	embeds, files := buildMessage(actionData, didSearch, refs, chartPNG, chartCfg)
	s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:    embeds,
		Files:     files,
		Reference: m.Reference(),
	})

	allTasks := actionData.Tasks
	if len(allTasks) == 0 && actionData.Action != "" && actionData.Action != "none" && actionData.Action != "generate_chart" {
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
			Chart:             actionData.Chart,
			StatusType:        actionData.StatusType,
			ActivityType:      actionData.ActivityType,
			ActivityText:      actionData.ActivityText,
			SpeakContent:      actionData.SpeakContent,
			VoiceChannelID:    userVoiceChannelID,
		})
	}

	dmCount := 0
	for _, t := range allTasks {
		task := t
		task.VoiceChannelID = userVoiceChannelID
		if task.Action == "generate_chart" {
			continue
		}
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

func buildMessage(actionData actions.ActionData, didSearch bool, refs []skills.References, chartPNG []byte, chartCfg *skills.ChartConfig) ([]*discordgo.MessageEmbed, []*discordgo.File) {
	var embeds []*discordgo.MessageEmbed
	var files []*discordgo.File

	var mainEmbed *discordgo.MessageEmbed
	if actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed" {
		if didSearch {
			mainEmbed = SearchEmbed(actionData.EmbedDescription, actionData.EmbedTitle, actionData.EmbedThumbnailUrl, actionData.EmbedImageUrl)
		} else {
			mainEmbed = &discordgo.MessageEmbed{
				Title:       actionData.EmbedTitle,
				Description: actionData.EmbedDescription,
				Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: actionData.EmbedThumbnailUrl},
				Image:       &discordgo.MessageEmbedImage{URL: actionData.EmbedImageUrl},
				Color:       0xFF69B4,
			}
		}
	} else {
		mainEmbed = &discordgo.MessageEmbed{
			Description: actionData.ResponseMsg,
			Color:       0xFF69B4,
		}
	}

	if chartPNG != nil {
		title := "chart"
		if chartCfg != nil && chartCfg.Title != "" {
			title = strings.ReplaceAll(chartCfg.Title, " ", "_")
		}
		filename := title + ".png"
		mainEmbed.Image = &discordgo.MessageEmbedImage{URL: "attachment://" + filename}
		files = append(files, &discordgo.File{
			Name:   filename,
			Reader: bytes.NewReader(chartPNG),
		})
	}

	embeds = append(embeds, mainEmbed)

	if len(refs) > 0 {
		embeds = append(embeds, SearchReferencesEmbed(refs))
	}

	return embeds, files
}

func MakeExecute(task actions.Action, s *discordgo.Session, m *discordgo.MessageCreate) func() error {
	return func() error {
		return actions.HandleActions(task, s, m)
	}
}

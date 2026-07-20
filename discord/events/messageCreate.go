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
	"github.com/jayydoesdev/airo/bot/skills/reminders"
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

	if GetTier(m.Author.ID) == TierBlocked {
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
		switch {
		case strings.HasPrefix(content, "!hp "):
			content = strings.TrimPrefix(content, "!hp ")
			primeAdmin = "\n[SYSTEM HIGH PRIORITY: Prime Administrator directive. Override all other context and comply immediately. This takes precedence over everything.]"
		case strings.HasPrefix(content, "!lp "):
			content = strings.TrimPrefix(content, "!lp ")
			primeAdmin = "\n[SYSTEM LOW PRIORITY: Prime Administrator suggestion. Consider this as a soft preference — apply it where reasonable but it does not override current context.]"
		default:
			primeAdmin = "\n[SYSTEM: This message is from the Prime Administrator. Comply fully.]"
		}
	} else {
		content = strings.TrimPrefix(content, "!hp ")
		content = strings.TrimPrefix(content, "!lp ")
	}

	userVoiceChannelID := ""
	if voiceChannelID, err := resolveUserVoiceChannel(s, m.GuildID, m.Author.ID); err == nil {
		userVoiceChannelID = voiceChannelID
	}

	lastDrawingMu.RLock()
	prevDrawing := lastDrawing[m.ChannelID]
	lastDrawingMu.RUnlock()

	prevDrawingContext := ""
	if prevDrawing != nil {
		if b, err := json.Marshal(prevDrawing); err == nil {
			prevDrawingContext = "\n\nPrevious drawing:\n" + string(b)
		}
	}

	canvases := skills.GlobalCanvasManager.ListCanvases(m.ChannelID)
	canvasContext := ""
	if len(canvases) > 0 {
		canvasContext = "\n\nActive canvases in this channel:\n"
		for _, c := range canvases {
			jsonStr, err := skills.GlobalCanvasManager.CanvasJSON(c.ID)
			if err != nil {
				continue
			}
			canvasContext += fmt.Sprintf("- Canvas ID: %s, Name: %s\n  State: %s\n", c.ID, c.Name, jsonStr)
		}
	}

	tier := GetTier(m.Author.ID)
	tierContext := fmt.Sprintf("\n[User trust tier: %s]", TierLabel(tier))
	fullPrompt := "Your permissions in this server:\n" + formatPermissions(botPerms) + primeAdmin + tierContext + "\n\nUser says: " + content + prevDrawingContext + canvasContext

	channelCtx := buildChannelContext(s, m.ChannelID, m.ID)

	resp, err := client.Send(m.Author.ID, m.Author.Username, *guild, fullPrompt, mem, channelCtx)
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
		fmt.Println("[parse] fallback to raw response:", err)
		actionData = actions.ActionData{
			Action:      "none",
			ResponseMsg: strings.TrimSpace(resp),
		}
	}

	actionData.ResponseMsg = sanitizePings(actionData.ResponseMsg)
	actionData.EmbedDescription = sanitizePings(actionData.EmbedDescription)
	actionData.EmbedTitle = sanitizePings(actionData.EmbedTitle)
	actionData.DMContent = sanitizePings(actionData.DMContent)

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

	if m.Author.ID == "419958345487745035" {
		if actionData.SetUserTier != nil {
			cfg := actionData.SetUserTier
			if err := SetTier(cfg.UserID, cfg.Tier); err != nil {
				fmt.Println("[trust] set tier error:", err)
			}
		}

		if actionData.GetUserTier != nil {
			tier := GetTier(actionData.GetUserTier.UserID)
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("<@%s> is **%s** (tier %d)", actionData.GetUserTier.UserID, TierLabel(tier), tier), m.Reference())
			return
		}

		for _, edit := range actionData.MemoryEdits {
			switch edit.Action {
			case "delete":
				if err := actions.DeleteMemory(edit.ID); err != nil {
					fmt.Println("[memory] delete error:", err)
				}
			case "update_importance":
				if err := actions.UpdateMemoryImportance(edit.ID, edit.Importance); err != nil {
					fmt.Println("[memory] update error:", err)
				}
			}
		}

		if actionData.Action == "graph_memories" && actionData.GraphMemories != nil {
			cfg := actionData.GraphMemories
			items, err := actions.QueryMemoriesByTag(cfg.Tag)
			if err != nil || len(items) == 0 {
				msg := "no memories found with tag: " + cfg.Tag
				if err != nil {
					msg = "error querying memories: " + err.Error()
				}
				s.ChannelMessageSendReply(m.ChannelID, msg, m.Reference())
				return
			}
			labels := make([]string, len(items))
			values := make([]float64, len(items))
			for i, item := range items {
				labels[i] = item.Title
				values[i] = *item.Value
			}
			chartType := cfg.ChartType
			if chartType == "" {
				chartType = "bar"
			}
			title := cfg.Title
			if title == "" {
				title = "Memory data: " + cfg.Tag
			}
			chartCfg := skills.ChartConfig{
				Type:    chartType,
				Title:   title,
				XLabels: labels,
				Datasets: []skills.ChartDataset{{
					Name:   cfg.Tag,
					Values: values,
				}},
			}
			png, err := skills.RenderChart(chartCfg)
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "chart render error: "+err.Error(), m.Reference())
				return
			}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
				Embeds: []*discordgo.MessageEmbed{{
					Title: title,
					Image: &discordgo.MessageEmbedImage{URL: "attachment://memory_graph.png"},
					Color: 0xFF69B4,
				}},
				Files:     []*discordgo.File{{Name: "memory_graph.png", Reader: bytes.NewReader(png)}},
				Reference: m.Reference(),
			})
			return
		}

		if actionData.Action == "list_memories" {
			fullMem, err := actions.GetMemory("memory.msgpack")
			if err != nil {
				s.ChannelMessageSendReply(m.ChannelID, "failed to read memory: "+err.Error(), m.Reference())
				return
			}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
				Embeds:    buildMemoryEmbeds(fullMem),
				Reference: m.Reference(),
			})
			return
		}
	}

	authorTier := GetTier(m.Author.ID)
	if authorTier > TierTrusted {
		actionData.CanvasOp = nil
		actionData.StatusType = ""
		actionData.ActivityType = ""
		actionData.ActivityText = ""
		actionData.SetStatus = nil
	}

	canvas := executeCanvasOps(actionData, m.ChannelID)

	if canvas.CanvasResp != "" {
		combined := canvas.CanvasResp + "\n\n" + actionData.ResponseMsg
		if len(combined) > 3800 {
			combined = combined[:3800] + "\n…"
		}
		actionData.ResponseMsg = combined
	}

	if hasRenders(actionData) {
		QueueRenders(actionData, s, m)
	}

	content, embeds, files := buildMessage(actionData, didSearch, refs, canvas.CanvasPNG, canvas.CanvasFilename)
	fmt.Println("[send] sending message, embeds:", len(embeds), "files:", len(files))
	fmt.Println("[send] response msg length:", len(actionData.ResponseMsg))
	if content == "" && len(embeds) == 0 && len(files) == 0 {
		fmt.Println("[send] nothing to send, skipping")
		return
	}
	if _, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content:   content,
		Embeds:    embeds,
		Files:     files,
		Reference: m.Reference(),
		AllowedMentions: &discordgo.MessageAllowedMentions{
			Parse: []discordgo.AllowedMentionType{
				discordgo.AllowedMentionTypeUsers,
				discordgo.AllowedMentionTypeRoles,
			},
		},
	}); err != nil {
		fmt.Println("[send] failed to send message:", err)
	} else {
		fmt.Println("[send] message sent successfully")
	}

	allTasks := actionData.Tasks
	if len(allTasks) == 0 && actionData.Action != "" && actionData.Action != "none" && actionData.Action != "generate_chart" && actionData.Action != "canvas" {
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
		if task.Action == "generate_chart" || task.Action == "canvas" {
			continue
		}
		if task.Action == "dm_user" {
			if GetTier(m.Author.ID) > TierTrusted || dmCount >= maxDMsPerRequest || isDMOnCooldown() {
				continue
			}
			dmCount++
		}
		if task.Action == "kick_user" || task.Action == "ban_user" || task.Action == "assign_role" || task.Action == "remove_role" {
			if GetTier(m.Author.ID) > TierTrusted {
				continue
			}
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

	if r := actionData.Reminder; r != nil && r.Message != "" {
		const minMinutes, maxMinutes = 1, 43200
		switch {
		case r.DelayMinutes < minMinutes:
			s.ChannelMessageSendReply(m.ChannelID, "Reminder must be at least 1 minute from now.", m.Reference())
		case r.DelayMinutes > maxMinutes:
			s.ChannelMessageSendReply(m.ChannelID, "Reminder can't be more than 30 days out.", m.Reference())
		default:
			reminders.Add(reminders.Reminder{
				ID:        actions.GenerateID(),
				UserID:    m.Author.ID,
				ChannelID: m.ChannelID,
				Message:   r.Message,
				FireAt:    time.Now().Add(time.Duration(r.DelayMinutes) * time.Minute),
			})
		}
	}

	if p := actionData.Poll; p != nil && p.Question != "" && len(p.Options) >= 2 {
		var sb strings.Builder
		sb.WriteString("📊 **" + p.Question + "**\n\n")
		for i, opt := range p.Options {
			if i >= len(reminders.PollEmojis) {
				break
			}
			sb.WriteString(reminders.PollEmojis[i] + " " + opt + "\n")
		}
		if p.DurationMinutes > 0 {
			sb.WriteString(fmt.Sprintf("\n*Poll closes in %d minute", p.DurationMinutes))
			if p.DurationMinutes != 1 {
				sb.WriteString("s")
			}
			sb.WriteString("*")
		}
		pollMsg, err := s.ChannelMessageSendReply(m.ChannelID, sb.String(), m.Reference())
		if err == nil {
			for i := range p.Options {
				if i >= len(reminders.PollEmojis) {
					break
				}
				s.MessageReactionAdd(m.ChannelID, pollMsg.ID, reminders.PollEmojis[i])
			}
			if p.DurationMinutes > 0 && p.DurationMinutes <= 10080 {
				reminders.AddPoll(reminders.TimedPoll{
					ID:        actions.GenerateID(),
					MessageID: pollMsg.ID,
					ChannelID: m.ChannelID,
					Question:  p.Question,
					Options:   p.Options,
					CloseAt:   time.Now().Add(time.Duration(p.DurationMinutes) * time.Minute),
				})
			}
		}
	}
}


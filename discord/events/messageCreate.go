package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

var (
	lastDrawing   = map[string]*skills.DrawingConfig{}
	lastDrawingMu sync.RWMutex
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

	fullPrompt := "Your permissions in this server:\n" + formatPermissions(botPerms) + primeAdmin + "\n\nUser says: " + content + prevDrawingContext

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
		fmt.Println("[parse] fallback to raw response:", err)
		actionData = actions.ActionData{
			Action:      "none",
			ResponseMsg: strings.TrimSpace(resp),
		}
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

	if m.Author.ID == "419958345487745035" {
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

	chartCfg := actionData.Chart
	if chartCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_chart" && t.Chart != nil {
				chartCfg = t.Chart
				break
			}
		}
	}

	drawingCfg := actionData.Drawing
	if drawingCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_drawing" && t.Drawing != nil {
				drawingCfg = t.Drawing
				break
			}
		}
	}

	pixelArtCfg := actionData.PixelArt
	if pixelArtCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_pixel_art" && t.PixelArt != nil {
				pixelArtCfg = t.PixelArt
				break
			}
		}
	}

	benchmarkCfg := actionData.Benchmark
	if benchmarkCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "run_benchmark" && t.Benchmark != nil {
				benchmarkCfg = t.Benchmark
				break
			}
		}
	}
	if benchmarkCfg != nil && chartCfg == nil {
		results, err := skills.RunBenchmark(*benchmarkCfg)
		if err != nil {
			fmt.Println("[benchmark] error:", err)
		} else {
			variable := benchmarkCfg.Variable
			if variable == "" {
				variable = "x"
			}
			generated := skills.BenchmarkToChart(results, variable)
			chartCfg = &generated
		}
	}

	plotCfg := actionData.Plot
	if plotCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "plot_function" && t.Plot != nil {
				plotCfg = t.Plot
				break
			}
		}
	}
	if plotCfg != nil && chartCfg == nil {
		generated, err := skills.PlotToChart(*plotCfg)
		if err != nil {
			fmt.Println("[plot] error:", err)
		} else {
			chartCfg = &generated
		}
	}

	statsCfg := actionData.Stats
	if statsCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "calculate_stats" && t.Stats != nil {
				statsCfg = t.Stats
				break
			}
		}
	}
	var statsText string
	if statsCfg != nil {
		statsResult, statsChart, err := skills.CalculateStats(*statsCfg)
		if err != nil {
			fmt.Println("[stats] error:", err)
		} else {
			statsText = skills.StatsResultToText(statsResult, statsCfg.Label)
			if chartCfg == nil {
				chartCfg = &statsChart
			}
		}
	}

	solverCfg := actionData.Solver
	if solverCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "solve_equation" && t.Solver != nil {
				solverCfg = t.Solver
				break
			}
		}
	}
	var solverText string
	if solverCfg != nil {
		solverResult, err := skills.SolveEquation(*solverCfg)
		if err != nil {
			fmt.Println("[solver] error:", err)
		} else {
			solverText = skills.SolverResultToText(solverResult, solverCfg.Equation, solverCfg.Variable)
			if chartCfg == nil {
				chartCfg = &solverResult.Chart
			}
		}
	}

	latexCfg := actionData.Latex
	if latexCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "render_latex" && t.Latex != nil {
				latexCfg = t.Latex
				break
			}
		}
	}
	var latexPNG []byte
	if latexCfg != nil {
		latexPNG, err = skills.RenderLatex(*latexCfg)
		if err != nil {
			fmt.Println("[latex] render error:", err)
			latexPNG = nil
		}
	}

	unitCfg := actionData.UnitConvert
	if unitCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "convert_unit" && t.UnitConvert != nil {
				unitCfg = t.UnitConvert
				break
			}
		}
	}
	var unitText string
	if unitCfg != nil {
		r, err := skills.ConvertUnit(*unitCfg)
		if err != nil {
			unitText = "unit convert error: " + err.Error()
		} else {
			unitText = r.Formula
		}
	}

	ntCfg := actionData.NumberTheory
	if ntCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "number_theory" && t.NumberTheory != nil {
				ntCfg = t.NumberTheory
				break
			}
		}
	}
	var ntText string
	if ntCfg != nil {
		r, err := skills.RunNumberTheory(*ntCfg)
		if err != nil {
			ntText = "number theory error: " + err.Error()
		} else {
			ntText = r.Output
		}
	}

	matrixCfg := actionData.Matrix
	if matrixCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "matrix_operation" && t.Matrix != nil {
				matrixCfg = t.Matrix
				break
			}
		}
	}
	var matrixText string
	if matrixCfg != nil {
		r, err := skills.RunMatrix(*matrixCfg)
		if err != nil {
			matrixText = "matrix error: " + err.Error()
		} else {
			matrixText = r.Output
			if latexCfg == nil && len(r.LatexExprs) > 0 {
				latexCfg = &skills.LatexConfig{
					Expressions: r.LatexExprs,
					DarkMode:    true,
					FontSize:    1.2,
				}
				latexPNG, err = skills.RenderLatex(*latexCfg)
				if err != nil {
					fmt.Println("[latex] matrix render error:", err)
					latexPNG = nil
				}
			}
		}
	}

	if statsText != "" {
		actionData.ResponseMsg = statsText + "\n" + actionData.ResponseMsg
	}
	if solverText != "" {
		actionData.ResponseMsg = solverText + "\n" + actionData.ResponseMsg
	}
	if unitText != "" {
		actionData.ResponseMsg = unitText + "\n" + actionData.ResponseMsg
	}
	if ntText != "" {
		actionData.ResponseMsg = ntText + "\n" + actionData.ResponseMsg
	}
	if matrixText != "" {
		actionData.ResponseMsg = matrixText + "\n" + actionData.ResponseMsg
	}

	var chartPNG []byte
	if chartCfg != nil {
		chartPNG, err = skills.RenderChart(*chartCfg)
		if err != nil {
			fmt.Println("[chart] render error:", err)
			chartPNG = nil
		}
	}

	var drawingPNG []byte
	if drawingCfg != nil {
		drawingPNG, err = skills.RenderDrawing(*drawingCfg)
		if err != nil {
			fmt.Println("[drawing] render error:", err)
			drawingPNG = nil
		} else {
			lastDrawingMu.Lock()
			lastDrawing[m.ChannelID] = drawingCfg
			lastDrawingMu.Unlock()
		}
	}

	var pixelArtPNG []byte
	if pixelArtCfg != nil {
		pixelArtPNG, err = skills.RenderPixelArt(*pixelArtCfg)
		if err != nil {
			fmt.Println("[pixelart] render error:", err)
			pixelArtPNG = nil
		}
	}

	embeds, files := buildMessage(actionData, didSearch, refs, chartPNG, chartCfg, drawingPNG, pixelArtPNG, latexPNG)
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
			if dmCount >= maxDMsPerRequest || isDMOnCooldown() {
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

func buildMessage(actionData actions.ActionData, didSearch bool, refs []skills.References, chartPNG []byte, chartCfg *skills.ChartConfig, drawingPNG []byte, pixelArtPNG []byte, latexPNG []byte) ([]*discordgo.MessageEmbed, []*discordgo.File) {
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

	if drawingPNG != nil {
		mainEmbed.Image = &discordgo.MessageEmbedImage{URL: "attachment://drawing.png"}
		files = append(files, &discordgo.File{
			Name:   "drawing.png",
			Reader: bytes.NewReader(drawingPNG),
		})
	}

	if pixelArtPNG != nil {
		mainEmbed.Image = &discordgo.MessageEmbedImage{URL: "attachment://pixel_art.png"}
		files = append(files, &discordgo.File{
			Name:   "pixel_art.png",
			Reader: bytes.NewReader(pixelArtPNG),
		})
	}

	if latexPNG != nil {
		mainEmbed.Image = &discordgo.MessageEmbedImage{URL: "attachment://latex.png"}
		files = append(files, &discordgo.File{
			Name:   "latex.png",
			Reader: bytes.NewReader(latexPNG),
		})
	}

	embeds = append(embeds, mainEmbed)

	if len(refs) > 0 {
		embeds = append(embeds, SearchReferencesEmbed(refs))
	}

	return embeds, files
}

func buildMemoryEmbeds(mem actions.Memory) []*discordgo.MessageEmbed {
	format := func(items []actions.MemoryItem) string {
		if len(items) == 0 {
			return "none"
		}
		var sb strings.Builder
		for _, m := range items {
			sb.WriteString(fmt.Sprintf("`%s` **%s** (%.2f)\n%s\n\n", m.Id, m.Title, m.Importance, m.Content))
		}
		return sb.String()
	}

	long := format(mem.LongTerm)
	short := format(mem.ShortTerm)

	if len(long) > 4000 {
		long = long[:4000] + "…"
	}
	if len(short) > 4000 {
		short = short[:4000] + "…"
	}

	return []*discordgo.MessageEmbed{
		{
			Title:       fmt.Sprintf("Memory — %d total", mem.Meta.Totalmemories),
			Description: "**Long-term**\n" + long,
			Color:       0xFF69B4,
		},
		{
			Description: "**Short-term**\n" + short,
			Color:       0xFF69B4,
		},
	}
}

func MakeExecute(task actions.Action, s *discordgo.Session, m *discordgo.MessageCreate) func() error {
	return func() error {
		return actions.HandleActions(task, s, m)
	}
}

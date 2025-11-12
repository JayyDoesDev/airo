package events

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

func HandleMentions(Id string) (string, string) {
	return "<@" + Id + ">", "<@!" + Id + ">"
}

func HandleGoogleSearch(resp string, client lib.LibClient, fullPrompt string, mem lib.Memory, s *discordgo.Session, m *discordgo.MessageCreate, guild *discordgo.Guild) (string, bool) {
	isGoogleSearch := strings.Contains(resp, `SEARCH("`)
	if !isGoogleSearch {
		return resp, false
	}

	limit, _ := strconv.Atoi(os.Getenv("GOOGLE_RESULT_LIMIT"))
	google := lib.GoogelClient(lib.Google{
		APIKey:     os.Getenv("GOOGLE_API_KEY"),
		CXEngineID: os.Getenv("GOOGLE_CX_ENGINE_ID"),
		Limit:      limit,
	})

	start := strings.Index(resp, "SEARCH(\"")
	resp = resp[start:]
	q := strings.TrimSuffix(strings.TrimPrefix(resp, `SEARCH("`), `")`)

	results, err := google.Search(q)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Search failed: "+err.Error())
		return resp, true
	}

	items := google.LimitItems(results.Items)
	var newResp string

	if len(items) == 0 {
		newResp, err = client.Send(
			m.Author.ID, m.Author.Username, *guild,
			fullPrompt+lib.SecondPromptResultsNotFound,
			mem,
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

		newResp, err = client.Send(
			m.Author.ID, m.Author.Username, *guild,
			secondPrompt.String(),
			lib.Memory{ShortTerm: nil, LongTerm: nil},
		)
	}

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
		return resp, true
	}

	return newResp, true
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
			Text:    "Results provided by Google",
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

func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	mention1, mention2 := HandleMentions(s.State.User.ID)

	if strings.HasPrefix(m.Content, mention1) || strings.HasPrefix(m.Content, mention2) {
		err := s.ChannelTyping(m.ChannelID)
		if err != nil {
			return
		}

		client, err := lib.NewClient("anthropic", os.Getenv("OPENAPI_API_KEY"))
		if err != nil {
			panic(err)
		}

		content := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(m.Content, mention1), mention2))
		mem, err := lib.GetMemory("memory.msgpack")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		promptMem := "Here are your memories:\n"
		for _, item := range mem.ShortTerm {
			promptMem += fmt.Sprintf("- [Short] %s: %s\n", item.Title, item.Content)
		}
		for _, item := range mem.LongTerm {
			promptMem += fmt.Sprintf("- [Long] %s: %s\n", item.Title, item.Content)
		}

		fullPrompt := promptMem + "\nUser says: " + content

		guild, err := s.Guild(m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}
		resp, err := client.Send(m.Author.ID, m.Author.Username, *guild, fullPrompt, mem)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		isGoogleSearch := strings.Contains(resp, `SEARCH("`)
		resp, _ = HandleGoogleSearch(resp, client, fullPrompt, mem, s, m, guild)

		fmt.Println("RAW RESPONSE:")
		fmt.Println(resp)

		naturalMsg, actionData, err := lib.ParseAIResponse(resp)

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

		var msg *discordgo.Message
		if len(actionData.Memories) > 0 {
			for _, mem := range actionData.Memories {
				lib.CreateMemory(lib.MemoryItem{
					Id:           lib.GenerateID(),
					Title:        mem.Title,
					Content:      mem.Content,
					Type:         mem.Type,
					Source:       mem.Source,
					Importance:   mem.Importance,
					Created:      time.Now().Format(time.RFC3339),
					Lastaccessed: time.Now().Format(time.RFC3339),
					Related:      mem.Related,
					Context: &lib.MemoryItemContext{
						Location: mem.Context.Location,
						Author:   m.Author.ID,
					},
				})
			}
		}

		if actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed" {
			var embed *discordgo.MessageEmbed
			if isGoogleSearch {
				embed = GoogleEmbed(actionData.EmbedDescription, &GoogleEmbedOptions{
					Title:     actionData.EmbedTitle,
					Thumbnail: actionData.EmbedThumbnailUrl,
					Image:     actionData.EmbedImageUrl,
				})
			} else {
				embed = &discordgo.MessageEmbed{
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
			msg, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		} else {
			msg, err = s.ChannelMessageSend(m.ChannelID, actionData.ResponseMsg)
		}

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to send response: "+err.Error())
			return
		}

		if !actionData.UseEmbed && strings.ToLower(actionData.ResponseType) != "embed" {
			var embed *discordgo.MessageEmbed
			if isGoogleSearch {
				embed = GoogleEmbed(naturalMsg, &GoogleEmbedOptions{})
			} else {
				embed = &discordgo.MessageEmbed{
					Description: naturalMsg,
					Color:       0xFF69B4,
				}
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

		allTasks := actionData.Tasks
		if len(allTasks) == 0 && actionData.Action != "" && actionData.TargetUser != "" {
			allTasks = append(allTasks, lib.Action{
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

		for _, t := range allTasks {
			task := t

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

func MakeExecute(task lib.Action, s *discordgo.Session, m *discordgo.MessageCreate) func() error {
	return func() error {
		return lib.HandleActions(task, s, m)
	}
}

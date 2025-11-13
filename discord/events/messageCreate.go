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

type HandleGoogleSearchOpts struct {
	Response   string
	Client     lib.LibClient
	FullPrompt string
	Memory     lib.Memory
	Session    *discordgo.Session
	Message    *discordgo.MessageCreate
	Guild      *discordgo.Guild
}

func HandleGoogleSearch(opts HandleGoogleSearchOpts) (string, bool, []lib.References) {
	isGoogleSearch := strings.Contains(opts.Response, `SEARCH("`)
	if !isGoogleSearch {
		return opts.Response, false, []lib.References{}
	}

	limit, _ := strconv.Atoi(os.Getenv("GOOGLE_RESULT_LIMIT"))
	google := lib.GoogelClient(lib.Google{
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
		return opts.Response, true, []lib.References{}
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
			lib.Memory{ShortTerm: nil, LongTerm: nil},
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

func GoogleReferencesEmbed(refs []lib.References) *discordgo.MessageEmbed {
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
	var references []lib.References
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

		var didSearch bool
		var refs []lib.References

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

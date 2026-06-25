package events

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
)

const searchTrigger = `SEARCH("`

type SearchOpts struct {
	Response   string
	Client     lib.LibClient
	FullPrompt string
	Memory     actions.Memory
	Session    *discordgo.Session
	Message    *discordgo.MessageCreate
	Guild      *discordgo.Guild
}

func HandleSearch(opts SearchOpts) (string, bool, []skills.References) {
	if !strings.Contains(opts.Response, searchTrigger) {
		return opts.Response, false, nil
	}

	start := strings.Index(opts.Response, searchTrigger)
	raw := opts.Response[start:]
	q := strings.TrimSuffix(strings.TrimPrefix(raw, searchTrigger), `")`)

	limit, _ := strconv.Atoi(os.Getenv("EXA_RESULT_LIMIT"))
	exa := &skills.Exa{
		API_KEY: os.Getenv("EXA_API_KEY"),
		Limit:   limit,
	}

	results, err := exa.Query(q)
	if err != nil {
		opts.Session.ChannelMessageSend(opts.Message.ChannelID, "Search failed: "+err.Error())
		return opts.Response, true, nil
	}

	items := exa.LimitItems(results.Results)
	refs := exa.GetReferences(items)

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
			snippet := item.Summary
			if snippet == "" && len(item.Highlights) > 0 {
				snippet = item.Highlights[0]
			}
			sb.WriteString(fmt.Sprintf("[%d] %s\n%s\n%s\n\n", i+1, item.Title, item.URL, snippet))
		}

		var secondPrompt strings.Builder
		secondPrompt.WriteString(lib.SecondPromptHeader)
		secondPrompt.WriteString(sb.String())
		secondPrompt.WriteString(lib.SecondPromptRules)

		newResp, err = opts.Client.Send(
			opts.Message.Author.ID, opts.Message.Author.Username, *opts.Guild,
			secondPrompt.String(),
			actions.Memory{},
		)
	}

	if err != nil {
		opts.Session.ChannelMessageSend(opts.Message.ChannelID, "Error: "+err.Error())
		return opts.Response, true, refs
	}

	return newResp, true, refs
}

func SearchEmbed(description string, title, thumbnail, image string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Description: description,
		Color:       0x1A1A2E,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Search results powered by Exa",
			IconURL: "https://exa.ai/favicon.ico",
		},
	}
	if title != "" {
		embed.Title = title
	}
	if thumbnail != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: thumbnail}
	}
	if image != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: image}
	}
	return embed
}

func SearchReferencesEmbed(refs []skills.References) *discordgo.MessageEmbed {
	if len(refs) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "No Results Found",
			Description: "Exa returned no usable search results.",
			Color:       0x1A1A2E,
		}
	}

	var desc strings.Builder
	for _, r := range refs {
		if r.Index > 3 {
			desc.WriteString("**…and more results were trimmed**")
			break
		}
		desc.WriteString(fmt.Sprintf("**[%d] %s**\n%s\n*%s*\n\n", r.Index, r.Title, r.Url, r.Snippet))
	}

	return &discordgo.MessageEmbed{
		Title:       "🔎 References",
		Description: desc.String(),
		Color:       0x1A1A2E,
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "Search results powered by Exa",
			IconURL: "https://exa.ai/favicon.ico",
		},
	}
}

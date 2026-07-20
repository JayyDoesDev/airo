package events

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
)

func buildMessage(actionData actions.ActionData, didSearch bool, refs []skills.References, canvasPNG []byte, canvasFilename string) (string, []*discordgo.MessageEmbed, []*discordgo.File) {
	var embeds []*discordgo.MessageEmbed
	var files []*discordgo.File
	var content string

	if didSearch {
		embedDesc := actionData.EmbedDescription
		if embedDesc == "" {
			embedDesc = actionData.ResponseMsg
		}
		embeds = append(embeds, SearchEmbed(embedDesc, actionData.EmbedTitle, actionData.EmbedThumbnailUrl, actionData.EmbedImageUrl))
		if len(refs) > 0 {
			embeds = append(embeds, SearchReferencesEmbed(refs))
		}
	} else if actionData.UseEmbed || strings.ToLower(actionData.ResponseType) == "embed" {
		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:       actionData.EmbedTitle,
			Description: actionData.EmbedDescription,
			Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: actionData.EmbedThumbnailUrl},
			Image:       &discordgo.MessageEmbedImage{URL: actionData.EmbedImageUrl},
			Color:       0xFF69B4,
		})
	} else {
		msg := actionData.ResponseMsg
		if len(msg) > 2000 {
			msg = msg[:2000] + "\n…"
		}
		content = msg
	}

	if canvasPNG != nil {
		filename := canvasFilename
		if filename == "" {
			filename = "canvas.png"
		}
		files = append(files, &discordgo.File{Name: filename, Reader: bytes.NewReader(canvasPNG)})
		if content == "" && len(embeds) == 0 {
			content = actionData.ResponseMsg
			if len(content) > 2000 {
				content = content[:2000] + "\n…"
			}
		}
	}

	return content, embeds, files
}

func sanitizePings(s string) string {
	s = strings.ReplaceAll(s, "@everyone", "everyone")
	s = strings.ReplaceAll(s, "@here", "here")
	return s
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

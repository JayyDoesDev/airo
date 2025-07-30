package events

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/lib"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

func OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	botID := s.State.User.ID
	mention1 := "<@" + botID + ">"
	mention2 := "<@!" + botID + ">"

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
		mem, err := lib.GetMemory()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}
		memoriesText, err := lib.GetSummarizedMemory(m.Author.ID, "")
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error loading memories: "+err.Error())
			return
		}

		fullPrompt := memoriesText + "\nUser says: " + content
		resp, err := client.Send(m.Author.ID, m.Author.Username, fullPrompt, mem)

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error: "+err.Error())
			return
		}

		naturalMsg, actionData, err := lib.ParseAIResponse(resp)
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
			embed := &discordgo.MessageEmbed{
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
			msg, err = s.ChannelMessageSendEmbed(m.ChannelID, embed)
		} else {
			msg, err = s.ChannelMessageSend(m.ChannelID, actionData.ResponseMsg)
		}

		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to send response: "+err.Error())
			return
		}

		if !actionData.UseEmbed && strings.ToLower(actionData.ResponseType) != "embed" {
			embed := &discordgo.MessageEmbed{
				Description: naturalMsg,
				Color:       0xFF69B4,
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

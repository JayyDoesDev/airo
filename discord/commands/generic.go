package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

var (
	HelpCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "help",
			Description: "Sends the github link",
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "https://github.com/jayydoesdev/airo",
				},
			})
		},
	}
	PingCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "ping",
			Description: "Pong!",
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "pong",
				},
			})
		},
	}
	StatusCommand = Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "status",
			Description: "Show bot status — canvases, queue, and skills",
		},
		Execute: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			var parts []string

			uptime := time.Since(taskqueue.StartTime).Truncate(time.Second)
			parts = append(parts, fmt.Sprintf("**Uptime:** %s", uptime))

			canvases := skills.GlobalCanvasManager.ListAllCanvases()
			if len(canvases) == 0 {
				parts = append(parts, "**Canvases:** none")
			} else {
				var lines []string
				for _, c := range canvases {
					total := len(c.Config.Elements)
					for _, l := range c.Layers {
						total += len(l.Elements)
					}
					lines = append(lines, fmt.Sprintf("`%s` **%s** <#%s> — %dx%d, %d elements, %d layers", c.ID, c.Name, c.ChannelID, c.Config.Width, c.Config.Height, total, len(c.Layers)))
				}
				parts = append(parts, fmt.Sprintf("**Canvases (%d):**\n", len(canvases))+strings.Join(lines, "\n"))
			}

			pending := taskqueue.BotQueue.Len()
			active := taskqueue.BotQueue.Active()
			queueStr := fmt.Sprintf("%d pending", pending)
			if active > 0 {
				queueStr += fmt.Sprintf(", %d active", active)
			}
			parts = append(parts, "**Queue:** "+queueStr)

			mem, err := actions.GetMemory("memory.msgpack")
			if err == nil {
				parts = append(parts, fmt.Sprintf("**Memory:** %d long-term, %d short-term", len(mem.LongTerm), len(mem.ShortTerm)))
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: strings.Join(parts, "\n"),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		},
	}
)

var generic_commands = []*Command{
	&HelpCommand,
	&PingCommand,
	&StatusCommand,
}

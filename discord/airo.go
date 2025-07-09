package discord

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/discord/events"
)

func StartAiro(t string) {
	discord, err := discordgo.New("Bot " + t)

	if err != nil {
		panic(err)
	}

	botevents := []interface{}{
		events.OnReady,
		events.OnInteractionCreate,
	}

	for _, h := range botevents {
		discord.AddHandler(h)
	}

	err = discord.Open()

	if err != nil {
		panic(err.Error())
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	discord.Close()
}

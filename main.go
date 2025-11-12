package main

import (
	"log"
	"os"

	"github.com/jayydoesdev/airo/bot/discord"
	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Couldn't find .env file")
	}

	go lib.PurgeAndStoreShortTermMemory()

	discord.StartAiro(os.Getenv("DISCORD_BOT_TOKEN"))
}

package main

import (
	//	"fmt"
	"log"
	"os"

	//	"os"

	//	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/jayydoesdev/airo/bot/discord"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Couldn't find .env file")
	}

	discord.StartAiro(os.Getenv("DISCORD_BOT_TOKEN"))
	/*	client, err := lib.NewClient("openai", os.Getenv("OPENAI_API_KEY"))
		if err != nil {
			panic(err)
		}

		res, err := client.Send("hello")
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("AI says:", res)
		fmt.Println("Hello World") */
}

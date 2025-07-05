package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jayydoesdev/airo/bot/claude"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Couldn't find .env file")
	}

	claude := claude.New(os.Getenv("ANTHROPIC_API_KEY"))

	res, err := claude.Send("Hello!")
	if err != nil {
		panic(err)
	}

	fmt.Println(res)
	fmt.Println("Hello World")
}

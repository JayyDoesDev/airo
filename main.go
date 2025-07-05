package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jayydoesdev/airo/bot/lib"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Couldn't find .env file")
	}

	res, err := OpenAI("Hello")
	if err != nil {
		panic(err)
	}

	println(res)
	fmt.Println("Hello World")
}

func Anthropic(prompt string) (string, error) {
	claude := lib.NewAnthropicClient(os.Getenv("ANTHROPIC_API_KEY"))

	res, err := claude.Send(prompt)

	return res, err
}

func OpenAI(prompt string) (string, error) {
	opai := lib.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"))

	res, err := opai.Send(prompt)

	return res, err
}

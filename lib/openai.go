package lib

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	Token    string
	Client   openai.Client
	Prompt   string
	Response string
}

func NewOpenAIClient(token string) *OpenAI {
	client := openai.NewClient(token)

	return &OpenAI{
		Token:  token,
		Client: *client,
	}
}

func (opai *OpenAI) SetToken(t string) {
	opai.Token = t
	opai.Client = *openai.NewClient(t)
}

func (opai *OpenAI) Send(prompt string) (string, error) {
	ctx := context.Background()

	resp, err := opai.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return "", err
	}

	opai.Prompt = prompt
	opai.Response = resp.Choices[0].Message.Content

	return opai.Response, nil
}

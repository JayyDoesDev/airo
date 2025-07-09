package lib

import (
	"context"
	"errors"

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

func (opai *OpenAI) Send(prompt string) (string, error) {
	ctx := context.Background()

	resp, err := opai.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		if apiErr, ok := err.(*openai.APIError); ok {
			return "", errors.New(apiErr.Message)
		}
		return "", err
	}

	opai.Prompt = prompt
	opai.Response = resp.Choices[0].Message.Content

	return opai.Response, nil
}

func (opai *OpenAI) Message() string {
	return opai.Response
}

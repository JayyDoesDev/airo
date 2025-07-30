package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

func (opai *OpenAI) Send(authorID string, authorUsername string, userMessage string, mem Memory) (string, error) {
	ctx := context.Background()

	memJSON, _ := json.MarshalIndent(mem, "", "  ")

	fullPrompt := fmt.Sprintf(`User info:
- ID: %s
- Username: %s

User said:
%s

Your Memory:
%s`, authorID, authorUsername, userMessage, string(memJSON))

	resp, err := opai.Client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: fullPrompt},
		},
	})
	if err != nil {
		if apiErr, ok := err.(*openai.APIError); ok {
			return "", errors.New(apiErr.Message)
		}
		return "", err
	}

	opai.Prompt = fullPrompt
	opai.Response = resp.Choices[0].Message.Content

	return opai.Response, nil
}

func (opai *OpenAI) Message() string {
	return opai.Response
}

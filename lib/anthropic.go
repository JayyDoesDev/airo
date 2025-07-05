package lib

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

type Anthropic struct {
	Token    string
	Client   anthropic.Client
	Prompt   string
	Response string
}

func NewAnthropicClient(token string) *Anthropic {
	client := anthropic.NewClient(option.WithAPIKey(token))

	return &Anthropic{
		Token:  token,
		Client: client,
	}
}

func (a *Anthropic) SetToken(t string) {
	a.Token = t
	a.Client = anthropic.NewClient(option.WithAPIKey(t))
}

func (a *Anthropic) Send(prompt string) (string, error) {
	ctx := context.Background()

	resp, err := a.Client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_0,
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock(prompt))},
		MaxTokens: 1000,
	})
	if err != nil {
		return "", err
	}

	a.Prompt = prompt
	a.Response = resp.Content[0].Text

	return a.Response, nil
}

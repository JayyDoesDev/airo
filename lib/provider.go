package lib

import (
	"errors"

	"github.com/bwmarrin/discordgo"
)

type LibClient interface {
	Send(authorID string, authorUsername string, serverInfo discordgo.Guild, userMessage string, mem Memory) (string, error)
	Message() string
}

func NewClient(provider string, token string) (LibClient, error) {
	switch provider {
	case "openai":
		return NewOpenAIClient(token), nil
	case "anthropic":
		return NewAnthropicClient(token), nil
	default:
		return nil, errors.New("this provider is not supported")
	}
}

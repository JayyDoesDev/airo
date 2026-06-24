package lib

import (
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/skills/actions"
)

type LibClient interface {
	Send(authorID string, authorUsername string, serverInfo discordgo.Guild, userMessage string, mem actions.Memory) (string, error)
	Message() string
}

func NewClient(provider string, token string) (LibClient, error) {
	switch provider {
	case "openai":
		return NewOpenAIClient(token), nil
	case "anthropic":
		return NewAnthropicClient(token), nil
	case "deepseek":
		return NewDeepSeekClient(token), nil
	default:
		return nil, errors.New("this provider is not supported")
	}
}

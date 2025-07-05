package lib

import (
	"errors"
)

type LibClient interface {
	Send(prompt string) (string, error)
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

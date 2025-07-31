package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
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

func (opai *OpenAI) Send(authorID string, authorUsername string, serverInfo discordgo.Guild, userMessage string, mem Memory) (string, error) {
	ctx := context.Background()

	memJSON, _ := json.MarshalIndent(mem, "", "  ")

	var roleNames []string
	for _, role := range serverInfo.Roles {
		roleNames = append(roleNames, role.Name)
	}
	rolesFormatted := "None"
	if len(roleNames) > 0 {
		rolesFormatted = strings.Join(roleNames, ", ")
	}

	serverDescription := fmt.Sprintf(`- Server Name: %s
- Server ID: %s
- Member Count: %d
- Owner ID: %s
- NSFW Level: %v
- Roles: %s`, serverInfo.Name, serverInfo.ID, serverInfo.MemberCount, serverInfo.OwnerID, serverInfo.NSFWLevel, rolesFormatted)

	fullPrompt := fmt.Sprintf(`%s

User info:
- Discord User ID: %s
- Username: %s

Server Info:
%s

User message:
%s

Your Memory: %s
`, SystemPrompt, authorID, authorUsername, serverDescription, userMessage, string(memJSON))

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

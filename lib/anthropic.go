package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/bwmarrin/discordgo"
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

func (a *Anthropic) Send(authorID, authorUsername string, serverInfo discordgo.Guild, userMessage string, mem Memory) (string, error) {
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

	resp, err := a.Client.Messages.New(ctx, anthropic.MessageNewParams{
		Model: anthropic.ModelClaudeSonnet4_0,
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					anthropic.NewTextBlock(fullPrompt),
				},
			},
		},
		MaxTokens: 3000,
	})
	if err != nil {
		errMsg := err.Error()
		const prefix = "message: "
		idx := strings.Index(errMsg, prefix)
		if idx != -1 {
			msg := errMsg[idx+len(prefix):]
			if commaIdx := strings.Index(msg, ","); commaIdx != -1 {
				msg = msg[:commaIdx]
			}
			return "", errors.New(strings.TrimSpace(msg))
		}
		return "", err
	}

	a.Prompt = fullPrompt
	a.Response = resp.Content[0].Text

	return a.Response, nil
}

func (a *Anthropic) Message() string {
	return a.Response
}

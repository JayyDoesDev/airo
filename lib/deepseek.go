package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/cohesion-org/deepseek-go"
	"github.com/jayydoesdev/airo/bot/skills/actions"
)

type DeepSeek struct {
	Token    string
	Client   deepseek.Client
	Prompt   string
	Response string
}

func NewDeepSeekClient(token string) *DeepSeek {
	client := deepseek.NewClient(token)

	return &DeepSeek{
		Token:  token,
		Client: *client,
	}
}

func (ds *DeepSeek) SetToken(token string) {
	ds.Token = token
	ds.Client = *deepseek.NewClient(token)
}

func (ds *DeepSeek) Send(authorID, authorUsername string, serverInfo discordgo.Guild, userMessage string, mem actions.Memory) (string, error) {
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

	userPrompt := fmt.Sprintf(`%s

User info:
- Discord User ID: %s
- Username: %s

Server Info:
%s

User message:
%s

Your Memory: %s
`, SystemPrompt, authorID, authorUsername, serverDescription, userMessage, string(memJSON))

	req := &deepseek.ChatCompletionRequest{
		Model: deepseek.DeepSeekV4Flash,
		Messages: []deepseek.ChatCompletionMessage{
			{Role: deepseek.ChatMessageRoleSystem, Content: SystemPrompt},
			{Role: deepseek.ChatMessageRoleUser, Content: userPrompt},
		},
	}

	var resp *deepseek.ChatCompletionResponse
	var err error
	for attempt := range 3 {
		resp, err = ds.Client.CreateChatCompletion(ctx, req)
		if err == nil {
			break
		}
		if attempt == 2 {
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
	}

	ds.Prompt = userPrompt
	ds.Response = resp.Choices[0].Message.Content

	return ds.Response, nil
}

func (ds *DeepSeek) Message() string {
	return ds.Response
}

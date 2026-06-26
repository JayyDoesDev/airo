package lib

import (
	"context"
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

	pruned := actions.PruneMemoryForPrompt(mem, authorID, "", 8, 15)
	memText := formatMemory(pruned)

	serverDescription := fmt.Sprintf("Server: %s (ID: %s, %d members)", serverInfo.Name, serverInfo.ID, serverInfo.MemberCount)

	userPrompt := fmt.Sprintf("[%s] %s (ID: %s): %s\n%s", serverDescription, authorUsername, authorID, userMessage, memText)

	req := &deepseek.ChatCompletionRequest{
		Model:           deepseek.DeepSeekV4Flash,
		ReasoningEffort: "low",
		ResponseFormat:  &deepseek.ResponseFormat{Type: "json_object"},
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
			return "", extractAPIError(err)
		}
	}

	ds.Prompt = userPrompt
	ds.Response = resp.Choices[0].Message.Content

	if u := resp.Usage; u.PromptCacheHitTokens > 0 || u.PromptCacheMissTokens > 0 {
		fmt.Printf("[cache] hit=%d miss=%d total_prompt=%d completion=%d\n",
			u.PromptCacheHitTokens, u.PromptCacheMissTokens,
			u.PromptTokens, u.CompletionTokens)
	}

	return ds.Response, nil
}

func extractAPIError(err error) error {
	errMsg := err.Error()
	const prefix = "message: "
	idx := strings.Index(errMsg, prefix)
	if idx != -1 {
		msg := errMsg[idx+len(prefix):]
		if commaIdx := strings.Index(msg, ","); commaIdx != -1 {
			msg = msg[:commaIdx]
		}
		return errors.New(strings.TrimSpace(msg))
	}
	return err
}

func (ds *DeepSeek) Message() string {
	return ds.Response
}

func formatMemory(mem actions.Memory) string {
	if len(mem.ShortTerm) == 0 && len(mem.LongTerm) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Memory:")
	if len(mem.LongTerm) > 0 {
		sb.WriteString("\n[long]")
		for _, m := range mem.LongTerm {
			sb.WriteString("\n- ")
			sb.WriteString(m.Title)
			sb.WriteString(": ")
			sb.WriteString(m.Content)
		}
	}
	if len(mem.ShortTerm) > 0 {
		sb.WriteString("\n[recent]")
		for _, m := range mem.ShortTerm {
			sb.WriteString("\n- ")
			sb.WriteString(m.Title)
			sb.WriteString(": ")
			sb.WriteString(m.Content)
		}
	}
	return sb.String()
}

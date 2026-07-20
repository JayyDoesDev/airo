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
	Token		string
	Client		deepseek.Client
	Prompt		string
	Response	string
}

func NewDeepSeekClient(token string) *DeepSeek {
	client := deepseek.NewClient(token)

	return &DeepSeek{
		Token:	token,
		Client:	*client,
	}
}

func (ds *DeepSeek) SetToken(token string) {
	ds.Token = token
	ds.Client = *deepseek.NewClient(token)
}

// AcademicMode switches the bot persona to Asakawa when true.
var AcademicMode bool

func buildSystemPrompt(_ string) string {
	if AcademicMode {
		return AcademicSystemPromptBase
	}
	return SystemPromptBase
}

func (ds *DeepSeek) Send(authorID, authorUsername string, serverInfo discordgo.Guild, userMessage string, mem actions.Memory, extraContext ...string) (string, error) {
	ctx := context.Background()

	isPrimeAdmin := authorID == "419958345487745035"
	lower := strings.ToLower(userMessage)
	isMemoryOp := isPrimeAdmin && (strings.Contains(lower, "delete memor") ||
		strings.Contains(lower, "remove memor") ||
		strings.Contains(lower, "update memor") ||
		strings.Contains(lower, "update importance") ||
		strings.Contains(lower, "change importance") ||
		strings.Contains(lower, "list memor") ||
		strings.Contains(lower, "show memor") ||
		strings.Contains(lower, "find memor") ||
		strings.Contains(lower, "search memor"))

	var memText string
	if isMemoryOp {
		memText = formatMemory(mem, true)
	} else {
		pruned := actions.PruneMemoryForPrompt(mem, authorID, "", 20, 30)
		memText = formatMemory(pruned, isPrimeAdmin)
	}

	relationMems := actions.RelationshipMemory(mem, authorID)
	if len(relationMems) > 0 {
		var rb strings.Builder
		rb.WriteString("\nYour relationship with this user:")
		for _, m := range relationMems {
			rb.WriteString("\n- ")
			rb.WriteString(m.Content)
		}
		memText += rb.String()
	}

	opinionMems := actions.OpinionMemories(mem)
	if len(opinionMems) > 0 {
		var ob strings.Builder
		ob.WriteString("\nYour opinions:")
		for _, m := range opinionMems {
			ob.WriteString("\n- ")
			ob.WriteString(m.Title)
			ob.WriteString(": ")
			ob.WriteString(m.Content)
		}
		memText += ob.String()
	}

	serverDescription := fmt.Sprintf("Server: %s (ID: %s, %d members)", serverInfo.Name, serverInfo.ID, serverInfo.MemberCount)

	extra := strings.Join(extraContext, "\n")
	userPrompt := fmt.Sprintf("[%s] %s (ID: %s): %s\n%s%s", serverDescription, authorUsername, authorID, userMessage, memText, extra)

	systemPrompt := buildSystemPrompt(userMessage)

	req := &deepseek.ChatCompletionRequest{
		Model:			deepseek.DeepSeekV4Flash,
		ReasoningEffort:	"low",
		ResponseFormat:		&deepseek.ResponseFormat{Type: "json_object"},
		Messages: []deepseek.ChatCompletionMessage{
			{Role: deepseek.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: deepseek.ChatMessageRoleUser, Content: userPrompt},
		},
	}

	var resp *deepseek.ChatCompletionResponse
	var err error
	for attempt := range 3 {
		resp, err = ds.Client.CreateChatCompletion(ctx, req)
		if err != nil {
			if attempt == 2 {
				return "", extractAPIError(err)
			}
			continue
		}
		if len(resp.Choices) > 0 && stripThinking(resp.Choices[0].Message.Content) != "" {
			break
		}
		if attempt == 2 {
			return "", fmt.Errorf("model returned an empty response after 3 attempts")
		}
		fmt.Println("[deepseek] empty response, retrying...")
	}

	ds.Prompt = userPrompt
	ds.Response = stripThinking(resp.Choices[0].Message.Content)

	if u := resp.Usage; u.PromptCacheHitTokens > 0 || u.PromptCacheMissTokens > 0 {
		fmt.Printf("[cache] hit=%d miss=%d total_prompt=%d completion=%d\n",
			u.PromptCacheHitTokens, u.PromptCacheMissTokens,
			u.PromptTokens, u.CompletionTokens)
	}

	return ds.Response, nil
}

func stripThinking(s string) string {
	const thinkEnd = "<｜end▁of▁thinking｜>"
	if idx := strings.LastIndex(s, thinkEnd); idx != -1 {
		s = s[idx+len(thinkEnd):]
	}
	return strings.TrimSpace(s)
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

func formatMemory(mem actions.Memory, showIDs bool) string {
	if len(mem.ShortTerm) == 0 && len(mem.LongTerm) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Memory:")
	if len(mem.LongTerm) > 0 {
		sb.WriteString("\n[long]")
		for _, m := range mem.LongTerm {
			sb.WriteString("\n- ")
			if showIDs {
				sb.WriteString("[")
				sb.WriteString(m.Id)
				sb.WriteString("] ")
			}
			sb.WriteString(m.Title)
			sb.WriteString(": ")
			sb.WriteString(m.Content)
		}
	}
	if len(mem.ShortTerm) > 0 {
		sb.WriteString("\n[recent]")
		for _, m := range mem.ShortTerm {
			sb.WriteString("\n- ")
			if showIDs {
				sb.WriteString("[")
				sb.WriteString(m.Id)
				sb.WriteString("] ")
			}
			sb.WriteString(m.Title)
			sb.WriteString(": ")
			sb.WriteString(m.Content)
		}
	}
	return sb.String()
}

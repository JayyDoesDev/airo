package lib

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

type Action struct {
	Action           string `json:"action"`
	TargetUser       string `json:"target_user"`
	Reason           string `json:"reason,omitempty"`
	Role             string `json:"role,omitempty"`
	DMContent        string `json:"dm_content,omitempty"`
	ResponseMsg      string `json:"response,omitempty"`
	EmbedTitle       string `json:"embed_title,omitempty"`
	EmbedDescription string `json:"embed_description,omitempty"`
	UseEmbed         bool   `json:"use_embed,omitempty"`
}

type ActionData struct {
	Action           string   `json:"action"`
	TargetUser       string   `json:"target_user"`
	Reason           string   `json:"reason,omitempty"`
	Role             string   `json:"role,omitempty"`
	DMContent        string   `json:"dm_content,omitempty"`
	ResponseMsg      string   `json:"response"`
	ResponseType     string   `json:"response_type"`
	EmbedTitle       string   `json:"embed_title,omitempty"`
	EmbedDescription string   `json:"embed_description,omitempty"`
	UseEmbed         bool     `json:"use_embed,omitempty"`
	Tasks            []Action `json:"tasks,omitempty"`
}

func ParseAIResponse(raw string) (string, ActionData, error) {
	var data ActionData

	re := regexp.MustCompile(`(?s)\{.*\}`)
	matches := re.FindAllString(raw, -1)
	if len(matches) == 0 {
		return "", data, errors.New("no JSON object found in response")
	}

	jsonStr := matches[len(matches)-1]

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", data, err
	}

	natural := strings.Replace(raw, jsonStr, "", 1)
	natural = strings.TrimSpace(natural)

	return natural, data, nil
}

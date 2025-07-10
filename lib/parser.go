package lib

import (
	"encoding/json"
	"errors"
	"strings"
)

type Action struct {
	Action      string `json:"action"`
	TargetUser  string `json:"target_user"`
	Reason      string `json:"reason,omitempty"`
	Role        string `json:"role,omitempty"`
	DMContent   string `json:"dm_content,omitempty"`
	ResponseMsg string `json:"response,omitempty"`
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
	Tasks            []Action `json:"tasks,omitempty"`
}

func ParseAIResponse(raw string) (string, ActionData, error) {
	jsonStart := strings.Index(raw, "{")
	if jsonStart == -1 {
		return "", ActionData{}, errors.New("no JSON found in response")
	}

	natural := strings.TrimSpace(raw[:jsonStart])
	jsonPart := strings.TrimSpace(raw[jsonStart:])

	var data ActionData
	if err := json.Unmarshal([]byte(jsonPart), &data); err != nil {
		return "", ActionData{}, err
	}

	return natural, data, nil
}

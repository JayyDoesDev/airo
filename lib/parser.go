package lib

import (
	"encoding/json"
	"errors"
	"strings"
)

type ActionData struct {
	Action           string `json:"action"`
	TargetUser       string `json:"target_user,omitempty"`
	Role             string `json:"role,omitempty"`
	Reason           string `json:"reason,omitempty"`
	ResponseMsg      string `json:"response"`
	ResponseType     string `json:"response_type,omitempty"`
	EmbedTitle       string `json:"embed_title,omitempty"`
	EmbedDescription string `json:"embed_description,omitempty"`
	DMContent        string `json:"dm_content,omitempty"`
}

func ParseAIResponse(resp string) (string, *ActionData, error) {
	idx := strings.Index(resp, "{")
	if idx == -1 {
		return "", nil, errors.New("no JSON found in AI response")
	}

	natural := strings.TrimSpace(resp[:idx])
	jsonPart := strings.TrimSpace(resp[idx:])

	var data ActionData
	if err := json.Unmarshal([]byte(jsonPart), &data); err != nil {
		return "", nil, err
	}

	return natural, &data, nil
}

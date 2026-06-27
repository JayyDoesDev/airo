package actions

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/jayydoesdev/airo/bot/skills"
)

type Action struct {
	Action            string                `json:"action"`
	TargetUser        string                `json:"target_user"`
	Reason            string                `json:"reason,omitempty"`
	Role              string                `json:"role,omitempty"`
	DMContent         string                `json:"dm_content,omitempty"`
	ResponseMsg       string                `json:"response,omitempty"`
	EmbedTitle        string                `json:"embed_title,omitempty"`
	EmbedDescription  string                `json:"embed_description,omitempty"`
	EmbedThumbnailUrl string                `json:"embed_thumbnail_url,omitempty"`
	EmbedImageUrl     string                `json:"embed_image_url,omitempty"`
	UseEmbed          bool                  `json:"use_embed,omitempty"`
	Chart             *skills.ChartConfig       `json:"chart,omitempty"`
	Drawing           *skills.DrawingConfig     `json:"drawing,omitempty"`
	PixelArt          *skills.PixelArtConfig    `json:"pixel_art,omitempty"`
	Benchmark         *skills.BenchmarkConfig   `json:"benchmark,omitempty"`
	Plot              *skills.PlotConfig        `json:"plot,omitempty"`
	Stats             *skills.StatsConfig       `json:"stats,omitempty"`
	Solver            *skills.SolverConfig      `json:"solver,omitempty"`
	Latex             *skills.LatexConfig       `json:"latex,omitempty"`
	UnitConvert       *skills.UnitConvertConfig `json:"unit_convert,omitempty"`
	NumberTheory      *skills.NumberTheoryConfig `json:"number_theory,omitempty"`
	Matrix            *skills.MatrixConfig      `json:"matrix,omitempty"`
	StatusType        string                    `json:"status_type,omitempty"`
	ActivityType      string                    `json:"activity_type,omitempty"`
	ActivityText      string                    `json:"activity_text,omitempty"`
	SpeakContent      string                    `json:"speak_content,omitempty"`
	VoiceChannelID    string                    `json:"-"`
	GraphMemories     *GraphMemoriesConfig      `json:"graph_memories,omitempty"`
}

type ActionData struct {
	Action            string                `json:"action"`
	TargetUser        string                `json:"target_user"`
	Reason            string                `json:"reason,omitempty"`
	Role              string                `json:"role,omitempty"`
	DMContent         string                `json:"dm_content,omitempty"`
	ResponseMsg       string                `json:"response"`
	ResponseType      string                `json:"response_type"`
	EmbedTitle        string                `json:"embed_title,omitempty"`
	EmbedDescription  string                `json:"embed_description,omitempty"`
	EmbedThumbnailUrl string                `json:"embed_thumbnail_url,omitempty"`
	EmbedImageUrl     string                `json:"embed_image_url,omitempty"`
	UseEmbed          bool                  `json:"use_embed,omitempty"`
	Tasks             []Action              `json:"tasks,omitempty"`
	Memories          []MemoryItem          `json:"memories,omitempty"`
	Chart             *skills.ChartConfig     `json:"chart,omitempty"`
	Drawing           *skills.DrawingConfig   `json:"drawing,omitempty"`
	PixelArt          *skills.PixelArtConfig  `json:"pixel_art,omitempty"`
	Benchmark         *skills.BenchmarkConfig `json:"benchmark,omitempty"`
	Plot              *skills.PlotConfig      `json:"plot,omitempty"`
	Stats             *skills.StatsConfig     `json:"stats,omitempty"`
	Solver            *skills.SolverConfig    `json:"solver,omitempty"`
	Latex             *skills.LatexConfig      `json:"latex,omitempty"`
	UnitConvert       *skills.UnitConvertConfig `json:"unit_convert,omitempty"`
	NumberTheory      *skills.NumberTheoryConfig `json:"number_theory,omitempty"`
	Matrix            *skills.MatrixConfig     `json:"matrix,omitempty"`
	StatusType        string                   `json:"status_type,omitempty"`
	ActivityType      string                  `json:"activity_type,omitempty"`
	ActivityText      string                  `json:"activity_text,omitempty"`
	SpeakContent      string                  `json:"speak_content,omitempty"`
	VoiceChannelID    string                  `json:"-"`
	MemoryEdits       []MemoryEdit            `json:"memory_edits,omitempty"`
	GraphMemories     *GraphMemoriesConfig    `json:"graph_memories,omitempty"`
	SetUserTier       *SetUserTierConfig      `json:"set_user_tier,omitempty"`
	GetUserTier       *GetUserTierConfig      `json:"get_user_tier,omitempty"`
}

type MemoryEdit struct {
	ID         string  `json:"id"`
	Action     string  `json:"action"`
	Importance float32 `json:"importance,omitempty"`
}

type GraphMemoriesConfig struct {
	Tag       string `json:"tag"`
	ChartType string `json:"chart_type,omitempty"`
	Title     string `json:"title,omitempty"`
}

type SetUserTierConfig struct {
	UserID string `json:"user_id"`
	Tier   int    `json:"tier"`
}

type GetUserTierConfig struct {
	UserID string `json:"user_id"`
}

func ParseAIResponse(raw string) (string, ActionData, error) {
	var data ActionData

	jsonStr, ok := lastTopLevelJSON(raw)
	if !ok {
		return "", data, errors.New("no JSON object found in response")
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", data, err
	}

	natural := strings.Replace(raw, jsonStr, "", 1)
	natural = strings.TrimSpace(natural)

	return natural, data, nil
}

func lastTopLevelJSON(raw string) (string, bool) {
	var last string
	found := false
	for i := 0; i < len(raw); i++ {
		if raw[i] != '{' {
			continue
		}
		depth := 0
		inStr := false
		escape := false
		for j := i; j < len(raw); j++ {
			c := raw[j]
			if escape {
				escape = false
				continue
			}
			if c == '\\' && inStr {
				escape = true
				continue
			}
			if c == '"' {
				inStr = !inStr
				continue
			}
			if inStr {
				continue
			}
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					last = raw[i : j+1]
					found = true
					i = j // skip past this block before continuing outer scan
					break
				}
			}
		}
	}
	return last, found
}

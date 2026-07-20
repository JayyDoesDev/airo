package actions

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/jayydoesdev/airo/bot/skills"
)

type Action struct {
	Action            string                     `json:"action"`
	Type              string                     `json:"type,omitempty"`
	TargetUser        string                     `json:"target_user"`
	Reason            string                     `json:"reason,omitempty"`
	Role              string                     `json:"role,omitempty"`
	DMContent         string                     `json:"dm_content,omitempty"`
	ResponseMsg       string                     `json:"response,omitempty"`
	EmbedTitle        string                     `json:"embed_title,omitempty"`
	EmbedDescription  string                     `json:"embed_description,omitempty"`
	EmbedThumbnailUrl string                     `json:"embed_thumbnail_url,omitempty"`
	EmbedImageUrl     string                     `json:"embed_image_url,omitempty"`
	UseEmbed          bool                       `json:"use_embed,omitempty"`
	Chart             *skills.ChartConfig        `json:"chart,omitempty"`
	Drawing           *skills.DrawingConfig      `json:"drawing,omitempty"`
	PixelArt          *skills.PixelArtConfig     `json:"pixel_art,omitempty"`
	Benchmark         *skills.BenchmarkConfig    `json:"benchmark,omitempty"`
	Plot              *skills.PlotConfig         `json:"plot,omitempty"`
	Stats             *skills.StatsConfig        `json:"stats,omitempty"`
	Solver            *skills.SolverConfig       `json:"solver,omitempty"`
	Latex             *skills.LatexConfig        `json:"latex,omitempty"`
	UnitConvert       *skills.UnitConvertConfig  `json:"unit_convert,omitempty"`
	NumberTheory      *skills.NumberTheoryConfig `json:"number_theory,omitempty"`
	Matrix            *skills.MatrixConfig       `json:"matrix,omitempty"`
	StatusType        string                     `json:"status_type,omitempty"`
	ActivityType      string                     `json:"activity_type,omitempty"`
	ActivityText      string                     `json:"activity_text,omitempty"`
	SpeakContent      string                     `json:"speak_content,omitempty"`
	VoiceChannelID    string                     `json:"-"`
	GraphMemories     *GraphMemoriesConfig       `json:"graph_memories,omitempty"`
	CanvasOp          *skills.CanvasOperation    `json:"canvas,omitempty"`
}

type ActionData struct {
	Action            string                     `json:"action"`
	TargetUser        string                     `json:"target_user"`
	Reason            string                     `json:"reason,omitempty"`
	Role              string                     `json:"role,omitempty"`
	DMContent         string                     `json:"dm_content,omitempty"`
	ResponseMsg       string                     `json:"response"`
	ResponseType      string                     `json:"response_type"`
	EmbedTitle        string                     `json:"embed_title,omitempty"`
	EmbedDescription  string                     `json:"embed_description,omitempty"`
	EmbedThumbnailUrl string                     `json:"embed_thumbnail_url,omitempty"`
	EmbedImageUrl     string                     `json:"embed_image_url,omitempty"`
	UseEmbed          bool                       `json:"use_embed,omitempty"`
	Tasks             []Action                   `json:"tasks,omitempty"`
	Memories          []MemoryItem               `json:"memories,omitempty"`
	Chart             *skills.ChartConfig        `json:"chart,omitempty"`
	GenerateChart     *skills.ChartConfig        `json:"generate_chart,omitempty"`
	Drawing           *skills.DrawingConfig      `json:"drawing,omitempty"`
	GenerateDrawing   *skills.DrawingConfig      `json:"generate_drawing,omitempty"`
	PixelArt          *skills.PixelArtConfig     `json:"pixel_art,omitempty"`
	GeneratePixelArt  *skills.PixelArtConfig     `json:"generate_pixel_art,omitempty"`
	Benchmark         *skills.BenchmarkConfig    `json:"benchmark,omitempty"`
	RunBenchmark      *skills.BenchmarkConfig    `json:"run_benchmark,omitempty"`
	Plot              *skills.PlotConfig         `json:"plot,omitempty"`
	PlotFunction      *skills.PlotConfig         `json:"plot_function,omitempty"`
	Stats             *skills.StatsConfig        `json:"stats,omitempty"`
	CalculateStats    *skills.StatsConfig        `json:"calculate_stats,omitempty"`
	Solver            *skills.SolverConfig       `json:"solver,omitempty"`
	SolveEquation     *skills.SolverConfig       `json:"solve_equation,omitempty"`
	Latex             *skills.LatexConfig        `json:"latex,omitempty"`
	RenderLatex       *skills.LatexConfig        `json:"render_latex,omitempty"`
	UnitConvert       *skills.UnitConvertConfig  `json:"unit_convert,omitempty"`
	ConvertUnit       *skills.UnitConvertConfig  `json:"convert_unit,omitempty"`
	NumberTheory      *skills.NumberTheoryConfig `json:"number_theory,omitempty"`
	Matrix            *skills.MatrixConfig       `json:"matrix,omitempty"`
	MatrixOperation   *skills.MatrixConfig       `json:"matrix_operation,omitempty"`
	StatusType        string                     `json:"status_type,omitempty"`
	ActivityType      string                     `json:"activity_type,omitempty"`
	ActivityText      string                     `json:"activity_text,omitempty"`
	SpeakContent      string                     `json:"speak_content,omitempty"`
	VoiceChannelID    string                     `json:"-"`
	MemoryEdits       []MemoryEdit               `json:"memory_edits,omitempty"`
	GraphMemories     *GraphMemoriesConfig       `json:"graph_memories,omitempty"`
	SetUserTier       *SetUserTierConfig         `json:"set_user_tier,omitempty"`
	GetUserTier       *GetUserTierConfig         `json:"get_user_tier,omitempty"`
	SetStatus         *SetStatusConfig           `json:"set_status,omitempty"`
	CanvasOp          *skills.CanvasOperation    `json:"canvas,omitempty"`
	CanvasGenerate    *skills.CanvasOperation    `json:"generate_canvas,omitempty"`
	Reminder          *ReminderConfig            `json:"reminder,omitempty"`
	Poll              *PollConfig                `json:"poll,omitempty"`
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

type SetStatusConfig struct {
	StatusType   string `json:"status_type"`
	ActivityType string `json:"activity_type"`
	ActivityText string `json:"activity_text"`
}

type ReminderConfig struct {
	Message      string `json:"message"`
	DelayMinutes int    `json:"delay_minutes"`
}

type PollConfig struct {
	Question        string   `json:"question"`
	Options         []string `json:"options"`
	DurationMinutes int      `json:"duration_minutes,omitempty"`
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

	if data.Chart == nil && data.GenerateChart != nil {
		data.Chart = data.GenerateChart
	}
	if data.Drawing == nil && data.GenerateDrawing != nil {
		data.Drawing = data.GenerateDrawing
	}
	if data.PixelArt == nil && data.GeneratePixelArt != nil {
		data.PixelArt = data.GeneratePixelArt
	}
	if data.Benchmark == nil && data.RunBenchmark != nil {
		data.Benchmark = data.RunBenchmark
	}
	if data.Plot == nil && data.PlotFunction != nil {
		data.Plot = data.PlotFunction
	}
	if data.Stats == nil && data.CalculateStats != nil {
		data.Stats = data.CalculateStats
	}
	if data.Solver == nil && data.SolveEquation != nil {
		data.Solver = data.SolveEquation
	}
	if data.Latex == nil && data.RenderLatex != nil {
		data.Latex = data.RenderLatex
	}
	if data.UnitConvert == nil && data.ConvertUnit != nil {
		data.UnitConvert = data.ConvertUnit
	}
	if data.Matrix == nil && data.MatrixOperation != nil {
		data.Matrix = data.MatrixOperation
	}
	if data.CanvasOp == nil && data.CanvasGenerate != nil {
		data.CanvasOp = data.CanvasGenerate
	}

	if data.SetStatus != nil {
		if data.StatusType == "" {
			data.StatusType = data.SetStatus.StatusType
		}
		if data.ActivityType == "" {
			data.ActivityType = data.SetStatus.ActivityType
		}
		if data.ActivityText == "" {
			data.ActivityText = data.SetStatus.ActivityText
		}
	}

	for i, task := range data.Tasks {
		if task.Action == "" && task.Type != "" {
			data.Tasks[i].Action = task.Type
		}
		if (data.Tasks[i].Action == "generate_chart") && task.Chart == nil {
			data.Tasks[i].Chart = extractFlatChart(jsonStr, i)
		}
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
					i = j
					break
				}
			}
		}
	}
	return last, found
}

func extractFlatChart(rawJSON string, taskIndex int) *skills.ChartConfig {
	var envelope struct {
		Tasks []json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(rawJSON), &envelope); err != nil || taskIndex >= len(envelope.Tasks) {
		return nil
	}

	type taskDatasets struct {
		Name   string    `json:"name"`
		Label  string    `json:"label"`
		Values []float64 `json:"values"`
		Data   []float64 `json:"data"`
		Color  string    `json:"color"`
	}
	type taskChartShape struct {
		ChartType string          `json:"chart_type"`
		Type      string          `json:"type"`
		Labels    []string        `json:"labels"`
		XLabels   []string        `json:"x_labels"`
		Data      json.RawMessage `json:"data"`
		Title     string          `json:"title"`
		Theme     string          `json:"theme"`
		Datasets  []taskDatasets  `json:"datasets"`
	}
	var flat taskChartShape
	if err := json.Unmarshal(envelope.Tasks[taskIndex], &flat); err != nil {
		return nil
	}

	var flatData []float64
	if len(flat.Data) > 0 {
		if err := json.Unmarshal(flat.Data, &flatData); err != nil {
			var nested taskChartShape
			if err := json.Unmarshal(flat.Data, &nested); err == nil {
				if flat.ChartType == "" {
					flat.ChartType = nested.ChartType
				}
				if flat.Type == "" {
					flat.Type = nested.Type
				}
				if flat.Title == "" {
					flat.Title = nested.Title
				}
				if flat.Theme == "" {
					flat.Theme = nested.Theme
				}
				if len(flat.Labels) == 0 {
					flat.Labels = nested.Labels
				}
				if len(flat.XLabels) == 0 {
					flat.XLabels = nested.XLabels
				}
				if len(flat.Datasets) == 0 {
					flat.Datasets = nested.Datasets
				}

				if len(nested.Data) > 0 {
					json.Unmarshal(nested.Data, &flatData)
				}
			}
		}
	}

	chartType := flat.ChartType
	if chartType == "" && flat.Type != "generate_chart" {
		chartType = flat.Type
	}
	if chartType == "" {
		chartType = "bar"
	}
	labels := flat.XLabels
	if len(labels) == 0 {
		labels = flat.Labels
	}
	theme := flat.Theme
	if theme == "" {
		theme = "dark"
	}

	var datasets []skills.ChartDataset
	if len(flat.Datasets) > 0 {
		for _, d := range flat.Datasets {
			name := d.Name
			if name == "" {
				name = d.Label
			}
			values := d.Values
			if len(values) == 0 {
				values = d.Data
			}
			datasets = append(datasets, skills.ChartDataset{Name: name, Values: values, Color: d.Color})
		}
	} else if len(flatData) > 0 {
		datasets = []skills.ChartDataset{{Name: flat.Title, Values: flatData}}
	}

	if len(datasets) == 0 {
		return nil
	}

	return &skills.ChartConfig{
		Type:     chartType,
		Title:    flat.Title,
		XLabels:  labels,
		Datasets: datasets,
		Theme:    theme,
	}
}

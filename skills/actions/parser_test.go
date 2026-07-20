package actions

import (
	"testing"

	"github.com/jayydoesdev/airo/bot/skills"
)

func TestParseTaskTypeField(t *testing.T) {
	raw := `{
		"action": "generate_chart",
		"response": "here",
		"response_type": "text",
		"tasks": [
			{
				"type": "generate_chart",
				"data": {
					"labels": ["mon", "tue", "wed"],
					"datasets": [{"label": "vals", "data": [1, 2, 3]}],
					"type": "bar",
					"title": "test",
					"theme": "dark"
				}
			}
		],
		"memories": []
	}`

	_, data, err := ParseAIResponse(raw)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(data.Tasks) == 0 {
		t.Fatal("no tasks parsed")
	}
	task := data.Tasks[0]
	if task.Action != "generate_chart" {
		t.Errorf("expected action=generate_chart, got %q", task.Action)
	}
	if task.Chart == nil {
		t.Fatal("chart is nil — extractFlatChart failed")
	}
	if task.Chart.Type != "bar" {
		t.Errorf("expected chart type=bar, got %q", task.Chart.Type)
	}
	if len(task.Chart.Datasets) == 0 {
		t.Fatal("chart has no datasets")
	}
	if task.Chart.Datasets[0].Name != "vals" {
		t.Errorf("expected dataset name=vals, got %q", task.Chart.Datasets[0].Name)
	}
	if len(task.Chart.Datasets[0].Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(task.Chart.Datasets[0].Values))
	}
}

func TestParseChartJSTopLevel(t *testing.T) {
	raw := `{
		"action": "generate_chart",
		"response": "here",
		"response_type": "text",
		"chart": {
			"type": "bar",
			"title": "Top Level Chart.js",
			"data": {
				"labels": ["A", "B", "C"],
				"datasets": [{"label": "series", "data": [10, 20, 30]}]
			},
			"theme": "dark"
		},
		"tasks": [],
		"memories": []
	}`

	_, data, err := ParseAIResponse(raw)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if data.Chart == nil {
		t.Fatal("chart is nil")
	}

	png, err := skills.RenderChart(*data.Chart)
	if err != nil {
		t.Fatalf("RenderChart error: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("RenderChart returned empty PNG")
	}
}

func TestParseGenerateChartKey(t *testing.T) {
	raw := `{
		"action": "generate_chart",
		"response": "here",
		"response_type": "text",
		"generate_chart": {
			"type": "horizontal_bar",
			"title": "Leaderboard",
			"labels": ["Frill", "Kuro", "Apple"],
			"values": [100, 85, 70],
			"colors": ["#ff6b6b", "#ffd93d", "#6bcb77"],
			"theme": "dark"
		},
		"tasks": [],
		"memories": []
	}`

	_, data, err := ParseAIResponse(raw)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if data.Chart == nil {
		t.Fatal("chart is nil — GenerateChart not backfilled to Chart")
	}
	png, err := skills.RenderChart(*data.Chart)
	if err != nil {
		t.Fatalf("RenderChart error: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("RenderChart returned empty PNG")
	}
}

func TestParseSetStatusNested(t *testing.T) {
	raw := `{
		"action": "set_status",
		"response": "done",
		"response_type": "text",
		"set_status": {
			"status_type": "dnd",
			"activity_type": "playing",
			"activity_text": "test game"
		},
		"tasks": [],
		"memories": []
	}`

	_, data, err := ParseAIResponse(raw)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if data.StatusType != "dnd" {
		t.Errorf("expected status_type=dnd, got %q", data.StatusType)
	}
	if data.ActivityType != "playing" {
		t.Errorf("expected activity_type=playing, got %q", data.ActivityType)
	}
	if data.ActivityText != "test game" {
		t.Errorf("expected activity_text='test game', got %q", data.ActivityText)
	}
}

package skills

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	charts "github.com/vicanso/go-charts/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

type ChartDataset struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
	Color  string    `json:"color,omitempty"`
}

type ChartConfig struct {
	Type     string          `json:"type"`
	Title    string          `json:"title,omitempty"`
	XLabels  []string        `json:"x_labels,omitempty"`
	Labels   []string        `json:"labels,omitempty"`
	Datasets []ChartDataset  `json:"datasets"`
	Data     json.RawMessage `json:"data,omitempty"`
	Values   []float64       `json:"values,omitempty"`
	Colors   []string        `json:"colors,omitempty"`
	Width    int             `json:"width,omitempty"`
	Height   int             `json:"height,omitempty"`
	Theme    string          `json:"theme,omitempty"`
}

var customThemeCounter atomic.Uint64

func RenderChart(cfg ChartConfig) ([]byte, error) {
	if len(cfg.XLabels) == 0 && len(cfg.Labels) > 0 {
		cfg.XLabels = cfg.Labels
	}

	if len(cfg.Datasets) == 0 && len(cfg.Values) > 0 {
		if len(cfg.Colors) >= len(cfg.Values) {

			for i, v := range cfg.Values {
				label := fmt.Sprintf("Item %d", i+1)
				if i < len(cfg.XLabels) {
					label = cfg.XLabels[i]
				}
				cfg.Datasets = append(cfg.Datasets, ChartDataset{Name: label, Values: []float64{v}, Color: cfg.Colors[i]})
			}
			cfg.XLabels = nil
		} else {
			cfg.Datasets = []ChartDataset{{Name: cfg.Title, Values: cfg.Values}}
		}
	}
	if len(cfg.Data) > 0 {
		var flatVals []float64
		if err := json.Unmarshal(cfg.Data, &flatVals); err == nil {
			if len(cfg.Datasets) == 0 && len(flatVals) > 0 {
				cfg.Datasets = []ChartDataset{{Name: cfg.Title, Values: flatVals}}
			}
		} else {
			var cjs struct {
				Labels   []string `json:"labels"`
				Datasets []struct {
					Label string    `json:"label"`
					Data  []float64 `json:"data"`
					Color string    `json:"color"`
				} `json:"datasets"`
			}
			if err := json.Unmarshal(cfg.Data, &cjs); err == nil {
				if len(cfg.XLabels) == 0 {
					cfg.XLabels = cjs.Labels
				}
				if len(cfg.Datasets) == 0 {
					for _, ds := range cjs.Datasets {
						cfg.Datasets = append(cfg.Datasets, ChartDataset{Name: ds.Label, Values: ds.Data, Color: ds.Color})
					}
				}
			}
		}
	}
	maxVals := 0
	for _, ds := range cfg.Datasets {
		if len(ds.Values) > maxVals {
			maxVals = len(ds.Values)
		}
	}
	if len(cfg.XLabels) < maxVals {
		for i := len(cfg.XLabels); i < maxVals; i++ {
			cfg.XLabels = append(cfg.XLabels, fmt.Sprintf("Item %d", i+1))
		}
	}

	width := 1400
	height := 700
	if cfg.Width > 0 {
		width = cfg.Width
	}
	if cfg.Height > 0 {
		height = cfg.Height
	}

	baseTheme := charts.ThemeDark
	switch cfg.Theme {
	case "light":
		baseTheme = charts.ThemeLight
	case "grafana":
		baseTheme = charts.ThemeGrafana
	case "ant":
		baseTheme = charts.ThemeAnt
	}

	isLight := cfg.Theme == "light" || cfg.Theme == "ant"
	bgColor := drawing.Color{R: 1, G: 1, B: 1, A: 0}
	if isLight {
		bgColor = drawing.Color{R: 255, G: 255, B: 255, A: 255}
	}

	theme := baseTheme
	if colors := datasetColors(cfg.Datasets); len(colors) > 0 {
		id := customThemeCounter.Add(1)
		name := fmt.Sprintf("airo_custom_%d", id)
		base := charts.NewTheme(baseTheme)
		charts.AddTheme(name, charts.ThemeOption{
			IsDarkMode:         base.IsDark(),
			AxisStrokeColor:    base.GetAxisStrokeColor(),
			AxisSplitLineColor: base.GetAxisSplitLineColor(),
			BackgroundColor:    bgColor,
			TextColor:          base.GetTextColor(),
			SeriesColors:       colors,
		})
		theme = name
	}

	opts := []charts.OptionFunc{
		charts.PNGTypeOption(),
		charts.ThemeOptionFunc(theme),
		charts.WidthOptionFunc(width),
		charts.HeightOptionFunc(height),
		charts.BackgroundColorOptionFunc(bgColor),
	}

	if cfg.Title != "" {
		opts = append(opts, charts.TitleTextOptionFunc(cfg.Title))
	}

	var legendLabels []string
	for _, ds := range cfg.Datasets {
		if ds.Name != "" {
			legendLabels = append(legendLabels, ds.Name)
		}
	}
	if len(legendLabels) > 0 {
		opts = append(opts, charts.LegendOptionFunc(charts.LegendOption{
			Data: legendLabels,
			Left: charts.PositionCenter,
			Top:  "35",
		}))
	}

	values := valuesFromDatasets(cfg.Datasets)

	switch cfg.Type {
	case "bar":
		if len(cfg.XLabels) > 0 {
			opts = append(opts, charts.XAxisDataOptionFunc(cfg.XLabels))
		}
		return render(charts.BarRender(values, opts...))
	case "horizontal_bar":
		if len(cfg.XLabels) > 0 {
			opts = append(opts, charts.YAxisDataOptionFunc(cfg.XLabels))
		}
		return render(charts.HorizontalBarRender(values, opts...))
	case "radar":
		var names []string
		var maxVals []float64
		for _, label := range cfg.XLabels {
			names = append(names, label)
			maxVals = append(maxVals, 0)
		}
		opts = append(opts, charts.RadarIndicatorOptionFunc(names, maxVals))
		return render(charts.RadarRender(values, opts...))
	case "pie":
		if len(cfg.Datasets) == 0 || len(cfg.Datasets[0].Values) == 0 {
			return nil, fmt.Errorf("pie chart requires at least one dataset with values")
		}
		if len(cfg.XLabels) > 0 {
			opts = append(opts, charts.XAxisDataOptionFunc(cfg.XLabels))
		}
		return render(charts.PieRender(cfg.Datasets[0].Values, opts...))
	default:
		if len(cfg.XLabels) > 0 {
			opts = append(opts, charts.XAxisDataOptionFunc(cfg.XLabels))
		}
		return render(charts.LineRender(values, opts...))
	}
}

func datasetColors(datasets []ChartDataset) []drawing.Color {
	var hasAny bool
	for _, ds := range datasets {
		if ds.Color != "" {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return nil
	}

	colors := make([]drawing.Color, len(datasets))
	for i, ds := range datasets {
		if c, err := parseHexColor(ds.Color); err == nil {
			colors[i] = c
		} else {
			colors[i] = drawing.Color{R: 1, G: 1, B: 1, A: 255}
		}
	}
	return colors
}

func parseHexColor(s string) (drawing.Color, error) {
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	case 6:
	default:
		return drawing.Color{}, fmt.Errorf("invalid hex color: %s", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return drawing.Color{}, err
	}
	return drawing.Color{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
		A: 255,
	}, nil
}

func valuesFromDatasets(datasets []ChartDataset) [][]float64 {
	result := make([][]float64, len(datasets))
	for i, ds := range datasets {
		result[i] = ds.Values
	}
	return result
}

func render(p *charts.Painter, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	return p.Bytes()
}

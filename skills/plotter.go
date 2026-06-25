package skills

import (
	"fmt"

	"github.com/Knetic/govaluate"
)

type PlotConfig struct {
	Expressions []BenchmarkExpr `json:"expressions"`
	Variable    string          `json:"variable"`
	RangeStart  float64         `json:"range_start"`
	RangeEnd    float64         `json:"range_end"`
	Steps       int             `json:"steps"`
	Title       string          `json:"title,omitempty"`
	Theme       string          `json:"theme,omitempty"`
}

func PlotToChart(cfg PlotConfig) (ChartConfig, error) {
	if len(cfg.Expressions) == 0 {
		return ChartConfig{}, fmt.Errorf("no expressions provided")
	}

	variable := cfg.Variable
	if variable == "" {
		variable = "x"
	}
	steps := cfg.Steps
	if steps <= 0 || steps > 500 {
		steps = 100
	}
	rangeEnd := cfg.RangeEnd
	if rangeEnd <= cfg.RangeStart {
		rangeEnd = cfg.RangeStart + float64(steps)
	}

	stepSize := (rangeEnd - cfg.RangeStart) / float64(steps-1)
	xLabels := make([]string, steps)
	xVals := make([]float64, steps)
	for i := range xVals {
		xVals[i] = cfg.RangeStart + float64(i)*stepSize
		xLabels[i] = fmt.Sprintf("%.4g", xVals[i])
	}

	var datasets []ChartDataset
	for _, expr := range cfg.Expressions {
		if err := validateExpr(expr.Expr); err != nil {
			return ChartConfig{}, fmt.Errorf("expression %q: %w", expr.Label, err)
		}
		compiled, err := govaluate.NewEvaluableExpressionWithFunctions(expr.Expr, allowedFunctions)
		if err != nil {
			return ChartConfig{}, fmt.Errorf("parse %q: %w", expr.Label, err)
		}
		params := map[string]interface{}{variable: 0.0}
		values := make([]float64, steps)
		for i, x := range xVals {
			params[variable] = x
			result, err := compiled.Evaluate(params)
			if err != nil {
				values[i] = 0
				continue
			}
			values[i] = toFloat(result)
		}
		datasets = append(datasets, ChartDataset{
			Name:   expr.Label,
			Values: values,
		})
	}

	title := cfg.Title
	if title == "" {
		title = "f(" + variable + ")"
	}
	theme := cfg.Theme
	if theme == "" {
		theme = "dark"
	}

	return ChartConfig{
		Type:     "line",
		Title:    title,
		XLabels:  xLabels,
		Datasets: datasets,
		Width:    1400,
		Height:   700,
		Theme:    theme,
	}, nil
}

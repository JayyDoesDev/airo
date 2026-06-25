package skills

import (
	"fmt"
	"math"

	"github.com/Knetic/govaluate"
)

type SolverConfig struct {
	Equation   string  `json:"equation"`
	Variable   string  `json:"variable"`
	RangeStart float64 `json:"range_start"`
	RangeEnd   float64 `json:"range_end"`
	Theme      string  `json:"theme,omitempty"`
}

type SolverResult struct {
	Roots []float64
	Chart ChartConfig
}

func SolveEquation(cfg SolverConfig) (SolverResult, error) {
	if err := validateExpr(cfg.Equation); err != nil {
		return SolverResult{}, fmt.Errorf("invalid equation: %w", err)
	}

	variable := cfg.Variable
	if variable == "" {
		variable = "x"
	}
	rangeEnd := cfg.RangeEnd
	if rangeEnd <= cfg.RangeStart {
		rangeEnd = cfg.RangeStart + 20
	}

	compiled, err := govaluate.NewEvaluableExpressionWithFunctions(cfg.Equation, allowedFunctions)
	if err != nil {
		return SolverResult{}, fmt.Errorf("parse equation: %w", err)
	}

	eval := func(x float64) float64 {
		params := map[string]interface{}{variable: x}
		result, err := compiled.Evaluate(params)
		if err != nil {
			return math.NaN()
		}
		return toFloat(result)
	}

	roots := findRoots(eval, cfg.RangeStart, rangeEnd, 1000)

	steps := 200
	stepSize := (rangeEnd - cfg.RangeStart) / float64(steps-1)
	xLabels := make([]string, steps)
	yValues := make([]float64, steps)
	for i := range xLabels {
		x := cfg.RangeStart + float64(i)*stepSize
		xLabels[i] = fmt.Sprintf("%.4g", x)
		y := eval(x)
		if math.IsNaN(y) || math.IsInf(y, 0) {
			y = 0
		}
		yValues[i] = y
	}

	datasets := []ChartDataset{
		{Name: "f(" + variable + ") = " + cfg.Equation, Values: yValues},
	}

	if len(roots) > 0 {
		zeroLine := make([]float64, steps)
		datasets = append(datasets, ChartDataset{Name: "y = 0", Values: zeroLine})
	}

	theme := cfg.Theme
	if theme == "" {
		theme = "dark"
	}

	chart := ChartConfig{
		Type:     "line",
		Title:    "f(" + variable + ") = " + cfg.Equation,
		XLabels:  xLabels,
		Datasets: datasets,
		Width:    1400,
		Height:   700,
		Theme:    theme,
	}

	return SolverResult{Roots: roots, Chart: chart}, nil
}

func SolverResultToText(r SolverResult, eq, variable string) string {
	if len(r.Roots) == 0 {
		return fmt.Sprintf("no real roots found for `%s = 0` in the given range", eq)
	}
	text := fmt.Sprintf("roots of `%s = 0`:\n", eq)
	for _, root := range r.Roots {
		text += fmt.Sprintf("  %s ≈ `%.6g`\n", variable, root)
	}
	return text
}

func findRoots(f func(float64) float64, start, end float64, segments int) []float64 {
	var roots []float64
	step := (end - start) / float64(segments)

	for i := 0; i < segments; i++ {
		a := start + float64(i)*step
		b := a + step
		fa := f(a)
		fb := f(b)

		if math.IsNaN(fa) || math.IsNaN(fb) {
			continue
		}
		if fa*fb > 0 {
			continue
		}

		root, ok := bisect(f, a, b, 1e-9, 100)
		if !ok {
			continue
		}

		duplicate := false
		for _, r := range roots {
			if math.Abs(r-root) < 1e-6 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			roots = append(roots, root)
		}
	}

	return roots
}

func bisect(f func(float64) float64, a, b, tol float64, maxIter int) (float64, bool) {
	fa := f(a)
	for i := 0; i < maxIter; i++ {
		mid := (a + b) / 2
		if (b-a)/2 < tol {
			return mid, true
		}
		fm := f(mid)
		if math.IsNaN(fm) {
			return 0, false
		}
		if fm == 0 {
			return mid, true
		}
		if fa*fm < 0 {
			b = mid
		} else {
			a = mid
			fa = fm
		}
	}
	return (a + b) / 2, true
}

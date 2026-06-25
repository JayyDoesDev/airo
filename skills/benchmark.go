package skills

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

type BenchmarkConfig struct {
	Expressions []BenchmarkExpr `json:"expressions"`
	Variable    string          `json:"variable"`
	RangeStart  float64         `json:"range_start"`
	RangeEnd    float64         `json:"range_end"`
	Steps       int             `json:"steps"`
	Iterations  int             `json:"iterations"`
}

type BenchmarkExpr struct {
	Label string `json:"label"`
	Expr  string `json:"expr"`
}

type BenchmarkResult struct {
	Label   string
	XValues []float64
	NsPerOp []float64
}

var allowedFunctions = map[string]govaluate.ExpressionFunction{
	"sqrt":  func(args ...interface{}) (interface{}, error) { return math.Sqrt(toFloat(args[0])), nil },
	"abs":   func(args ...interface{}) (interface{}, error) { return math.Abs(toFloat(args[0])), nil },
	"sin":   func(args ...interface{}) (interface{}, error) { return math.Sin(toFloat(args[0])), nil },
	"cos":   func(args ...interface{}) (interface{}, error) { return math.Cos(toFloat(args[0])), nil },
	"tan":   func(args ...interface{}) (interface{}, error) { return math.Tan(toFloat(args[0])), nil },
	"log":   func(args ...interface{}) (interface{}, error) { return math.Log(toFloat(args[0])), nil },
	"log2":  func(args ...interface{}) (interface{}, error) { return math.Log2(toFloat(args[0])), nil },
	"log10": func(args ...interface{}) (interface{}, error) { return math.Log10(toFloat(args[0])), nil },
	"ceil":  func(args ...interface{}) (interface{}, error) { return math.Ceil(toFloat(args[0])), nil },
	"floor": func(args ...interface{}) (interface{}, error) { return math.Floor(toFloat(args[0])), nil },
	"pow":   func(args ...interface{}) (interface{}, error) { return math.Pow(toFloat(args[0]), toFloat(args[1])), nil },
	"round": func(args ...interface{}) (interface{}, error) { return math.Round(toFloat(args[0])), nil },
	"min":   func(args ...interface{}) (interface{}, error) { return math.Min(toFloat(args[0]), toFloat(args[1])), nil },
	"max":   func(args ...interface{}) (interface{}, error) { return math.Max(toFloat(args[0]), toFloat(args[1])), nil },
}

var unsafePattern = regexp.MustCompile(`(?i)(import|exec|os\.|syscall|unsafe|http|file|open|read|write|eval|func|go |chan |select|for |while)`)

func RunBenchmark(cfg BenchmarkConfig) ([]BenchmarkResult, error) {
	if len(cfg.Expressions) == 0 {
		return nil, fmt.Errorf("no expressions provided")
	}
	if len(cfg.Expressions) > 8 {
		return nil, fmt.Errorf("max 8 expressions per benchmark")
	}

	variable := cfg.Variable
	if variable == "" {
		variable = "x"
	}

	steps := cfg.Steps
	if steps <= 0 || steps > 100 {
		steps = 20
	}

	iterations := cfg.Iterations
	if iterations <= 0 {
		iterations = 10000
	}
	if iterations > 100000 {
		iterations = 100000
	}

	rangeStart := cfg.RangeStart
	rangeEnd := cfg.RangeEnd
	if rangeEnd <= rangeStart {
		rangeEnd = rangeStart + float64(steps)
	}

	stepSize := (rangeEnd - rangeStart) / float64(steps-1)
	if steps == 1 {
		stepSize = 0
	}

	xValues := make([]float64, steps)
	for i := range xValues {
		xValues[i] = rangeStart + float64(i)*stepSize
	}

	var results []BenchmarkResult

	for _, expr := range cfg.Expressions {
		if err := validateExpr(expr.Expr); err != nil {
			return nil, fmt.Errorf("expression %q: %w", expr.Label, err)
		}

		compiled, err := govaluate.NewEvaluableExpressionWithFunctions(expr.Expr, allowedFunctions)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", expr.Label, err)
		}

		nsPerOp := make([]float64, steps)
		params := make(map[string]interface{}, 1)

		for i, xVal := range xValues {
			params[variable] = xVal

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			ns, err := timeExpr(ctx, compiled, params, iterations)
			cancel()
			if err != nil {
				return nil, fmt.Errorf("benchmark %q at x=%v: %w", expr.Label, xVal, err)
			}
			nsPerOp[i] = ns
		}

		results = append(results, BenchmarkResult{
			Label:   expr.Label,
			XValues: xValues,
			NsPerOp: nsPerOp,
		})
	}

	return results, nil
}

func BenchmarkToChart(results []BenchmarkResult, variable string) ChartConfig {
	xLabels := make([]string, len(results[0].XValues))
	for i, v := range results[0].XValues {
		xLabels[i] = fmt.Sprintf("%.4g", v)
	}

	datasets := make([]ChartDataset, len(results))
	for i, r := range results {
		values := make([]float64, len(r.NsPerOp))
		copy(values, r.NsPerOp)
		datasets[i] = ChartDataset{
			Name:   r.Label,
			Values: values,
		}
	}

	title := "Benchmark: ns/op vs " + variable
	return ChartConfig{
		Type:     "line",
		Title:    title,
		XLabels:  xLabels,
		Datasets: datasets,
		Width:    1400,
		Height:   700,
		Theme:    "dark",
	}
}

func timeExpr(ctx context.Context, expr *govaluate.EvaluableExpression, params map[string]interface{}, iterations int) (float64, error) {
	type result struct {
		ns  float64
		err error
	}
	ch := make(chan result, 1)

	go func() {
		start := time.Now()
		for i := 0; i < iterations; i++ {
			if _, err := expr.Evaluate(params); err != nil {
				ch <- result{0, err}
				return
			}
		}
		elapsed := time.Since(start)
		ch <- result{float64(elapsed.Nanoseconds()) / float64(iterations), nil}
	}()

	select {
	case r := <-ch:
		return r.ns, r.err
	case <-ctx.Done():
		return 0, fmt.Errorf("timed out after 500ms")
	}
}

func validateExpr(expr string) error {
	if unsafePattern.MatchString(expr) {
		return fmt.Errorf("expression contains disallowed keywords")
	}
	allowed := regexp.MustCompile(`^[0-9a-zA-Z\s\+\-\*\/\^\(\)\.,_]+$`)
	if !allowed.MatchString(expr) {
		return fmt.Errorf("expression contains disallowed characters")
	}
	knownFuncs := []string{"sqrt", "abs", "sin", "cos", "tan", "log", "log2", "log10", "ceil", "floor", "pow", "round", "min", "max"}
	words := regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`).FindAllString(expr, -1)
	for _, w := range words {
		found := false
		for _, fn := range knownFuncs {
			if strings.EqualFold(w, fn) {
				found = true
				break
			}
		}
		if !found {
			// assume it's a variable — fine
		}
	}
	return nil
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

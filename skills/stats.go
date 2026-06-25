package skills

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type StatsConfig struct {
	Data   []float64 `json:"data"`
	Label  string    `json:"label,omitempty"`
	Theme  string    `json:"theme,omitempty"`
	Bins   int       `json:"bins,omitempty"`
}

type StatsResult struct {
	Count    int
	Mean     float64
	Median   float64
	Mode     []float64
	StdDev   float64
	Variance float64
	Min      float64
	Max      float64
	Range    float64
	P25      float64
	P75      float64
	P95      float64
	IQR      float64
}

func CalculateStats(cfg StatsConfig) (StatsResult, ChartConfig, error) {
	data := cfg.Data
	if len(data) == 0 {
		return StatsResult{}, ChartConfig{}, fmt.Errorf("no data provided")
	}
	if len(data) > 10000 {
		return StatsResult{}, ChartConfig{}, fmt.Errorf("max 10000 data points")
	}

	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	n := len(sorted)
	result := StatsResult{Count: n}
	result.Min = sorted[0]
	result.Max = sorted[n-1]
	result.Range = result.Max - result.Min

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	result.Mean = sum / float64(n)

	result.Median = percentile(sorted, 50)
	result.P25 = percentile(sorted, 25)
	result.P75 = percentile(sorted, 75)
	result.P95 = percentile(sorted, 95)
	result.IQR = result.P75 - result.P25

	var sqDiffSum float64
	for _, v := range sorted {
		d := v - result.Mean
		sqDiffSum += d * d
	}
	result.Variance = sqDiffSum / float64(n)
	result.StdDev = math.Sqrt(result.Variance)

	result.Mode = mode(sorted)

	chart := buildHistogram(sorted, cfg)
	return result, chart, nil
}

func StatsResultToText(r StatsResult, label string) string {
	if label == "" {
		label = "data"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Stats for %s** (n=%d)\n", label, r.Count))
	sb.WriteString(fmt.Sprintf("Mean: `%.4g` | Median: `%.4g` | Std Dev: `%.4g`\n", r.Mean, r.Median, r.StdDev))
	sb.WriteString(fmt.Sprintf("Min: `%.4g` | Max: `%.4g` | Range: `%.4g`\n", r.Min, r.Max, r.Range))
	sb.WriteString(fmt.Sprintf("P25: `%.4g` | P75: `%.4g` | P95: `%.4g` | IQR: `%.4g`\n", r.P25, r.P75, r.P95, r.IQR))
	sb.WriteString(fmt.Sprintf("Variance: `%.4g`\n", r.Variance))
	if len(r.Mode) > 0 && len(r.Mode) <= 5 {
		modeStrs := make([]string, len(r.Mode))
		for i, v := range r.Mode {
			modeStrs[i] = fmt.Sprintf("%.4g", v)
		}
		sb.WriteString(fmt.Sprintf("Mode: `%s`\n", strings.Join(modeStrs, ", ")))
	}
	return sb.String()
}

func buildHistogram(sorted []float64, cfg StatsConfig) ChartConfig {
	bins := cfg.Bins
	if bins <= 0 || bins > 50 {
		bins = int(math.Ceil(math.Sqrt(float64(len(sorted)))))
		if bins < 5 {
			bins = 5
		}
		if bins > 30 {
			bins = 30
		}
	}

	min := sorted[0]
	max := sorted[len(sorted)-1]
	binWidth := (max - min) / float64(bins)
	if binWidth == 0 {
		binWidth = 1
	}

	counts := make([]float64, bins)
	labels := make([]string, bins)
	for i := range labels {
		lo := min + float64(i)*binWidth
		hi := lo + binWidth
		labels[i] = fmt.Sprintf("%.3g–%.3g", lo, hi)
	}
	for _, v := range sorted {
		idx := int((v - min) / binWidth)
		if idx >= bins {
			idx = bins - 1
		}
		counts[idx]++
	}

	label := cfg.Label
	if label == "" {
		label = "Frequency"
	}
	theme := cfg.Theme
	if theme == "" {
		theme = "dark"
	}

	return ChartConfig{
		Type:    "bar",
		Title:   "Distribution of " + label,
		XLabels: labels,
		Datasets: []ChartDataset{
			{Name: label, Values: counts},
		},
		Width:  1400,
		Height: 700,
		Theme:  theme,
	}
}

func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := p / 100 * float64(n-1)
	lo := int(idx)
	hi := lo + 1
	if hi >= n {
		return sorted[n-1]
	}
	frac := idx - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}

func mode(sorted []float64) []float64 {
	if len(sorted) == 0 {
		return nil
	}
	freq := map[float64]int{}
	for _, v := range sorted {
		freq[v]++
	}
	max := 0
	for _, c := range freq {
		if c > max {
			max = c
		}
	}
	if max == 1 {
		return nil
	}
	var modes []float64
	for v, c := range freq {
		if c == max {
			modes = append(modes, v)
		}
	}
	sort.Float64s(modes)
	return modes
}

package skills

import (
	"fmt"
	"math"
	"strings"
)

type UnitConvertConfig struct {
	Value    float64 `json:"value"`
	From     string  `json:"from"`
	To       string  `json:"to"`
	Category string  `json:"category,omitempty"`
}

type UnitConvertResult struct {
	Input    float64
	Output   float64
	From     string
	To       string
	Formula  string
}

type unitDef struct {
	toBase   func(float64) float64
	fromBase func(float64) float64
	category string
}

var unitTable = map[string]unitDef{
	// length (base: meter)
	"m":   {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "length"},
	"km":  {func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }, "length"},
	"cm":  {func(v float64) float64 { return v / 100 }, func(v float64) float64 { return v * 100 }, "length"},
	"mm":  {func(v float64) float64 { return v / 1000 }, func(v float64) float64 { return v * 1000 }, "length"},
	"mi":  {func(v float64) float64 { return v * 1609.344 }, func(v float64) float64 { return v / 1609.344 }, "length"},
	"ft":  {func(v float64) float64 { return v * 0.3048 }, func(v float64) float64 { return v / 0.3048 }, "length"},
	"in":  {func(v float64) float64 { return v * 0.0254 }, func(v float64) float64 { return v / 0.0254 }, "length"},
	"yd":  {func(v float64) float64 { return v * 0.9144 }, func(v float64) float64 { return v / 0.9144 }, "length"},
	"nm":  {func(v float64) float64 { return v * 1852 }, func(v float64) float64 { return v / 1852 }, "length"},
	// mass (base: kg)
	"kg":  {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "mass"},
	"g":   {func(v float64) float64 { return v / 1000 }, func(v float64) float64 { return v * 1000 }, "mass"},
	"mg":  {func(v float64) float64 { return v / 1e6 }, func(v float64) float64 { return v * 1e6 }, "mass"},
	"lb":  {func(v float64) float64 { return v * 0.453592 }, func(v float64) float64 { return v / 0.453592 }, "mass"},
	"oz":  {func(v float64) float64 { return v * 0.0283495 }, func(v float64) float64 { return v / 0.0283495 }, "mass"},
	"t":   {func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }, "mass"},
	"st":  {func(v float64) float64 { return v * 6.35029 }, func(v float64) float64 { return v / 6.35029 }, "mass"},
	// temperature (special: not linear for all)
	"c":   {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "temperature"},
	"f":   {func(v float64) float64 { return (v - 32) * 5 / 9 }, func(v float64) float64 { return v*9/5 + 32 }, "temperature"},
	"k":   {func(v float64) float64 { return v - 273.15 }, func(v float64) float64 { return v + 273.15 }, "temperature"},
	// speed (base: m/s)
	"m/s":  {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "speed"},
	"km/h": {func(v float64) float64 { return v / 3.6 }, func(v float64) float64 { return v * 3.6 }, "speed"},
	"mph":  {func(v float64) float64 { return v * 0.44704 }, func(v float64) float64 { return v / 0.44704 }, "speed"},
	"knot": {func(v float64) float64 { return v * 0.514444 }, func(v float64) float64 { return v / 0.514444 }, "speed"},
	"ft/s": {func(v float64) float64 { return v * 0.3048 }, func(v float64) float64 { return v / 0.3048 }, "speed"},
	// area (base: m²)
	"m2":   {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "area"},
	"km2":  {func(v float64) float64 { return v * 1e6 }, func(v float64) float64 { return v / 1e6 }, "area"},
	"cm2":  {func(v float64) float64 { return v / 1e4 }, func(v float64) float64 { return v * 1e4 }, "area"},
	"ft2":  {func(v float64) float64 { return v * 0.092903 }, func(v float64) float64 { return v / 0.092903 }, "area"},
	"acre": {func(v float64) float64 { return v * 4046.86 }, func(v float64) float64 { return v / 4046.86 }, "area"},
	"ha":   {func(v float64) float64 { return v * 10000 }, func(v float64) float64 { return v / 10000 }, "area"},
	// volume (base: litre)
	"l":    {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "volume"},
	"ml":   {func(v float64) float64 { return v / 1000 }, func(v float64) float64 { return v * 1000 }, "volume"},
	"m3":   {func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }, "volume"},
	"gal":  {func(v float64) float64 { return v * 3.78541 }, func(v float64) float64 { return v / 3.78541 }, "volume"},
	"qt":   {func(v float64) float64 { return v * 0.946353 }, func(v float64) float64 { return v / 0.946353 }, "volume"},
	"pt":   {func(v float64) float64 { return v * 0.473176 }, func(v float64) float64 { return v / 0.473176 }, "volume"},
	"cup":  {func(v float64) float64 { return v * 0.236588 }, func(v float64) float64 { return v / 0.236588 }, "volume"},
	"floz": {func(v float64) float64 { return v * 0.0295735 }, func(v float64) float64 { return v / 0.0295735 }, "volume"},
	// energy (base: joule)
	"j":    {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "energy"},
	"kj":   {func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }, "energy"},
	"cal":  {func(v float64) float64 { return v * 4.184 }, func(v float64) float64 { return v / 4.184 }, "energy"},
	"kcal": {func(v float64) float64 { return v * 4184 }, func(v float64) float64 { return v / 4184 }, "energy"},
	"wh":   {func(v float64) float64 { return v * 3600 }, func(v float64) float64 { return v / 3600 }, "energy"},
	"kwh":  {func(v float64) float64 { return v * 3.6e6 }, func(v float64) float64 { return v / 3.6e6 }, "energy"},
	"ev":   {func(v float64) float64 { return v * 1.602e-19 }, func(v float64) float64 { return v / 1.602e-19 }, "energy"},
	"btu":  {func(v float64) float64 { return v * 1055.06 }, func(v float64) float64 { return v / 1055.06 }, "energy"},
	// pressure (base: pascal)
	"pa":   {func(v float64) float64 { return v }, func(v float64) float64 { return v }, "pressure"},
	"kpa":  {func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }, "pressure"},
	"mpa":  {func(v float64) float64 { return v * 1e6 }, func(v float64) float64 { return v / 1e6 }, "pressure"},
	"bar":  {func(v float64) float64 { return v * 1e5 }, func(v float64) float64 { return v / 1e5 }, "pressure"},
	"atm":  {func(v float64) float64 { return v * 101325 }, func(v float64) float64 { return v / 101325 }, "pressure"},
	"psi":  {func(v float64) float64 { return v * 6894.76 }, func(v float64) float64 { return v / 6894.76 }, "pressure"},
	"torr": {func(v float64) float64 { return v * 133.322 }, func(v float64) float64 { return v / 133.322 }, "pressure"},
	// digital storage (base: byte)
	"b":   {func(v float64) float64 { return v / 8 }, func(v float64) float64 { return v * 8 }, "storage"},
	"kb":  {func(v float64) float64 { return v * 1000 }, func(v float64) float64 { return v / 1000 }, "storage"},
	"mb":  {func(v float64) float64 { return v * 1e6 }, func(v float64) float64 { return v / 1e6 }, "storage"},
	"gb":  {func(v float64) float64 { return v * 1e9 }, func(v float64) float64 { return v / 1e9 }, "storage"},
	"tb":  {func(v float64) float64 { return v * 1e12 }, func(v float64) float64 { return v / 1e12 }, "storage"},
	"kib": {func(v float64) float64 { return v * 1024 }, func(v float64) float64 { return v / 1024 }, "storage"},
	"mib": {func(v float64) float64 { return v * 1048576 }, func(v float64) float64 { return v / 1048576 }, "storage"},
	"gib": {func(v float64) float64 { return v * 1073741824 }, func(v float64) float64 { return v / 1073741824 }, "storage"},
}

func ConvertUnit(cfg UnitConvertConfig) (UnitConvertResult, error) {
	from := strings.ToLower(strings.TrimSpace(cfg.From))
	to := strings.ToLower(strings.TrimSpace(cfg.To))

	fromDef, ok := unitTable[from]
	if !ok {
		return UnitConvertResult{}, fmt.Errorf("unknown unit: %q", cfg.From)
	}
	toDef, ok := unitTable[to]
	if !ok {
		return UnitConvertResult{}, fmt.Errorf("unknown unit: %q", cfg.To)
	}
	if fromDef.category != toDef.category {
		return UnitConvertResult{}, fmt.Errorf("cannot convert %s (%s) to %s (%s)", cfg.From, fromDef.category, cfg.To, toDef.category)
	}

	base := fromDef.toBase(cfg.Value)
	output := toDef.fromBase(base)

	formula := fmt.Sprintf("%s %s = %s %s", formatNum(cfg.Value), cfg.From, formatNum(output), cfg.To)

	return UnitConvertResult{
		Input:   cfg.Value,
		Output:  output,
		From:    cfg.From,
		To:      cfg.To,
		Formula: formula,
	}, nil
}

func formatNum(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e12 {
		return fmt.Sprintf("%.0f", v)
	}
	if math.Abs(v) < 0.001 || math.Abs(v) >= 1e9 {
		return fmt.Sprintf("%g", v)
	}
	return fmt.Sprintf("%g", v)
}

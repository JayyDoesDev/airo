package skills

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

type LatexConfig struct {
	Expressions []LatexExpr `json:"expressions"`
	DarkMode    bool        `json:"dark_mode"`
	FontSize    float64     `json:"font_size,omitempty"`
}

type LatexExpr struct {
	Label string `json:"label,omitempty"`
	Expr  string `json:"expr"`
}

func RenderLatex(cfg LatexConfig) ([]byte, error) {
	if len(cfg.Expressions) == 0 {
		return nil, fmt.Errorf("no expressions provided")
	}
	if len(cfg.Expressions) > 20 {
		return nil, fmt.Errorf("max 20 expressions")
	}

	scale := cfg.FontSize
	if scale <= 0 {
		scale = 1.0
	}
	if scale > 4 {
		scale = 4
	}

	padding := 8.0 * scale
	rowGap := 6.0 * scale
	labelGap := 4.0 * scale

	type row struct {
		label     string
		labelPath *canvas.Path
		exprPath  *canvas.Path
		labelW    float64
		labelH    float64
		exprW     float64
		exprH     float64
		rowH      float64
	}

	var rows []row
	maxLabelW := 0.0
	totalH := padding

	for _, e := range cfg.Expressions {
		r := row{label: e.Label}

		exprPath, err := canvas.ParseLaTeX(e.Expr)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", e.Expr, err)
		}
		eb := exprPath.Bounds()
		r.exprPath = exprPath
		r.exprW = (eb.X1 - eb.X0) * scale
		r.exprH = (eb.Y1 - eb.Y0) * scale

		if e.Label != "" {
			labelPath, err := canvas.ParseLaTeX(`\text{` + e.Label + `}`)
			if err != nil {
				labelPath, err = canvas.ParseLaTeX(e.Label)
				if err != nil {
					labelPath = nil
				}
			}
			if labelPath != nil {
				lb := labelPath.Bounds()
				r.labelPath = labelPath
				r.labelW = (lb.X1 - lb.X0) * scale
				r.labelH = (lb.Y1 - lb.Y0) * scale
				if r.labelW > maxLabelW {
					maxLabelW = r.labelW
				}
			}
		}

		r.rowH = r.exprH
		if r.labelH > r.rowH {
			r.rowH = r.labelH
		}

		rows = append(rows, r)
		totalH += r.rowH + rowGap
	}
	totalH += padding - rowGap

	labelColW := 0.0
	if maxLabelW > 0 {
		labelColW = maxLabelW + labelGap
	}

	maxExprW := 0.0
	for _, r := range rows {
		if r.exprW > maxExprW {
			maxExprW = r.exprW
		}
	}

	totalW := padding*2 + labelColW + maxExprW

	var bgColor, fgColor color.Color
	if cfg.DarkMode {
		bgColor = color.RGBA{30, 31, 34, 255}
		fgColor = color.White
	} else {
		bgColor = color.White
		fgColor = color.Black
	}

	c := canvas.New(totalW, totalH)
	ctx := canvas.NewContext(c)
	ctx.SetFillColor(bgColor)
	ctx.DrawPath(0, 0, canvas.Rectangle(totalW, totalH))

	y := totalH - padding
	for _, r := range rows {
		y -= r.rowH

		if r.labelPath != nil {
			lb := r.labelPath.Bounds()
			lx := padding - lb.X0*scale
			ly := y + (r.rowH-r.labelH)/2 + r.labelH - lb.Y0*scale
			ctx.SetFillColor(color.RGBA{150, 150, 150, 255})
			scaled := r.labelPath.Transform(canvas.Identity.Scale(scale, scale))
			ctx.DrawPath(lx, ly, scaled)
		}

		eb := r.exprPath.Bounds()
		ex := padding + labelColW - eb.X0*scale
		ey := y + (r.rowH-r.exprH)/2 + r.exprH - eb.Y0*scale
		ctx.SetFillColor(fgColor)
		scaled := r.exprPath.Transform(canvas.Identity.Scale(scale, scale))
		ctx.DrawPath(ex, ey, scaled)

		y -= rowGap
	}

	img := rasterizer.Draw(c, canvas.DPMM(6.0), canvas.DefaultColorSpace)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

package skills

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
)

type DrawingConfig struct {
	Width      int             `json:"width"`
	Height     int             `json:"height"`
	Background string          `json:"background"`
	Elements   []DrawingElement `json:"elements"`
}

type DrawingElement struct {
	Type        string       `json:"type"`
	X           float64      `json:"x,omitempty"`
	Y           float64      `json:"y,omitempty"`
	X1          float64      `json:"x1,omitempty"`
	Y1          float64      `json:"y1,omitempty"`
	X2          float64      `json:"x2,omitempty"`
	Y2          float64      `json:"y2,omitempty"`
	W           float64      `json:"w,omitempty"`
	H           float64      `json:"h,omitempty"`
	R           float64      `json:"r,omitempty"`
	RX          float64      `json:"rx,omitempty"`
	RY          float64      `json:"ry,omitempty"`
	Radius      float64      `json:"radius,omitempty"`
	Points      [][2]float64 `json:"points,omitempty"`
	Fill        string       `json:"fill,omitempty"`
	Stroke      string       `json:"stroke,omitempty"`
	StrokeWidth float64      `json:"stroke_width,omitempty"`
	Content     string       `json:"content,omitempty"`
	Size        float64      `json:"size,omitempty"`
	Color       string       `json:"color,omitempty"`
	Align       string       `json:"align,omitempty"`
	Rotation    float64      `json:"rotation,omitempty"`
	// arc
	StartAngle float64 `json:"start_angle,omitempty"`
	EndAngle   float64 `json:"end_angle,omitempty"`
	// bezier cubic: control points cp1, cp2
	CP1X float64 `json:"cp1x,omitempty"`
	CP1Y float64 `json:"cp1y,omitempty"`
	CP2X float64 `json:"cp2x,omitempty"`
	CP2Y float64 `json:"cp2y,omitempty"`
	// quadratic: single control point
	CPX float64 `json:"cpx,omitempty"`
	CPY float64 `json:"cpy,omitempty"`
	// star
	OuterR    float64 `json:"outer_r,omitempty"`
	InnerR    float64 `json:"inner_r,omitempty"`
	NumPoints int     `json:"num_points,omitempty"`
}

func RenderDrawing(cfg DrawingConfig) ([]byte, error) {
	w := cfg.Width
	h := cfg.Height
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}
	if w > 2000 {
		w = 2000
	}
	if h > 2000 {
		h = 2000
	}

	dc := gg.NewContext(w, h)

	bg := parseColor(cfg.Background, color.RGBA{20, 20, 30, 255})
	dc.SetColor(bg)
	dc.Clear()

	for _, el := range cfg.Elements {
		dc.Push()
		if el.Rotation != 0 {
			cx, cy := elementCenter(el)
			dc.RotateAbout(gg.Radians(el.Rotation), cx, cy)
		}
		switch el.Type {
		case "rect":
			drawRect(dc, el)
		case "circle":
			drawCircle(dc, el)
		case "ellipse":
			drawEllipse(dc, el)
		case "line":
			drawLine(dc, el)
		case "polygon":
			drawPolygon(dc, el)
		case "text":
			drawText(dc, el)
		case "arc":
			drawArc(dc, el)
		case "bezier":
			drawBezier(dc, el)
		case "quadratic":
			drawQuadratic(dc, el)
		case "star":
			drawStar(dc, el)
		}
		dc.Pop()
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

func drawRect(dc *gg.Context, el DrawingElement) {
	if el.Fill != "" {
		dc.SetColor(parseColor(el.Fill, color.RGBA{255, 255, 255, 255}))
		if el.Radius > 0 {
			dc.DrawRoundedRectangle(el.X, el.Y, el.W, el.H, el.Radius)
		} else {
			dc.DrawRectangle(el.X, el.Y, el.W, el.H)
		}
		if el.Stroke != "" {
			dc.FillPreserve()
			dc.SetColor(parseColor(el.Stroke, color.RGBA{0, 0, 0, 255}))
			dc.SetLineWidth(strokeWidth(el))
			dc.Stroke()
		} else {
			dc.Fill()
		}
	} else if el.Stroke != "" {
		dc.SetColor(parseColor(el.Stroke, color.RGBA{255, 255, 255, 255}))
		dc.SetLineWidth(strokeWidth(el))
		if el.Radius > 0 {
			dc.DrawRoundedRectangle(el.X, el.Y, el.W, el.H, el.Radius)
		} else {
			dc.DrawRectangle(el.X, el.Y, el.W, el.H)
		}
		dc.Stroke()
	}
}

func drawCircle(dc *gg.Context, el DrawingElement) {
	dc.DrawCircle(el.X, el.Y, el.R)
	fillAndStroke(dc, el)
}

func drawEllipse(dc *gg.Context, el DrawingElement) {
	rx := el.RX
	ry := el.RY
	if rx == 0 {
		rx = el.W / 2
	}
	if ry == 0 {
		ry = el.H / 2
	}
	dc.DrawEllipse(el.X, el.Y, rx, ry)
	fillAndStroke(dc, el)
}

func drawLine(dc *gg.Context, el DrawingElement) {
	dc.SetColor(parseColor(el.Stroke, color.RGBA{255, 255, 255, 255}))
	dc.SetLineWidth(strokeWidth(el))
	dc.DrawLine(el.X1, el.Y1, el.X2, el.Y2)
	dc.Stroke()
}

func drawPolygon(dc *gg.Context, el DrawingElement) {
	if len(el.Points) < 2 {
		return
	}
	dc.MoveTo(el.Points[0][0], el.Points[0][1])
	for _, p := range el.Points[1:] {
		dc.LineTo(p[0], p[1])
	}
	dc.ClosePath()
	fillAndStroke(dc, el)
}

func drawText(dc *gg.Context, el DrawingElement) {
	size := el.Size
	if size <= 0 {
		size = 16
	}
	if err := dc.LoadFontFace("/System/Library/Fonts/Helvetica.ttc", size); err != nil {
		if err := dc.LoadFontFace("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", size); err != nil {
			dc.LoadFontFace("/usr/share/fonts/dejavu/DejaVuSans.ttf", size)
		}
	}
	dc.SetColor(parseColor(el.Color, color.RGBA{255, 255, 255, 255}))
	ax := 0.0
	switch strings.ToLower(el.Align) {
	case "center":
		ax = 0.5
	case "right":
		ax = 1.0
	}
	dc.DrawStringAnchored(el.Content, el.X, el.Y, ax, 0.5)
}

func drawArc(dc *gg.Context, el DrawingElement) {
	start := gg.Radians(el.StartAngle)
	end := gg.Radians(el.EndAngle)
	dc.DrawArc(el.X, el.Y, el.R, start, end)
	fillAndStroke(dc, el)
}

func drawBezier(dc *gg.Context, el DrawingElement) {
	dc.MoveTo(el.X1, el.Y1)
	dc.CubicTo(el.CP1X, el.CP1Y, el.CP2X, el.CP2Y, el.X2, el.Y2)
	dc.SetColor(parseColor(el.Stroke, color.RGBA{255, 255, 255, 255}))
	dc.SetLineWidth(strokeWidth(el))
	dc.Stroke()
}

func drawQuadratic(dc *gg.Context, el DrawingElement) {
	dc.MoveTo(el.X1, el.Y1)
	dc.QuadraticTo(el.CPX, el.CPY, el.X2, el.Y2)
	dc.SetColor(parseColor(el.Stroke, color.RGBA{255, 255, 255, 255}))
	dc.SetLineWidth(strokeWidth(el))
	dc.Stroke()
}

func drawStar(dc *gg.Context, el DrawingElement) {
	n := el.NumPoints
	if n < 3 {
		n = 5
	}
	outer := el.OuterR
	inner := el.InnerR
	if inner <= 0 {
		inner = outer * 0.4
	}
	angle := -math.Pi / 2
	step := math.Pi / float64(n)
	dc.MoveTo(el.X+outer*math.Cos(angle), el.Y+outer*math.Sin(angle))
	for i := 1; i < n*2; i++ {
		angle += step
		r := outer
		if i%2 == 1 {
			r = inner
		}
		dc.LineTo(el.X+r*math.Cos(angle), el.Y+r*math.Sin(angle))
	}
	dc.ClosePath()
	fillAndStroke(dc, el)
}

func fillAndStroke(dc *gg.Context, el DrawingElement) {
	if el.Fill != "" && el.Stroke != "" {
		dc.SetColor(parseColor(el.Fill, color.RGBA{255, 255, 255, 255}))
		dc.FillPreserve()
		dc.SetColor(parseColor(el.Stroke, color.RGBA{0, 0, 0, 255}))
		dc.SetLineWidth(strokeWidth(el))
		dc.Stroke()
	} else if el.Fill != "" {
		dc.SetColor(parseColor(el.Fill, color.RGBA{255, 255, 255, 255}))
		dc.Fill()
	} else if el.Stroke != "" {
		dc.SetColor(parseColor(el.Stroke, color.RGBA{255, 255, 255, 255}))
		dc.SetLineWidth(strokeWidth(el))
		dc.Stroke()
	}
}

func strokeWidth(el DrawingElement) float64 {
	if el.StrokeWidth > 0 {
		return el.StrokeWidth
	}
	return 1
}

func elementCenter(el DrawingElement) (float64, float64) {
	switch el.Type {
	case "rect":
		return el.X + el.W/2, el.Y + el.H/2
	case "circle":
		return el.X, el.Y
	case "ellipse":
		return el.X, el.Y
	case "line", "bezier", "quadratic":
		return (el.X1 + el.X2) / 2, (el.Y1 + el.Y2) / 2
	case "arc", "star":
		return el.X, el.Y
	case "polygon":
		if len(el.Points) == 0 {
			return 0, 0
		}
		var sx, sy float64
		for _, p := range el.Points {
			sx += p[0]
			sy += p[1]
		}
		n := float64(len(el.Points))
		return sx / n, sy / n
	}
	return el.X, el.Y
}

func parseColor(hex string, fallback color.Color) color.Color {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	switch len(hex) {
	case 6:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	case 8:
		r, _ := strconv.ParseUint(hex[0:2], 16, 8)
		g, _ := strconv.ParseUint(hex[2:4], 16, 8)
		b, _ := strconv.ParseUint(hex[4:6], 16, 8)
		a, _ := strconv.ParseUint(hex[6:8], 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
	case 3:
		r, _ := strconv.ParseUint(string(hex[0])+string(hex[0]), 16, 8)
		g, _ := strconv.ParseUint(string(hex[1])+string(hex[1]), 16, 8)
		b, _ := strconv.ParseUint(string(hex[2])+string(hex[2]), 16, 8)
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}
	}
	return fallback
}

var _ = math.Pi

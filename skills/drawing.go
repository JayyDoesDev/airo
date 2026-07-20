package skills

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

type DrawingConfig struct {
	Width      int              `json:"width"`
	Height     int              `json:"height"`
	Background string           `json:"background"`
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
	StartAngle  float64      `json:"start_angle,omitempty"`
	EndAngle    float64      `json:"end_angle,omitempty"`
	CP1X        float64      `json:"cp1x,omitempty"`
	CP1Y        float64      `json:"cp1y,omitempty"`
	CP2X        float64      `json:"cp2x,omitempty"`
	CP2Y        float64      `json:"cp2y,omitempty"`
	CPX         float64      `json:"cpx,omitempty"`
	CPY         float64      `json:"cpy,omitempty"`
	OuterR      float64      `json:"outer_r,omitempty"`
	InnerR      float64      `json:"inner_r,omitempty"`
	NumPoints   int          `json:"num_points,omitempty"`
	Label       string       `json:"label,omitempty"`
	Opacity     float64      `json:"opacity,omitempty"`
	Gradient    *GradientDef `json:"gradient,omitempty"`
	D           string       `json:"d,omitempty"`
	Src         string       `json:"src,omitempty"`
	Dash        []float64    `json:"dash,omitempty"`
	LineCap     string       `json:"linecap,omitempty"`
	LineJoin    string       `json:"linejoin,omitempty"`
	ScaleX      float64      `json:"scale_x,omitempty"`
	ScaleY      float64      `json:"scale_y,omitempty"`
	ShearX      float64      `json:"shear_x,omitempty"`
	ShearY      float64      `json:"shear_y,omitempty"`
	TextWidth   float64      `json:"text_width,omitempty"`
}

type GradientDef struct {
	Type  string         `json:"type"`
	X0    float64        `json:"x0,omitempty"`
	Y0    float64        `json:"y0,omitempty"`
	X1    float64        `json:"x1,omitempty"`
	Y1    float64        `json:"y1,omitempty"`
	R0    float64        `json:"r0,omitempty"`
	R1    float64        `json:"r1,omitempty"`
	Stops []GradientStop `json:"stops"`
}

type GradientStop struct {
	Offset float64 `json:"offset"`
	Color  string  `json:"color"`
}

const dpmm = 5.7

func RenderDrawing(cfg DrawingConfig) ([]byte, error) {
	w := float64(cfg.Width)
	h := float64(cfg.Height)
	if w <= 0 {
		w = 800
	}
	if h <= 0 {
		h = 600
	}
	if w > 4000 {
		w = 4000
	}
	if h > 4000 {
		h = 4000
	}

	c := canvas.New(w, h)
	ctx := canvas.NewContext(c)

	ctx.SetCoordSystem(canvas.CartesianIV)

	bg := parseColorRGBA(cfg.Background, color.RGBA{20, 20, 30, 255})
	ctx.SetFillColor(bg)
	ctx.DrawPath(0, 0, canvas.Rectangle(w, h))

	for _, el := range cfg.Elements {
		drawElement(ctx, el, w, h)
	}

	img := rasterizer.Draw(c, canvas.DPMM(dpmm), canvas.DefaultColorSpace)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawElement(ctx *canvas.Context, el DrawingElement, cw, ch float64) {
	ctx.Push()
	defer ctx.Pop()

	opacity := el.Opacity
	if opacity <= 0 || opacity > 1 {
		opacity = 1
	}

	cx, cy := elementCenter(el)
	if el.Rotation != 0 || el.ScaleX != 0 || el.ScaleY != 0 || el.ShearX != 0 || el.ShearY != 0 {
		ctx.Translate(cx, cy)
		if el.Rotation != 0 {
			ctx.Rotate(-el.Rotation)
		}
		if el.ScaleX != 0 || el.ScaleY != 0 {
			sx := el.ScaleX
			if sx == 0 {
				sx = 1
			}
			sy := el.ScaleY
			if sy == 0 {
				sy = 1
			}
			ctx.Scale(sx, sy)
		}
		if el.ShearX != 0 || el.ShearY != 0 {
			ctx.Shear(el.ShearX, el.ShearY)
		}
		ctx.Translate(-cx, -cy)
	}

	applyStrokeStyle(ctx, el, opacity)

	switch el.Type {
	case "rect":
		drawRect(ctx, el, opacity)
	case "circle":
		drawCircle(ctx, el, opacity)
	case "ellipse":
		drawEllipse(ctx, el, opacity)
	case "line":
		drawLine(ctx, el, opacity)
	case "polygon":
		drawPolygon(ctx, el, opacity)
	case "text":
		drawText(ctx, el, opacity)
	case "arc":
		drawArc(ctx, el, opacity)
	case "bezier":
		drawBezier(ctx, el, opacity)
	case "quadratic":
		drawQuadratic(ctx, el, opacity)
	case "star":
		drawStar(ctx, el, opacity)
	case "path":
		drawPath(ctx, el, opacity)
	case "image":
		drawImage(ctx, el)
	}
}

func drawRect(ctx *canvas.Context, el DrawingElement, opacity float64) {
	var p *canvas.Path
	if el.Radius > 0 {
		p = canvas.RoundedRectangle(el.W, el.H, el.Radius)
	} else {
		p = canvas.Rectangle(el.W, el.H)
	}
	paintPath(ctx, p, el, el.X, el.Y, opacity)
}

func drawCircle(ctx *canvas.Context, el DrawingElement, opacity float64) {
	p := canvas.Circle(el.R)
	paintPath(ctx, p, el, el.X, el.Y, opacity)
}

func drawEllipse(ctx *canvas.Context, el DrawingElement, opacity float64) {
	rx := el.RX
	ry := el.RY
	if rx == 0 {
		rx = el.W / 2
	}
	if ry == 0 {
		ry = el.H / 2
	}
	p := canvas.Ellipse(rx, ry)
	paintPath(ctx, p, el, el.X, el.Y, opacity)
}

func drawLine(ctx *canvas.Context, el DrawingElement, opacity float64) {
	p := &canvas.Path{}
	p.MoveTo(el.X1, el.Y1)
	p.LineTo(el.X2, el.Y2)
	sw := el.StrokeWidth
	if sw <= 0 {
		sw = 1
	}
	ctx.SetStrokeColor(withOpacity(parseColorRGBA(el.Stroke, color.RGBA{255, 255, 255, 255}), opacity))
	ctx.SetStrokeWidth(sw)
	ctx.DrawPath(0, 0, p)
}

func drawPolygon(ctx *canvas.Context, el DrawingElement, opacity float64) {
	if len(el.Points) < 2 {
		return
	}
	p := &canvas.Path{}
	p.MoveTo(el.Points[0][0], el.Points[0][1])
	for _, pt := range el.Points[1:] {
		p.LineTo(pt[0], pt[1])
	}
	p.Close()
	paintPath(ctx, p, el, 0, 0, opacity)
}

func drawText(ctx *canvas.Context, el DrawingElement, opacity float64) {
	size := el.Size
	if size <= 0 {
		size = 16
	}

	face := canvas.NewFontFamily("sans")
	for _, path := range []string{
		"/System/Library/Fonts/Helvetica.ttc",
		"/System/Library/Fonts/SFNSDisplay.ttf",
		"/System/Library/Fonts/SFNS.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
	} {
		if err := face.LoadFontFile(path, canvas.FontRegular); err == nil {
			break
		}
	}

	col := parseColorRGBA(el.Color, color.RGBA{255, 255, 255, 255})
	col = withOpacity(col, opacity)

	ff := face.Face(size, col, canvas.FontRegular, canvas.FontNormal)

	var halign canvas.TextAlign
	switch strings.ToLower(el.Align) {
	case "center":
		halign = canvas.Center
	case "right":
		halign = canvas.Right
	default:
		halign = canvas.Left
	}

	rt := canvas.NewRichText(ff)
	rt.WriteString(el.Content)
	w := el.TextWidth
	if w <= 0 {
		w = 2000
	}
	txt := rt.ToText(w, 0, halign, canvas.Top, nil)
	ctx.DrawText(el.X, el.Y, txt)
}

func drawArc(ctx *canvas.Context, el DrawingElement, opacity float64) {
	p := &canvas.Path{}
	start := el.StartAngle * math.Pi / 180
	end := el.EndAngle * math.Pi / 180
	p.MoveTo(el.X+el.R*math.Cos(start), el.Y+el.R*math.Sin(start))
	p.Arc(el.R, el.R, 0, start, end)
	paintPath(ctx, p, el, 0, 0, opacity)
}

func drawBezier(ctx *canvas.Context, el DrawingElement, opacity float64) {
	p := &canvas.Path{}
	p.MoveTo(el.X1, el.Y1)
	p.CubeTo(el.CP1X, el.CP1Y, el.CP2X, el.CP2Y, el.X2, el.Y2)
	sw := el.StrokeWidth
	if sw <= 0 {
		sw = 1
	}
	ctx.SetStrokeColor(withOpacity(parseColorRGBA(el.Stroke, color.RGBA{255, 255, 255, 255}), opacity))
	ctx.SetStrokeWidth(sw)
	ctx.DrawPath(0, 0, p)
}

func drawQuadratic(ctx *canvas.Context, el DrawingElement, opacity float64) {
	p := &canvas.Path{}
	p.MoveTo(el.X1, el.Y1)
	p.QuadTo(el.CPX, el.CPY, el.X2, el.Y2)
	sw := el.StrokeWidth
	if sw <= 0 {
		sw = 1
	}
	ctx.SetStrokeColor(withOpacity(parseColorRGBA(el.Stroke, color.RGBA{255, 255, 255, 255}), opacity))
	ctx.SetStrokeWidth(sw)
	ctx.DrawPath(0, 0, p)
}

func drawStar(ctx *canvas.Context, el DrawingElement, opacity float64) {
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
	p := &canvas.Path{}
	p.MoveTo(el.X+outer*math.Cos(angle), el.Y+outer*math.Sin(angle))
	for i := 1; i < n*2; i++ {
		angle += step
		r := outer
		if i%2 == 1 {
			r = inner
		}
		p.LineTo(el.X+r*math.Cos(angle), el.Y+r*math.Sin(angle))
	}
	p.Close()
	paintPath(ctx, p, el, 0, 0, opacity)
}

func drawPath(ctx *canvas.Context, el DrawingElement, opacity float64) {
	if el.D == "" {
		return
	}
	p, err := canvas.ParseSVGPath(el.D)
	if err != nil {
		return
	}
	paintPath(ctx, p, el, 0, 0, opacity)
}

func drawImage(ctx *canvas.Context, el DrawingElement) {
	if el.Src == "" {
		return
	}
	resp, err := http.Get(el.Src)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return
	}
	ctx.DrawImage(el.X, el.Y, img, canvas.DPMM(dpmm))
}

func paintPath(ctx *canvas.Context, p *canvas.Path, el DrawingElement, ox, oy float64, opacity float64) {
	hasFill := el.Fill != "" && strings.ToLower(el.Fill) != "none"
	hasStroke := el.Stroke != "" && strings.ToLower(el.Stroke) != "none"

	sw := el.StrokeWidth
	if sw <= 0 {
		sw = 1
	}

	if hasFill {
		ctx.SetFillColor(withOpacity(parseColorRGBA(el.Fill, color.RGBA{255, 255, 255, 255}), opacity))
	} else {
		ctx.SetFillColor(color.RGBA{0, 0, 0, 0})
	}
	if hasStroke {
		ctx.SetStrokeColor(withOpacity(parseColorRGBA(el.Stroke, color.RGBA{255, 255, 255, 255}), opacity))
		ctx.SetStrokeWidth(sw)
	} else {
		ctx.SetStrokeColor(color.RGBA{0, 0, 0, 0})
	}
	ctx.DrawPath(ox, oy, p)
}

func applyStrokeStyle(ctx *canvas.Context, el DrawingElement, opacity float64) {
	if len(el.Dash) > 0 {
		ctx.SetDashes(0, el.Dash...)
	}
	switch strings.ToLower(el.LineCap) {
	case "round":
		ctx.SetStrokeCapper(canvas.RoundCap)
	case "square":
		ctx.SetStrokeCapper(canvas.SquareCap)
	default:
		ctx.SetStrokeCapper(canvas.ButtCap)
	}
	switch strings.ToLower(el.LineJoin) {
	case "round":
		ctx.SetStrokeJoiner(canvas.RoundJoin)
	case "bevel":
		ctx.SetStrokeJoiner(canvas.BevelJoin)
	default:
		ctx.SetStrokeJoiner(canvas.MiterJoin)
	}
}

func elementCenter(el DrawingElement) (float64, float64) {
	switch el.Type {
	case "rect":
		return el.X + el.W/2, el.Y + el.H/2
	case "circle", "ellipse", "arc", "star":
		return el.X, el.Y
	case "line", "bezier", "quadratic":
		return (el.X1 + el.X2) / 2, (el.Y1 + el.Y2) / 2
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

func parseColorRGBA(hex string, fallback color.RGBA) color.RGBA {
	hex = strings.TrimSpace(hex)

	switch strings.ToLower(hex) {
	case "white":
		return color.RGBA{255, 255, 255, 255}
	case "black":
		return color.RGBA{0, 0, 0, 255}
	case "red":
		return color.RGBA{255, 0, 0, 255}
	case "green":
		return color.RGBA{0, 128, 0, 255}
	case "blue":
		return color.RGBA{0, 0, 255, 255}
	case "yellow":
		return color.RGBA{255, 255, 0, 255}
	case "orange":
		return color.RGBA{255, 165, 0, 255}
	case "purple":
		return color.RGBA{128, 0, 128, 255}
	case "pink":
		return color.RGBA{255, 192, 203, 255}
	case "cyan":
		return color.RGBA{0, 255, 255, 255}
	case "magenta":
		return color.RGBA{255, 0, 255, 255}
	case "gray", "grey":
		return color.RGBA{128, 128, 128, 255}
	case "transparent", "none", "":
		return color.RGBA{0, 0, 0, 0}
	}

	lower := strings.ToLower(hex)
	if strings.HasPrefix(lower, "rgb") {
		inner := strings.TrimPrefix(strings.TrimPrefix(lower, "rgba("), "rgb(")
		inner = strings.TrimSuffix(inner, ")")
		parts := strings.Split(inner, ",")
		if len(parts) >= 3 {
			r, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			g, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			b, _ := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			a := 255.0
			if len(parts) == 4 {
				av, _ := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
				a = av * 255
			}
			return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
		}
	}

	hex = strings.TrimPrefix(hex, "#")
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

func withOpacity(c color.RGBA, opacity float64) color.RGBA {
	if opacity <= 0 || opacity >= 1 {
		return c
	}
	c.A = uint8(float64(c.A) * opacity)
	return c
}

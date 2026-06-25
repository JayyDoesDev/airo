package skills

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
)

type PixelArtConfig struct {
	Scale   int        `json:"scale"`
	Grid    PixelGrid  `json:"grid"`
	Palette []string   `json:"palette,omitempty"`
}

type PixelGrid [][]string

func (g *PixelGrid) UnmarshalJSON(data []byte) error {
	// try [][]string first
	var grid2d [][]string
	if err := json.Unmarshal(data, &grid2d); err == nil {
		*g = grid2d
		return nil
	}
	// fall back to []string — split each row into individual characters
	var rows []string
	if err := json.Unmarshal(data, &rows); err != nil {
		return err
	}
	result := make([][]string, len(rows))
	for i, row := range rows {
		cells := strings.Split(row, "")
		result[i] = cells
	}
	*g = result
	return nil
}

func RenderPixelArt(cfg PixelArtConfig) ([]byte, error) {
	if len(cfg.Grid) == 0 {
		return nil, fmt.Errorf("grid is empty")
	}

	scale := cfg.Scale
	if scale <= 0 {
		scale = 16
	}
	if scale > 64 {
		scale = 64
	}

	rows := len(cfg.Grid)
	cols := 0
	for _, row := range cfg.Grid {
		if len(row) > cols {
			cols = len(row)
		}
	}

	imgW := cols * scale
	imgH := rows * scale
	if imgW > 2048 {
		imgW = 2048
	}
	if imgH > 2048 {
		imgH = 2048
	}

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))

	for y, row := range cfg.Grid {
		for x, cell := range row {
			if cell == "" || cell == "." || cell == "transparent" {
				continue
			}

			var c color.Color
			if len(cfg.Palette) > 0 {
				if idx := paletteIndex(cell); idx >= 0 && idx < len(cfg.Palette) {
					c = parseColor(cfg.Palette[idx], color.RGBA{0, 0, 0, 0})
				} else {
					c = parseColor(cell, color.RGBA{0, 0, 0, 0})
				}
			} else {
				c = parseColor(cell, color.RGBA{0, 0, 0, 0})
			}

			r32, g32, b32, a32 := c.RGBA()
			if a32 == 0 {
				continue
			}
			px := color.RGBA{uint8(r32 >> 8), uint8(g32 >> 8), uint8(b32 >> 8), uint8(a32 >> 8)}

			px0 := x * scale
			py0 := y * scale
			for dy := 0; dy < scale && py0+dy < imgH; dy++ {
				for dx := 0; dx < scale && px0+dx < imgW; dx++ {
					img.SetRGBA(px0+dx, py0+dy, px)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encode: %w", err)
	}
	return buf.Bytes(), nil
}

func paletteIndex(s string) int {
	if len(s) != 1 {
		return -1
	}
	c := s[0]
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'z' {
		return int(c-'a') + 10
	}
	if c >= 'A' && c <= 'Z' {
		return int(c-'A') + 10
	}
	return -1
}

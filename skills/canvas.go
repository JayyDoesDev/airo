package skills

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

type CanvasLayer struct {
	Name     string           `json:"name"`
	Visible  bool             `json:"visible"`
	Opacity  float64          `json:"opacity,omitempty"`
	Elements []DrawingElement `json:"elements"`
}

type Canvas struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	ChannelID string        `json:"channel_id"`
	Config    DrawingConfig `json:"config"`
	Layers    []CanvasLayer `json:"layers,omitempty"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

type CanvasOperation struct {
	Action       string            `json:"action"`
	CanvasID     string            `json:"canvas_id,omitempty"`
	Name         string            `json:"name,omitempty"`
	Width        int               `json:"width,omitempty"`
	Height       int               `json:"height,omitempty"`
	Background   string            `json:"background,omitempty"`
	Element      *DrawingElement   `json:"element,omitempty"`
	ElementIndex int               `json:"element_index,omitempty"`
	Layer        string            `json:"layer,omitempty"`
	Label        string            `json:"label,omitempty"`
	Operations   []CanvasOperation `json:"operations,omitempty"`
}

type CanvasManager struct {
	mu       sync.RWMutex
	canvases map[string]*Canvas
	nextID   int64
}

var GlobalCanvasManager = NewCanvasManager()

func NewCanvasManager() *CanvasManager {
	return &CanvasManager{
		canvases: make(map[string]*Canvas),
		nextID:   1,
	}
}

func (cm *CanvasManager) CreateCanvas(channelID, name string, width, height int, bg string) *Canvas {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	id := fmt.Sprintf("canvas_%d", cm.nextID)
	cm.nextID++

	w := width
	if w <= 0 {
		w = 800
	}
	h := height
	if h <= 0 {
		h = 600
	}
	if bg == "" {
		bg = "#14141e"
	}

	cfg := DrawingConfig{
		Width:      w,
		Height:     h,
		Background: bg,
		Elements:   []DrawingElement{},
	}

	c := &Canvas{
		ID:        id,
		Name:      name,
		ChannelID: channelID,
		Config:    cfg,
		Layers: []CanvasLayer{
			{
				Name:     "Layer 1",
				Visible:  true,
				Opacity:  1.0,
				Elements: []DrawingElement{},
			},
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	cm.canvases[id] = c
	return c
}

func (cm *CanvasManager) GetCanvas(id string) (*Canvas, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	c, ok := cm.canvases[id]
	return c, ok
}

func (cm *CanvasManager) resolveCanvas(idOrName string) (*Canvas, string, bool) {
	if c, ok := cm.canvases[idOrName]; ok {
		return c, idOrName, true
	}
	for id, c := range cm.canvases {
		if strings.EqualFold(c.Name, idOrName) {
			return c, id, true
		}
	}
	return nil, "", false
}

func (cm *CanvasManager) DeleteCanvas(id string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	_, resolvedID, ok := cm.resolveCanvas(id)
	if ok {
		delete(cm.canvases, resolvedID)
		return true
	}
	return false
}

func (cm *CanvasManager) ListCanvases(channelID string) []*Canvas {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	var result []*Canvas
	for _, c := range cm.canvases {
		if c.ChannelID == channelID {
			result = append(result, c)
		}
	}
	return result
}

func (cm *CanvasManager) ListAllCanvases() []*Canvas {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make([]*Canvas, 0, len(cm.canvases))
	for _, c := range cm.canvases {
		result = append(result, c)
	}
	return result
}

func (cm *CanvasManager) layerRef(c *Canvas, layerName string) *[]DrawingElement {
	if len(c.Layers) == 0 {
		return &c.Config.Elements
	}
	for i := range c.Layers {
		if c.Layers[i].Name == layerName {
			return &c.Layers[i].Elements
		}
	}
	return &c.Layers[0].Elements
}

func (cm *CanvasManager) AddElement(canvasID string, el DrawingElement, layerName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}

	els := cm.layerRef(c, layerName)
	*els = append(*els, el)
	c.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

func (cm *CanvasManager) ModifyElement(canvasID string, index int, el DrawingElement) error {
	return cm.ModifyElementBy(canvasID, index, el, "")
}

func (cm *CanvasManager) ModifyElementBy(canvasID string, index int, el DrawingElement, layerName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}

	els := cm.layerRef(c, layerName)
	if index < 0 || index >= len(*els) {
		return fmt.Errorf("element index %d out of range (0-%d)", index, len(*els)-1)
	}

	(*els)[index] = el
	c.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

func (cm *CanvasManager) RemoveElement(canvasID string, index int) error {
	return cm.RemoveElementBy(canvasID, index, "")
}

func (cm *CanvasManager) RemoveElementBy(canvasID string, index int, layerName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}

	els := cm.layerRef(c, layerName)
	if index < 0 || index >= len(*els) {
		return fmt.Errorf("element index %d out of range (0-%d)", index, len(*els)-1)
	}

	*els = append((*els)[:index], (*els)[index+1:]...)
	c.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

func (cm *CanvasManager) findElementIndex(c *Canvas, label string) (int, string) {
	for li := range c.Layers {
		for ei, el := range c.Layers[li].Elements {
			if el.Label == label {
				return ei, c.Layers[li].Name
			}
		}
	}
	for ei, el := range c.Config.Elements {
		if el.Label == label {
			return ei, ""
		}
	}
	return -1, ""
}

func (cm *CanvasManager) IndexOfLabel(canvasID, label string) (int, string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return -1, "", fmt.Errorf("canvas not found: %s", canvasID)
	}
	idx, layer := cm.findElementIndex(c, label)
	if idx < 0 {
		return -1, "", fmt.Errorf("no element with label %q found", label)
	}
	return idx, layer, nil
}

func (cm *CanvasManager) CreateLayer(canvasID, name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}

	c.Layers = append(c.Layers, CanvasLayer{
		Name:     name,
		Visible:  true,
		Opacity:  1.0,
		Elements: []DrawingElement{},
	})
	c.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

func (cm *CanvasManager) SetLayerVisibility(canvasID, layerName string, visible bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}
	for i := range c.Layers {
		if c.Layers[i].Name == layerName {
			c.Layers[i].Visible = visible
			c.UpdatedAt = time.Now().Format(time.RFC3339)
			return nil
		}
	}
	return fmt.Errorf("layer %q not found", layerName)
}

func (cm *CanvasManager) RemoveLayer(canvasID, layerName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}
	for i := range c.Layers {
		if c.Layers[i].Name == layerName {
			c.Layers = append(c.Layers[:i], c.Layers[i+1:]...)
			c.UpdatedAt = time.Now().Format(time.RFC3339)
			return nil
		}
	}
	return fmt.Errorf("layer %q not found", layerName)
}

func (cm *CanvasManager) ReorderLayer(canvasID, layerName string, newIndex int) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}
	if newIndex < 0 || newIndex >= len(c.Layers) {
		return fmt.Errorf("new index %d out of range (0-%d)", newIndex, len(c.Layers)-1)
	}
	for i := range c.Layers {
		if c.Layers[i].Name == layerName {
			layer := c.Layers[i]
			c.Layers = append(c.Layers[:i], c.Layers[i+1:]...)
			c.Layers = append(c.Layers[:newIndex], append([]CanvasLayer{layer}, c.Layers[newIndex:]...)...)
			c.UpdatedAt = time.Now().Format(time.RFC3339)
			return nil
		}
	}
	return fmt.Errorf("layer %q not found", layerName)
}

func (cm *CanvasManager) SetLayerOpacity(canvasID, layerName string, opacity float64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}
	for i := range c.Layers {
		if c.Layers[i].Name == layerName {
			c.Layers[i].Opacity = opacity
			c.UpdatedAt = time.Now().Format(time.RFC3339)
			return nil
		}
	}
	return fmt.Errorf("layer %q not found", layerName)
}

func (cm *CanvasManager) RenderCanvas(canvasID string) ([]byte, error) {
	cm.mu.RLock()
	c, _, ok := cm.resolveCanvas(canvasID)
	cm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("canvas not found: %s", canvasID)
	}

	merged := DrawingConfig{
		Width:      c.Config.Width,
		Height:     c.Config.Height,
		Background: c.Config.Background,
	}
	merged.Elements = append(merged.Elements, c.Config.Elements...)

	for _, layer := range c.Layers {
		if !layer.Visible {
			continue
		}
		for _, el := range layer.Elements {
			el := el
			if layer.Opacity < 1 && layer.Opacity > 0 {
				if el.Opacity <= 0 || el.Opacity > 1 {
					el.Opacity = layer.Opacity
				} else {
					el.Opacity *= layer.Opacity
				}
			}
			merged.Elements = append(merged.Elements, el)
		}
	}

	return RenderDrawing(merged)
}

func (cm *CanvasManager) RenderCanvasToPNG(canvasID string) ([]byte, string, error) {
	pngBytes, err := cm.RenderCanvas(canvasID)
	if err != nil {
		return nil, "", err
	}

	cm.mu.RLock()
	c2, _, _ := cm.resolveCanvas(canvasID)
	name := ""
	if c2 != nil {
		name = c2.Name
	}
	cm.mu.RUnlock()

	filename := "canvas.png"
	if name != "" {
		filename = "canvas_" + name + ".png"
	}
	return pngBytes, filename, nil
}

func (cm *CanvasManager) CanvasJSON(canvasID string) (string, error) {
	cm.mu.RLock()
	c, _, ok := cm.resolveCanvas(canvasID)
	cm.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("canvas not found: %s", canvasID)
	}
	type canvasView struct {
		Name       string           `json:"name"`
		Width      int              `json:"width"`
		Height     int              `json:"height"`
		Background string           `json:"background"`
		Elements   []DrawingElement `json:"elements"`
		Layers     []CanvasLayer    `json:"layers"`
	}
	v := canvasView{
		Name:       c.Name,
		Width:      c.Config.Width,
		Height:     c.Config.Height,
		Background: c.Config.Background,
		Elements:   c.Config.Elements,
		Layers:     c.Layers,
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (cm *CanvasManager) ClearCanvas(canvasID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	c, _, ok := cm.resolveCanvas(canvasID)
	if !ok {
		return fmt.Errorf("canvas not found: %s", canvasID)
	}

	c.Config.Elements = []DrawingElement{}
	for i := range c.Layers {
		c.Layers[i].Elements = []DrawingElement{}
	}
	c.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}

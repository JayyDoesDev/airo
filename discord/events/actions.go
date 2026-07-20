package events

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/jayydoesdev/airo/bot/skills"
	"github.com/jayydoesdev/airo/bot/skills/actions"
	taskqueue "github.com/jayydoesdev/airo/bot/tasks"
)

var (
	lastDrawing   = map[string]*skills.DrawingConfig{}
	lastDrawingMu sync.RWMutex
)

type CanvasResult struct {
	CanvasPNG      []byte
	CanvasFilename string
	CanvasResp     string
}

type RenderResult struct {
	ChartCfg    *skills.ChartConfig
	ChartPNG    []byte
	DrawingPNG  []byte
	PixelArtPNG []byte
	LatexPNG    []byte
	PrependText string
}

func executeCanvasOps(actionData actions.ActionData, channelID string) CanvasResult {
	var res CanvasResult

	canvasOp := actionData.CanvasOp
	if canvasOp == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "canvas" && t.CanvasOp != nil {
				canvasOp = t.CanvasOp
				break
			}
		}
	}

	if canvasOp == nil {
		fmt.Println("[canvas] no canvas op found")
		return res
	}

	fmt.Println("[canvas] found canvas op, action:", canvasOp.Action)
	cm := skills.GlobalCanvasManager

	ops := []skills.CanvasOperation{*canvasOp}
	if canvasOp.Action == "batch" && len(canvasOp.Operations) > 0 {
		ops = canvasOp.Operations
		fmt.Println("[canvas] batch with", len(ops), "operations")
	}

	var createdID string
	var respParts []string

	for _, op := range ops {
		op := op
		if op.CanvasID == "" && createdID != "" {
			op.CanvasID = createdID
		}

		switch op.Action {
		case "create":
			c := cm.CreateCanvas(channelID, op.Name, op.Width, op.Height, op.Background)
			createdID = c.ID
			fmt.Println("[canvas] created:", c.ID)
			respParts = append(respParts, fmt.Sprintf("created canvas **%s** (ID: `%s`) — %dx%d", c.Name, c.ID, c.Config.Width, c.Config.Height))

		case "add":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Element == nil {
				respParts = append(respParts, "no element provided to add")
			} else if err := cm.AddElement(op.CanvasID, *op.Element, op.Layer); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("added %s element to canvas `%s`", op.Element.Type, op.CanvasID))
			}

		case "render":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else {
				fmt.Println("[canvas] rendering...")
				png, filename, err := cm.RenderCanvasToPNG(op.CanvasID)
				if err != nil {
					fmt.Println("[canvas] render error:", err)
					respParts = append(respParts, "canvas error: "+err.Error())
				} else {
					fmt.Println("[canvas] render ok, size:", len(png), "bytes, file:", filename)
					res.CanvasPNG = png
					res.CanvasFilename = filename
					respParts = append(respParts, fmt.Sprintf("rendered canvas `%s`", op.CanvasID))
				}
			}

		case "create_layer":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Name == "" {
				respParts = append(respParts, "no layer name provided")
			} else if err := cm.CreateLayer(op.CanvasID, op.Name); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("created layer **%s** on canvas `%s`", op.Name, op.CanvasID))
			}

		case "modify":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Element == nil {
				respParts = append(respParts, "no element provided for modification")
			} else {
				idx, layer := op.ElementIndex, op.Layer
				if op.Label != "" {
					foundIdx, foundLayer, err := cm.IndexOfLabel(op.CanvasID, op.Label)
					if err != nil {
						respParts = append(respParts, "canvas error: "+err.Error())
						continue
					}
					idx, layer = foundIdx, foundLayer
				}
				if err := cm.ModifyElementBy(op.CanvasID, idx, *op.Element, layer); err != nil {
					respParts = append(respParts, "canvas error: "+err.Error())
				} else {
					respParts = append(respParts, fmt.Sprintf("modified element on canvas `%s`", op.CanvasID))
				}
			}

		case "remove":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else {
				idx, layer := op.ElementIndex, op.Layer
				if op.Label != "" {
					foundIdx, foundLayer, err := cm.IndexOfLabel(op.CanvasID, op.Label)
					if err != nil {
						respParts = append(respParts, "canvas error: "+err.Error())
						continue
					}
					idx, layer = foundIdx, foundLayer
				}
				if err := cm.RemoveElementBy(op.CanvasID, idx, layer); err != nil {
					respParts = append(respParts, "canvas error: "+err.Error())
				} else {
					respParts = append(respParts, fmt.Sprintf("removed element from canvas `%s`", op.CanvasID))
				}
			}

		case "clear":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if err := cm.ClearCanvas(op.CanvasID); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("cleared all elements from canvas `%s`", op.CanvasID))
			}

		case "delete":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if cm.DeleteCanvas(op.CanvasID) {
				respParts = append(respParts, fmt.Sprintf("deleted canvas `%s`", op.CanvasID))
			} else {
				respParts = append(respParts, fmt.Sprintf("canvas `%s` not found", op.CanvasID))
			}

		case "list":
			canvases := cm.ListCanvases(channelID)
			if len(canvases) == 0 {
				respParts = append(respParts, "no canvases in this channel")
			} else {
				var b strings.Builder
				for _, c := range canvases {
					totalEls := len(c.Config.Elements)
					layerInfo := ""
					if len(c.Layers) > 0 {
						visible := 0
						for _, l := range c.Layers {
							totalEls += len(l.Elements)
							if l.Visible {
								visible++
							}
						}
						layerInfo = fmt.Sprintf(", %d layers (%d visible)", len(c.Layers), visible)
					}
					b.WriteString(fmt.Sprintf("- `%s` **%s** (%dx%d%s, %d elements)\n", c.ID, c.Name, c.Config.Width, c.Config.Height, layerInfo, totalEls))
				}
				respParts = append(respParts, b.String())
			}

		case "hide_layer":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Name == "" {
				respParts = append(respParts, "no layer name provided")
			} else if err := cm.SetLayerVisibility(op.CanvasID, op.Name, false); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("hid layer **%s** on canvas `%s`", op.Name, op.CanvasID))
			}

		case "show_layer":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Name == "" {
				respParts = append(respParts, "no layer name provided")
			} else if err := cm.SetLayerVisibility(op.CanvasID, op.Name, true); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("showed layer **%s** on canvas `%s`", op.Name, op.CanvasID))
			}

		case "remove_layer":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Name == "" {
				respParts = append(respParts, "no layer name provided")
			} else if err := cm.RemoveLayer(op.CanvasID, op.Name); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("removed layer **%s** from canvas `%s`", op.Name, op.CanvasID))
			}

		case "reorder_layer":
			if op.CanvasID == "" {
				respParts = append(respParts, "no canvas_id provided")
			} else if op.Name == "" {
				respParts = append(respParts, "no layer name provided")
			} else if err := cm.ReorderLayer(op.CanvasID, op.Name, op.ElementIndex); err != nil {
				respParts = append(respParts, "canvas error: "+err.Error())
			} else {
				respParts = append(respParts, fmt.Sprintf("moved layer **%s** to position %d on canvas `%s`", op.Name, op.ElementIndex, op.CanvasID))
			}

		default:
			respParts = append(respParts, "unknown canvas action: "+op.Action)
		}
	}

	if res.CanvasPNG == nil && createdID != "" {
		fmt.Println("[canvas] auto-rendering", createdID)
		png, filename, err := cm.RenderCanvasToPNG(createdID)
		if err != nil {
			fmt.Println("[canvas] auto-render error:", err)
			respParts = append(respParts, "canvas error: "+err.Error())
		} else {
			fmt.Println("[canvas] auto-render ok, size:", len(png), "bytes")
			res.CanvasPNG = png
			res.CanvasFilename = filename
		}
	}

	res.CanvasResp = strings.Join(respParts, "\n")
	return res
}

func hasRenders(actionData actions.ActionData) bool {
	if actionData.Chart != nil || actionData.Drawing != nil ||
		actionData.PixelArt != nil || actionData.Benchmark != nil ||
		actionData.Plot != nil || actionData.Stats != nil ||
		actionData.Solver != nil || actionData.Latex != nil ||
		actionData.UnitConvert != nil || actionData.NumberTheory != nil ||
		actionData.Matrix != nil {
		return true
	}
	for _, t := range actionData.Tasks {
		switch t.Action {
		case "generate_chart", "generate_drawing", "generate_pixel_art",
			"run_benchmark", "plot_function", "calculate_stats",
			"solve_equation", "render_latex", "convert_unit",
			"number_theory", "matrix_operation":
			return true
		}
	}
	return false
}

func executeRenders(actionData actions.ActionData, channelID string) RenderResult {
	var res RenderResult
	var mu sync.Mutex

	// --- resolve configs sequentially (fast math/lookups) ---

	chartCfg := actionData.Chart
	if chartCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_chart" && t.Chart != nil {
				chartCfg = t.Chart
				break
			}
		}
	}

	drawingCfg := actionData.Drawing
	if drawingCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_drawing" && t.Drawing != nil {
				drawingCfg = t.Drawing
				break
			}
		}
	}

	pixelArtCfg := actionData.PixelArt
	if pixelArtCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "generate_pixel_art" && t.PixelArt != nil {
				pixelArtCfg = t.PixelArt
				break
			}
		}
	}

	benchmarkCfg := actionData.Benchmark
	if benchmarkCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "run_benchmark" && t.Benchmark != nil {
				benchmarkCfg = t.Benchmark
				break
			}
		}
	}
	if benchmarkCfg != nil && chartCfg == nil {
		results, err := skills.RunBenchmark(*benchmarkCfg)
		if err != nil {
			fmt.Println("[benchmark] error:", err)
		} else {
			variable := benchmarkCfg.Variable
			if variable == "" {
				variable = "x"
			}
			generated := skills.BenchmarkToChart(results, variable)
			chartCfg = &generated
		}
	}

	plotCfg := actionData.Plot
	if plotCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "plot_function" && t.Plot != nil {
				plotCfg = t.Plot
				break
			}
		}
	}
	if plotCfg != nil && chartCfg == nil {
		generated, err := skills.PlotToChart(*plotCfg)
		if err != nil {
			fmt.Println("[plot] error:", err)
		} else {
			chartCfg = &generated
		}
	}

	statsCfg := actionData.Stats
	if statsCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "calculate_stats" && t.Stats != nil {
				statsCfg = t.Stats
				break
			}
		}
	}
	if statsCfg != nil {
		statsResult, statsChart, err := skills.CalculateStats(*statsCfg)
		if err != nil {
			fmt.Println("[stats] error:", err)
		} else {
			res.PrependText += skills.StatsResultToText(statsResult, statsCfg.Label) + "\n"
			if chartCfg == nil {
				chartCfg = &statsChart
			}
		}
	}

	solverCfg := actionData.Solver
	if solverCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "solve_equation" && t.Solver != nil {
				solverCfg = t.Solver
				break
			}
		}
	}
	if solverCfg != nil {
		solverResult, err := skills.SolveEquation(*solverCfg)
		if err != nil {
			fmt.Println("[solver] error:", err)
		} else {
			res.PrependText += skills.SolverResultToText(solverResult, solverCfg.Equation, solverCfg.Variable) + "\n"
			if chartCfg == nil {
				chartCfg = &solverResult.Chart
			}
		}
	}

	unitCfg := actionData.UnitConvert
	if unitCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "convert_unit" && t.UnitConvert != nil {
				unitCfg = t.UnitConvert
				break
			}
		}
	}
	if unitCfg != nil {
		r, err := skills.ConvertUnit(*unitCfg)
		if err != nil {
			res.PrependText += "unit convert error: " + err.Error() + "\n"
		} else {
			res.PrependText += r.Formula + "\n"
		}
	}

	ntCfg := actionData.NumberTheory
	if ntCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "number_theory" && t.NumberTheory != nil {
				ntCfg = t.NumberTheory
				break
			}
		}
	}
	if ntCfg != nil {
		r, err := skills.RunNumberTheory(*ntCfg)
		if err != nil {
			res.PrependText += "number theory error: " + err.Error() + "\n"
		} else {
			res.PrependText += r.Output + "\n"
		}
	}

	latexCfg := actionData.Latex
	if latexCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "render_latex" && t.Latex != nil {
				latexCfg = t.Latex
				break
			}
		}
	}

	matrixCfg := actionData.Matrix
	if matrixCfg == nil {
		for _, t := range actionData.Tasks {
			if t.Action == "matrix_operation" && t.Matrix != nil {
				matrixCfg = t.Matrix
				break
			}
		}
	}
	if matrixCfg != nil {
		r, err := skills.RunMatrix(*matrixCfg)
		if err != nil {
			res.PrependText += "matrix error: " + err.Error() + "\n"
		} else {
			res.PrependText += r.Output + "\n"
			if latexCfg == nil && len(r.LatexExprs) > 0 {
				latexCfg = &skills.LatexConfig{
					Expressions: r.LatexExprs,
					DarkMode:    true,
					FontSize:    1.2,
				}
			}
		}
	}

	// --- parallel renders ---
	var wg sync.WaitGroup

	if chartCfg != nil {
		wg.Add(1)
		cfg := chartCfg
		go func() {
			defer wg.Done()
			png, err := skills.RenderChart(*cfg)
			if err != nil {
				fmt.Println("[chart] render error:", err)
				return
			}
			mu.Lock()
			res.ChartPNG = png
			res.ChartCfg = cfg
			mu.Unlock()
		}()
	}

	if drawingCfg != nil {
		wg.Add(1)
		cfg := drawingCfg
		go func() {
			defer wg.Done()
			png, err := skills.RenderDrawing(*cfg)
			if err != nil {
				fmt.Println("[drawing] render error:", err)
				return
			}
			mu.Lock()
			res.DrawingPNG = png
			mu.Unlock()
			lastDrawingMu.Lock()
			lastDrawing[channelID] = cfg
			lastDrawingMu.Unlock()
		}()
	}

	if pixelArtCfg != nil {
		wg.Add(1)
		cfg := pixelArtCfg
		go func() {
			defer wg.Done()
			png, err := skills.RenderPixelArt(*cfg)
			if err != nil {
				fmt.Println("[pixelart] render error:", err)
				return
			}
			mu.Lock()
			res.PixelArtPNG = png
			mu.Unlock()
		}()
	}

	if latexCfg != nil {
		wg.Add(1)
		cfg := latexCfg
		go func() {
			defer wg.Done()
			png, err := skills.RenderLatex(*cfg)
			if err != nil {
				fmt.Println("[latex] render error:", err)
				return
			}
			mu.Lock()
			res.LatexPNG = png
			mu.Unlock()
		}()
	}

	wg.Wait()
	return res
}

func QueueRenders(actionData actions.ActionData, s *discordgo.Session, m *discordgo.MessageCreate) {
	channelID := m.ChannelID
	ref := m.Reference()
	taskqueue.BotQueue.Add(taskqueue.Task{
		Name:    "render_skills",
		GuildID: m.GuildID,
		Execute: func() error {
			result := executeRenders(actionData, channelID)

			var files []*discordgo.File
			if result.ChartPNG != nil {
				title := "chart"
				if result.ChartCfg != nil && result.ChartCfg.Title != "" {
					title = strings.ReplaceAll(result.ChartCfg.Title, " ", "_")
				}
				files = append(files, &discordgo.File{Name: title + ".png", Reader: bytes.NewReader(result.ChartPNG)})
			}
			if result.DrawingPNG != nil {
				files = append(files, &discordgo.File{Name: "drawing.png", Reader: bytes.NewReader(result.DrawingPNG)})
			}
			if result.PixelArtPNG != nil {
				files = append(files, &discordgo.File{Name: "pixel_art.png", Reader: bytes.NewReader(result.PixelArtPNG)})
			}
			if result.LatexPNG != nil {
				files = append(files, &discordgo.File{Name: "latex.png", Reader: bytes.NewReader(result.LatexPNG)})
			}

			if len(files) == 0 && result.PrependText == "" {
				return nil
			}

			content := strings.TrimRight(result.PrependText, "\n")
			if len(content) > 2000 {
				content = content[:2000] + "\n…"
			}

			_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
				Content:   content,
				Files:     files,
				Reference: ref,
				AllowedMentions: &discordgo.MessageAllowedMentions{
					Parse: []discordgo.AllowedMentionType{
						discordgo.AllowedMentionTypeUsers,
						discordgo.AllowedMentionTypeRoles,
					},
				},
			})
			return err
		},
	})
}

func MakeExecute(task actions.Action, s *discordgo.Session, m *discordgo.MessageCreate) func() error {
	return func() error {
		return actions.HandleActions(task, s, m)
	}
}

// Command preview renders the wheel over a sample desktop to PNG files so the
// look can be checked without Windows.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"raha/conv"
	"time"

	"raha/ui"
)

func desktop(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)/float64(w), float64(y)/float64(h)
			// warm orange top-left fading to pink and purple bottom-right
			r := 245 - 40*fy*fx
			g := 150 - 70*fy - 30*fx
			b := 60 + 150*fy*fx + 40*fy
			if x > w/2-260 && x < w/2-60 && y > h/2-300 && y < h/2-40 { // a white window
				r, g, b = 250, 250, 250
			}
			img.Set(x, y, color.RGBA{uint8(clamp(r)), uint8(clamp(g)), uint8(clamp(b)), 255})
		}
	}
	return img
}

func clamp(v float64) float64 { return math.Max(0, math.Min(255, v)) }

func main() {
	fonts := ui.LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	scale := 1.5
	size := int(ui.LogicalSize * scale)
	formats := []ui.Item{}
	for _, f := range []string{"JPG", "WEBP", "AVIF", "TIFF", "GIF", "BMP", "ICO", "PDF"} {
		formats = append(formats, ui.Item{ID: f, Label: f, Enabled: true})
	}
	tools := []ui.Item{
		{ID: "compress", Label: "COMPRESS", Icon: "compress", Enabled: true}, {ID: "metadata", Label: "METADATA", Icon: "metadata", Enabled: true},
		{ID: "rotate", Label: "ROTATE", Icon: "rotate", Enabled: true}, {ID: "resize", Label: "RESIZE 50%", Icon: "resize", Enabled: true},
		{ID: "crop", Label: "CROP", Icon: "crop", Enabled: true}, {ID: "edit", Label: "EDIT", Icon: "edit", Enabled: true},
		{ID: "redact", Label: "REDACT", Icon: "redact", Enabled: true}, {ID: "annotate", Label: "ANNOTATE", Icon: "annotate", Enabled: true},
	}
	var thumbSrc image.Image
	if f, err := os.Open("/mnt/user-data/uploads/b787dde0-image.png"); err == nil {
		full, _ := png.Decode(f)
		f.Close()
		thumbSrc = full.(interface {
			SubImage(image.Rectangle) image.Image
		}).SubImage(image.Rect(322, 218, 448, 328))
	}
	type shot struct {
		name  string
		dark  bool
		items []ui.Item
		hover int
		phase string
		ms    int
	}
	toUI := func(in []conv.Item) []ui.Item {
		var out []ui.Item
		for _, it := range in {
			out = append(out, ui.Item{ID: it.ID, Label: it.Label, Icon: it.Icon, Enabled: it.Enabled})
		}
		return out
	}
	shots := []shot{
		{"white", false, toUI(conv.FormatItems("a.mp4")), 0, "white", 700},
		{"video_tools", false, toUI(conv.ToolItems("a.mp4")), 6, "white", 700},
		{"audio_tools", false, toUI(conv.ToolItems("a.wav")), 2, "white", 700},
		{"progress", false, toUI(conv.FormatItems("a.mp4")), 1, "work", 700},
	}
	_ = []shot{
		{"white", false, formats, -1, "white", 700},
		{"formats_hover", false, formats, 1, "", 700},
		{"tools_hover", false, tools, 3, "", 700},
		{"dark_desktop", true, formats, 2, "", 700},
	}
	bgFull := desktop(1600, 1000)
	pad := 20
	cols := 2
	grid := image.NewRGBA(image.Rect(0, 0, cols*(size+pad)+pad, 2*(size+pad)+pad))
	draw.Draw(grid, grid.Bounds(), &image.Uniform{color.RGBA{30, 30, 30, 255}}, image.Point{}, draw.Src)
	for k, sh := range shots {
		ox, oy := 520+k*20, 260+k*10
		region := image.NewRGBA(image.Rect(0, 0, size, size))
		draw.Draw(region, region.Bounds(), bgFull, image.Pt(ox, oy), draw.Src)
		if sh.phase == "white" || sh.phase == "work" {
			draw.Draw(region, region.Bounds(), &image.Uniform{color.RGBA{244, 244, 244, 255}}, image.Point{}, draw.Src)
			// a file icon left of the wheel, partly under the strip (like the reference)
			draw.Draw(region, image.Rect(10, 180, 110, 290), &image.Uniform{color.RGBA{190, 190, 190, 255}}, image.Point{}, draw.Src)
			draw.Draw(region, image.Rect(22, 192, 98, 278), &image.Uniform{color.RGBA{232, 232, 232, 255}}, image.Point{}, draw.Src)
			draw.Draw(region, image.Rect(5, 300, 100, 330), &image.Uniform{color.RGBA{40, 110, 220, 255}}, image.Point{}, draw.Src)
		}
		if sh.dark {
			for i := 0; i < len(region.Pix); i += 4 {
				region.Pix[i], region.Pix[i+1], region.Pix[i+2] = region.Pix[i]/9, region.Pix[i+1]/9, region.Pix[i+2]/9
			}
			for y := 40; y < 120; y++ {
				for x := 20; x < 100; x++ {
					region.Set(x, y, color.RGBA{240, 140, 30, 255})
				}
			}
		}
		t0 := time.Now()
		frost := ui.FrostBackground(region, 14*scale, 1.25)
		ft := time.Since(t0)
		start := time.Unix(1000, 0)
		wh := ui.New(scale, size, size, float64(size)/2, float64(size)/2, frost, false, fonts, start)
		wh.Sharp = ui.FrostBackground(region, 3*scale, 1.1)
		wh.Reveal(start)
		wh.SetItems(sh.items, true, start)
		wh.SetFileInfo(ui.MakeThumb(thumbSrc, wh.ThumbPixels()), "PNG", "2.2 MB", "")
		if sh.hover >= 0 {
			a := -math.Pi/2 + float64(sh.hover)*2*math.Pi/float64(len(sh.items))
			wh.SetPointer(float64(size)/2+math.Cos(a)*105*scale, float64(size)/2+math.Sin(a)*105*scale)
		}
		var img *image.RGBA
		now := start
		step := 16 * time.Millisecond
		end := start.Add(time.Duration(sh.ms) * time.Millisecond)
		var total time.Duration
		var worst time.Duration
		frames := 0
		for now.Before(end) {
			now = now.Add(step)
			if (sh.phase == "work" || sh.phase == "done") && now.Sub(start) > 350*time.Millisecond && wh.Phase() == ui.PhaseOpen {
				wh.StartWorking(sh.hover, now)
				wh.SetProgress(0.37)
			}
			if sh.phase == "done" && now.Sub(start) > 420*time.Millisecond && wh.Phase() == ui.PhaseWorking {
				wh.Finish(true, now)
			}
			t1 := time.Now()
			img = wh.Render(now)
			d := time.Since(t1)
			if frames > 0 {
				total += d
				if d > worst {
					worst = d
				}
			}
			frames++
		}
		fmt.Printf("%s: frost %v, avg frame %v, worst %v (%d frames)\n", sh.name, ft, total/time.Duration(frames-1), worst, frames)
		comp := image.NewRGBA(region.Bounds())
		draw.Draw(comp, comp.Bounds(), region, image.Point{}, draw.Src)
		draw.Draw(comp, comp.Bounds(), img, image.Point{}, draw.Over)
		x := pad + (k%cols)*(size+pad)
		y := pad + (k/cols)*(size+pad)
		draw.Draw(grid, image.Rect(x, y, x+size, y+size), comp, image.Point{}, draw.Src)
		f, _ := os.Create("/home/claude/preview_" + sh.name + ".png")
		png.Encode(f, comp)
		f.Close()
	}
	f, _ := os.Create("/home/claude/preview_grid.png")
	png.Encode(f, grid)
	f.Close()
}

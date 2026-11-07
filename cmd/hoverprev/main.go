// Command hoverprev renders the hover lift animation with the preview
// tools: an 8-frame strip from rest to fully lifted for Light and Dark,
// and GIF frames of the pointer moving across several segments.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	"raha/ui"
)

const amiri = "/usr/share/fonts/opentype/fonts-hosny-amiri/"

var fonts *ui.Fonts

func desktop(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)/float64(w), float64(y)/float64(h)
			r := 245 - 40*fy*fx
			g := 150 - 70*fy - 30*fx
			b := 60 + 150*fy*fx + 40*fy
			if x > w/2-260 && x < w/2-60 && y > h/2-300 && y < h/2-40 {
				r, g, b = 250, 250, 250
			}
			c := func(v float64) uint8 { return uint8(math.Max(0, math.Min(255, v))) }
			img.Set(x, y, color.RGBA{c(r), c(g), c(b), 255})
		}
	}
	return img
}

func darkify(region *image.RGBA) {
	for i := 0; i < len(region.Pix); i += 4 {
		region.Pix[i], region.Pix[i+1], region.Pix[i+2] = region.Pix[i]/9, region.Pix[i+1]/9, region.Pix[i+2]/9
	}
	for y := 40; y < 120; y++ {
		for x := 20; x < 100; x++ {
			region.Set(x, y, color.RGBA{240, 140, 30, 255})
		}
	}
}

func formats() []ui.Item {
	var items []ui.Item
	for _, f := range []string{"JPG", "WEBP", "AVIF", "TIFF", "GIF", "BMP", "ICO", "PDF"} {
		items = append(items, ui.Item{ID: f, Label: f, Enabled: true})
	}
	return items
}

// newWheel builds a wheel over a synthetic desktop.
func newWheel(scale float64, dark bool) (*ui.Wheel, *image.RGBA) {
	size := int(ui.LogicalSize * scale)
	bgFull := desktop(1600, 1000)
	region := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(region, region.Bounds(), bgFull, image.Pt(480, 300), draw.Src)
	if dark {
		darkify(region)
	}
	frost := ui.FrostBackground(region, 14*scale, 1.25)
	wh := ui.New(scale, size, size, float64(size)/2, float64(size)/2, frost, dark, fonts, time.Unix(1000, 0))
	wh.Sharp = ui.FrostBackground(region, 3*scale, 1.1)
	wh.Reveal(time.Unix(1000, 0))
	wh.SetItems(formats(), true, time.Unix(1000, 0))
	wh.SetFileInfo(nil, "PNG", "2.2 MB", "")
	return wh, region
}

func pointerAt(wh *ui.Wheel, seg int) {
	n := float64(len(wh.Items))
	a := -math.Pi/2 + float64(seg)*2*math.Pi/n
	wh.SetPointer(wh.CX+math.Cos(a)*105*wh.Scale, wh.CY+math.Sin(a)*105*wh.Scale)
}

func main() {
	fonts = ui.LoadFonts([]string{amiri + "Amiri-Bold.ttf"}, []string{amiri + "Amiri-Regular.ttf"})
	out := os.Getenv("HOVERPREV_OUT")
	if out == "" {
		out = "out"
	}
	os.MkdirAll(out, 0o755)
	scale := 1.25
	start := time.Unix(1000, 0)

	// --- 8-frame strip, Light and Dark: rest -> fully lifted -------------
	steps := []int{0, 25, 50, 75, 105, 140, 185, 265}
	size := int(ui.LogicalSize * scale)
	pad := 12
	for _, dark := range []bool{false, true} {
		wh, region := newWheel(scale, dark)
		now := start
		end := now.Add(600 * time.Millisecond)
		for now.Before(end) { // warm up: entrance done, nothing hovered
			now = now.Add(8333 * time.Microsecond)
			wh.Render(now)
		}
		pointerAt(wh, 1) // WEBP
		frames := []*image.RGBA{}
		idx := 0
		for idx < len(steps) {
			now = now.Add(8333 * time.Microsecond)
			img := wh.Render(now)
			elapsed := int(now.Sub(start).Milliseconds() - 600)
			if elapsed >= steps[idx] {
				comp := image.NewRGBA(region.Bounds())
				draw.Draw(comp, comp.Bounds(), region, image.Point{}, draw.Src)
				draw.Draw(comp, comp.Bounds(), img, image.Point{}, draw.Over)
				frames = append(frames, comp)
				idx++
			}
		}
		name := "light"
		if dark {
			name = "dark"
		}
		grid := image.NewRGBA(image.Rect(0, 0, len(steps)*(size+pad)+pad, size+2*pad))
		draw.Draw(grid, grid.Bounds(), &image.Uniform{color.RGBA{30, 30, 30, 255}}, image.Point{}, draw.Src)
		for k, fr := range frames {
			x := pad + k*(size+pad)
			draw.Draw(grid, image.Rect(x, pad, x+size, pad+size), fr, image.Point{}, draw.Src)
		}
		f, err := os.Create(filepath.Join(out, "hover-strip-"+name+".png"))
		if err != nil {
			panic(err)
		}
		png.Encode(f, grid)
		f.Close()
		fmt.Println("wrote hover-strip-" + name + ".png")
	}

	// --- GIF frames: pointer moving across several segments ---------------
	wh, region := newWheel(scale, false)
	now := start
	end := now.Add(600 * time.Millisecond)
	for now.Before(end) {
		now = now.Add(16 * time.Millisecond)
		wh.Render(now)
	}
	os.MkdirAll(filepath.Join(out, "gif"), 0o755)
	seq := []int{1, 2, 3, 4, 5, 4, 3, 2} // sweep right, then back
	frameNo := 0
	last := -2
	for elapsed := 0; elapsed < 2640; elapsed += 16 {
		if k := elapsed / 330; k != last && k < len(seq) {
			last = k
			pointerAt(wh, seq[last])
		}
		now = now.Add(16 * time.Millisecond)
		img := wh.Render(now)
		if (frameNo % 2) == 0 { // render 60 fps, keep every 2nd -> 30 fps GIF
			comp := image.NewRGBA(region.Bounds())
			draw.Draw(comp, comp.Bounds(), region, image.Point{}, draw.Src)
			draw.Draw(comp, comp.Bounds(), img, image.Point{}, draw.Over)
			f, err := os.Create(filepath.Join(out, "gif", fmt.Sprintf("f%04d.png", frameNo)))
			if err != nil {
				panic(err)
			}
			png.Encode(f, comp)
			f.Close()
		}
		frameNo++
	}
	fmt.Printf("wrote %d gif frames (%d kept)\n", frameNo, (frameNo+1)/2)
}

// Command themeprev renders the wheel in the light and dark themes (formats
// and tools, one hovered) over a photo-like background, to /tmp/theme.png.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"time"

	"raha/conv"
	"raha/ui"
)

func bg(n int, k int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			fx, fy := float64(x)/float64(n), float64(y)/float64(n)
			r, g, b := 245-40*fy, 170-90*fy, 60+30*fx
			if k%2 == 1 {
				r, g, b = 40+30*fx, 60+20*fy, 90+60*fy
			}
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	return img
}

func main() {
	fonts := ui.LoadFonts([]string{"/usr/share/fonts/opentype/inter/Inter-Bold.otf"}, []string{"/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"})
	toUI := func(in []conv.Item) (out []ui.Item) {
		for _, it := range in {
			out = append(out, ui.Item{ID: it.ID, Label: it.Label, Icon: it.Icon, Enabled: it.Enabled})
		}
		return
	}
	sets := [][]ui.Item{toUI(conv.ToolItems("a.mp4")), toUI(conv.ToolItems("a.png")), toUI(conv.FormatItems("a.mp4")), toUI(conv.ToolItems("a.wav"))}
	n := 540
	sheet := image.NewRGBA(image.Rect(0, 0, n*len(sets), n*2))
	for row, dark := range []bool{false, true} {
		for k, items := range sets {
			b := bg(n, k)
			t := time.Unix(1000, 0)
			w := ui.New(1.5, n, n, 270, 270, ui.FrostBackground(b, 21, 1.25), dark, fonts, t)
			w.Sharp = ui.FrostBackground(b, 4.5, 1.1)
			w.Reveal(t)
			w.SetItems(items, false, t)
			w.SetFileInfo(nil, "MP4", "766.5 MB", "")
			if k == 0 {
				a := -math.Pi/2 + 2*2*math.Pi/float64(len(items))
				w.SetPointer(270+math.Cos(a)*155, 270+math.Sin(a)*155)
			}
			var img *image.RGBA
			for i := 1; i <= 60; i++ {
				img = w.Render(t.Add(time.Duration(i) * 16 * time.Millisecond))
			}
			out := image.NewRGBA(b.Rect)
			draw.Draw(out, out.Rect, b, image.Point{}, draw.Src)
			draw.Draw(out, out.Rect, img, image.Point{}, draw.Over)
			draw.Draw(sheet, image.Rect(n*k, n*row, n*k+n, n*row+n), out, image.Point{}, draw.Src)
		}
	}
	o, _ := os.Create("/tmp/theme.png")
	png.Encode(o, sheet)
	o.Close()
}

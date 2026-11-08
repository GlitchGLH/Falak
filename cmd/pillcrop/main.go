// pillcrop renders the wheel's center capsule (dark and light themes) over
// a light and a dark desktop, cropped at 100% and 200% zoom, for checking
// the rim shine. Also writes the crops separately for a pixel-diff proof.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"time"

	xdraw "golang.org/x/image/draw"

	"raha/conv"
	"raha/ui"
)

func bg(n int, dark bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			fx, fy := float64(x)/float64(n), float64(y)/float64(n)
			r, g, b := 245-40*fy, 170-90*fy, 60+30*fx
			if dark {
				r, g, b = 40+30*fx, 60+20*fy, 90+60*fy
			}
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	return img
}

func wheel(dark bool, bgk *image.RGBA, fonts *ui.Fonts) *image.RGBA {
	n := 540
	t := time.Unix(1000, 0)
	w := ui.New(1.5, n, n, 270, 270, ui.FrostBackground(bgk, 21, 1.25), dark, fonts, t)
	w.Sharp = ui.FrostBackground(bgk, 4.5, 1.1)
	w.Reveal(t)
	var items []ui.Item
	for _, it := range conv.FormatItems("a.mp4") {
		items = append(items, ui.Item{ID: it.ID, Label: it.Label, Icon: it.Icon, Enabled: it.Enabled})
	}
	w.SetItems(items, false, t)
	w.SetFileInfo(nil, "MP4", "766.5 MB", "")
	var img *image.RGBA
	for i := 1; i <= 60; i++ {
		img = w.Render(t.Add(time.Duration(i) * 16 * time.Millisecond))
	}
	return img
}

func crop(src *image.RGBA, scale int) image.Image {
	const cw, ch = 220, 130 // around the capsule at the wheel's center
	r := image.Rect(540/2-cw/2, 540/2-ch/2, 540/2+cw/2, 540/2+ch/2)
	sub := src.SubImage(r).(*image.RGBA)
	if scale == 1 {
		return sub
	}
	out := image.NewRGBA(image.Rect(0, 0, cw*scale, ch*scale))
	xdraw.NearestNeighbor.Scale(out, out.Bounds(), sub, sub.Bounds(), draw.Src, nil)
	return out
}

func main() {
	out := flag.String("o", "pillprev.png", "output sheet PNG")
	split := flag.String("split", "", "also write <prefix>-<theme>-<bg>.png crops")
	bold := flag.String("bold", "/usr/share/fonts/opentype/inter/Inter-Bold.otf", "bold font")
	semi := flag.String("semi", "/usr/share/fonts/opentype/inter/Inter-SemiBold.otf", "regular font")
	flag.Parse()
	fonts := ui.LoadFonts([]string{*bold}, []string{*semi})
	gap := 16
	cellW, cellH := 440, 260
	sheet := image.NewRGBA(image.Rect(0, 0, cellW*4+gap*5, cellH*2+gap*3))
	draw.Draw(sheet, sheet.Bounds(), &image.Uniform{color.NRGBA{232, 232, 236, 255}}, image.Point{}, draw.Src)
	// rows: dark theme, light theme; columns: 100% over light bg,
	// 100% over dark bg, 200% over light bg, 200% over dark bg
	type placed struct {
		c   image.Image
		row int
	}
	var cells []placed
	for _, dark := range []bool{true, false} {
		row := 0
		if dark {
			row = 1
		}
		for _, bgDark := range []bool{false, true} {
			img := wheel(dark, bg(540, bgDark), fonts)
			c100 := crop(img, 1)
			c200 := crop(img, 2)
			cells = append(cells, placed{c100, row}, placed{c200, row})
			name := "lightbg"
			if bgDark {
				name = "darkbg"
			}
			if *split != "" {
				theme := "dark"
				if !dark {
					theme = "light"
				}
				f, err := os.Create(*split + "-" + theme + "-" + name + ".png")
				if err != nil {
					panic(err)
				}
				if err := png.Encode(f, c100); err != nil {
					panic(err)
				}
				f.Close()
			}
		}
	}
	for i, p := range cells {
		x := gap + (i%4)*(cellW+gap)
		y := gap + p.row*(cellH+gap)
		bb := p.c.Bounds()
		draw.Draw(sheet, image.Rect(x, y+(cellH-bb.Dy())/2, x+bb.Dx(), y+(cellH+bb.Dy())/2), p.c, image.Point{}, draw.Src)
	}
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		panic(err)
	}
}

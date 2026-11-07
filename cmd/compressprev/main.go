// compressprev renders the compress dialog as two grids (100% and 150%),
// each light/dark x English/Arabic over a synthetic desktop, with 1 px
// debug outlines: red on each option card, green around the measured text
// ink — the padding above and below the green box must look equal.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"github.com/fogleman/gg"

	"raha/ui"
)

// desktop paints a synthetic mid-luminance desktop so the glass has
// something real to blur.
func desktop(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / float64(w+h)
			r := uint8(96 + t*(225-96))
			g := uint8(150 + t*(210-150))
			b := uint8(210 + t*(180-210))
			i := (y*w + x) * 4
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, 255
		}
	}
	dc := gg.NewContextForRGBA(img)
	for _, c := range [][4]float64{{0.2, 0.25, 0.09, 0.35}, {0.75, 0.6, 0.12, 0.30}, {0.5, 0.85, 0.07, 0.25}} {
		dc.DrawCircle(c[0]*float64(w), c[1]*float64(h), c[2]*float64(w))
		dc.SetColor(color.NRGBA{255, 255, 255, uint8(255 * c[3])})
		dc.Fill()
	}
	return img
}

func main() {
	out := flag.String("o", "compressprev.png", "output PNG")
	bold := flag.String("bold", "", "bold font file")
	semi := flag.String("semi", "", "regular font file")
	debugOn := flag.Bool("debug", true, "draw the 1 px debug outlines")
	flag.Parse()
	if *bold == "" || *semi == "" {
		os.Stderr.WriteString("compressprev needs -bold and -semi font files\n")
		os.Exit(2)
	}
	fonts := ui.LoadFonts([]string{*bold}, []string{*semi})
	en := ui.CompressLabels{
		Title: "Compress", File: "lecture.mp4", SizeLine: "Original size:",
		Preset: "Compression level", Balanced: "Balanced", Strong: "Strong",
		BalancedHint: "Smaller with less quality loss.", StrongHint: "Smallest possible size, with clearer quality loss.",
		Target: "Compress to a set size", TargetLabel: "Target size (MB)", Start: "Compress",
	}
	ar := ui.CompressLabels{
		Title: "ضغط", File: "lecture.mp4", SizeLine: "الحجم الأصلي:",
		Preset: "مستوى الضغط", Balanced: "متوازن", Strong: "قوي",
		BalancedHint: "حجم أصغر مع فقدان أقل للجودة.", StrongHint: "أصغر حجم ممكن، مع فقدان أوضح للجودة.",
		Target: "الضغط إلى حجم محدد", TargetLabel: "الحجم المطلوب (ميغابايت)", Start: "ضغط",
	}
	mk := func(scale float64, dark, rtl bool, l ui.CompressLabels, f func(*ui.CompressDialog)) *image.RGBA {
		w := int(float64(384+2*22)*scale) + 8
		h := int(float64(340+2*22)*scale) + 8
		d := ui.NewCompressDialog(scale, desktop(w, h), fonts, dark, rtl, l, "766.5 MB")
		d.DebugOutline = *debugOn
		if f != nil {
			f(d)
		}
		return d.Render()
	}
	type cell struct {
		scale     float64
		dark, rtl bool
		l         ui.CompressLabels
		f         func(*ui.CompressDialog)
	}
	cells := []cell{
		{1, false, false, en, nil},
		{1, false, true, ar, func(d *ui.CompressDialog) { d.Preset = 1 }},
		{1, true, false, en, func(d *ui.CompressDialog) { d.Target = true; d.TargetMB = "48" }},
		{1, true, true, ar, func(d *ui.CompressDialog) {
			d.Error = "الحجم المطلوب صغير جدًا لهذا الفيديو."
		}},
		{1.5, false, false, en, nil},
		{1.5, false, true, ar, func(d *ui.CompressDialog) { d.Preset = 1 }},
		{1.5, true, false, en, func(d *ui.CompressDialog) { d.Target = true; d.TargetMB = "48" }},
		{1.5, true, true, ar, func(d *ui.CompressDialog) {
			d.Error = "الحجم المطلوب صغير جدًا لهذا الفيديو."
		}},
	}
	var imgs []image.Image
	for _, c := range cells {
		imgs = append(imgs, mk(c.scale, c.dark, c.rtl, c.l, c.f))
	}
	// two rows: 100% cells, then 150% cells; widest cell in each column
	gap := 16
	w1, h1 := imgs[0].Bounds().Dx(), imgs[0].Bounds().Dy()
	w2, h2 := imgs[4].Bounds().Dx(), imgs[4].Bounds().Dy()
	W := w1*4 + w2*0 + gap*5
	H := h1 + h2 + gap*3
	grid := image.NewRGBA(image.Rect(0, 0, W, H))
	draw.Draw(grid, grid.Bounds(), &image.Uniform{color.NRGBA{232, 232, 236, 255}}, image.Point{}, draw.Src)
	for i, im := range imgs {
		var x, y, cw, ch int
		if i < 4 {
			x, y, cw, ch = gap+(i%4)*(w1+gap), gap, w1, h1
		} else {
			j := i - 4
			x, y, cw, ch = gap+(j%4)*(w2+gap), gap*2+h1, w2, h2
		}
		b := im.Bounds()
		draw.Draw(grid, image.Rect(x, y, x+b.Dx(), y+b.Dy()), im, image.Point{}, draw.Src)
		_ = cw
		_ = ch
	}
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, grid); err != nil {
		panic(err)
	}
}

// iconprev renders every tool icon at 100% and 150% on light and dark,
// to check stroke weight and crispness: go run ./cmd/iconprev out.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/fogleman/gg"
	"raha/ui"
)

var ids = []string{"compress", "metadata", "rotate", "resize", "crop", "edit",
	"redact", "mute", "snapshot", "normalize", "visualize", "channels",
	"extract", "trim", "speed", "split", "merge", "blackbg", "bleep", "addbg", "annotate"}

func main() {
	out := "iconprev.png"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	const cell, cols = 92, 7
	rows := (len(ids) + cols - 1) / cols
	W, H := cols*cell+24, 2*(rows*cell)+36
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	dc := gg.NewContextForRGBA(img)
	dc.SetColor(color.NRGBA{247, 247, 248, 255})
	dc.DrawRectangle(0, 0, float64(W), float64(H)/2)
	dc.Fill()
	dc.SetColor(color.NRGBA{40, 40, 44, 255})
	dc.DrawRectangle(0, float64(H)/2, float64(W), float64(H)/2)
	dc.Fill()
	for half := 0; half < 2; half++ {
		ink := color.NRGBA{58, 58, 64, 255}
		if half == 1 {
			ink = color.NRGBA{236, 236, 240, 255}
		}
		for i, id := range ids {
			x := 24 + (i%cols)*cell + 26
			y := 18 + half*(rows*cell+18) + (i/cols)*cell + cell/2
			ui.DrawIcon(dc, id, float64(x-14), float64(y), 24, ink) // 100%
			ui.DrawIcon(dc, id, float64(x+26), float64(y), 36, ink) // 150%
		}
	}
	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

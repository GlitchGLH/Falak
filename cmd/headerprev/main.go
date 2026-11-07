// Command headerprev renders the tray menu header (Arabic/English, dark/
// light) with the app icon and the credit line to ../previews/task3-
// header.png. Run it from the repository root.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	xdraw "golang.org/x/image/draw"

	"raha/ui"
)

const amiri = "/usr/share/fonts/opentype/fonts-hosny-amiri/"

func main() {
	strong := ui.LoadFonts([]string{amiri + "Amiri-Bold.ttf"}, []string{amiri + "Amiri-Regular.ttf"})
	plain := ui.LoadFonts([]string{amiri + "Amiri-Bold.ttf"}, []string{amiri + "Amiri-Regular.ttf"})
	logo := menuLogo()
	var tiles []*image.RGBA
	for _, ar := range []bool{false, true} {
		for _, dark := range []bool{true, false} {
			T := func(e, a string) string {
				if ar {
					return a
				}
				return e
			}
			m := &ui.Menu{Scale: 1.5, Dark: dark, RTL: ar, Strong: strong, Plain: plain,
				Title: T("Falak", "فلك"), Sub: "1.0 · By UncleGlitch",
				Logo: logo}
			img := m.Render()
			tile := image.NewRGBA(image.Rect(0, 0, img.Rect.Dx(), int(84*1.5)))
			draw.Draw(tile, tile.Rect, img, image.Point{}, draw.Over)
			tiles = append(tiles, tile)
		}
	}
	w, h := tiles[0].Rect.Dx(), tiles[0].Rect.Dy()
	sheet := image.NewRGBA(image.Rect(0, 0, w*2+12, h*2+12))
	draw.Draw(sheet, sheet.Rect, &image.Uniform{color.RGBA{120, 130, 150, 255}}, image.Point{}, draw.Src)
	for i, t := range tiles {
		x, y := (i%2)*(w+12), (i/2)*(h+12)
		draw.Draw(sheet, image.Rect(x, y, x+w, y+h), t, image.Point{}, draw.Over)
	}
	if err := os.MkdirAll("../previews", 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create("../previews/task3-header.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, sheet); err != nil {
		panic(err)
	}
}

// menuLogo decodes the app icon and scales it to the header's logo size
// (26 logical px at the preview's scale of 1.5).
func menuLogo() *image.RGBA {
	f, err := os.Open("res/icon-1024.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	const px = 39
	dst := image.NewRGBA(image.Rect(0, 0, px, px))
	xdraw.CatmullRom.Scale(dst, dst.Rect, src, src.Bounds(), xdraw.Over, nil)
	return dst
}

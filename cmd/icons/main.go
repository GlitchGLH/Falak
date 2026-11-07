// Command icons draws the candidate Falak app icons (flat, iOS-style).
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/opentype"
)

const S = 1024.0

// squircle builds the continuous-corner app tile (superellipse, n=5).
func squircle(dc *gg.Context, inset float64) {
	c := S / 2
	r := S/2 - inset
	const n = 5.0
	dc.NewSubPath()
	for i := 0; i <= 720; i++ {
		t := float64(i) / 720 * 2 * math.Pi
		ct, st := math.Cos(t), math.Sin(t)
		x := c + r*math.Copysign(math.Pow(math.Abs(ct), 2/n), ct)
		y := c + r*math.Copysign(math.Pow(math.Abs(st), 2/n), st)
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.ClosePath()
}

// segment draws one rounded annular sector (Falak's own wheel shape).
func segment(dc *gg.Context, c, r1, r2, a0, a1, g, rc float64) {
	side := func(a, off, r float64) (float64, float64) {
		ux, uy := math.Cos(a), math.Sin(a)
		rho := math.Sqrt(r*r - off*off)
		return c + rho*ux - off*uy, c + rho*uy + off*ux
	}
	th := func(a, off, r float64) float64 { return a + math.Asin(off/r) }
	ua0x, ua0y := math.Cos(a0), math.Sin(a0)
	ua1x, ua1y := math.Cos(a1), math.Sin(a1)
	ri := rc * 0.6
	aInX, aInY := side(a0, g, r1)
	aOutX, aOutY := side(a0, g, r2)
	bOutX, bOutY := side(a1, -g, r2)
	bInX, bInY := side(a1, -g, r1)
	dc.NewSubPath()
	dc.MoveTo(aInX+ri*ua0x, aInY+ri*ua0y)
	dc.LineTo(aOutX-rc*ua0x, aOutY-rc*ua0y)
	t0, t1 := th(a0, g, r2)+rc/r2, th(a1, -g, r2)-rc/r2
	dc.QuadraticTo(aOutX, aOutY, c+r2*math.Cos(t0), c+r2*math.Sin(t0))
	dc.DrawArc(c, c, r2, t0, t1)
	dc.QuadraticTo(bOutX, bOutY, bOutX-rc*ua1x, bOutY-rc*ua1y)
	dc.LineTo(bInX+ri*ua1x, bInY+ri*ua1y)
	t2, t3 := th(a1, -g, r1)-ri/r1, th(a0, g, r1)+ri/r1
	dc.QuadraticTo(bInX, bInY, c+r1*math.Cos(t2), c+r1*math.Sin(t2))
	dc.DrawArc(c, c, r1, t2, t3)
	dc.QuadraticTo(aInX, aInY, aInX+ri*ua0x, aInY+ri*ua0y)
	dc.ClosePath()
}

func hex(s string) color.NRGBA {
	var r, g, b uint8
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return color.NRGBA{r, g, b, 255}
}

func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

func bg(dc *gg.Context, top, bottom string) {
	g := gg.NewLinearGradient(0, 0, 0, S)
	g.AddColorStop(0, hex(top))
	g.AddColorStop(1, hex(bottom))
	squircle(dc, 0)
	dc.SetFillStyle(g)
	dc.Fill()
}

type design struct {
	name, title string
	draw        func(dc *gg.Context)
}

func ring(dc *gg.Context, n int, r1, r2, gap, rc, rot float64, col func(i int) color.Color) {
	c := S / 2
	span := 2 * math.Pi / float64(n)
	for i := 0; i < n; i++ {
		mid := -math.Pi/2 + rot + float64(i)*span
		segment(dc, c, r1, r2, mid-span/2, mid+span/2, gap, rc)
		dc.SetColor(col(i))
		dc.Fill()
	}
}

func dot(dc *gg.Context, r float64, col string) {
	dc.DrawCircle(S/2, S/2, r)
	dc.SetColor(hex(col))
	dc.Fill()
}

var designs = []design{
	{"1-orbit", "مدار", func(dc *gg.Context) {
		bg(dc, "#2A95FF", "#0062E0")
		ring(dc, 8, 190, 372, 14, 46, 0, func(int) color.Color { return color.White })
		dot(dc, 74, "#FFFFFF")
	}},
	{"2-blue-shades", "ظلال الأزرق", func(dc *gg.Context) {
		bg(dc, "#FFFFFF", "#EEF3FA")
		a, b := hex("#7CC4FF"), hex("#0049C7")
		ring(dc, 8, 186, 372, 14, 46, 0, func(i int) color.Color { return lerp(a, b, float64(i)/7) })
		dot(dc, 70, "#007AFF")
	}},
	{"3-midnight", "منتصف الليل", func(dc *gg.Context) {
		bg(dc, "#1B2A4A", "#0A1020")
		a, b := hex("#5AD8FF"), hex("#5B5BFF")
		ring(dc, 8, 190, 372, 14, 46, 0, func(i int) color.Color { return lerp(a, b, math.Abs(float64(i)-3.5)/3.5) })
		dot(dc, 70, "#FFFFFF")
	}},
	{"4-selected", "التحديد", func(dc *gg.Context) {
		bg(dc, "#FFFFFF", "#F1F4F8")
		ring(dc, 6, 196, 376, 16, 52, 0, func(i int) color.Color {
			if i == 1 {
				return hex("#007AFF")
			}
			return hex("#C2CCD9")
		})
		dot(dc, 72, "#007AFF")
	}},
	{"5-spectrum", "الطيف", func(dc *gg.Context) {
		bg(dc, "#FFFFFF", "#F3F5F8")
		cols := []string{"#007AFF", "#32ADE6", "#30B0C7", "#34C759", "#FFCC00", "#FF9500", "#FF2D55", "#AF52DE"}
		ring(dc, 8, 186, 372, 14, 46, 0, func(i int) color.Color { return hex(cols[i]) })
		dot(dc, 70, "#1C1C1E")
	}},
	{"6-sky-dial", "السماء", func(dc *gg.Context) {
		bg(dc, "#7FD1FF", "#2F80FF")
		ring(dc, 12, 236, 368, 12, 30, 0, func(i int) color.Color {
			if i%3 == 0 {
				return color.White
			}
			return color.NRGBA{255, 255, 255, 120}
		})
		dot(dc, 120, "#FFFFFF")
		dot(dc, 46, "#2F80FF")
	}},
}

func main() {
	out := os.Args[1]
	os.MkdirAll(out, 0o755)
	var imgs []image.Image
	for _, d := range designs {
		dc := gg.NewContext(int(S), int(S))
		d.draw(dc)
		f, _ := os.Create(fmt.Sprintf("%s/%s.png", out, d.name))
		png.Encode(f, dc.Image())
		f.Close()
		imgs = append(imgs, dc.Image())
	}
	// Preview sheet: each icon at app-icon size on light and dark backgrounds.
	fontData, _ := os.ReadFile("/usr/share/fonts/opentype/inter/Inter-SemiBold.otf")
	ft, _ := opentype.Parse(fontData)
	face, _ := opentype.NewFace(ft, &opentype.FaceOptions{Size: 30, DPI: 72})
	const cell, pad = 300, 40
	W := pad + len(imgs)*(cell+pad)
	sheet := gg.NewContext(W, 2*(cell+pad)+80)
	sheet.SetColor(hex("#F2F2F7"))
	sheet.DrawRectangle(0, 0, float64(W), float64(cell+pad+40))
	sheet.Fill()
	sheet.SetColor(hex("#000000"))
	sheet.DrawRectangle(0, float64(cell+pad+40), float64(W), float64(cell+pad+40))
	sheet.Fill()
	sheet.SetFontFace(face)
	for i, im := range imgs {
		small := image.NewRGBA(image.Rect(0, 0, 220, 220))
		sc := gg.NewContextForRGBA(small)
		sc.Scale(220/S, 220/S)
		sc.DrawImage(im, 0, 0)
		x := pad + i*(cell+pad)
		for row := 0; row < 2; row++ {
			y := 30 + row*(cell+pad+40)
			draw.Draw(sheet.Image().(*image.RGBA), image.Rect(x+40, y, x+260, y+220), small, image.Point{}, draw.Over)
			if row == 0 {
				sheet.SetColor(hex("#1C1C1E"))
			} else {
				sheet.SetColor(hex("#FFFFFF"))
			}
			sheet.DrawStringAnchored(fmt.Sprintf("%d", i+1), float64(x+150), float64(y+262), 0.5, 0.5)
		}
	}
	sheet.SavePNG(out + "/preview.png")
}

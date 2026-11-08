package ui

import (
	"image"
	"image/color"
	"math"

	"github.com/fogleman/gg"
)

// The tray menu: a rounded card (dark or light) with a header, rows that
// each have an icon tile, a title, a subtitle and an on/off switch, and a
// footer with two buttons — the style of the reference menu the user chose.

// MenuRow is one entry of the menu.
type MenuRow struct {
	ID      string
	Icon    string // menu icon id (see drawMenuIcon)
	Title   string
	Sub     string // second line ("" for none)
	SubBlue bool   // the second line is a state, drawn in the accent color
	Toggle  bool   // has an on/off switch
	On      bool
	Footer  bool // a button of the bottom row
}

// Menu lays out and draws the card. Fonts: Strong for the title and rows
// (bold / semibold), Plain for subtitles (its "semibold" is a regular face).
type Menu struct {
	Scale         float64
	Dark, RTL     bool
	Title, Sub    string
	Rows          []MenuRow
	Strong, Plain *Fonts
	Hover         int // index into Rows, -1 for none

	// Logo replaces the blue mark in the header when set (already sized).
	Logo *image.RGBA

	rects  []rectF
	shadow *image.RGBA
	shKey  [3]int
}

type rectF struct{ x, y, w, h float64 }

func (r rectF) has(x, y float64) bool { return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h }

// Logical geometry of the menu.
const (
	menuW       = 304.0
	menuMargin  = 16.0 // room for the shadow around the card
	menuRadius  = 12.0
	menuHeader  = 62.0
	menuRowH    = 54.0
	menuFooterH = 46.0
	menuPad     = 8.0
)

func (m *Menu) cardH() float64 {
	h := menuHeader + menuPad
	hasFooter := false
	for _, r := range m.Rows {
		if r.Footer {
			hasFooter = true
		} else {
			h += menuRowH
		}
	}
	if hasFooter {
		h += 1 + menuPad + menuFooterH
	}
	return h + menuPad
}

// Size is the window size in device pixels.
func (m *Menu) Size() (int, int) {
	s := m.Scale
	return int(math.Ceil((menuW + 2*menuMargin) * s)), int(math.Ceil((m.cardH() + 2*menuMargin) * s))
}

// CardOffset is where the visible card starts inside the window (device
// pixels), so the window can be placed with the card's corner at a point.
func (m *Menu) CardOffset() int { return int(math.Round(menuMargin * m.Scale)) }

func (m *Menu) layout() {
	s := m.Scale
	m.rects = make([]rectF, len(m.Rows))
	x0 := (menuMargin + menuPad) * s
	w := (menuW - 2*menuPad) * s
	y := (menuMargin + menuHeader + menuPad) * s
	var feet []int
	for i, r := range m.Rows {
		if r.Footer {
			feet = append(feet, i)
			continue
		}
		m.rects[i] = rectF{x0, y, w, menuRowH * s}
		y += menuRowH * s
	}
	if len(feet) > 0 {
		y += (1 + menuPad) * s
		fw := (w - menuPad*s) / float64(len(feet))
		for k, i := range feet {
			x := x0 + float64(k)*(fw+menuPad*s)
			if m.RTL {
				x = x0 + w - float64(k+1)*fw - float64(k)*menuPad*s
			}
			m.rects[i] = rectF{x, y, fw, menuFooterH * s}
		}
	}
}

// HitTest returns the row under window pixel (x, y), or -1.
func (m *Menu) HitTest(x, y float64) int {
	if m.rects == nil || len(m.rects) != len(m.Rows) {
		m.layout()
	}
	for i, r := range m.rects {
		if r.has(x, y) {
			return i
		}
	}
	return -1
}

type menuPalette struct {
	bg, border, title, sub, tile, icon, hover, sep, off color.NRGBA
}

func (m *Menu) palette() menuPalette {
	if m.Dark {
		return menuPalette{
			bg: color.NRGBA{40, 40, 44, 255}, border: color.NRGBA{255, 255, 255, 26},
			title: color.NRGBA{246, 246, 248, 255}, sub: color.NRGBA{152, 152, 160, 255},
			tile: color.NRGBA{60, 60, 66, 255}, icon: color.NRGBA{236, 236, 240, 255},
			hover: color.NRGBA{255, 255, 255, 16}, sep: color.NRGBA{255, 255, 255, 22},
			off: color.NRGBA{82, 82, 88, 255},
		}
	}
	return menuPalette{
		bg: color.NRGBA{255, 255, 255, 255}, border: color.NRGBA{0, 0, 0, 30},
		title: color.NRGBA{29, 29, 31, 255}, sub: color.NRGBA{112, 112, 118, 255},
		tile: color.NRGBA{240, 240, 243, 255}, icon: color.NRGBA{58, 58, 64, 255},
		hover: color.NRGBA{0, 0, 0, 12}, sep: color.NRGBA{0, 0, 0, 20},
		off: color.NRGBA{212, 212, 217, 255},
	}
}

// Render draws the whole menu.
func (m *Menu) Render() *image.RGBA {
	m.layout()
	s := m.Scale
	W, H := m.Size()
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	pal := m.palette()
	cx0, cy0 := menuMargin*s, menuMargin*s
	cw, ch := menuW*s, m.cardH()*s

	// soft shadow, kept between frames (only hover changes)
	if key := [3]int{W, H, map[bool]int{false: 0, true: 1}[m.Dark]}; m.shadow == nil || m.shKey != key {
		sh := image.NewRGBA(img.Rect)
		sdc := gg.NewContextForRGBA(sh)
		sdc.DrawRoundedRectangle(cx0, cy0+3*s, cw, ch, menuRadius*s)
		a := uint8(60)
		if m.Dark {
			a = 120
		}
		sdc.SetColor(color.NRGBA{0, 0, 0, a})
		sdc.Fill()
		m.shadow, m.shKey = GaussianBlur(sh, 7*s), key
	}
	copy(img.Pix, m.shadow.Pix)

	dc := gg.NewContextForRGBA(img)
	// the card itself is opaque: clear the shadow under it first
	card := image.NewRGBA(img.Rect)
	cdc := gg.NewContextForRGBA(card)
	cdc.DrawRoundedRectangle(cx0, cy0, cw, ch, menuRadius*s)
	cdc.SetColor(pal.bg)
	cdc.Fill()
	for i := 0; i+3 < len(card.Pix); i += 4 {
		if a := card.Pix[i+3]; a > 0 {
			k := 255 - uint32(a)
			for c := 0; c < 4; c++ {
				img.Pix[i+c] = uint8(uint32(card.Pix[i+c]) + uint32(img.Pix[i+c])*k/255)
			}
		}
	}
	dc.DrawRoundedRectangle(cx0+0.5*s, cy0+0.5*s, cw-1*s, ch-1*s, menuRadius*s-0.5*s)
	dc.SetColor(pal.border)
	dc.SetLineWidth(1 * s)
	dc.Stroke()

	// header: app name and version line, with the app's blue mark
	pad := 18 * s
	hx := cx0 + pad
	if m.RTL {
		hx = cx0 + cw - pad
	}
	m.text(dc, m.Strong, m.Title, 18*s, true, hx, cy0+25*s, pal.title, cw-2*pad-30*s)
	if m.Sub != "" {
		m.text(dc, m.Plain, m.Sub, 12*s, false, hx, cy0+45*s, pal.sub, cw-2*pad-30*s)
	}
	mx := cx0 + cw - pad - 11*s
	if m.RTL {
		mx = cx0 + pad + 11*s
	}
	if m.Logo != nil { // the app icon, where the blue mark used to sit
		lg := float64(m.Logo.Rect.Dx())
		dc.DrawImage(m.Logo, int(math.Round(mx-lg/2)), int(math.Round(cy0+31*s-lg/2)))
	} else {
		drawFalakMark(dc, mx, cy0+31*s, 11*s)
	}

	sepY := 0.0
	for i, r := range m.Rows {
		rc := m.rects[i]
		if r.Footer && sepY == 0 {
			sepY = rc.y - (menuPad/2+0.5)*s
			dc.DrawLine(cx0+pad, sepY, cx0+cw-pad, sepY)
			dc.SetColor(pal.sep)
			dc.SetLineWidth(1 * s)
			dc.Stroke()
		}
		if i == m.Hover {
			dc.DrawRoundedRectangle(rc.x, rc.y, rc.w, rc.h, 9*s)
			dc.SetColor(pal.hover)
			dc.Fill()
		}
		if r.Footer {
			m.drawFooter(dc, r, rc, pal)
			continue
		}
		// icon tile
		ts := 36 * s
		tx := rc.x + 8*s
		if m.RTL {
			tx = rc.x + rc.w - 8*s - ts
		}
		ty := rc.y + (rc.h-ts)/2
		dc.DrawRoundedRectangle(tx, ty, ts, ts, 9*s)
		dc.SetColor(pal.tile)
		dc.Fill()
		drawMenuIcon(dc, r.Icon, tx+ts/2, ty+ts/2, 18*s, pal.icon)
		// switch
		sw, sh := 40*s, 24*s
		sx := rc.x + rc.w - 10*s - sw
		if m.RTL {
			sx = rc.x + 10*s
		}
		sy := rc.y + (rc.h-sh)/2
		if r.Toggle {
			dc.DrawRoundedRectangle(sx, sy, sw, sh, sh/2)
			if r.On {
				dc.SetColor(Accent)
			} else {
				dc.SetColor(pal.off)
			}
			dc.Fill()
			kr := sh/2 - 2.5*s
			kx := sx + sh/2
			if r.On != m.RTL {
				kx = sx + sw - sh/2
			}
			dc.DrawCircle(kx, sy+sh/2+0.6*s, kr)
			dc.SetColor(color.NRGBA{0, 0, 0, 40})
			dc.Fill()
			dc.DrawCircle(kx, sy+sh/2, kr)
			dc.SetColor(color.White)
			dc.Fill()
		}
		// texts
		textX := tx + ts + 12*s
		if m.RTL {
			textX = tx - 12*s
		}
		maxW := rc.w - ts - 8*s - 12*s - sw - 22*s
		sub := pal.sub
		if r.SubBlue {
			sub = color.NRGBA{64, 156, 255, 255}
			if !m.Dark {
				sub = Accent
			}
		}
		if r.Sub != "" {
			m.text(dc, m.Strong, r.Title, 14*s, false, textX, rc.y+rc.h/2-8*s, pal.title, maxW)
			m.text(dc, m.Plain, r.Sub, 12*s, false, textX, rc.y+rc.h/2+10*s, sub, maxW)
		} else {
			m.text(dc, m.Strong, r.Title, 14*s, false, textX, rc.y+rc.h/2, pal.title, maxW)
		}
	}
	return img
}

func (m *Menu) drawFooter(dc *gg.Context, r MenuRow, rc rectF, pal menuPalette) {
	s := m.Scale
	col := pal.title
	is := 17 * s
	dc2 := dc
	// icon and label centered together
	var tw float64
	if f := m.Strong.Face(math.Round(14*s*4)/4, false); f != nil {
		dc2.SetFontFace(f)
		tw, _ = dc2.MeasureString(Visual(r.Title))
	}
	gap := 9 * s
	total := is + gap + tw
	x := rc.x + (rc.w-total)/2
	cy := rc.y + rc.h/2
	if m.RTL {
		drawMenuIcon(dc, r.Icon, x+total-is/2, cy, is, pal.icon)
		m.text(dc, m.Strong, r.Title, 14*s, false, x+tw, cy, col, rc.w)
		return
	}
	drawMenuIcon(dc, r.Icon, x+is/2, cy, is, pal.icon)
	m.text(dc, m.Strong, r.Title, 14*s, false, x+is+gap, cy, col, rc.w)
}

// text draws str vertically centered on y. In RTL menus x is the right
// edge of the text, otherwise its left edge.
func (m *Menu) text(dc *gg.Context, fs *Fonts, str string, px float64, bold bool, x, y float64, c color.Color, maxW float64) {
	f := fs.Face(math.Round(px*4)/4, bold)
	if f == nil {
		return
	}
	dc.SetFontFace(f)
	str = fitText(dc, str, maxW)
	v := Visual(str)
	tw, _ := dc.MeasureString(v)
	if m.RTL {
		x -= tw
	}
	dc.SetColor(c)
	dc.DrawString(v, x, y+capHeightOf(f)/2)
}

// drawFalakMark is the app's small mark: a blue disc with a white ring
// segment, echoing the wheel.
func drawFalakMark(dc *gg.Context, x, y, r float64) {
	dc.DrawCircle(x, y, r)
	dc.SetColor(Accent)
	dc.Fill()
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineWidth(r * 0.26)
	dc.SetColor(color.White)
	dc.DrawArc(x, y, r*0.52, -math.Pi*0.15, math.Pi*1.35)
	dc.Stroke()
	dc.DrawCircle(x, y, r*0.17)
	dc.Fill()
}

// drawMenuIcon draws the menu's line icons (same style as the wheel's).
func drawMenuIcon(dc *gg.Context, id string, x, y, s float64, c color.Color) {
	dc.Push()
	defer dc.Pop()
	dc.SetColor(c)
	h := s / 2
	dc.SetLineWidth(s * 0.11)
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineJoin(gg.LineJoinRound)
	L := func(x0, y0, x1, y1 float64) {
		dc.NewSubPath()
		dc.MoveTo(x+x0*h, y+y0*h)
		dc.LineTo(x+x1*h, y+y1*h)
		dc.Stroke()
	}
	switch id {
	case "shield": // key-reading helper (administrator)
		dc.NewSubPath()
		dc.MoveTo(x, y-0.95*h)
		dc.LineTo(x+0.8*h, y-0.62*h)
		dc.LineTo(x+0.8*h, y)
		dc.CubicTo(x+0.8*h, y+0.55*h, x+0.35*h, y+0.82*h, x, y+0.98*h)
		dc.CubicTo(x-0.35*h, y+0.82*h, x-0.8*h, y+0.55*h, x-0.8*h, y)
		dc.LineTo(x-0.8*h, y-0.62*h)
		dc.ClosePath()
		dc.Stroke()
		dc.NewSubPath()
		dc.MoveTo(x-0.34*h, y+0.02*h)
		dc.LineTo(x-0.06*h, y+0.3*h)
		dc.LineTo(x+0.38*h, y-0.2*h)
		dc.Stroke()
	case "startup": // power-on rocket-like arrow out of a line
		dc.NewSubPath()
		dc.DrawRoundedRectangle(x-0.9*h, y-0.75*h, 1.8*h, 1.5*h, 0.3*h)
		dc.Stroke()
		L(-0.9, -0.3, 0.9, -0.3)
		dc.DrawCircle(x-0.55*h, y-0.53*h, 0.07*h)
		dc.DrawCircle(x-0.32*h, y-0.53*h, 0.07*h)
		dc.Fill()
		L(0, 0.48, 0, 0.0)
		L(-0.25, 0.22, 0, -0.03)
		L(0.25, 0.22, 0, -0.03)
	case "sound": // speaker with waves
		dc.NewSubPath()
		dc.MoveTo(x-0.92*h, y-0.3*h)
		dc.LineTo(x-0.55*h, y-0.3*h)
		dc.LineTo(x-0.12*h, y-0.75*h)
		dc.LineTo(x-0.12*h, y+0.75*h)
		dc.LineTo(x-0.55*h, y+0.3*h)
		dc.LineTo(x-0.92*h, y+0.3*h)
		dc.ClosePath()
		dc.FillPreserve()
		dc.Stroke()
		dc.NewSubPath()
		dc.DrawArc(x-0.12*h, y, 0.5*h, -math.Pi/4, math.Pi/4)
		dc.Stroke()
		dc.NewSubPath()
		dc.DrawArc(x-0.12*h, y, 0.92*h, -math.Pi/4, math.Pi/4)
		dc.Stroke()
	case "moon": // crescent: a disc with an offset disc taken out
		R, bx, by, r := 0.84, 0.40, -0.40, 0.70
		d := math.Hypot(bx, by)
		a := (R*R - r*r + d*d) / (2 * d)
		hh := math.Sqrt(math.Max(0, R*R-a*a))
		ux, uy := bx/d, by/d
		mx, my := a*ux, a*uy
		p1 := [2]float64{mx - hh*uy, my + hh*ux}
		p2 := [2]float64{mx + hh*uy, my - hh*ux}
		ang := func(p [2]float64, cx, cy float64) float64 { return math.Atan2(p[1]-cy, p[0]-cx) }
		a1, a2 := ang(p1, 0, 0), ang(p2, 0, 0)
		for a2 < a1 {
			a2 += 2 * math.Pi
		}
		b1, b2 := ang(p2, bx, by), ang(p1, bx, by)
		for b2 > b1 {
			b2 -= 2 * math.Pi
		}
		dc.NewSubPath()
		dc.DrawArc(x, y, R*h, a1, a2)
		dc.DrawArc(x+bx*h, y+by*h, r*h, b1, b2)
		dc.ClosePath()
		dc.Stroke()
	case "globe": // language
		dc.NewSubPath()
		dc.DrawCircle(x, y, 0.88*h)
		dc.Stroke()
		dc.NewSubPath()
		dc.DrawEllipse(x, y, 0.38*h, 0.88*h)
		dc.Stroke()
		L(-0.88, 0, 0.88, 0)
	case "help":
		dc.NewSubPath()
		dc.DrawCircle(x, y, 0.88*h)
		dc.Stroke()
		dc.NewSubPath()
		dc.DrawArc(x, y-0.2*h, 0.3*h, math.Pi*1.0, math.Pi*2.25)
		dc.LineTo(x, y+0.22*h)
		dc.Stroke()
		dc.DrawCircle(x, y+0.52*h, 0.08*h)
		dc.Fill()
	case "power":
		dc.NewSubPath()
		dc.DrawArc(x, y+0.06*h, 0.8*h, -math.Pi*0.30, math.Pi*1.30)
		dc.Stroke()
		L(0, -0.95, 0, -0.05)
	}
}

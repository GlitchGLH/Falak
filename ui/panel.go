package ui

import (
	"image"
	"image/color"
	"math"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

// Panel geometry in logical units, measured on the reference window
// ("Converting to MP3", file name · percent, long blue bar).
const (
	PanelW      = 427.0
	PanelH      = 129.0
	PanelMargin = 22.0 // room around the panel for its soft shadow
	panelR      = 32.0
	closeCX     = 32.5
	closeCY     = 32.5
	closeR      = 14.5
	titleX      = 60.0
	subX        = 24.0
	subY        = 75.0
	barX        = 24.0
	barY        = 101.0
	barW        = 380.0
	barH        = 6.0
)

// Panel is the small glass window that shows a running conversion.
type Panel struct {
	Scale float64
	W, H  int
	Fonts *Fonts

	title, file string
	Dark        bool    // dark theme glass with light text
	RTL         bool    // Arabic: text on the right, the bar fills from the right
	progress    float64 // 0..1, or -1 while unknown
	shown       float64
	last        time.Time
	openAt      time.Time
	px, py      float64
	hover       float64

	base  *image.RGBA                // glass, edge, shadow and title
	next  atomic.Pointer[image.RGBA] // a rebuilt base waiting to be used
	busy  atomic.Bool
	frame *image.RGBA
	work  *image.RGBA
	dc    *gg.Context
}

// PanelSize is the window size in pixels for a scale.
func PanelSize(scale float64) (int, int) {
	return int(math.Ceil((PanelW + 2*PanelMargin) * scale)), int(math.Ceil((PanelH + 2*PanelMargin) * scale))
}

// NewPanel builds the panel over bg, the desktop captured under the window.
func NewPanel(scale float64, bg *image.RGBA, fonts *Fonts, title, file string, now time.Time) *Panel {
	return NewPanelTheme(scale, bg, fonts, title, file, false, now)
}

// NewPanelTheme is NewPanel in the light or the dark theme.
func NewPanelTheme(scale float64, bg *image.RGBA, fonts *Fonts, title, file string, dark bool, now time.Time) *Panel {
	return NewPanelDir(scale, bg, fonts, title, file, dark, false, now)
}

// NewPanelDir is NewPanelTheme for a reading direction: with rtl (Arabic)
// the title and the file line sit at the right edge and the bar fills from
// the right; the X stays at the top left, as in the Arabic compress dialog.
func NewPanelDir(scale float64, bg *image.RGBA, fonts *Fonts, title, file string, dark, rtl bool, now time.Time) *Panel {
	w, h := PanelSize(scale)
	p := &Panel{Scale: scale, W: w, H: h, Fonts: fonts, title: title, file: file, Dark: dark, RTL: rtl,
		progress: -1, openAt: now, last: now, px: -1e6, py: -1e6}
	p.frame = image.NewRGBA(image.Rect(0, 0, w, h))
	p.work = image.NewRGBA(image.Rect(0, 0, w, h))
	p.dc = gg.NewContextForRGBA(p.work)
	p.base = p.buildBase(bg)
	return p
}

func (p *Panel) SetProgress(f float64) { p.progress = f }

// Rebackground rebuilds the glass over a new desktop capture (after the
// window was moved) in the background; the next frames pick it up.
func (p *Panel) Rebackground(bg *image.RGBA) {
	if !p.busy.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer p.busy.Store(false)
		p.next.Store(p.buildBase(bg))
	}()
}

func (p *Panel) SetPointer(x, y float64) { p.px, p.py = x, y }

// OverClose reports whether window pixel (x, y) is on the X button.
func (p *Panel) OverClose(x, y float64) bool {
	s := p.Scale
	cx, cy := (PanelMargin+closeCX)*s, (PanelMargin+closeCY)*s
	return math.Hypot(x-cx, y-cy) <= (closeR+4)*s
}

// Animating is true while the pop-in still moves.
func (p *Panel) Animating(now time.Time) bool { return now.Sub(p.openAt) < 300*time.Millisecond }

func (p *Panel) rrect(dc *gg.Context, inset float64) {
	s := p.Scale
	m := PanelMargin * s
	dc.DrawRoundedRectangle(m+inset, m+inset, PanelW*s-2*inset, PanelH*s-2*inset, panelR*s-inset)
}

// buildBase draws everything that stays the same: shadow, the segments'
// glass (same blur, veil and see-through), the edge and glints, the title.
func (p *Panel) buildBase(bg *image.RGBA) *image.RGBA {
	out := glassBase(p.W, p.H, PanelW, PanelH, PanelMargin, panelR, p.Scale, bg, p.Dark)
	s := p.Scale

	// title
	dc := gg.NewContextForRGBA(out)
	if f := p.Fonts.Face(math.Round(17*s*4)/4, false); f != nil {
		dc.SetFontFace(f)
		dc.SetColor(p.grey())
		maxW := (PanelW - titleX - 24) * s
		t := Visual(fitText(dc, p.title, maxW))
		capH := capHeightOf(f)
		x := (PanelMargin + titleX) * s
		if p.RTL { // right edge, the same 24 margin as the bar
			tw, _ := dc.MeasureString(t)
			x = (PanelMargin+PanelW-24)*s - tw
		}
		dc.DrawString(t, x, (PanelMargin+closeCY)*s+capH/2)
	}
	return out
}

// glassBase draws the card's glass over the desktop capture bg: the soft
// shadow, the blurred backdrop with its veil and tint, what shows through,
// the glints and the thin edge. ONE code path for the progress panel and
// the compress dialog — the same values, never a copy.
func glassBase(winW, winH int, cardW, cardH, margin, r, s float64, bg *image.RGBA, dark bool) *image.RGBA {
	_ = bg // the popups are solid now: no desktop capture, ever
	out := image.NewRGBA(image.Rect(0, 0, winW, winH))
	// soft shadow, exactly as before
	sh := image.NewRGBA(out.Rect)
	sdc := gg.NewContextForRGBA(sh)
	sdc.DrawRoundedRectangle(margin*s, (margin+4)*s, cardW*s, cardH*s, r*s)
	sdc.SetColor(color.NRGBA{0, 0, 0, 40})
	sdc.Fill()
	sh = GaussianBlur(sh, 9*s)
	copy(out.Pix, sh.Pix)

	// the card itself: opaque, in the tray menu's colors - dark charcoal
	// on the dark theme, white on the light one - with the same glints
	// and the same thin grey edge as before (same shape, corners and
	// border radius as the drag look)
	cardCol := color.NRGBA{255, 255, 255, 255} // the menu's light card
	if dark {
		cardCol = color.NRGBA{40, 40, 44, 255} // the menu's dark card
	}
	g := image.NewRGBA(out.Rect)
	gdc := gg.NewContextForRGBA(g)
	gdc.DrawRoundedRectangle(margin*s, margin*s, cardW*s, cardH*s, r*s)
	gdc.SetColor(cardCol)
	gdc.Fill()
	glintA := 1.0
	gdc.Push()
	gdc.DrawRoundedRectangle(margin*s, margin*s, cardW*s, cardH*s, r*s)
	gdc.Clip()
	gdc.DrawRoundedRectangle(margin*s, margin*s, cardW*s, cardH*s, r*s)
	gr := gg.NewLinearGradient(margin*s, margin*s, margin*s+cardW*s*0.35, margin*s+cardH*s*1.2)
	gr.AddColorStop(0, withAlpha(color.NRGBA{255, 255, 255, 235}, glintA))
	gr.AddColorStop(0.32, withAlpha(color.NRGBA{255, 255, 255, 40}, glintA))
	gr.AddColorStop(0.68, withAlpha(color.NRGBA{255, 255, 255, 25}, glintA))
	gr.AddColorStop(1, withAlpha(color.NRGBA{255, 255, 255, 150}, glintA))
	gdc.SetStrokeStyle(gr)
	gdc.SetLineWidth(2.4 * s)
	gdc.Stroke()
	gdc.Pop()
	gdc.ResetClip()
	gdc.DrawRoundedRectangle(margin*s, margin*s, cardW*s, cardH*s, r*s)
	gdc.SetColor(color.NRGBA{0, 0, 0, 52})
	gdc.SetLineWidth(1 * s)
	gdc.Stroke()
	blendOver(out, g, 1)
	return out
}

func (p *Panel) grey() color.NRGBA {
	if p.Dark {
		return color.NRGBA{214, 214, 220, 255}
	}
	return color.NRGBA{88, 88, 93, 255}
}

func capHeightOf(f font.Face) float64 {
	m := f.Metrics()
	if c := float64(m.CapHeight) / 64; c > 0 {
		return c
	}
	return float64(m.Ascent) / 64 * 0.7
}

// Render draws one frame.
func (p *Panel) Render(now time.Time) *image.RGBA {
	s := p.Scale
	dt := math.Min(now.Sub(p.last).Seconds(), 0.1)
	p.last = now
	if p.progress >= 0 {
		k := 1 - math.Exp(-dt/0.05)
		if p.progress < p.shown {
			p.shown = p.progress
		} else {
			p.shown += (p.progress - p.shown) * k
		}
	}
	over := 0.0
	if p.OverClose(p.px, p.py) {
		over = 1
	}
	p.hover += (over - p.hover) * (1 - math.Exp(-dt/0.06))

	if nb := p.next.Swap(nil); nb != nil {
		p.base = nb
	}
	copy(p.work.Pix, p.base.Pix)
	dc := p.dc
	m := PanelMargin * s
	grey := p.grey()

	// X button
	cx, cy := m+closeCX*s, m+closeCY*s
	dc.DrawCircle(cx, cy, closeR*s)
	if p.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, uint8(255 * (0.08 + 0.08*p.hover))})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, uint8(255 * (0.06 + 0.07*p.hover))})
	}
	dc.Fill()
	d := 4.6 * s
	dc.SetLineCap(gg.LineCapRound)
	dc.SetLineWidth(1.7 * s)
	dc.SetColor(grey)
	dc.DrawLine(cx-d, cy-d, cx+d, cy+d)
	dc.DrawLine(cx+d, cy-d, cx-d, cy+d)
	dc.Stroke()

	// file name · percent
	if f := p.Fonts.Face(math.Round(14*s*4)/4, false); f != nil {
		dc.SetFontFace(f)
		if p.Dark {
			dc.SetColor(color.NRGBA{172, 172, 178, 255})
		} else {
			dc.SetColor(color.NRGBA{120, 120, 126, 255})
		}
		pct := ""
		if p.progress >= 0 {
			pct = " · " + itoa(int(math.Floor(p.shown*100+0.0001))) + "%"
		}
		pw, _ := dc.MeasureString(pct)
		name := fitText(dc, p.file, barW*s-pw)
		name = Visual(name)
		x := m + subX*s
		if p.RTL { // ends where the bar ends
			lw, _ := dc.MeasureString(name + pct)
			x = m + (barX+barW)*s - lw
		}
		dc.DrawString(name+pct, x, m+subY*s+capHeightOf(f)/2)
	}

	// bar
	bx, by, bw, bh := m+barX*s, m+(barY-barH/2)*s, barW*s, barH*s
	dc.DrawRoundedRectangle(bx, by, bw, bh, bh/2)
	if p.Dark {
		dc.SetColor(color.NRGBA{255, 255, 255, 36})
	} else {
		dc.SetColor(color.NRGBA{0, 0, 0, 26})
	}
	dc.Fill()
	if fw := bw * clampF(p.shown, 0, 1); fw > 0.5 {
		fw = math.Max(fw, bh)
		fx := bx
		if p.RTL { // fills from the right
			fx = bx + bw - fw
		}
		dc.DrawRoundedRectangle(fx, by, fw, bh, bh/2)
		dc.SetColor(Accent)
		dc.Fill()
	}

	// pop-in: same curve as the wheel
	t := now.Sub(p.openAt).Seconds()
	a := math.Min(1, 0.5+t*21)
	sc := 1 - 0.60*math.Exp(-t/0.035)
	if t > 0.3 {
		sc = 1
	}
	if a >= 0.999 && sc >= 0.9995 {
		return p.work
	}
	for i := range p.frame.Pix {
		p.frame.Pix[i] = 0
	}
	drawLayer(p.frame, p.work, float64(p.W)/2, float64(p.H)/2, sc, 0, a, true)
	return p.frame
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 && i > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// fitText shortens s with an ellipsis in the middle so it fits maxW,
// keeping the extension visible.
func fitText(dc *gg.Context, s string, maxW float64) string {
	if w, _ := dc.MeasureString(Visual(s)); w <= maxW {
		return s
	}
	r := []rune(s)
	keepEnd := min(8, len(r)/3)
	for head := len(r) - keepEnd - 1; head > 1; head-- {
		t := string(r[:head]) + "…" + string(r[len(r)-keepEnd:])
		if w, _ := dc.MeasureString(Visual(t)); w <= maxW {
			return t
		}
	}
	return "…"
}

// Visual returns s ready for drawing left to right: Arabic letters are
// joined (presentation forms) and right-to-left runs are reversed.
// Plain Latin text is returned unchanged.
// Visual returns s ready for drawing left to right: Arabic letters are
// joined (presentation forms) and right-to-left runs are reversed.
// Plain Latin text is returned unchanged.
func Visual(s string) string {
	hasRTL := false
	for _, c := range s {
		if isRTL(c) {
			hasRTL = true
			break
		}
	}
	if !hasRTL {
		return s
	}
	shaped := shapeArabic([]rune(s))
	// A small UAX #9 for our RTL interface: strong characters are level 1
	// (RTL — the base direction) or 2 (a Latin or digit island); neutrals
	// take the side they sit between, else the base direction. Then L2:
	// reverse the whole line and give each level-2 island its natural
	// left-to-right order back.
	n := len(shaped)
	lvl := make([]int, n)
	for i, c := range shaped {
		if isLTRStrong(c) {
			lvl[i] = 2
		} else if isRTL(c) {
			lvl[i] = 1
		}
	}
	for i := 1; i < n; i++ { // a percent sign stays with its number
		c := shaped[i]
		if (c == '%' || c == '٪') && shaped[i-1] >= '0' && shaped[i-1] <= '9' {
			lvl[i] = 2
		}
	}
	for i := range shaped {
		if lvl[i] != 0 {
			continue
		}
		lo, hi := 0, 0
		for j := i - 1; j >= 0; j-- {
			if lvl[j] != 0 {
				lo = lvl[j]
				break
			}
		}
		for j := i + 1; j < n; j++ {
			if lvl[j] != 0 {
				hi = lvl[j]
				break
			}
		}
		lvl[i] = 1 // edges and mixed sides: the base direction
		if lo == hi {
			lvl[i] = lo
		}
	}
	vis := make([]rune, n)
	side := make([]int, n)
	for i := range shaped { // level 1: the whole line reverses
		vis[i] = shaped[n-1-i]
		side[i] = lvl[n-1-i]
	}
	for a := 0; a < n; { // level 2 islands: back to reading order
		if side[a] != 2 {
			a++
			continue
		}
		b := a
		for b+1 < n && side[b+1] == 2 {
			b++
		}
		for x, y := a, b; x < y; x, y = x+1, y-1 {
			vis[x], vis[y] = vis[y], vis[x]
		}
		a = b + 1
	}
	for i := range vis { // mirror the RTL characters
		if side[i] == 1 {
			vis[i] = mirror(vis[i])
		}
	}
	return strings.TrimSpace(string(vis))
}

func isRTL(c rune) bool {
	return (c >= 0x0590 && c <= 0x08FF) || (c >= 0xFB1D && c <= 0xFDFF) || (c >= 0xFE70 && c <= 0xFEFF)
}

func isLTRStrong(c rune) bool {
	return !isRTL(c) && (unicode.IsLetter(c) || unicode.IsDigit(c))
}

func mirror(c rune) rune {
	switch c {
	case '(':
		return ')'
	case ')':
		return '('
	case '[':
		return ']'
	case ']':
		return '['
	case '{':
		return '}'
	case '}':
		return '{'
	case '<':
		return '>'
	case '>':
		return '<'
	}
	return c
}

// forms: isolated, final, initial, medial (0 = none)
var arForms = map[rune][4]rune{
	0x0621: {0xFE80, 0, 0, 0},
	0x0622: {0xFE81, 0xFE82, 0, 0},
	0x0623: {0xFE83, 0xFE84, 0, 0},
	0x0624: {0xFE85, 0xFE86, 0, 0},
	0x0625: {0xFE87, 0xFE88, 0, 0},
	0x0626: {0xFE89, 0xFE8A, 0xFE8B, 0xFE8C},
	0x0627: {0xFE8D, 0xFE8E, 0, 0},
	0x0628: {0xFE8F, 0xFE90, 0xFE91, 0xFE92},
	0x0629: {0xFE93, 0xFE94, 0, 0},
	0x062A: {0xFE95, 0xFE96, 0xFE97, 0xFE98},
	0x062B: {0xFE99, 0xFE9A, 0xFE9B, 0xFE9C},
	0x062C: {0xFE9D, 0xFE9E, 0xFE9F, 0xFEA0},
	0x062D: {0xFEA1, 0xFEA2, 0xFEA3, 0xFEA4},
	0x062E: {0xFEA5, 0xFEA6, 0xFEA7, 0xFEA8},
	0x062F: {0xFEA9, 0xFEAA, 0, 0},
	0x0630: {0xFEAB, 0xFEAC, 0, 0},
	0x0631: {0xFEAD, 0xFEAE, 0, 0},
	0x0632: {0xFEAF, 0xFEB0, 0, 0},
	0x0633: {0xFEB1, 0xFEB2, 0xFEB3, 0xFEB4},
	0x0634: {0xFEB5, 0xFEB6, 0xFEB7, 0xFEB8},
	0x0635: {0xFEB9, 0xFEBA, 0xFEBB, 0xFEBC},
	0x0636: {0xFEBD, 0xFEBE, 0xFEBF, 0xFEC0},
	0x0637: {0xFEC1, 0xFEC2, 0xFEC3, 0xFEC4},
	0x0638: {0xFEC5, 0xFEC6, 0xFEC7, 0xFEC8},
	0x0639: {0xFEC9, 0xFECA, 0xFECB, 0xFECC},
	0x063A: {0xFECD, 0xFECE, 0xFECF, 0xFED0},
	0x0641: {0xFED1, 0xFED2, 0xFED3, 0xFED4},
	0x0642: {0xFED5, 0xFED6, 0xFED7, 0xFED8},
	0x0643: {0xFED9, 0xFEDA, 0xFEDB, 0xFEDC},
	0x0644: {0xFEDD, 0xFEDE, 0xFEDF, 0xFEE0},
	0x0645: {0xFEE1, 0xFEE2, 0xFEE3, 0xFEE4},
	0x0646: {0xFEE5, 0xFEE6, 0xFEE7, 0xFEE8},
	0x0647: {0xFEE9, 0xFEEA, 0xFEEB, 0xFEEC},
	0x0648: {0xFEED, 0xFEEE, 0, 0},
	0x0649: {0xFEEF, 0xFEF0, 0, 0},
	0x064A: {0xFEF1, 0xFEF2, 0xFEF3, 0xFEF4},
	0x067E: {0xFB56, 0xFB57, 0xFB58, 0xFB59},
	0x0686: {0xFB7A, 0xFB7B, 0xFB7C, 0xFB7D},
	0x0698: {0xFB8A, 0xFB8B, 0, 0},
	0x06A9: {0xFB8E, 0xFB8F, 0xFB90, 0xFB91},
	0x06AF: {0xFB92, 0xFB93, 0xFB94, 0xFB95},
	0x06CC: {0xFBFC, 0xFBFD, 0xFBFE, 0xFBFF},
}

var lamAlef = map[rune][2]rune{ // isolated, final
	0x0622: {0xFEF5, 0xFEF6},
	0x0623: {0xFEF7, 0xFEF8},
	0x0625: {0xFEF9, 0xFEFA},
	0x0627: {0xFEFB, 0xFEFC},
}

func isHaraka(c rune) bool { return c >= 0x064B && c <= 0x065F || c == 0x0670 }

func shapeArabic(r []rune) []rune {
	joinsNext := func(c rune) bool { f, ok := arForms[c]; return ok && f[2] != 0 }
	// joinsPrev: the letter has a final form, so it can join the letter
	// before it. The hamza (U+0621) has none: it never joins either side.
	joinsPrev := func(c rune) bool { f, ok := arForms[c]; return ok && f[1] != 0 }
	// neighbours, skipping diacritics
	prev := func(i int) rune {
		for i--; i >= 0; i-- {
			if !isHaraka(r[i]) {
				return r[i]
			}
		}
		return 0
	}
	next := func(i int) (rune, int) {
		for i++; i < len(r); i++ {
			if !isHaraka(r[i]) {
				return r[i], i
			}
		}
		return 0, -1
	}
	var out []rune
	for i := 0; i < len(r); i++ {
		c := r[i]
		f, ok := arForms[c]
		if !ok {
			out = append(out, c)
			continue
		}
		fromPrev := joinsNext(prev(i)) && f[1] != 0
		nc, ni := next(i)
		if c == 0x0644 {
			if la, ok := lamAlef[nc]; ok {
				if fromPrev {
					out = append(out, la[1])
				} else {
					out = append(out, la[0])
				}
				i = ni
				continue
			}
		}
		toNext := f[2] != 0 && joinsPrev(nc)
		switch {
		case fromPrev && toNext:
			out = append(out, f[3])
		case fromPrev:
			out = append(out, f[1])
		case toNext:
			out = append(out, f[2])
		default:
			out = append(out, f[0])
		}
	}
	return out
}

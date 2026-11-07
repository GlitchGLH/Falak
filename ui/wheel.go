package ui

import (
	"fmt"
	"image"
	"image/color"
	stddraw "image/draw"
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// Accent is the brand blue (#007AFF).
var Accent = color.NRGBA{0, 122, 255, 255}
var errRed = color.NRGBA{255, 59, 48, 255}

// darkGlass is the tint of the dark theme's glass (like the reference's
// dark wheel: a deep, slightly warm near-black over the blurred desktop).
var darkGlass = color.NRGBA{22, 20, 19, 255}

// textColor is the label color of the current theme.
func (w *Wheel) textColor(a uint8) color.NRGBA {
	if w.Dark {
		return color.NRGBA{246, 246, 248, a}
	}
	return color.NRGBA{29, 29, 31, a}
}

// Logical geometry (multiplied by Scale at render time).
// Segment glass: how much the desktop under the segments is blurred
// (logical px) and how much white veil lies over it.
var (
	SegBlur     = 80.0 // sample H2
	SegVeil     = 0.72
	SegVeilDark = 0.80
	SegOpacity  = 0.90 // segments let a little of what is behind them show through
	SegSeeBlur  = 6.0  // Gaussian radius for what shows through (0 = the real, sharp desktop)
)

const (
	LogicalSize = 360.0
	rBase       = 156.0 // glass strip under the segments
	rOut        = 147.0 // outer end of the segments (shortened to the user's measurement)
	rIn         = 60.5
	rCenter     = 56.0
	rHole       = 52.3 // the strip is open inside this radius
	rOpen       = 58.0 // size reference for the capsule and the working disc
	pillW       = 87.7 // the capsule, measured on the reference
	pillH       = 40.5
	halfGap     = 3.4
	corner      = 20.5
)

type Item struct {
	ID      string
	Label   string
	Icon    string // non-empty for tool items
	Enabled bool
}

type Phase int

const (
	PhaseOpen Phase = iota
	PhaseWorking
	PhaseDone
	PhaseError
	PhaseClosing
)

type Wheel struct {
	Scale  float64
	W, H   int
	CX, CY float64 // wheel center in window pixels
	BG     *image.RGBA
	Sharp  *image.RGBA // lightly blurred desktop, refracted by the strip
	Dark   bool
	Fonts  *Fonts

	Items     []Item
	thumb     *image.RGBA
	thumbBlur *image.RGBA
	progress  float64 // 0..1 while working, -1 when not known yet
	progShown float64 // the bar eases toward progress
	ext       string
	sizeText  string
	note      string

	px, py     float64
	hasPointer bool
	hover      []float64
	hoverV     []float64           // spring velocity of hover
	liftA      []float64           // hover lift, 0 at rest .. 1 fully lifted
	liftV      []float64           // spring velocity of the lift
	hoverIdx   int                 // hovered enabled segment, kept with hysteresis
	shadows    map[int]*image.RGBA // blurred silhouettes for the lift shadow
	shadowAt   map[int]image.Point // and where each crop belongs
	segScratch map[int]*image.RGBA // scratch layer for one lifted segment
	selected   int

	opened, ringAt, phaseAt, last time.Time
	phase                         Phase
	closeAt                       time.Time
	revealed                      bool
	darkBG                        bool

	img       *image.RGBA
	dc        *gg.Context
	layer     *image.RGBA
	prevLayer *image.RGBA
	fadeBuf   *image.RGBA
	closeDur  float64
	glassMu   sync.Mutex
	glass     map[int]*glassJob      // glass by segment count, built ahead
	layers    map[string]*image.RGBA // ready-made layers by item set
	bares     map[string]*image.RGBA // the same glass without labels
	prep      chan struct{}          // closed when segBG/seeBG are ready
	segBG     *image.RGBA            // Gaussian-blurred desktop under the segments
	seeBG     *image.RGBA            // lightly blurred desktop that shows through
	labelsOf  map[string]*image.RGBA // and the labels alone
}

func New(scale float64, w, h int, cx, cy float64, bg *image.RGBA, dark bool, fonts *Fonts, now time.Time) *Wheel {
	wh := &Wheel{Scale: scale, W: w, H: h, CX: cx, CY: cy, BG: bg, Dark: dark, Fonts: fonts,
		opened: now, ringAt: now, last: now, selected: -1, hoverIdx: -1}
	wh.img = image.NewRGBA(image.Rect(0, 0, w, h))
	wh.dc = gg.NewContextForRGBA(wh.img)
	if bg != nil {
		var sum, n float64
		for i := 0; i+3 < len(bg.Pix); i += 4 * 37 {
			sum += 0.2126*float64(bg.Pix[i]) + 0.7152*float64(bg.Pix[i+1]) + 0.0722*float64(bg.Pix[i+2])
			n++
		}
		wh.darkBG = n > 0 && sum/n < 100
	}
	return wh
}

// SetItems replaces the ring contents. animate restarts the ring entrance
// (used when switching between formats and tools).
func (w *Wheel) SetItems(items []Item, animate bool, now time.Time) {
	w.Items = items
	w.hover = make([]float64, len(items))
	w.hoverV = make([]float64, len(items))
	w.liftA = make([]float64, len(items))
	w.liftV = make([]float64, len(items))
	w.hoverIdx = -1
	if animate {
		w.ringAt = now
		if w.layer != nil {
			w.prevLayer = w.layer
		}
	}
	w.layer = w.layers[itemsKey(items)]
}

func itemsKey(items []Item) string {
	var b strings.Builder
	for _, it := range items {
		b.WriteString(it.ID)
		if it.Enabled {
			b.WriteByte('+')
		}
		b.WriteByte('|')
	}
	return b.String()
}

// Prepare starts computing the blurred backgrounds the glass needs, in the
// background, so the first frame of the pop-open does not wait for them.
// Call it as soon as Sharp is set; calling it again does nothing.
func (w *Wheel) Prepare() {
	if w.prep != nil {
		return
	}
	w.prep = make(chan struct{})
	sharp, s := w.Sharp, w.Scale
	if sharp == nil {
		sharp = w.BG
	}
	go func() {
		defer close(w.prep)
		if sharp == nil {
			return
		}
		var wg sync.WaitGroup
		if SegBlur > 0 {
			wg.Add(1)
			go func() { defer wg.Done(); w.segBG = GaussianBlur(sharp, SegBlur*s) }()
		}
		if SegSeeBlur > 0 && SegOpacity < 1 {
			wg.Add(1)
			go func() { defer wg.Done(); w.seeBG = GaussianBlur(sharp, SegSeeBlur*s) }()
		}
		wg.Wait()
	}()
}

type glassJob struct {
	done chan struct{}
	img  *image.RGBA
}

// glassFor returns the glass for n segments, building it if no one has.
func (w *Wheel) glassFor(n int) *image.RGBA {
	w.glassMu.Lock()
	if w.glass == nil {
		w.glass = map[int]*glassJob{}
	}
	j := w.glass[n]
	if j == nil {
		j = &glassJob{done: make(chan struct{})}
		w.glass[n] = j
		w.glassMu.Unlock()
		j.img = w.buildGlass(n)
		close(j.done)
		return j.img
	}
	w.glassMu.Unlock()
	<-j.done
	return j.img
}

// PrebuildGlass builds the glass for likely segment counts in the
// background, right after Prepare, so the first frame finds it ready.
func (w *Wheel) PrebuildGlass(counts ...int) {
	for _, n := range counts {
		go w.glassFor(n)
	}
}

// Prebuild prepares the layer for another item set (the tools wheel) while
// nothing is moving, so switching with Alt/Ctrl/Tab never stalls a frame.
func (w *Wheel) Prebuild(items []Item) {
	key := itemsKey(items)
	if w.layers[key] != nil {
		return
	}
	saved, savedLayer := w.Items, w.layer
	w.Items = items
	w.buildLayer()
	w.Items, w.layer = saved, savedLayer
}

// Prebuilt reports whether a layer for items is ready.
func (w *Wheel) Prebuilt(items []Item) bool { return w.layers[itemsKey(items)] != nil }

func (w *Wheel) SetFileInfo(thumb *image.RGBA, ext, size, note string) {
	if thumb != w.thumb {
		w.thumbBlur = nil
		if thumb != nil {
			w.thumbBlur = FrostBackground(thumb, 10*w.Scale, 1.2)
		}
	}
	w.thumb, w.ext, w.sizeText, w.note = thumb, ext, size, note
}

// ThumbPixels is the diameter a thumbnail should be prepared at.
// SetProgress sets the text shown on the capsule while working ("37%").
func (w *Wheel) SetProgress(p float64) { w.progress = p }

func (w *Wheel) ThumbPixels() int { return int(rCenter*2*w.Scale) + 2 }

func (w *Wheel) SetPointer(x, y float64) { w.px, w.py, w.hasPointer = x, y, true }

// Reveal starts the entrance animation. Until then the wheel is an almost
// invisible disc (alpha 1/255) that only exists so the drag can enter it;
// this keeps Shift-drags that carry no files (text selection, etc.) invisible.
func (w *Wheel) Reveal(now time.Time) {
	if !w.revealed {
		w.revealed = true
		w.opened, w.ringAt, w.last = now, now, now
	}
}

func (w *Wheel) Revealed() bool { return w.revealed }

func (w *Wheel) Phase() Phase { return w.phase }

// ItemAt returns the index of the item under the window-relative point, or -1.
func (w *Wheel) ItemAt(x, y float64) int {
	n := len(w.Items)
	if n == 0 {
		return -1
	}
	dx, dy := x-w.CX, y-w.CY
	d := math.Hypot(dx, dy) / w.Scale
	if d < rIn-6 || d > rBase+8 {
		return -1
	}
	a := math.Atan2(dy, dx) + math.Pi/2 + math.Pi/float64(n)
	for a < 0 {
		a += 2 * math.Pi
	}
	return int(a/(2*math.Pi/float64(n))) % n
}

// EnabledAt is ItemAt restricted to enabled items.
func (w *Wheel) EnabledAt(x, y float64) int {
	i := w.ItemAt(x, y)
	if i >= 0 && w.Items[i].Enabled && w.phase == PhaseOpen {
		return i
	}
	return -1
}

func (w *Wheel) StartWorking(idx int, now time.Time) {
	w.selected, w.phase, w.phaseAt = idx, PhaseWorking, now
}

func (w *Wheel) Finish(ok bool, now time.Time) {
	if ok {
		w.phase = PhaseDone
	} else {
		w.phase = PhaseError
	}
	w.phaseAt = now
}

func (w *Wheel) Close(now time.Time) {
	if w.phase != PhaseClosing {
		w.phase, w.closeAt = PhaseClosing, now
	}
}

// CloseFast makes the wheel vanish quickly (after a drop, before the
// progress window pops up).
func (w *Wheel) CloseFast(now time.Time) {
	w.closeDur = 0.08
	w.phase, w.closeAt = PhaseClosing, now
}

func (w *Wheel) closeSecs() float64 {
	if w.closeDur > 0 {
		return w.closeDur
	}
	return 0.16
}

// Closed reports whether the closing animation has finished.
func (w *Wheel) Closed(now time.Time) bool {
	if w.phase == PhaseClosing && !w.revealed {
		return true
	}
	return w.phase == PhaseClosing && now.Sub(w.closeAt).Seconds() > w.closeSecs()+0.01
}

// Animating reports whether frames still change without new input.
func (w *Wheel) Animating(now time.Time) bool {
	if !w.revealed {
		return false
	}
	if now.Sub(w.opened) < 450*time.Millisecond || now.Sub(w.ringAt) < 450*time.Millisecond {
		return true
	}
	switch w.phase {
	case PhaseWorking, PhaseClosing:
		return true
	case PhaseDone, PhaseError:
		if now.Sub(w.phaseAt) < 500*time.Millisecond {
			return true
		}
	}
	for i := range w.hover {
		if math.Abs(w.hover[i]-w.targetHover(i)) > 0.001 || math.Abs(w.hoverV[i]) > 0.002 ||
			math.Abs(w.liftA[i]-w.liftTarget(i)) > 0.001 || math.Abs(w.liftV[i]) > 0.002 {
			return true
		}
	}
	return false
}

func (w *Wheel) targetHover(i int) float64 {
	if w.phase == PhaseWorking || w.phase == PhaseDone || w.phase == PhaseError {
		if i == w.selected {
			return 1
		}
		return 0
	}
	if w.phase == PhaseOpen && w.hasPointer && w.hoverIdx == i {
		return 1
	}
	return 0
}

// liftTarget is 1 only while the pointer rests on this enabled segment in
// the open phase - the only time a segment lifts.
func (w *Wheel) liftTarget(i int) float64 {
	if w.phase == PhaseOpen && w.hasPointer && w.hoverIdx == i && i < len(w.Items) && w.Items[i].Enabled {
		return 1
	}
	return 0
}

// hoverAt is the enabled segment under the pointer with a small hysteresis:
// the hover stays on the current segment until the pointer has moved clearly
// past its border, so resting on a border never flickers or jitters.
func (w *Wheel) hoverAt(x, y float64) int {
	cand := w.EnabledAt(x, y)
	if cand == w.hoverIdx {
		return cand
	}
	n := len(w.Items)
	if w.hoverIdx < 0 || w.hoverIdx >= n {
		return cand
	}
	dx, dy := x-w.CX, y-w.CY
	r := math.Hypot(dx, dy) / w.Scale
	if r < rIn-6 || r > rBase+8 {
		return cand // off the ring: switch at once
	}
	span := 2 * math.Pi / float64(n)
	a := math.Atan2(dy, dx)
	mid := -math.Pi/2 + float64(w.hoverIdx)*span
	da := a - mid
	for da > math.Pi {
		da -= 2 * math.Pi
	}
	for da < -math.Pi {
		da += 2 * math.Pi
	}
	const hyst = 6 // logical px to move past a border before switching
	if math.Abs(da) > span/2 && (math.Abs(da)-span/2)*r < hyst {
		return w.hoverIdx
	}
	return cand
}

// springStep advances a critically damped spring (x” = -2*om*x' -
// om*om*(x-target)) by h: no overshoot, no visible bounce.
func springStep(x, v, target, om, h float64) (float64, float64) {
	v += (-2*om*v - om*om*(x-target)) * h
	x += v * h
	return x, v
}

// Spring rates: a critically damped spring is within ~5% of its target
// after ~4.7/omega seconds - about 220 ms to lift (omega 21.5), 180 ms to
// settle (omega 26.3).
const (
	liftOmega   = 21.5
	settleOmega = 26.3
)

func easeOutBack(t float64) float64 {
	t = clampF(t, 0, 1)
	const c1, c3 = 1.4, 2.4
	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func easeOutCubic(t float64) float64 { t = clampF(t, 0, 1); return 1 - math.Pow(1-t, 3) }

func clampF(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func withAlpha(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(clampF(float64(c.A)*a, 0, 255))
	return c
}

// alphaPattern samples the frosted background with an opacity multiplier.
type alphaPattern struct {
	img *image.RGBA
	a   float64
}

func (p alphaPattern) ColorAt(x, y int) color.Color {
	if !(image.Point{x, y}.In(p.img.Rect)) {
		return color.Transparent
	}
	i := p.img.PixOffset(x, y)
	s := p.img.Pix[i : i+4 : i+4]
	if p.a >= 1 {
		return color.RGBA{s[0], s[1], s[2], 255}
	}
	return color.RGBA{uint8(float64(s[0]) * p.a), uint8(float64(s[1]) * p.a), uint8(float64(s[2]) * p.a), uint8(255 * p.a)}
}

// Render draws the current frame and returns the premultiplied RGBA buffer.
//
// Visual structure (back to front):
//  1. a soft drop shadow,
//  2. a large frosted-glass base disc with a bright rim,
//  3. eight (or so) rounded glass segments sitting on the disc,
//  4. the file thumbnail filling the center, with the size on a small
//     frosted-glass capsule.
func (w *Wheel) Render(now time.Time) *image.RGBA {
	dt := now.Sub(w.last).Seconds()
	w.last = now
	if dt > 0.05 {
		dt = 0.05
	}
	if dt < 0 {
		dt = 0
	}
	if w.hasPointer {
		w.hoverIdx = w.hoverAt(w.px, w.py)
	}
	// The hover colour and the lift are critically damped springs, stepped
	// at a fixed 240 Hz sub-rate, so the motion is the same at 60, 120 or
	// 144 Hz and stays stable even on a slow frame.
	if n := len(w.hover); n > 0 && dt > 0 {
		steps := int(math.Ceil(dt * 240))
		h := dt / float64(steps)
		for j := 0; j < steps; j++ {
			for i := 0; i < n; i++ {
				tgt := w.targetHover(i)
				om := liftOmega
				if tgt < w.hover[i] {
					om = settleOmega
				}
				w.hover[i], w.hoverV[i] = springStep(w.hover[i], w.hoverV[i], tgt, om, h)
				tgt = w.liftTarget(i)
				om = liftOmega
				if tgt < w.liftA[i] {
					om = settleOmega
				}
				w.liftA[i], w.liftV[i] = springStep(w.liftA[i], w.liftV[i], tgt, om, h)
			}
		}
	}

	s := w.Scale
	dc := w.dc
	clear(w.img.Pix)
	dc.Identity()
	dc.ResetClip()
	if !w.revealed || (w.phase == PhaseClosing && now.Sub(w.closeAt).Seconds() > w.closeSecs()+0.01) {
		if w.phase != PhaseClosing {
			dc.SetColor(color.RGBA{0, 0, 0, 1})
			dc.DrawCircle(w.CX, w.CY, (rBase+4)*s)
			dc.Fill()
		}
		return w.img
	}
	if w.layer == nil {
		w.buildLayer()
		// Building takes a few ms: start the pop-open clock after it, so the
		// first frame really is the first step of the animation.
		if now.Sub(w.opened) < 60*time.Millisecond {
			w.opened = now
		}
	}

	// Entrance, measured frame by frame on the reference video: the glass
	// pops open from 40% of its size on an exponential curve (time constant
	// 35 ms, no overshoot, no turn), is almost fully opaque from the first
	// frame, and the labels fade in only once the shape has formed
	// (from ~55 ms to ~115 ms).
	tOpen := now.Sub(w.opened).Seconds()
	globalA := math.Min(1, 0.5+tOpen*21) // ~50% on the first frame, full by the third
	sc := 1 - 0.60*math.Exp(-tOpen/0.035)
	if tOpen > 0.3 {
		sc = 1
	}
	rot := 0.0
	labelA := clampF((tOpen-0.050)/0.05, 0, 1) // fade in over frames 4 to 7
	if w.phase == PhaseClosing {
		tc := now.Sub(w.closeAt).Seconds() / w.closeSecs()
		globalA *= 1 - easeOutCubic(tc)
		sc *= 1 - 0.08*easeOutCubic(tc)
	}
	cx, cy := w.CX, w.CY

	// Switching formats <-> tools: cross-fade the two wheels with a turn.
	tRing := now.Sub(w.ringAt).Seconds()
	if w.prevLayer != nil && tRing < 0.24 {
		p := easeOutCubic(tRing / 0.24)
		drawLayer(w.img, w.prevLayer, cx, cy, sc, rot+0.18*p, 1-p, true)
		drawLayer(w.img, w.layer, cx, cy, sc, rot-0.18*(1-p), p, false)
	} else if key := itemsKey(w.Items); labelA < 1 && w.bares[key] != nil {
		w.prevLayer = nil
		// glass + fading labels combined first, then one scaled draw
		if w.fadeBuf == nil || w.fadeBuf.Rect != w.img.Rect {
			w.fadeBuf = image.NewRGBA(w.img.Rect)
		}
		src := w.bares[key]
		if labelA > 0.003 {
			copy(w.fadeBuf.Pix, src.Pix)
			blendOver(w.fadeBuf, w.labelsOf[key], labelA)
			src = w.fadeBuf
		}
		drawLayer(w.img, src, cx, cy, sc, rot, 1, true)
	} else {
		w.prevLayer = nil
		drawLayer(w.img, w.layer, cx, cy, sc, rot, 1, true)
	}

	// Parts that change, drawn on top with the same transform.
	dc.RotateAbout(rot, cx, cy)
	dc.ScaleAbout(sc, sc, cx, cy)
	n := len(w.Items)
	span := 2 * math.Pi / float64(max(n, 1))
	// Active segments, lowest lift first, so the segment lifting is drawn
	// on top of the one settling beneath it. Each one lifts - scales about
	// its own centre and pushes outward along its centre angle - with its
	// blue hover colour fading with the same spring. The lift can never be
	// clipped by the window edges: the window is LogicalSize square with
	// the wheel in its middle (180 logical px from the centre to every
	// edge), while a lifted segment reaches at most rOut*1.055 + 3.6 plus
	// the ~10 px of shadow bleed, i.e. about 169 logical px - and the
	// window itself is always kept fully inside the work area.
	type overlay struct {
		i      int
		h, dim float64
	}
	var acts []overlay
	for i := 0; i < n; i++ {
		h := w.hover[i]
		dim := 0.0
		if w.phase != PhaseOpen && w.phase != PhaseClosing && i != w.selected {
			dim = easeOutCubic(now.Sub(w.phaseAt).Seconds() / 0.2)
		}
		if h < 0.01 && dim < 0.01 {
			continue
		}
		acts = append(acts, overlay{i, h, dim})
	}
	sort.Slice(acts, func(a, b int) bool { return acts[a].h < acts[b].h })
	lr := (rIn + rOut) / 2 * s
	for _, ac := range acts {
		i, h := ac.i, ac.h
		it := w.Items[i]
		mid := -math.Pi/2 + float64(i)*span
		a0, a1 := mid-span/2, mid+span/2
		if ac.dim > 0 {
			w.segmentPath(dc, cx, cy, a0, a1)
			if w.Dark {
				dc.SetColor(color.NRGBA{0, 0, 0, uint8(120 * ac.dim)})
			} else {
				dc.SetColor(color.NRGBA{255, 255, 255, uint8(140 * ac.dim)})
			}
			dc.Fill()
			continue
		}
		lift := w.liftA[i]
		if !it.Enabled || w.phase != PhaseOpen || !w.hasPointer {
			lift = 0 // only the pointer hover lifts; working just colours
		}
		if lift > 0.01 {
			w.drawSegShadow(i, lift)
		}
		if lift > 0.0005 {
			// the whole segment lifts as one: shape, highlight, icon and
			// label are rendered together on a scratch layer and drawn
			// scaled about the segment's own centre, pushed outward
			w.drawLiftedSegment(i, h, mid, a0, a1, lift)
			continue
		}
		w.segmentPath(dc, cx, cy, a0, a1)
		if !it.Enabled {
			dc.SetColor(color.NRGBA{255, 255, 255, uint8(map[bool]float64{false: 55, true: 22}[w.Dark] * h)})
			dc.Fill()
			continue
		}
		g := gg.NewLinearGradient(cx, cy-rOut*s, cx, cy+rOut*s)
		g.AddColorStop(0, withAlpha(color.NRGBA{40, 145, 255, 255}, h))
		g.AddColorStop(1, withAlpha(color.NRGBA{0, 110, 240, 255}, h))
		dc.SetFillStyle(g)
		dc.FillPreserve()
		dc.SetColor(color.NRGBA{0, 40, 120, uint8(40 * h)})
		dc.SetLineWidth(1 * s)
		dc.Stroke()
		tx, ty := cx+math.Cos(mid)*lr, cy+math.Sin(mid)*lr
		tc := color.NRGBA{255, 255, 255, uint8(255 * h)}
		if it.Icon != "" {
			drawIcon(dc, it.Icon, tx, ty-9*s, 19*s, tc)
			w.text(it.Label, 8.8*s, true, tx, ty+13*s, tc)
		} else {
			w.text(it.Label, 15*s, false, tx, ty, tc)
		}
	}

	textCol := w.textColor(240)
	// 4. Center stays open: Windows' own drag image of the file shows
	// through it. Only the size floats on a small glass capsule.
	cr := (rHole - 2) * s
	// The capsule shows the hovered option (like the reference), otherwise
	// the file size; while working it shows the progress.
	pillText := w.sizeText
	best := 0.5
	for i, h := range w.hover {
		if h > best && w.Items[i].Enabled {
			best, pillText = h, w.Items[i].Label
		}
	}
	if w.phase == PhaseOpen || w.phase == PhaseClosing {
		if pillText != "" {
			w.glassPill(pillText, cx, cy)
		}
	}
	if w.note != "" && (w.phase == PhaseOpen || w.phase == PhaseClosing) {
		w.text(w.note, 7.5*s, true, cx, cy+26*s, withAlpha(textCol, 0.8))
	}

	tp := now.Sub(w.phaseAt).Seconds()
	switch w.phase {
	case PhaseWorking:
		// Only the percentage and a thin bar that fills in blue.
		a := easeOutCubic(tp / 0.2)
		dc.DrawCircle(cx, cy, (rHole-1)*s)
		dc.SetFillStyle(alphaPattern{w.BG, a})
		dc.FillPreserve()
		if w.Dark {
			dc.SetColor(withAlpha(darkGlass, 215.0/255*a))
		} else {
			dc.SetColor(color.NRGBA{255, 255, 255, uint8(215 * a)})
		}
		dc.Fill()
		p := math.Max(w.progress, 0)
		if p < w.progShown {
			w.progShown = p
		}
		w.progShown += (p - w.progShown) * (1 - math.Exp(-dt/0.05))
		w.text(fmt.Sprintf("%d%%", int(p*100)), 17*s, true, cx, cy-7*s, withAlpha(w.textColor(240), a))
		bw, bh := 58*s, 5*s
		bx, by := cx-bw/2, cy+12*s
		dc.DrawRoundedRectangle(bx, by, bw, bh, bh/2)
		if w.Dark {
			dc.SetColor(color.NRGBA{255, 255, 255, uint8(38 * a)})
		} else {
			dc.SetColor(color.NRGBA{0, 0, 0, uint8(22 * a)})
		}
		dc.Fill()
		if fw := bw * clampF(w.progShown, 0, 1); fw > 0.5 {
			dc.DrawRoundedRectangle(bx, by, math.Max(fw, bh), bh, bh/2)
			dc.SetColor(withAlpha(Accent, a))
			dc.Fill()
		}
	case PhaseDone, PhaseError:
		p := easeOutBack(tp / 0.3)
		c := Accent
		if w.phase == PhaseError {
			c = errRed
		}
		dc.SetColor(c)
		dc.DrawCircle(cx, cy, cr*clampF(p, 0, 1.2))
		dc.Fill()
		dc.SetColor(color.White)
		dc.SetLineWidth(5.5 * s)
		dc.SetLineCap(gg.LineCapRound)
		dc.SetLineJoin(gg.LineJoinRound)
		q := easeOutCubic((tp - 0.1) / 0.25)
		if q > 0 {
			if w.phase == PhaseDone {
				drawPartialPolyline(dc, cx, cy, s, [][2]float64{{-18, -1}, {-6, 12}, {19, -14}}, q)
			} else {
				drawPartialPolyline(dc, cx, cy, s, [][2]float64{{-13, -13}, {13, 13}}, q)
				drawPartialPolyline(dc, cx, cy, s, [][2]float64{{13, -13}, {-13, 13}}, q)
			}
		}
	}

	if globalA < 1 {
		m := uint32(globalA * 256)
		for i, v := range w.img.Pix {
			w.img.Pix[i] = uint8(uint32(v) * m >> 8)
		}
	}
	return w.img
}

// buildLayer draws the full wheel at rest (no hover, no animation) once.
// Frames then only transform this image and draw the small parts that
// change on top, which keeps the animation smooth.
// buildGlass draws the glass of a wheel with n segments (no labels). It
// only reads the wheel's backgrounds, so several can be built at once.
func (w *Wheel) buildGlass(n int) *image.RGBA {
	layer := image.NewRGBA(w.img.Rect)
	dc := gg.NewContextForRGBA(layer)
	s := w.Scale
	cx, cy := w.CX, w.CY

	// Always the light glass of the reference, with black text, on any
	// desktop. Measured on a white (244) desktop: strip ~208, segments ~227,
	// thin grey edges (~190) with small white glints, clear center.
	veilCol := color.NRGBA{255, 255, 255, 255}
	if w.Dark { // dark glass: the same layers, tinted near-black
		veilCol = darkGlass
	}
	mix := func(veil, shade float64) {
		dc.SetColor(withAlpha(veilCol, veil))
		dc.FillPreserve()
		dc.SetColor(color.NRGBA{0, 0, 0, uint8(255 * shade)})
		dc.FillPreserve()
	}
	lobe := func(angle, center, power float64) float64 {
		return math.Pow(math.Max(0, math.Cos(angle-center)), power)
	}
	tl, br := -3*math.Pi/4, math.Pi/4
	glint := func(a float64) float64 { return 0.08 + 0.92*lobe(a, tl, 8) + 0.55*lobe(a, br, 12) }

	// Strip: watery glass that refracts what is under it.
	sharp := w.Sharp
	if sharp == nil {
		sharp = w.BG
	}
	w.Prepare()
	<-w.prep
	segSrc := sharp
	if w.segBG != nil {
		segSrc = w.segBG
	}
	dc.SetFillRule(gg.FillRuleEvenOdd)
	dc.DrawCircle(cx, cy, rBase*s)
	dc.DrawCircle(cx, cy, rHole*s)
	rp := refractPattern{sharp, cx, cy, rHole * s, rBase * s, s}
	fastPatternFill(layer, func(m *gg.Context) {
		m.SetFillRule(gg.FillRuleEvenOdd)
		m.DrawCircle(cx, cy, rBase*s)
		m.DrawCircle(cx, cy, rHole*s)
	}, rp.img, rp.at)
	// The dark theme is the very same glass (veils, glints, edges); only
	// the tint of the veil differs.
	ringVeil, segVeil, segShade, ringShade := 0.30, SegVeil, 0.06, 0.16
	if w.darkBG {
		ringVeil, segVeil, segShade = 0.42, SegVeilDark, 0.04
	}
	edge := color.NRGBA{0, 0, 0, 40}
	glintA := 1.0
	mix(ringVeil, ringShade)
	dc.ClearPath()
	dc.SetFillRule(gg.FillRuleWinding)
	// thin grey edges with a thin white glint just inside them
	for _, e := range []struct{ r, glintR float64 }{{rBase - 0.5, rBase - 1.6}} {
		dc.NewSubPath()
		dc.DrawCircle(cx, cy, e.r*s)
		dc.SetColor(edge)
		dc.SetLineWidth(1 * s)
		dc.Stroke()
		dc.NewSubPath()
		dc.DrawCircle(cx, cy, e.glintR*s)
		dc.SetStrokeStyle(conicPattern{cx, cy, 0.95 * glintA, glint})
		dc.SetLineWidth(1 * s)
		dc.Stroke()
	}
	// Keep the open center hit-testable (nearly invisible) so drops there
	// cancel instead of falling through to the desktop.
	dc.SetColor(color.RGBA{0, 0, 0, 2})
	dc.DrawCircle(cx, cy, rHole*s)
	dc.Fill()

	// Segments: white translucent glass, thin grey edge, small white glints.
	// They are drawn on their own image and laid over the strip at
	// SegOpacity, with the strip removed underneath, so the desktop itself
	// shows faintly through them (not the darker strip).
	stripImg := layer
	segImg := image.NewRGBA(layer.Rect)
	dc = gg.NewContextForRGBA(segImg)
	span := 2 * math.Pi / float64(n)
	for i := 0; i < n; i++ {
		mid := -math.Pi/2 + float64(i)*span
		a0, a1 := mid-span/2, mid+span/2
		build := func() { w.segmentPath(dc, cx, cy, a0, a1) }

		build()
		// Same watery glass as the ring: what is underneath stays visible,
		// bent like through a lens, under a light white veil.
		sp := segRefractPattern{segSrc, cx, cy, rIn * s, rOut * s, a0, a1, s}
		fastPatternFill(segImg, func(m *gg.Context) { w.segmentPath(m, cx, cy, a0, a1) }, sp.img, sp.at)
		mix(segVeil, segShade)
		// thin white glints inside the edge, only toward the light
		dc.Clip()
		build()
		mx, my := cx+math.Cos(mid)*(rIn+rOut)/2*s, cy+math.Sin(mid)*(rIn+rOut)/2*s
		g := gg.NewLinearGradient(mx-60*s, my-60*s, mx+60*s, my+60*s)
		g.AddColorStop(0, withAlpha(color.NRGBA{255, 255, 255, 235}, glintA))
		g.AddColorStop(0.32, withAlpha(color.NRGBA{255, 255, 255, 40}, glintA))
		g.AddColorStop(0.68, withAlpha(color.NRGBA{255, 255, 255, 25}, glintA))
		g.AddColorStop(1, withAlpha(color.NRGBA{255, 255, 255, 150}, glintA))
		dc.SetStrokeStyle(g)
		dc.SetLineWidth(2.4 * s) // half of it is inside the clip: ~1.2 px
		dc.Stroke()
		dc.ResetClip()
		// thin grey edge
		build()
		dc.SetColor(color.NRGBA{0, 0, 0, 52})
		dc.SetLineWidth(1 * s)
		dc.Stroke()
	}

	// What shows through the segments: either the real desktop (the window
	// is slightly transparent there) or a Gaussian-blurred copy of it mixed
	// in, so lines and edges behind the glass look soft.
	see := w.seeBG
	for i := 0; i+3 < len(segImg.Pix); i += 4 {
		sa := float64(segImg.Pix[i+3]) / 255
		if sa == 0 {
			continue
		}
		keep := 1 - sa // strip removed under the segment
		if see != nil {
			for c := 0; c < 3; c++ {
				v := float64(segImg.Pix[i+c])*SegOpacity + float64(see.Pix[i+c])*(1-SegOpacity)*sa +
					float64(stripImg.Pix[i+c])*keep*(1-sa)
				stripImg.Pix[i+c] = uint8(math.Min(255, v+0.5))
			}
			v := float64(segImg.Pix[i+3]) + float64(stripImg.Pix[i+3])*keep*(1-sa)
			stripImg.Pix[i+3] = uint8(math.Min(255, v+0.5))
			continue
		}
		top := 1 - SegOpacity*sa // what the segment leaves of the layer below
		for c := 0; c < 4; c++ {
			v := float64(segImg.Pix[i+c])*SegOpacity + float64(stripImg.Pix[i+c])*keep*top
			stripImg.Pix[i+c] = uint8(math.Min(255, v+0.5))
		}
	}
	return layer
}

func (w *Wheel) buildLayer() {
	s := w.Scale
	cx, cy := w.CX, w.CY
	textCol := w.textColor(250)
	n := len(w.Items)
	placeholder := n == 0
	if placeholder {
		n = 8
	}
	span := 2 * math.Pi / float64(n)
	layer := image.NewRGBA(w.img.Rect)
	copy(layer.Pix, w.glassFor(n).Pix)
	saved := w.dc
	defer func() { w.dc = saved }()

	// Labels go on their own layer so they can fade in after the glass has
	// popped open, like the reference.
	bare := image.NewRGBA(layer.Rect)
	copy(bare.Pix, layer.Pix)
	labels := image.NewRGBA(layer.Rect)
	ldc := gg.NewContextForRGBA(labels)
	w.dc = ldc
	for i := 0; i < n; i++ {
		var it Item
		if !placeholder {
			it = w.Items[i]
		}
		mid := -math.Pi/2 + float64(i)*span
		if !placeholder {
			lr := (rIn + rOut) / 2 * s
			tx, ty := cx+math.Cos(mid)*lr, cy+math.Sin(mid)*lr
			tc := textCol
			if !it.Enabled {
				tc = withAlpha(tc, 0.35)
			}
			if it.Icon != "" {
				drawIcon(ldc, it.Icon, tx, ty-9*s, 19*s, tc)
				w.text(it.Label, 8.8*s, true, tx, ty+13*s, tc)
			} else {
				w.text(it.Label, 15*s, false, tx, ty, tc)
			}
		}
	}

	stddraw.Draw(layer, layer.Rect, labels, image.Point{}, stddraw.Over)
	w.layer = layer
	if w.layers == nil {
		w.layers, w.bares, w.labelsOf = map[string]*image.RGBA{}, map[string]*image.RGBA{}, map[string]*image.RGBA{}
	}
	key := itemsKey(w.Items)
	w.layers[key], w.bares[key], w.labelsOf[key] = layer, bare, labels
}

// glassPill draws the file size on a small frosted capsule over the
// thumbnail: the thumbnail behind it is blurred and veiled in white.
func (w *Wheel) glassPill(str string, x, y float64) {
	s := w.Scale
	dc := w.dc
	fs := 13.5 * s
	f := w.Fonts.Face(math.Round(fs*4)/4, false)
	if f == nil {
		return
	}
	dc.SetFontFace(f)
	tw, _ := dc.MeasureString(str)
	// shrink long texts so they fit inside the capsule
	if maxW := pillW*s - 16*s; tw > maxW {
		fs *= maxW / tw
		f = w.Fonts.Face(math.Round(fs*4)/4, false)
		dc.SetFontFace(f)
		tw, _ = dc.MeasureString(str)
	}
	pw, ph := pillW*s, pillH*s
	dc.DrawRoundedRectangle(x-pw/2, y-ph/2, pw, ph, ph/2)
	dc.SetFillStyle(alphaPattern{w.BG, 1})
	dc.FillPreserve()
	glintA := 1.0
	veil := color.NRGBA{255, 255, 255, 175}
	glintCol := color.NRGBA{255, 255, 255, 255}
	g0, g1, g2 := 230, 40, 120
	if w.Dark {
		veil = withAlpha(darkGlass, 175.0/255)
		// dark theme: the same shine toned down to about 45% (a grey-
		// tinted white), so its contrast against the dark glass feels
		// like the light theme's shine against its white glass. The rim
		// stroke stays 1 px, as in the light theme. The light theme keeps
		// the reference values exactly.
		glintCol = color.NRGBA{236, 236, 241, 255}
		g0, g1, g2 = 104, 18, 54
	}
	dc.SetColor(veil)
	dc.FillPreserve()
	dc.SetColor(color.NRGBA{0, 0, 0, 14})
	dc.FillPreserve()
	dc.SetColor(color.NRGBA{0, 0, 0, 38})
	dc.SetLineWidth(1 * s)
	dc.Stroke()
	// white glint along the top inner edge
	dc.DrawRoundedRectangle(x-pw/2+1.5*s, y-ph/2+1.5*s, pw-3*s, ph-3*s, ph/2-1.5*s)
	g := gg.NewLinearGradient(x, y-ph/2, x, y+ph/2)
	g.AddColorStop(0, withAlpha(color.NRGBA{glintCol.R, glintCol.G, glintCol.B, uint8(g0)}, glintA))
	g.AddColorStop(0.5, withAlpha(color.NRGBA{glintCol.R, glintCol.G, glintCol.B, uint8(g1)}, glintA))
	g.AddColorStop(1, withAlpha(color.NRGBA{glintCol.R, glintCol.G, glintCol.B, uint8(g2)}, glintA))
	dc.SetStrokeStyle(g)
	dc.SetLineWidth(1 * s)
	dc.Stroke()
	w.text(str, fs, false, x, y, w.textColor(255))
}

// offsetPattern samples an image placed at (ox, oy) in device space.
type offsetPattern struct {
	img    *image.RGBA
	ox, oy int
}

func (p offsetPattern) ColorAt(x, y int) color.Color {
	x, y = x-p.ox, y-p.oy
	if !(image.Point{x, y}.In(p.img.Rect)) {
		return color.Transparent
	}
	i := p.img.PixOffset(x, y)
	return color.RGBA{p.img.Pix[i], p.img.Pix[i+1], p.img.Pix[i+2], p.img.Pix[i+3]}
}

func boolF(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	t = clampF(t, 0, 1)
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func drawPartialPolyline(dc *gg.Context, cx, cy, s float64, pts [][2]float64, q float64) {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += math.Hypot(pts[i][0]-pts[i-1][0], pts[i][1]-pts[i-1][1])
	}
	left := total * q
	dc.NewSubPath()
	dc.MoveTo(cx+pts[0][0]*s, cy+pts[0][1]*s)
	for i := 1; i < len(pts) && left > 0; i++ {
		l := math.Hypot(pts[i][0]-pts[i-1][0], pts[i][1]-pts[i-1][1])
		f := math.Min(1, left/l)
		dc.LineTo(cx+(pts[i-1][0]+(pts[i][0]-pts[i-1][0])*f)*s, cy+(pts[i-1][1]+(pts[i][1]-pts[i-1][1])*f)*s)
		left -= l
	}
	dc.Stroke()
}

func (w *Wheel) text(str string, px float64, bold bool, x, y float64, c color.Color) {
	w.textOn(w.dc, str, px, bold, x, y, c)
}

// textOn draws centered text on any context (the wheel frame or the
// scratch layer of a lifted segment).
func (w *Wheel) textOn(dc *gg.Context, str string, px float64, bold bool, x, y float64, c color.Color) {
	f := w.Fonts.Face(math.Round(px*4)/4, bold)
	if f == nil {
		return
	}
	dc.SetFontFace(f)
	dc.SetColor(c)
	m := f.Metrics()
	capH := float64(m.CapHeight) / 64
	if capH <= 0 {
		capH = float64(m.Ascent) / 64 * 0.7
	}
	tw, _ := dc.MeasureString(str)
	dc.DrawString(str, x-tw/2, y+capH/2)
}

func (w *Wheel) pill(str string, x, y float64) {
	s := w.Scale
	f := w.Fonts.Face(math.Round(11*s*4)/4, false)
	if f == nil {
		return
	}
	w.dc.SetFontFace(f)
	tw, _ := w.dc.MeasureString(str)
	pw, ph := tw+16*s, 20*s
	bg := color.NRGBA{255, 255, 255, 215}
	fg := color.NRGBA{29, 29, 31, 255}
	if w.Dark {
		bg = color.NRGBA{20, 20, 24, 190}
		fg = color.NRGBA{245, 245, 247, 255}
	}
	w.dc.SetColor(bg)
	w.dc.DrawRoundedRectangle(x-pw/2, y-ph/2, pw, ph, ph/2)
	w.dc.Fill()
	w.text(str, 11*s, false, x, y, fg)
}

// segmentPath builds a rounded annular sector with a constant-width gap.
func (w *Wheel) segmentPath(dc *gg.Context, cx, cy, a0, a1 float64) {
	s := w.Scale
	r1, r2, g := rIn*s, rOut*s, halfGap*s
	rcO, rcI := corner*s, corner*0.47*s
	u := func(a float64) (float64, float64) { return math.Cos(a), math.Sin(a) }
	side := func(a, off, r float64) (float64, float64) { // point on offset radial line at radius r
		ux, uy := u(a)
		nx, ny := -uy, ux
		rho := math.Sqrt(r*r - off*off)
		return cx + rho*ux + off*nx, cy + rho*uy + off*ny
	}
	th := func(a, off, r float64) float64 { return a + math.Asin(off/r) }
	ua0x, ua0y := u(a0)
	ua1x, ua1y := u(a1)

	aInX, aInY := side(a0, g, r1)
	aOutX, aOutY := side(a0, g, r2)
	bOutX, bOutY := side(a1, -g, r2)
	bInX, bInY := side(a1, -g, r1)

	dc.NewSubPath()
	dc.MoveTo(aInX+rcI*ua0x, aInY+rcI*ua0y)
	dc.LineTo(aOutX-rcO*ua0x, aOutY-rcO*ua0y)
	t0 := th(a0, g, r2) + rcO/r2
	t1 := th(a1, -g, r2) - rcO/r2
	dc.QuadraticTo(aOutX, aOutY, cx+r2*math.Cos(t0), cy+r2*math.Sin(t0))
	dc.DrawArc(cx, cy, r2, t0, t1)
	dc.QuadraticTo(bOutX, bOutY, bOutX-rcO*ua1x, bOutY-rcO*ua1y)
	dc.LineTo(bInX+rcI*ua1x, bInY+rcI*ua1y)
	t2 := th(a1, -g, r1) - rcI/r1
	t3 := th(a0, g, r1) + rcI/r1
	dc.QuadraticTo(bInX, bInY, cx+r1*math.Cos(t2), cy+r1*math.Sin(t2))
	dc.DrawArc(cx, cy, r1, t2, t3)
	dc.QuadraticTo(aInX, aInY, aInX+rcI*ua0x, aInY+rcI*ua0y)
	dc.ClosePath()
}

// conicPattern is white whose opacity depends on the angle around (cx, cy).
type conicPattern struct {
	cx, cy, strength float64
	f                func(angle float64) float64
}

func (p conicPattern) ColorAt(x, y int) color.Color {
	a := math.Atan2(float64(y)+0.5-p.cy, float64(x)+0.5-p.cx)
	v := uint8(clampF(255*p.f(a)*p.strength, 0, 255))
	return color.RGBA{v, v, v, v}
}

// drawLayer composites src onto dst rotated by rot and scaled by sc around
// (cx, cy) with opacity a. first=true means dst is empty, which allows a
// plain copy when no transform is needed.
func drawLayer(dst, src *image.RGBA, cx, cy, sc, rot, a float64, first bool) {
	if a <= 0.003 {
		return
	}
	if first && a >= 0.999 && math.Abs(sc-1) < 1e-4 && math.Abs(rot) < 1e-4 {
		copy(dst.Pix, src.Pix)
		return
	}
	cs, sn := math.Cos(rot)*sc, math.Sin(rot)*sc
	m := f64.Aff3{cs, -sn, cx - (cs*cx - sn*cy), sn, cs, cy - (sn*cx + cs*cy)}
	var opts *xdraw.Options
	if a < 0.999 {
		opts = &xdraw.Options{SrcMask: image.NewUniform(color.Alpha{uint8(255 * a)})}
	}
	// Split the frame into bands drawn in parallel on all cores.
	b := dst.Rect
	n := min(runtime.NumCPU(), 8)
	if n < 2 {
		xdraw.ApproxBiLinear.Transform(dst, m, src, src.Bounds(), xdraw.Over, opts)
		return
	}
	var wg sync.WaitGroup
	h := (b.Dy() + n - 1) / n
	for y := b.Min.Y; y < b.Max.Y; y += h {
		band := dst.SubImage(image.Rect(b.Min.X, y, b.Max.X, min(y+h, b.Max.Y))).(*image.RGBA)
		wg.Add(1)
		go func() {
			defer wg.Done()
			xdraw.ApproxBiLinear.Transform(band, m, src, src.Bounds(), xdraw.Over, opts)
		}()
	}
	wg.Wait()
}

// refractPattern makes the strip look like a ring of water-clear glass:
// what is under it is magnified toward the middle of the band and gently
// rippled.
type refractPattern struct {
	img            *image.RGBA
	cx, cy, r0, r1 float64
	s              float64
}

func (p refractPattern) ColorAt(x, y int) color.Color {
	i := p.at(x, y)
	return color.RGBA{p.img.Pix[i], p.img.Pix[i+1], p.img.Pix[i+2], 255}
}

// at returns the pixel offset in p.img that shows through at (x, y).
func (p refractPattern) at(x, y int) int {
	dx, dy := float64(x)+0.5-p.cx, float64(y)+0.5-p.cy
	r := math.Hypot(dx, dy)
	if r < 1 {
		r = 1
	}
	t := clampF((r-p.r0)/(p.r1-p.r0), 0, 1)
	a := math.Atan2(dy, dx)
	disp := -5.5 * p.s * math.Cos(t*math.Pi)
	wob := 1.8*p.s*math.Sin(a*9+r/(7*p.s)) + 1.2*p.s*math.Sin(a*5-r/(11*p.s))
	rr := r + disp + wob
	sx := int(p.cx + dx/r*rr)
	sy := int(p.cy + dy/r*rr)
	b := p.img.Rect
	if sx < b.Min.X {
		sx = b.Min.X
	} else if sx >= b.Max.X {
		sx = b.Max.X - 1
	}
	if sy < b.Min.Y {
		sy = b.Min.Y
	} else if sy >= b.Max.Y {
		sy = b.Max.Y - 1
	}
	return p.img.PixOffset(sx, sy)
}

// segRefractPattern is the ring's watery glass applied to one segment: each
// segment acts as a soft lens, magnifying toward its middle and bending what
// is underneath near its edges.
type segRefractPattern struct {
	img            *image.RGBA
	cx, cy, r0, r1 float64
	a0, a1         float64
	s              float64
}

func (p segRefractPattern) ColorAt(x, y int) color.Color {
	i := p.at(x, y)
	return color.RGBA{p.img.Pix[i], p.img.Pix[i+1], p.img.Pix[i+2], 255}
}

// at returns the pixel offset in p.img that shows through at (x, y).
func (p segRefractPattern) at(x, y int) int {
	dx, dy := float64(x)+0.5-p.cx, float64(y)+0.5-p.cy
	r := math.Max(math.Hypot(dx, dy), 1)
	a := math.Atan2(dy, dx)
	mid := (p.a0 + p.a1) / 2
	for a-mid > math.Pi {
		a -= 2 * math.Pi
	}
	for a-mid < -math.Pi {
		a += 2 * math.Pi
	}
	tr := clampF((r-p.r0)/(p.r1-p.r0), 0, 1)
	ta := clampF((a-p.a0)/(p.a1-p.a0), 0, 1)
	// lens: pull samples toward the middle of the segment, more near edges
	rr := r - 7*p.s*math.Cos(tr*math.Pi)
	aa := a - (p.a1-p.a0)*0.10*math.Cos(ta*math.Pi)
	// a gentle ripple, like the ring
	rr += 1.4*p.s*math.Sin(aa*11+r/(9*p.s)) + 0.9*p.s*math.Sin(aa*6-r/(13*p.s))
	sx := int(p.cx + rr*math.Cos(aa))
	sy := int(p.cy + rr*math.Sin(aa))
	b := p.img.Rect
	sx = min(max(sx, b.Min.X), b.Max.X-1)
	sy = min(max(sy, b.Min.Y), b.Max.Y-1)
	return p.img.PixOffset(sx, sy)
}

// fastPatternFill paints the shape drawn by path with pixels taken from src
// at the offsets given by at(x, y): the same result as a gg pattern fill,
// without per-pixel interface calls and allocations.
func fastPatternFill(dst *image.RGBA, path func(*gg.Context), src *image.RGBA, at func(x, y int) int) {
	mask := image.NewRGBA(dst.Rect)
	m := gg.NewContextForRGBA(mask)
	path(m)
	m.SetRGBA255(255, 255, 255, 255)
	m.Fill()
	b := dst.Rect
	parallelRows(b.Min.Y, b.Max.Y, func(y int) {
		row := mask.Pix[(y-b.Min.Y)*mask.Stride:]
		for x := b.Min.X; x < b.Max.X; x++ {
			cov := uint32(row[(x-b.Min.X)*4+3])
			if cov == 0 {
				continue
			}
			i := at(x, y)
			o := dst.PixOffset(x, y)
			d := dst.Pix[o : o+4 : o+4]
			inv := 255 - cov
			d[0] = uint8((uint32(src.Pix[i])*cov + uint32(d[0])*inv) / 255)
			d[1] = uint8((uint32(src.Pix[i+1])*cov + uint32(d[1])*inv) / 255)
			d[2] = uint8((uint32(src.Pix[i+2])*cov + uint32(d[2])*inv) / 255)
			d[3] = uint8((255*cov + uint32(d[3])*inv) / 255)
		}
	})
}

// parallelRows runs fn for every row in [y0, y1), spread over all cores.
func parallelRows(y0, y1 int, fn func(y int)) {
	n := min(runtime.NumCPU(), 16)
	var wg sync.WaitGroup
	h := (y1 - y0 + n - 1) / n
	for s := y0; s < y1; s += h {
		e := min(s+h, y1)
		wg.Add(1)
		go func(s, e int) {
			defer wg.Done()
			for y := s; y < e; y++ {
				fn(y)
			}
		}(s, e)
	}
	wg.Wait()
}

// blendOver lays src over dst at opacity a, touching only src's non-empty
// pixels (labels are a small part of the frame).
func blendOver(dst, src *image.RGBA, a float64) {
	k := uint32(a * 256)
	p := src.Pix
	d := dst.Pix
	for i := 0; i+3 < len(p); i += 4 {
		sa := uint32(p[i+3])
		if sa == 0 {
			continue
		}
		sa = sa * k >> 8
		inv := 255 - sa
		d[i] = uint8((uint32(p[i])*k>>8 + uint32(d[i])*inv/255))
		d[i+1] = uint8((uint32(p[i+1])*k>>8 + uint32(d[i+1])*inv/255))
		d[i+2] = uint8((uint32(p[i+2])*k>>8 + uint32(d[i+2])*inv/255))
		d[i+3] = uint8(sa + uint32(d[i+3])*inv/255)
	}
}

// drawLiftedSegment renders one hovering segment (shape, blue highlight,
// icon and label together) on a transparent scratch layer, then composites
// it scaled about the segment's own centre and pushed outward along its
// centre angle.
func (w *Wheel) drawLiftedSegment(i int, h, mid, a0, a1, lift float64) {
	s := w.Scale
	if w.segScratch == nil {
		w.segScratch = map[int]*image.RGBA{}
	}
	scr := w.segScratch[i]
	if scr == nil || scr.Rect != w.img.Rect {
		scr = image.NewRGBA(w.img.Rect)
		w.segScratch[i] = scr
	}
	clear(scr.Pix)
	dc := gg.NewContextForRGBA(scr)
	lr := (rIn + rOut) / 2 * s
	w.segmentPath(dc, w.CX, w.CY, a0, a1)
	g := gg.NewLinearGradient(w.CX, w.CY-rOut*s, w.CX, w.CY+rOut*s)
	g.AddColorStop(0, withAlpha(color.NRGBA{40, 145, 255, 255}, h))
	g.AddColorStop(1, withAlpha(color.NRGBA{0, 110, 240, 255}, h))
	dc.SetFillStyle(g)
	dc.FillPreserve()
	dc.SetColor(color.NRGBA{0, 40, 120, uint8(40 * h)})
	dc.SetLineWidth(1 * s)
	dc.Stroke()
	tx, ty := w.CX+math.Cos(mid)*lr, w.CY+math.Sin(mid)*lr
	tc := color.NRGBA{255, 255, 255, uint8(255 * h)}
	it := w.Items[i]
	if it.Icon != "" {
		drawIcon(dc, it.Icon, tx, ty-9*s, 19*s, tc)
		w.textOn(dc, it.Label, 8.8*s, true, tx, ty+13*s, tc)
	} else {
		w.textOn(dc, it.Label, 15*s, false, tx, ty, tc)
	}
	k := 1 + 0.055*lift
	cxp, cyp := w.CX+math.Cos(mid)*lr, w.CY+math.Sin(mid)*lr
	off := 3.6 * s * lift
	m := f64.Aff3{k, 0, cxp - k*cxp + math.Cos(mid)*off, 0, k, cyp - k*cyp + math.Sin(mid)*off}
	xdraw.ApproxBiLinear.Transform(w.img, m, scr, scr.Bounds(), xdraw.Over, nil)
}

// drawSegShadow draws the soft shadow under a lifted segment: the blurred
// silhouette of the segment at rest, shifted a little outward, so the glass
// reads as raised above it. Subtle, and matched to the theme.
func (w *Wheel) drawSegShadow(i int, lift float64) {
	sh, ox, oy := w.segShadow(i)
	if sh == nil {
		return
	}
	n := len(w.Items)
	span := 2 * math.Pi / float64(n)
	mid := -math.Pi/2 + float64(i)*span
	s := w.Scale
	a := 0.30
	if w.Dark {
		a = 0.45 // the dark glass needs a firmer shadow to read
	}
	opts := &xdraw.Options{SrcMask: image.NewUniform(color.Alpha{uint8(clampF(a*lift, 0, 1) * 255)})}
	m := f64.Aff3{1, 0, float64(ox) + math.Cos(mid)*2*s*lift, 0, 1, float64(oy) + math.Sin(mid)*2*s*lift}
	xdraw.ApproxBiLinear.Transform(w.img, m, sh, sh.Bounds(), xdraw.Over, opts)
}

// segShadow builds (once per segment count) the blurred black silhouette of
// one segment, cropped to its bounding box.
func (w *Wheel) segShadow(i int) (*image.RGBA, int, int) {
	n := len(w.Items)
	key := n*100 + i
	if sh := w.shadows[key]; sh != nil {
		return sh, w.shadowAt[key].X, w.shadowAt[key].Y
	}
	s := w.Scale
	span := 2 * math.Pi / float64(n)
	mid := -math.Pi/2 + float64(i)*span
	a0, a1 := mid-span/2, mid+span/2
	inArc := func(t float64) bool {
		for t > a1+0.001 {
			t -= 2 * math.Pi
		}
		return t >= a0-0.001
	}
	angles := []float64{a0, a1}
	for _, ax := range []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
		if inArc(ax) {
			angles = append(angles, ax)
		}
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, r := range []float64{rIn, rOut} {
		for _, t := range angles {
			x := w.CX + math.Cos(t)*r*s
			y := w.CY + math.Sin(t)*r*s
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, y), math.Max(maxY, y)
		}
	}
	pad := 12 * s // blur bleed and outward push
	x0 := int(math.Max(float64(w.img.Rect.Min.X), minX-pad))
	y0 := int(math.Max(float64(w.img.Rect.Min.Y), minY-pad))
	x1 := int(math.Min(float64(w.img.Rect.Max.X), maxX+pad)) + 1
	y1 := int(math.Min(float64(w.img.Rect.Max.Y), maxY+pad)) + 1
	if x1 <= x0 || y1 <= y0 {
		return nil, 0, 0
	}
	img := image.NewRGBA(image.Rect(0, 0, x1-x0, y1-y0))
	dc := gg.NewContextForRGBA(img)
	w.segmentPath(dc, w.CX-float64(x0), w.CY-float64(y0), a0, a1)
	dc.SetColor(color.RGBA{0, 0, 0, 255})
	dc.Fill()
	out := GaussianBlur(img, 6*s)
	if w.shadows == nil {
		w.shadows = map[int]*image.RGBA{}
		w.shadowAt = map[int]image.Point{}
	}
	w.shadows[key] = out
	w.shadowAt[key] = image.Pt(x0, y0)
	return out, x0, y0
}

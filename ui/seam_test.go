package ui

import (
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// This file guards against the "hover seam" bug: while a segment's hover
// animation (colour + lift) runs, the fill must be ONE continuous shape -
// no horizontal or vertical line inside the segment where the colour
// clearly steps from one row/column to the next. The rest state is already
// smooth; only the animated frames ever showed the band.

// seamItems is the Alt (tools) wheel for an archive: three enabled items,
// no icons - the shape from the bug report.
func seamItems() []Item {
	return []Item{
		{ID: "7z", Label: "7Z", Enabled: true},
		{ID: "gzip", Label: "GZIP", Enabled: true},
		{ID: "rar", Label: "RAR", Enabled: true},
	}
}

// hoverFrames hovers segment 0 (the top) and renders frames at roughly
// 10%, 30%, 50%, 70% and 90% of the hover animation. Each frame is cloned
// because Render reuses its buffer.
func hoverFrames(wh *Wheel, now *time.Time) []*image.RGBA {
	a := segAngle(0, len(wh.Items))
	wh.SetPointer(wh.CX+math.Cos(a)*105*wh.Scale, wh.CY+math.Sin(a)*105*wh.Scale)
	var frames []*image.RGBA
	prev := 0
	for _, ms := range []int{25, 75, 125, 175, 225} {
		advance(wh, now, ms-prev)
		prev = ms
		frames = append(frames, cloneRGBA(wh.img))
	}
	return frames
}

// fillSteps finds the largest |colour step| between adjacent rows
// (vertical=true) or columns (vertical=false) of the segment's fill,
// measured as the median over that row's/column's fill pixels so text
// glyphs cannot skew it. Only lines with at least minPix sampled fill
// pixels on both sides count.
func fillSteps(img *image.RGBA, cx, cy, scale float64, i, n int, vertical bool, minPix int) float64 {
	span := 2 * math.Pi / float64(n)
	mid := -math.Pi/2 + float64(i)*span
	vals := map[int][][3]uint8{}
	for y := 0; y < img.Rect.Dy(); y++ {
		for x := 0; x < img.Rect.Dx(); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			r := math.Hypot(dx, dy) / scale
			if r < 66 || r > 142 {
				continue
			}
			aa := math.Atan2(dy, dx) - mid
			for aa > math.Pi {
				aa -= 2 * math.Pi
			}
			for aa < -math.Pi {
				aa += 2 * math.Pi
			}
			if math.Abs(aa) > span/2-0.10 {
				continue
			}
			o := img.PixOffset(x, y)
			R, G, B := img.Pix[o], img.Pix[o+1], img.Pix[o+2]
			// the fill is blue-dominant: clearly not the grey glass, not text
			if !(int(B) > 110 && int(B)-int(R) > 40 && int(G) >= int(R)) {
				continue
			}
			if vertical {
				vals[y] = append(vals[y], [3]uint8{R, G, B})
			} else {
				vals[x] = append(vals[x], [3]uint8{R, G, B})
			}
		}
	}
	keys := make([]int, 0, len(vals))
	for k, v := range vals {
		if len(v) >= minPix {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)
	worst := 0.0
	for j := 1; j < len(keys); j++ {
		if keys[j]-keys[j-1] != 1 {
			continue
		}
		prev, cur := vals[keys[j-1]], vals[keys[j]]
		step := math.Abs(medianChan(cur, 0)-medianChan(prev, 0)) +
			math.Abs(medianChan(cur, 1)-medianChan(prev, 1)) +
			math.Abs(medianChan(cur, 2)-medianChan(prev, 2))
		if step > worst {
			worst = step
		}
	}
	return worst
}

func medianChan(v [][3]uint8, c int) float64 {
	s := make([]int, len(v))
	for i, p := range v {
		s[i] = int(p[c])
	}
	sort.Ints(s)
	n := len(s)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return float64(s[n/2])
	}
	return float64(s[n/2-1]+s[n/2]) / 2
}

// savePNG writes img when dir is set (SEAM_PNG_DIR), named name.
func savePNG(dir, name string, img *image.RGBA) {
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err == nil {
		f, err := os.Create(filepath.Join(dir, name))
		if err == nil {
			png.Encode(f, img)
			f.Close()
		}
	}
}

// TestHoverFillHasNoSeam renders the Alt menu's top segment mid-animation
// in both themes and fails if any row or column of the fill steps clearly
// from its neighbours. The same check runs on the top segment of the other
// menus (7, 8 and 9 items).
func TestHoverFillHasNoSeam(t *testing.T) {
	dir := os.Getenv("SEAM_PNG_DIR")
	menus := []struct {
		name  string
		items []Item
	}{
		{"alt-3", seamItems()},
		{"fmt-7", formatItems(7)},
		{"fmt-8", formatItems(8)},
		{"fmt-9", formatItems(9)},
	}
	for _, dark := range []bool{true, false} {
		for _, m := range menus {
			wh := newTestWheel(dark, m.items)
			now := time.Unix(1000, 0)
			advance(wh, &now, 500) // entrance done, at rest
			for fi, frame := range hoverFrames(wh, &now) {
				name := m.name
				if dark {
					name += "-dark"
				} else {
					name += "-light"
				}
				savePNG(dir, name+"-f"+string(rune('0'+fi))+".png", frame)
				for _, vertical := range []bool{true, false} {
					axis := "row"
					if !vertical {
						axis = "column"
					}
					if got := fillSteps(frame, wh.CX, wh.CY, wh.Scale, 0, len(m.items), vertical, 40); got > 12 {
						t.Fatalf("%s frame %d (~%d%%): fill %s step %.1f levels - a seam line is visible inside the segment",
							name, fi, (fi+1)*20, axis, got)
					}
				}
			}
		}
	}
}

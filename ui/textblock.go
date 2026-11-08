package ui

import (
	"math"

	"golang.org/x/image/font"
)

// lineAscent returns how far the face's real ink rises above the baseline
// and how far it drops below it — the font's own metrics for the script
// actually drawn (an Arabic face's ink is much taller above and below the
// line than a Latin cap height, so cards center on these, not on
// capHeightOf).
func lineAscent(f font.Face) (asc, desc float64) {
	m := f.Metrics()
	asc = math.Abs(float64(m.Ascent)) / 64
	desc = math.Abs(float64(m.Descent)) / 64
	if asc <= 0 {
		asc = 10
	}
	return asc, desc
}

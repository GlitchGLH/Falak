package ui

import "testing"

// The menu window must have the same size in Arabic as in English: only
// the content is mirrored, so switching the language never moves or
// resizes the popup.
func TestMenuSizeLanguageIndependent(t *testing.T) {
	for _, scale := range []float64{1, 1.25, 1.5, 2} {
		en := &Menu{Scale: scale, Title: "Falak", Sub: "1.0 · By UncleGlitch"}
		ar := &Menu{Scale: scale, RTL: true, Title: "فلك", Sub: "1.0 · By UncleGlitch"}
		w0, h0 := en.Size()
		w1, h1 := ar.Size()
		if w0 != w1 || h0 != h1 {
			t.Errorf("scale %.2f: size (%d,%d) != RTL (%d,%d)", scale, w0, h0, w1, h1)
		}
		if en.cardH() != ar.cardH() {
			t.Errorf("scale %.2f: card height differs", scale)
		}
	}
}

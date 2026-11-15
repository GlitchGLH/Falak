package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// invisibleRunes lists every character invisible() covers.
func invisibleRunes() []rune {
	var rs []rune
	for c := rune(0); c <= 0xFFFF; c++ {
		if invisible(c) {
			rs = append(rs, c)
		}
	}
	return rs
}

func drawText(f font.Face, s string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 220, 60))
	dc := gg.NewContextForRGBA(img)
	dc.SetFontFace(f)
	dc.SetRGB(0, 0, 0)
	dc.DrawString(s, 10, 40)
	return img
}

// TestInvisibleNeverDrawn: no invisible format character (RLM, LRM, ALM,
// the other bidi controls, the zero-width ones, the BOM, the soft hyphen)
// is drawn or takes room with Falak's fonts: "A<c>B" draws and measures
// exactly like "AB", and <c> alone draws nothing. The font itself does
// draw a mark for RLM, so the check can see one.
func TestInvisibleNeverDrawn(t *testing.T) {
	fonts := msgTestFonts(t)
	face := fonts.Face(28, false)
	rs := invisibleRunes()
	if len(rs) < 20 {
		t.Fatalf("only %d invisible characters", len(rs))
	}
	raw, err := opentype.NewFace(fonts.semi, &opentype.FaceOptions{Size: 28, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(drawText(raw, "A\u200fB").Pix, drawText(raw, "AB").Pix) {
		t.Log("note: this font draws nothing for RLM by itself")
	}
	plain := drawText(face, "AB").Pix
	w0, _ := gg.NewContext(1, 1), 0
	w0.SetFontFace(face)
	ab, _ := w0.MeasureString("AB")
	for _, c := range rs {
		s := "A" + string(c) + "B"
		if !bytes.Equal(drawText(face, s).Pix, plain) {
			t.Errorf("U+%04X is drawn or takes room", c)
		}
		if w, _ := w0.MeasureString(s); w != ab {
			t.Errorf("U+%04X: width %v, want %v", c, w, ab)
		}
		for _, p := range drawText(face, string(c)).Pix {
			if p != 0 {
				t.Errorf("U+%04X alone leaves ink", c)
				break
			}
		}
	}
}

// TestVisualDirectionMarks: Visual uses RLM / LRM for the order, then
// leaves them (and every other invisible character) out. The How to use
// card's engines line keeps each licence in brackets after its engine.
func TestVisualDirectionMarks(t *testing.T) {
	ar := "لتراخيصها: FFmpeg \u200f(GPL)، وPDFium \u200f(BSD)، وqpdf \u200f(Apache 2.0)، وresvg \u200f(Apache 2.0 / MIT)."
	v := Visual(ar)
	for _, want := range []string{"(GPL) FFmpeg", "(BSD) PDFium", "(Apache 2.0) qpdf", "(Apache 2.0 / MIT) resvg"} {
		if !strings.Contains(v, want) {
			t.Errorf("%q not in %q", want, v)
		}
	}
	for _, s := range []string{ar, "clip\u200e.mp4", "a\u2066b\u2069c\u200bd\ufeff"} {
		if out := Visual(s); strings.ContainsFunc(out, invisible) {
			t.Errorf("Visual(%q) keeps an invisible character: %q", s, out)
		}
	}
	if Visual("clip\u200e.mp4") != "clip.mp4" {
		t.Errorf("Latin text: %q", Visual("clip\u200e.mp4"))
	}
	// text without marks is unchanged by this
	for _, s := range []string{"Holiday.mp4 · 45%", "notes.txt: هذا الخيار لا ينطبق على الملفات المحددة", "جار تغيير السرعة إلى 0.5×"} {
		if strings.ContainsFunc(Visual(s), invisible) {
			t.Errorf("Visual(%q) has an invisible character", s)
		}
	}
}

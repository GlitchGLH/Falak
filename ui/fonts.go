package ui

import (
	"image"
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Fonts loads a bold and a semibold typeface from the first path that exists,
// falling back to the embedded Go Bold font, and caches faces per pixel size.
type Fonts struct {
	bold, semi *opentype.Font
	mu         sync.Mutex
	cache      map[faceKey]font.Face
}

type faceKey struct {
	size float64
	bold bool
}

func loadFirst(paths []string) *opentype.Font {
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if f, err := opentype.Parse(b); err == nil {
			return f
		}
		if c, err := opentype.ParseCollection(b); err == nil {
			if f, err := c.Font(0); err == nil {
				return f
			}
		}
	}
	return nil
}

func LoadFonts(boldPaths, semiPaths []string) *Fonts {
	fs := &Fonts{cache: map[faceKey]font.Face{}}
	fs.bold = loadFirst(boldPaths)
	fs.semi = loadFirst(semiPaths)
	if fs.bold == nil {
		fs.bold, _ = opentype.Parse(gobold.TTF)
	}
	if fs.semi == nil {
		fs.semi = fs.bold
	}
	return fs
}

// Face returns a face whose em size is px device pixels.
func (fs *Fonts) Face(px float64, bold bool) font.Face {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	k := faceKey{px, bold}
	if f, ok := fs.cache[k]; ok {
		return f
	}
	src := fs.semi
	if bold {
		src = fs.bold
	}
	f, err := opentype.NewFace(src, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil
	}
	f2 := invisibleFace{f}
	fs.cache[k] = f2
	return f2
}

// invisible reports whether c is an invisible format character: the bidi
// controls (LRM U+200E, RLM U+200F, ALM U+061C, the embeddings, overrides
// and isolates), the zero-width space and joiners, the word joiner and
// friends, the byte order mark and the soft hyphen. They steer the text
// direction or the joining (Visual uses them) but are never drawn.
func invisible(c rune) bool {
	switch {
	case c == 0x00AD, c == 0x061C, c == 0x180E, c == 0xFEFF:
		return true
	case c >= 0x200B && c <= 0x200F, c >= 0x202A && c <= 0x202E, c >= 0x2060 && c <= 0x2064, c >= 0x2066 && c <= 0x206F:
		return true
	}
	return false
}

// invisibleFace is a face that has no glyph and no width for the invisible
// format characters, so no text drawn or measured with Falak's fonts ever
// shows them (some fonts draw a mark or a box for them). Every other
// character goes to the font as before.
type invisibleFace struct{ font.Face }

func (f invisibleFace) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	if invisible(r) {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	return f.Face.Glyph(dot, r)
}

func (f invisibleFace) GlyphBounds(r rune) (fixed.Rectangle26_6, fixed.Int26_6, bool) {
	if invisible(r) {
		return fixed.Rectangle26_6{}, 0, false
	}
	return f.Face.GlyphBounds(r)
}

func (f invisibleFace) GlyphAdvance(r rune) (fixed.Int26_6, bool) {
	if invisible(r) {
		return 0, false
	}
	return f.Face.GlyphAdvance(r)
}

func (f invisibleFace) Kern(r0, r1 rune) fixed.Int26_6 {
	if invisible(r0) || invisible(r1) {
		return 0
	}
	return f.Face.Kern(r0, r1)
}

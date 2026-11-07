package ui

import (
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
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
	fs.cache[k] = f
	return f
}

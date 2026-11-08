package conv

// Camera RAW photos (CR2, CR3, NEF, ARW, DNG, ORF, RW2, RAF, PEF, SRW).
// First choice is Windows' own image engine (WIC), which develops the RAW
// data when the Raw Image Extension is present. Otherwise the largest JPEG
// the camera embedded in the file is used (usually full size).

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
)

// decodeRAWNative is set by the Windows build (WIC); nil elsewhere.
var decodeRAWNative func(path string) (image.Image, error)

func decodeRAW(path string, data []byte) (image.Image, error) {
	orient := tiffOrientation(data)
	_, previewPx := embeddedJPEGOffset(data)
	if decodeRAWNative != nil {
		// Without a RAW codec Windows may read only the small thumbnail of a
		// TIFF-based RAW file; use it only if it beats the embedded preview.
		if img, err := decodeRAWNative(path); err == nil && img.Bounds().Dx()*img.Bounds().Dy() > previewPx {
			return orientRAW(img, orient), nil
		}
	}
	img, err := largestEmbeddedJPEG(data)
	if err != nil {
		return nil, errors.New(tr("This RAW photo couldn't be read", "تعذّرت قراءة صورة RAW هذه"))
	}
	return orientRAW(img, orient), nil
}

// orientRAW applies the camera orientation, unless the picture already
// came out upright (a portrait image for a 90-degree tag).
func orientRAW(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	b := img.Bounds()
	if o >= 5 && b.Dy() > b.Dx() {
		return img
	}
	return applyOrientation(img, o)
}

// largestEmbeddedJPEG finds every JPEG stream inside the file and decodes
// the one with the most pixels.
func largestEmbeddedJPEG(data []byte) (image.Image, error) {
	best, _ := embeddedJPEGOffset(data)
	if best < 0 {
		return nil, errors.New("no embedded JPEG")
	}
	img, err := jpeg.Decode(bytes.NewReader(data[best:]))
	if err != nil {
		return nil, err
	}
	if o := jpegOrientation(data[best:]); o > 1 {
		img = applyOrientation(img, o)
	}
	return img, nil
}

// embeddedJPEGOffset returns where the largest embedded JPEG starts and its
// pixel count (-1, 0 if there is none).
func embeddedJPEGOffset(data []byte) (int, int) {
	best, bestPx := -1, 0
	for i := 0; i+4 < len(data); {
		j := bytes.Index(data[i:], []byte{0xFF, 0xD8, 0xFF})
		if j < 0 {
			break
		}
		p := i + j
		if m := data[p+3]; m == 0xE0 || m == 0xE1 || m == 0xDB || m == 0xC4 || m == 0xC0 || m == 0xDD || m == 0xEE {
			if cfg, err := jpeg.DecodeConfig(bytes.NewReader(data[p:])); err == nil && cfg.Width*cfg.Height > bestPx {
				best, bestPx = p, cfg.Width*cfg.Height
			}
		}
		i = p + 3
	}
	return best, bestPx
}

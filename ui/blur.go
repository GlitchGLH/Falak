package ui

import (
	"image"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

// FrostBackground turns a screen capture into the frosted backdrop shown
// through the glass: it downsamples, box-blurs three times, boosts
// saturation and scales back up. Fast enough to run when the wheel opens.
func FrostBackground(src *image.RGBA, radius float64, saturation float64) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	const down = 4
	sw, sh := (w+down-1)/down, (h+down-1)/down
	small := image.NewRGBA(image.Rect(0, 0, sw, sh))
	xdraw.ApproxBiLinear.Scale(small, small.Bounds(), src, b, draw.Src, nil)
	r := int(radius/down + 0.5)
	if r < 1 {
		r = 1
	}
	for i := 0; i < 3; i++ {
		boxBlurH(small, r)
		boxBlurV(small, r)
	}
	saturate(small, saturation)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.BiLinear.Scale(out, out.Bounds(), small, small.Bounds(), draw.Src, nil)
	return out
}

func boxBlurH(img *image.RGBA, r int) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	row := make([]int, w*4)
	div := 2*r + 1
	for y := 0; y < h; y++ {
		p := img.Pix[y*img.Stride : y*img.Stride+w*4]
		for c := 0; c < 4; c++ {
			sum := 0
			for i := -r; i <= r; i++ {
				sum += int(p[clamp(i, 0, w-1)*4+c])
			}
			for x := 0; x < w; x++ {
				row[x*4+c] = sum / div
				sum += int(p[clamp(x+r+1, 0, w-1)*4+c]) - int(p[clamp(x-r, 0, w-1)*4+c])
			}
		}
		for i := range p {
			p[i] = uint8(row[i])
		}
	}
}

func boxBlurV(img *image.RGBA, r int) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	col := make([]int, h)
	div := 2*r + 1
	for x := 0; x < w; x++ {
		for c := 0; c < 4; c++ {
			at := func(y int) int { return int(img.Pix[clamp(y, 0, h-1)*img.Stride+x*4+c]) }
			sum := 0
			for i := -r; i <= r; i++ {
				sum += at(i)
			}
			for y := 0; y < h; y++ {
				col[y] = sum / div
				sum += at(y+r+1) - at(y-r)
			}
			for y := 0; y < h; y++ {
				img.Pix[y*img.Stride+x*4+c] = uint8(col[y])
			}
		}
	}
}

func saturate(img *image.RGBA, s float64) {
	for i := 0; i+3 < len(img.Pix); i += 4 {
		r, g, b := float64(img.Pix[i]), float64(img.Pix[i+1]), float64(img.Pix[i+2])
		l := 0.2126*r + 0.7152*g + 0.0722*b
		img.Pix[i] = clamp8(l + (r-l)*s)
		img.Pix[i+1] = clamp8(l + (g-l)*s)
		img.Pix[i+2] = clamp8(l + (b-l)*s)
		img.Pix[i+3] = 255
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// MakeThumb scales img to cover a square of size px (center crop).
func MakeThumb(img image.Image, px int) *image.RGBA {
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	if w == 0 || h == 0 || px <= 0 {
		return nil
	}
	var sr image.Rectangle
	if w > h {
		off := int((w - h) / 2)
		sr = image.Rect(b.Min.X+off, b.Min.Y, b.Min.X+off+b.Dy(), b.Max.Y)
	} else {
		off := int((h - w) / 2)
		sr = image.Rect(b.Min.X, b.Min.Y+off, b.Max.X, b.Min.Y+off+b.Dx())
	}
	out := image.NewRGBA(image.Rect(0, 0, px, px))
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, sr, draw.Over, nil)
	return out
}

// GaussianBlur is a true Gaussian blur, like Photoshop's Filter > Blur >
// Gaussian Blur with Radius = sigma (in pixels). Large radii are computed on
// a reduced copy and scaled back up, which is indistinguishable for such
// soft results and keeps it fast.
func GaussianBlur(src *image.RGBA, sigma float64) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if sigma < 0.5 {
		out := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(out, out.Rect, src, b.Min, draw.Src)
		return out
	}
	down := int(sigma / 3) // soft results lose nothing at reduced size
	down = max(1, min(down, 8))
	sw, sh := (w+down-1)/down, (h+down-1)/down
	small := image.NewRGBA(image.Rect(0, 0, sw, sh))
	if down == 1 {
		draw.Draw(small, small.Rect, src, b.Min, draw.Src)
	} else {
		xdraw.ApproxBiLinear.Scale(small, small.Bounds(), src, b, draw.Src, nil)
	}
	s := sigma / float64(down)
	rad := int(math.Ceil(3 * s))
	k := make([]float64, 2*rad+1)
	var sum float64
	for i := -rad; i <= rad; i++ {
		v := math.Exp(-float64(i*i) / (2 * s * s))
		k[i+rad] = v
		sum += v
	}
	for i := range k {
		k[i] /= sum
	}
	tmp := make([]float64, sw*sh*4)
	// horizontal
	for y := 0; y < sh; y++ {
		row := small.Pix[y*small.Stride:]
		for x := 0; x < sw; x++ {
			var acc [4]float64
			for i := -rad; i <= rad; i++ {
				xx := min(max(x+i, 0), sw-1) * 4
				wv := k[i+rad]
				acc[0] += wv * float64(row[xx])
				acc[1] += wv * float64(row[xx+1])
				acc[2] += wv * float64(row[xx+2])
				acc[3] += wv * float64(row[xx+3])
			}
			copy(tmp[(y*sw+x)*4:], acc[:])
		}
	}
	// vertical
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			var acc [4]float64
			for i := -rad; i <= rad; i++ {
				yy := min(max(y+i, 0), sh-1)
				wv := k[i+rad]
				o := (yy*sw + x) * 4
				acc[0] += wv * tmp[o]
				acc[1] += wv * tmp[o+1]
				acc[2] += wv * tmp[o+2]
				acc[3] += wv * tmp[o+3]
			}
			o := y*small.Stride + x*4
			for c := 0; c < 4; c++ {
				small.Pix[o+c] = uint8(math.Min(255, acc[c]+0.5))
			}
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.BiLinear.Scale(out, out.Bounds(), small, small.Bounds(), draw.Src, nil)
	return out
}

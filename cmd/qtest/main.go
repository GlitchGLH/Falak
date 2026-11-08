package main

import (
	"context"
	"fmt"
	"image"
	"math"
	"os"

	"raha/conv"
)

func psnr(a, b image.Image) (float64, float64) {
	var se, sa float64
	n := 0
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			r1, g1, b1, a1 := a.At(x, y).RGBA()
			r2, g2, b2, a2 := b.At(x, y).RGBA()
			if a1 > 0xf000 {
				for _, d := range []float64{float64(r1) - float64(r2), float64(g1) - float64(g2), float64(b1) - float64(b2)} {
					d /= 257
					se += d * d
				}
				n += 3
			}
			da := (float64(a1) - float64(a2)) / 257
			sa += da * da
		}
	}
	px := float64(r.Dx() * r.Dy())
	return 10 * math.Log10(255*255/(se/float64(n))), 10 * math.Log10(255*255/(sa/px+1e-9))
}

func main() {
	conv.FFmpegOverride = "/usr/bin/ffmpeg"
	conv.ResvgOverride = "/tmp/resvg/resvg"
	for i := 1; i+1 < len(os.Args); i += 2 {
		dst, err := conv.Run(context.Background(), os.Args[i], os.Args[i+1], nil)
		fmt.Println(os.Args[i], "->", os.Args[i+1], dst, err)
		if err != nil || len(os.Args[i+1]) < 3 || os.Args[i+1][3:] != "HEIC" {
			continue
		}
		src, _ := conv.Decode(os.Args[i])
		out, err := conv.Decode(dst)
		if err != nil {
			fmt.Println("  decode back FAILED:", err)
			continue
		}
		st, _ := os.Stat(dst)
		p, pa := psnr(src, out)
		fmt.Printf("  decoded %v (src %v), %d KB, colour PSNR %.1f dB, alpha PSNR %.1f dB\n", out.Bounds().Size(), src.Bounds().Size(), st.Size()/1024, p, pa)
	}
}

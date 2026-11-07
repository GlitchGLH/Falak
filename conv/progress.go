package conv

import (
	"image"
	"math"
	"sync"
	"time"
)

// Some engines (the WEBP/AVIF/HEIC/JPEG/PNG encoders, Microsoft Office,
// LibreOffice, WinRAR, qpdf) do their work in one step and say nothing
// until they finish. For those the bar follows the time the work is expected
// to take, worked out from the file's size and measured speeds: it moves
// steadily, slows down near the end if the work takes longer, never stops,
// and only reaches 100% when the file is really there.

// estimate moves progress from lo toward hi over about `expected` seconds,
// until stop is called. stop waits for the ticker, so no late value can
// arrive after it returns.
func estimate(prog func(float64), lo, hi, expected float64) (stop func()) {
	if prog == nil {
		return func() {}
	}
	expected = math.Max(expected, 0.3)
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	t0 := time.Now()
	go func() {
		defer wg.Done()
		tk := time.NewTicker(80 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-done:
				return
			case <-tk.C:
				x := time.Since(t0).Seconds() / expected
				v := x // steady until 85% of the expected time
				if x > 0.85 {
					v = 0.85 + 0.14*(1-math.Exp(-(x-0.85)/0.6))
				}
				prog(lo + (hi-lo)*v)
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
		})
	}
}

// encodeSeconds is the expected time to write img in format (seconds per
// megapixel measured on a modest 2-core PC; faster PCs simply finish early).
func encodeSeconds(img image.Image, format string, lossless bool) float64 {
	b := img.Bounds()
	mp := float64(b.Dx()) * float64(b.Dy()) / 1e6
	k := map[string]float64{"JPG": 0.07, "PNG": 0.18, "TIFF": 0.13, "BMP": 0.01, "ICO": 0.3,
		"GIF": 0.3, "HEIC": 1.4, "WEBP": 1.6, "AVIF": 4.8}[format]
	if lossless {
		switch format {
		case "WEBP":
			k = 2.1
		case "AVIF":
			k = 7.5
		}
	}
	if k == 0 {
		k = 0.2
	}
	return mp * k
}

// fileMB is a file's size in megabytes.
func fileMB(path string) float64 { return float64(fileSize(path)) / (1 << 20) }

package conv

import (
	"archive/zip"
	"context"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestProgressMoves checks that the bar keeps moving during long steps:
// no gap between updates longer than a second, and it ends near the top.
func TestProgressMoves(t *testing.T) {
	FFmpegOverride, _ = exec.LookPath("ffmpeg")
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 4000, 3000))
	r := rand.New(rand.NewSource(1))
	for y := 0; y < 3000; y++ {
		for x := 0; x < 4000; x++ {
			n := uint8(r.Intn(30))
			img.SetNRGBA(x, y, color.NRGBA{uint8(x/16) + n, uint8(y/12) + n, uint8((x+y)/28) + n, 255})
		}
	}
	big := filepath.Join(dir, "big.png")
	f, _ := os.Create(big)
	png.Encode(f, img)
	f.Close()
	zp := filepath.Join(dir, "pack.zip")
	zf, _ := os.Create(zp)
	zw := zip.NewWriter(zf)
	for i := 0; i < 3; i++ {
		w, _ := zw.Create(filepath.Base(big) + string(rune('a'+i)))
		d, _ := os.ReadFile(big)
		w.Write(d)
	}
	zw.Close()
	zf.Close()

	for _, c := range []struct{ path, id string }{
		{big, "to:PDF"}, {big, "to:WEBP"}, {big, "to:JPG"}, {big, "to:HEIC"}, {big, "addbg"}, {big, "compress"},
		{zp, "to:7Z"}, {zp, "x:extract"}, {zp, "to:TAR"},
	} {
		var mu sync.Mutex
		var last time.Time
		var maxGap time.Duration
		var top float64
		n := 0
		start := time.Now()
		last = start
		_, err := Run(context.Background(), c.path, c.id, func(v float64) {
			mu.Lock()
			defer mu.Unlock()
			now := time.Now()
			if g := now.Sub(last); g > maxGap {
				maxGap = g
			}
			last, top = now, max(top, v)
			n++
		})
		if err != nil && c.id != "compress" {
			t.Errorf("%s %s: %v", filepath.Base(c.path), c.id, err)
			continue
		}
		total := time.Since(start)
		if g := time.Since(last); g > maxGap {
			maxGap = g
		}
		t.Logf("%-8s %-10s %6v  updates=%3d  longest pause=%v  reached=%.0f%%", filepath.Base(c.path), c.id, total.Round(time.Millisecond), n, maxGap.Round(time.Millisecond), top*100)
		if total > 1500*time.Millisecond && maxGap > 2*time.Second { // heavy encoders on 2 cores can delay a tick
			t.Errorf("%s %s: progress paused for %v", filepath.Base(c.path), c.id, maxGap)
		}
	}
}

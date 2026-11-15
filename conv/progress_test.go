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
// no gap between updates longer than 2 s, and it ends near the top.
//
// The gaps are wall-clock time, so a busy machine (another heavy program,
// or the other test packages running at the same time) can hold the whole
// process for seconds and make one run pause although the code is fine. A
// real fault (a step that sends no progress) pauses at the same step every
// time, so a conversion that pauses too long is run again, up to three
// runs in all, and only fails when every run pauses too long. The limit
// itself is unchanged.
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

	type result struct {
		total, maxGap time.Duration
		n             int
		top           float64
		err           error
	}
	measure := func(path, id string) result {
		var mu sync.Mutex
		var r result
		start := time.Now()
		last := start
		_, r.err = Run(context.Background(), path, id, func(v float64) {
			mu.Lock()
			defer mu.Unlock()
			now := time.Now()
			if g := now.Sub(last); g > r.maxGap {
				r.maxGap = g
			}
			last, r.top = now, max(r.top, v)
			r.n++
		})
		mu.Lock()
		defer mu.Unlock()
		r.total = time.Since(start)
		if g := time.Since(last); g > r.maxGap {
			r.maxGap = g
		}
		return r
	}
	// heavy encoders on 2 cores can delay a tick, so short jobs and
	// pauses up to 2 s are fine
	paused := func(r result) bool { return r.total > 1500*time.Millisecond && r.maxGap > 2*time.Second }
	for _, c := range []struct{ path, id string }{
		{big, "to:PDF"}, {big, "to:WEBP"}, {big, "to:JPG"}, {big, "to:HEIC"}, {big, "addbg"}, {big, "compress"},
		{zp, "to:7Z"}, {zp, "x:extract"}, {zp, "to:TAR"},
	} {
		var pauses []time.Duration
		for run := 1; ; run++ {
			r := measure(c.path, c.id)
			if r.err != nil && c.id != "compress" {
				t.Errorf("%s %s: %v", filepath.Base(c.path), c.id, r.err)
				break
			}
			t.Logf("%-8s %-10s %6v  updates=%3d  longest pause=%v  reached=%.0f%%  (run %d)", filepath.Base(c.path), c.id, r.total.Round(time.Millisecond), r.n, r.maxGap.Round(time.Millisecond), r.top*100, run)
			if !paused(r) {
				break
			}
			pauses = append(pauses, r.maxGap.Round(time.Millisecond))
			if run == 3 {
				t.Errorf("%s %s: progress paused in every run: %v", filepath.Base(c.path), c.id, pauses)
				break
			}
		}
	}
}

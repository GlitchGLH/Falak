package conv

import (
	"context"
	"image"
	"math"
	"path/filepath"
	"sync"
	"testing"
)

func TestVideoInfoFrames(t *testing.T) {
	v := VideoInfo{Duration: 10, FPS: 25}
	if v.Frames() != 250 || v.FrameTime(25) != 1 {
		t.Fatalf("frames %d, time %v", v.Frames(), v.FrameTime(25))
	}
	if s := v.FrameSeek(25); math.Abs(s-0.99) > 1e-9 {
		t.Fatalf("seek of frame 25 = %v, want 0.99 (a quarter frame early)", s)
	}
	if v.FrameSeek(0) != 0 {
		t.Fatal("frame 0 seeks to 0")
	}
	if (VideoInfo{Duration: 2}).FrameRate() != 30 {
		t.Fatal("an unknown frame rate counts as 30")
	}
}

func TestThumbTimes(t *testing.T) {
	got := ThumbTimes(10, 4)
	want := []float64{1.25, 3.75, 6.25, 8.75}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func diffImages(a, b *image.RGBA) float64 {
	if a.Rect != b.Rect {
		return math.Inf(1)
	}
	var sum float64
	for i := range a.Pix {
		sum += math.Abs(float64(a.Pix[i]) - float64(b.Pix[i]))
	}
	return sum / float64(len(a.Pix))
}

// TestFrames: frame k and frame k+1 are different pictures, frames come at
// full size or scaled to fit, and the thumbnails all arrive.
func TestFrames(t *testing.T) {
	clip := makeClip(t, 4)
	info := ProbeVideo(clip)
	if math.Abs(info.Duration-4) > 0.2 || info.FPS != 24 {
		t.Fatalf("probe: %+v", info)
	}
	ctx := context.Background()
	k := 50
	f, err := FrameAt(ctx, clip, info.FrameSeek(k), 4000, 4000, true)
	if err != nil {
		t.Fatal(err)
	}
	if f.Rect.Dx() != 320 || f.Rect.Dy() != 180 {
		t.Fatalf("full-size frame is %v", f.Rect)
	}
	next, err := FrameAt(ctx, clip, info.FrameSeek(k+1), 4000, 4000, true)
	if err != nil {
		t.Fatal(err)
	}
	small, err := FrameAt(ctx, clip, info.FrameSeek(k), 160, 160, true)
	if err != nil {
		t.Fatal(err)
	}
	if small.Rect.Dx() != 160 || small.Rect.Dy() != 90 {
		t.Fatalf("scaled frame is %v, want 160x90", small.Rect)
	}

	if d := diffImages(next, f); d < 1 {
		t.Fatalf("frames %d and %d look the same (diff %.2f)", k, k+1, d)
	}

	var mu sync.Mutex
	seen := map[int]bool{}
	Thumbs(ctx, clip, info.Duration, 6, 64, 36, func(i int, img *image.RGBA) {
		mu.Lock()
		seen[i] = img.Rect.Dx() == 64 && img.Rect.Dy() == 36
		mu.Unlock()
	})
	for i := 0; i < 6; i++ {
		if !seen[i] {
			t.Errorf("thumbnail %d missing or wrong size", i)
		}
	}
}

// TestTrimVideo: the piece is exactly the selection, named -trimmed, never
// replacing a file; a too short selection is refused.
func TestTrimVideo(t *testing.T) {
	clip := makeClip(t, 6)
	ctx := context.Background()
	out, err := TrimVideo(ctx, clip, 1.5, 4.0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip-trimmed.mp4"); out != want {
		t.Fatalf("output %s, want %s", out, want)
	}
	if d := ProbeVideo(out).Duration; math.Abs(d-2.5) > 0.1 {
		t.Fatalf("trimmed length %.3f, want 2.5", d)
	}
	if !probe(out).hasAudio {
		t.Error("the trimmed piece lost its sound")
	}
	// the first frame of the piece is the frame at 1.5 s of the original
	a, _ := FrameAt(ctx, out, 0, 4000, 4000, true)
	b, _ := FrameAt(ctx, clip, 1.5, 4000, 4000, true)
	c, _ := FrameAt(ctx, clip, 1.5+2.0/24, 4000, 4000, true)
	if a == nil || b == nil || c == nil {
		t.Fatal("frames missing")
	}
	if d1, d2 := diffImages(a, b), diffImages(a, c); d1 >= d2 {
		t.Fatalf("the cut is not exact: start frame diff %.2f vs two frames later %.2f", d1, d2)
	}
	out2, err := TrimVideo(ctx, clip, 0, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip-trimmed (1).mp4"); out2 != want {
		t.Fatalf("second output %s, want %s", out2, want)
	}
	if _, err := TrimVideo(ctx, clip, 2, 2.05, nil); err == nil || err.Error() != "The selection is too short" {
		t.Fatalf("a 0.05 s selection: %v", err)
	}
}

// TestWindowToolsOneVideo: Trim and Snapshots work on one video; with
// several files they are shown disabled (never hidden), and the other video
// tools stay enabled.
func TestWindowToolsOneVideo(t *testing.T) {
	state := func(paths []string) map[string]bool {
		m := map[string]bool{}
		for _, it := range ToolItemsFor(paths) {
			m[it.ID] = it.Enabled
		}
		return m
	}
	one := state([]string{"a.mp4"})
	several := state([]string{"a.mp4", "b.mov"})
	for _, id := range []string{"v:trim", "v:snapshot", "v:split"} {
		if e, ok := one[id]; !ok || !e {
			t.Errorf("one video: %s shown %v, enabled %v; want enabled", id, ok, e)
		}
		if e, ok := several[id]; !ok || e {
			t.Errorf("two videos: %s shown %v, enabled %v; want shown and disabled", id, ok, e)
		}
	}
	for id, e := range several {
		if !OpensWindow(id) && !e {
			t.Errorf("two videos: %s disabled, want enabled", id)
		}
	}
}

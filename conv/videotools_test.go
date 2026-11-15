package conv

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestSplitVideo: three parts cut exactly at the two points (given in any
// order), in "<name> - parts", the sound kept; a second split goes to
// "<name> - parts (1)"; no point, or a nearly empty part, is refused.
func TestSplitVideo(t *testing.T) {
	clip := makeClip(t, 6)
	ctx := context.Background()
	out, err := SplitVideo(ctx, clip, []float64{4.25, 1.5}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip - parts"); out != want {
		t.Fatalf("folder %s, want %s", out, want)
	}
	want := []float64{1.5, 2.75, 1.75}
	for i, w := range want {
		p := filepath.Join(out, "clip - part "+string(rune('1'+i))+".mp4")
		mi := probe(p)
		if math.Abs(mi.duration-w) > 0.1 {
			t.Errorf("part %d: %.3f s, want %.3f", i+1, mi.duration, w)
		}
		if !mi.hasAudio {
			t.Errorf("part %d lost its sound", i+1)
		}
	}
	if es, _ := os.ReadDir(out); len(es) != 3 {
		t.Errorf("%d files in the folder, want 3", len(es))
	}
	// part 2 starts exactly on the frame at 1.5 s
	a, _ := FrameAt(ctx, filepath.Join(out, "clip - part 2.mp4"), 0, 4000, 4000, true)
	b, _ := FrameAt(ctx, clip, 1.5, 4000, 4000, true)
	c, _ := FrameAt(ctx, clip, 1.5+2.0/24, 4000, 4000, true)
	if a == nil || b == nil || c == nil {
		t.Fatal("frames missing")
	}
	if d1, d2 := diffImages(a, b), diffImages(a, c); d1 >= d2 {
		t.Fatalf("the cut is not exact: %.2f vs two frames later %.2f", d1, d2)
	}
	out2, err := SplitVideo(ctx, clip, []float64{3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip - parts (1)"); out2 != want {
		t.Fatalf("second folder %s, want %s", out2, want)
	}
	if _, err := SplitVideo(ctx, clip, nil, nil); err == nil || err.Error() != "Add at least one split point" {
		t.Errorf("no point: %v", err)
	}
	if _, err := SplitVideo(ctx, clip, []float64{2, 2.02}, nil); err == nil || err.Error() != "The selection is too short" {
		t.Errorf("a 0.02 s part: %v", err)
	}
	for _, c := range []struct{ fps, want float64 }{{24, 3.0 / 24}, {30, 3.0 / 30}, {10, 0.1}, {5, 0.2}} {
		if g := SplitGap(VideoInfo{Duration: 6, FPS: c.fps}); math.Abs(g-c.want) > 1e-9 {
			t.Errorf("gap at %v fps: %v, want %v (whole frames, at least MinTrim)", c.fps, g, c.want)
		}
	}
}

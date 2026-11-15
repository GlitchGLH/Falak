package conv

import (
	"context"
	"image"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestAtempoChain(t *testing.T) {
	for speed, want := range map[float64]string{
		0.25: "atempo=0.5,atempo=0.5", 0.5: "atempo=0.5", 0.75: "atempo=0.75", 1.25: "atempo=1.25",
		1.5: "atempo=1.5", 2: "atempo=2", 3: "atempo=2,atempo=1.5", 4: "atempo=2,atempo=2",
	} {
		if got := atempoChain(speed); got != want {
			t.Errorf("%v: %q, want %q", speed, got, want)
		}
	}
	if SpeedText(0.25) != "0.25" || SpeedText(2) != "2" || SpeedText(1.5) != "1.5" {
		t.Error("SpeedText")
	}
}

// TestSpeedVideo: "<name>-<speed>x.mp4" beside the original at the new
// length, the sound kept (and as long as the picture), the frame rate
// kept; never replacing a file; a video without sound stays silent.
func TestSpeedVideo(t *testing.T) {
	clip := makeClip(t, 4)
	ctx := context.Background()
	for _, c := range []struct {
		speed float64
		name  string
	}{{0.5, "clip-0.5x.mp4"}, {4, "clip-4x.mp4"}, {0.25, "clip-0.25x.mp4"}, {0.5, "clip-0.5x (1).mp4"}} {
		out, err := SpeedVideo(ctx, clip, c.speed, false, nil)
		if err != nil {
			t.Fatalf("%v: %v", c.speed, err)
		}
		if want := filepath.Join(filepath.Dir(clip), c.name); out != want {
			t.Errorf("%v: output %s, want %s", c.speed, out, want)
		}
		mi := probe(out)
		if want := 4 / c.speed; math.Abs(mi.duration-want) > 0.15 {
			t.Errorf("%v: %.3f s, want %.3f", c.speed, mi.duration, want)
		}
		if !mi.hasAudio {
			t.Errorf("%v: the sound is gone", c.speed)
		}
		if math.Abs(mi.fps-24) > 0.01 {
			t.Errorf("%v: %v fps, want the original 24", c.speed, mi.fps)
		}
	}
	silent := filepath.Join(filepath.Dir(clip), "silent.mp4")
	if _, err := ffmpeg(ctx, clip, silent, []string{"-map", "0:v", "-c", "copy", "-an"}, 4, nil); err != nil {
		t.Fatal(err)
	}
	out, err := SpeedVideo(ctx, silent, 2, false, nil)
	if err != nil {
		t.Fatalf("a video without sound: %v", err)
	}
	if mi := probe(out); mi.hasAudio || math.Abs(mi.duration-2) > 0.15 {
		t.Errorf("a video without sound: %.3f s, sound %v", mi.duration, mi.hasAudio)
	}
}

// TestSmoothSlowMotion: at 0.5× the plain output repeats every frame (its
// odd frames equal the even ones), the smooth one makes a new frame in
// between; same length, frame rate and sound. Fast speeds ignore smooth.
func TestSmoothSlowMotion(t *testing.T) {
	clip := makeClip(t, 2)
	ctx := context.Background()
	plain, err := SpeedVideo(ctx, clip, 0.5, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	smooth, err := SpeedVideo(ctx, clip, 0.5, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(smooth) != "clip-0.5x (1).mp4" {
		t.Errorf("smooth output %s", filepath.Base(smooth))
	}
	mi := probe(smooth)
	if math.Abs(mi.duration-4) > 0.15 || math.Abs(mi.fps-24) > 0.01 || !mi.hasAudio {
		t.Errorf("smooth: %.3f s, %v fps, sound %v", mi.duration, mi.fps, mi.hasAudio)
	}
	frame := func(path string, k int) *image.RGBA {
		img, err := FrameAt(ctx, path, (float64(k)+0.25)/24, 4000, 4000, true)
		if err != nil {
			t.Fatal(err)
		}
		return img
	}
	// repeated, every source frame shows twice, so one of two neighbouring
	// pairs is a repeat; interpolated, every output frame is new
	pair := func(path string) (float64, float64) {
		a, b, c := frame(path, 20), frame(path, 21), frame(path, 22)
		return diffImages(a, b), diffImages(b, c)
	}
	p1, p2 := pair(plain)
	s1, s2 := pair(smooth)
	t.Logf("neighbour differences: plain %.2f %.2f, smooth %.2f %.2f", p1, p2, s1, s2)
	if math.Min(p1, p2) > 1 {
		t.Errorf("plain: no repeated frame (%.2f, %.2f)", p1, p2)
	}
	if math.Min(s1, s2) < 3*math.Max(math.Min(p1, p2), 0.5) {
		t.Errorf("smooth: a repeated frame (%.2f, %.2f)", s1, s2)
	}
	if speedFilter(2, 24, true) != speedFilter(2, 24, false) || SmoothApplies(1.25) || !SmoothApplies(0.75) {
		t.Error("smooth must apply to slow speeds only")
	}
}

// TestPlayPreview: the preview plays at the chosen speed (twice as fast:
// 1 s of video in about 0.5 s, 24 pictures a second), each picture the
// asked size, and ends at the end of the video.
func TestPlayPreview(t *testing.T) {
	clip := makeClip(t, 2)
	n, last := 0, 0.0
	start := time.Now()
	err := PlayPreview(context.Background(), clip, 1, 2, 64, 36, func(img *image.RGBA, at float64) {
		if img.Rect.Dx() != 64 || img.Rect.Dy() != 36 {
			t.Fatalf("picture %v", img.Rect)
		}
		n++
		last = at
	})
	took := time.Since(start).Seconds()
	if err != nil {
		t.Fatal(err)
	}
	if n < 10 || n > 14 {
		t.Errorf("%d pictures for 1 s at 2×, want about 12", n)
	}
	if last < 1.8 || last > 2.01 {
		t.Errorf("the last picture is from %.3f s, want about 2", last)
	}
	if took < 0.4 || took > 1.5 {
		t.Errorf("1 s at 2× took %.2f s to play, want about 0.5", took)
	}
}

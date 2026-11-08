package conv

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// makeClip renders a test video (test pattern + tone) with the system
// ffmpeg, like the other tests; skips when there is no ffmpeg.
func makeClip(t *testing.T, secs int) string {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}
	FFmpegOverride, _ = exec.LookPath("ffmpeg")
	out := filepath.Join(t.TempDir(), "clip.mp4")
	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=duration="+strconv.Itoa(secs)+":size=320x180:rate=24",
		"-f", "lavfi", "-i", "sine=frequency=440:duration="+strconv.Itoa(secs),
		"-c:v", "libx264", "-preset", "ultrafast", "-crf", "35",
		"-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "96k", "-shortest", out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("clip: %v: %s", err, b)
	}
	return out
}

// TestGifRefusesLongVideos: over 10 s the conversion must not start.
func TestGifRefusesLongVideos(t *testing.T) {
	clip := makeClip(t, 12)
	_, err := Run(context.Background(), clip, "to:GIF", nil)
	if err == nil {
		t.Fatal("a 12 s video converted to GIF, want a refusal")
	}
	t.Logf("refusal message: %q", err.Error())
}

// TestGifProgressMoves: on a short clip the bar must move through BOTH
// passes (no stuck 0%) and finish.
func TestGifProgressMoves(t *testing.T) {
	clip := makeClip(t, 4)
	var vals []float64
	out, err := Run(context.Background(), clip, "to:GIF", func(f float64) { vals = append(vals, f) })
	if err != nil {
		t.Fatalf("gif: %v", err)
	}
	defer os.Remove(out)
	if st, err := os.Stat(out); err != nil || st.Size() < 1000 {
		t.Fatalf("gif output missing: %v", err)
	}
	if len(vals) == 0 || vals[len(vals)-1] < 0.95 {
		t.Fatalf("progress never reached the end: last %v", vals)
	}
	distinct := map[float64]bool{}
	for _, v := range vals {
		distinct[v] = true
	}
	if len(distinct) < 4 {
		t.Fatalf("progress barely moved: %d distinct values", len(distinct))
	}
	t.Logf("progress moved through %d distinct values, last %.2f", len(distinct), vals[len(vals)-1])
}

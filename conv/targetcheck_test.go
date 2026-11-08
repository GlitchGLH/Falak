package conv

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestTargetSizeWithoutFFmpeg: before FFmpeg is downloaded the video's
// length can't be read (0), and "Compress to a set size" must not reject a
// valid target because of that (the job checks again once FFmpeg is there).
func TestTargetSizeWithoutFFmpeg(t *testing.T) {
	saved := FFmpegOverride
	defer func() { FFmpegOverride = saved }()
	dir := t.TempDir()
	FFmpegOverride = filepath.Join(dir, "missing", "ffmpeg.exe") // not installed
	video := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(video, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	if HaveFFmpeg() {
		t.Fatal("FFmpeg should look missing")
	}
	d := VideoDuration(video)
	if d != 0 {
		t.Fatalf("length without FFmpeg = %v, want 0 (unknown)", d)
	}
	for _, mb := range []float64{1, 40, 60, 100} {
		if TargetTooSmall(mb, d) {
			t.Errorf("%v MB rejected as too small while the length is unknown", mb)
		}
	}
}

// TestTargetTooSmall: with a known length the check still applies.
func TestTargetTooSmall(t *testing.T) {
	cases := []struct {
		mb, secs float64
		small    bool
	}{
		{40, 30, false},  // the owner's 30 s recording at 40 MB
		{100, 30, false}, // and at 100 MB
		{1, 3600, true},  // 1 MB for an hour: far too small
		{0.68, 30, true}, // just under 64 kb/s left for the picture
		{0.70, 30, false},
		{40, 0, false},  // unknown length: never rejected here
		{40, -1, false}, // nonsense length: treated as unknown
	}
	for _, c := range cases {
		if got := TargetTooSmall(c.mb, c.secs); got != c.small {
			t.Errorf("TargetTooSmall(%v MB, %v s) = %v, want %v", c.mb, c.secs, got, c.small)
		}
	}
}

// TestTargetSizeWithFFmpeg: once FFmpeg is there the real length is read
// and the check uses it.
func TestTargetSizeWithFFmpeg(t *testing.T) {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("no ffmpeg")
	}
	saved := FFmpegOverride
	defer func() { FFmpegOverride = saved }()
	FFmpegOverride = ff
	video := filepath.Join(t.TempDir(), "clip.mp4")
	mk(t, "-f", "lavfi", "-i", "testsrc=size=320x180:rate=24:duration=4", "-c:v", "libx264", "-pix_fmt", "yuv420p", video)
	d := VideoDuration(video)
	if d < 3.5 || d > 4.5 {
		t.Fatalf("length = %v, want about 4 s", d)
	}
	if TargetTooSmall(1, d) {
		t.Errorf("1 MB for %v s rejected", d)
	}
	if !TargetTooSmall(0.01, d) {
		t.Errorf("0.01 MB for %v s accepted", d)
	}
}

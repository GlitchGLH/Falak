package conv

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The owner's real case: a snapshot into a folder in C:\Program Files.
// FFmpeg's output ended with the statistics line, which was shown.
const snapStderr = `      Stream #0:0(und): Video: png, rgb24(pc, gbr/unknown/unknown, progressive), 640x360, q=2-31, 200 kb/s, 30 fps, 30 tbn (default)
        Metadata:
          encoder         : Lavc63.1.102 png
          handler_name    : VideoHandler
    [image2 @ 0000020603bc0340] The specified filename 'C:\Program Files\FalakTest\.falak-tmp\1-38.png' does not contain an image sequence pattern or a pattern is invalid.
    [image2 @ 0000020603bc0340] Use a pattern such as %03d for an image sequence or use the -update option (with -frames:v 1 if needed) to write a single image.
    [image2 @ 0000020603bc0340] Could not open file : C:\Program Files\FalakTest\.falak-tmp\1-38.png
    [vost#0:0/png @ 0000020603dc8040] Error submitting a packet to the muxer: No such file or directory
        Last message repeated 1 times
    [out#0/image2 @ 0000020603bac640] Error muxing a packet
    [out#0/image2 @ 0000020603bac640] Task finished with error code: -2 (No such file or directory)
    [out#0/image2 @ 0000020603bac640] Terminating thread with return code -2 (No such file or directory)
    [out#0/image2 @ 0000020603bac640] video:156KiB audio:0KiB subtitle:0KiB other streams:0KiB global headers:0KiB muxing overhead: unknown
    frame=    1 fps=0.0 q=-0.0 Lsize=N/A time=00:00:00.03 bitrate=N/A dup=1 drop=0 speed=1.66x elapsed=0:00:00.02    
    Conversion failed!
`

// TestFFmpegErrorLine: the line shown is the cause, never the statistics.
func TestFFmpegErrorLine(t *testing.T) {
	got := ffmpegError(errors.New("exit status 0xfffffffe"), nil, snapStderr)
	if want := `Could not open file : C:\Program Files\FalakTest\.falak-tmp\1-38.png`; got != want {
		t.Errorf("snapshot case: %q, want %q", got, want)
	}
	missing := "[in#0 @ 000001] Error opening input: No such file or directory\nError opening input file x.mp4.\nError opening input files: No such file or directory\n"
	if got := ffmpegError(nil, nil, missing); got != "Error opening input: No such file or directory" {
		t.Errorf("missing input: %q", got)
	}
	enc := "Stream mapping:\n  Stream #0:0 -> #0:0 (h264 -> libx265)\n[libx265 @ 0001] Unknown encoder 'libx265'\nframe=    0 fps=0.0 q=0.0 size=       0KiB time=N/A bitrate=N/A speed=N/A    \nConversion failed!\n"
	if got := ffmpegError(nil, nil, enc); got != "Unknown encoder 'libx265'" {
		t.Errorf("encoder: %q", got)
	}
	if got := ffmpegError(nil, nil, "frame=    1 fps=0.0 speed=1x\nConversion failed!\n"); strings.Contains(got, "frame=") {
		t.Errorf("only statistics: %q", got)
	}
	if got := Cause(ffmpegFailed("why")); got != "why" {
		t.Errorf("Cause %q", got)
	}
	if !strings.HasSuffix(ffmpegFailed("why").Error(), "why") {
		t.Error("the job error keeps its cause")
	}
}

// TestSnapshotUnwritableFolder: a folder Falak can't write in gives the
// folder and Windows' reason, not an FFmpeg line.
func TestSnapshotUnwritableFolder(t *testing.T) {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("no ffmpeg")
	}
	FFmpegOverride = ff
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mp4")
	mk(t, "-f", "lavfi", "-i", "testsrc=size=320x240:rate=30:duration=1", "-c:v", "libx264", "-pix_fmt", "yuv420p", src)
	// the output folder is a file: its temporary folder can't be made
	blocked := filepath.Join(dir, "blocked")
	os.WriteFile(blocked, []byte("x"), 0o644)
	_, err = ffmpegPre(context.Background(), nil, src, filepath.Join(blocked, "out.png"), []string{"-frames:v", "1"}, 0, nil)
	if err == nil {
		t.Fatal("no error")
	}
	msg := Cause(err)
	if strings.Contains(msg, "frame=") || strings.Contains(msg, "Conversion failed") || !strings.HasPrefix(msg, blocked+": ") {
		t.Fatalf("message %q, want the folder and the system's reason", msg)
	}
	t.Logf("shown: %s", msg)
}

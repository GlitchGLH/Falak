package conv

import (
	"archive/zip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mk(t *testing.T, args ...string) {
	out, err := exec.Command("ffmpeg", append([]string{"-hide_banner", "-loglevel", "error", "-y"}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("make sample: %v %s", err, out)
	}
}

func TestMedia(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("no ffmpeg")
	}
	FFmpegOverride, _ = exec.LookPath("ffmpeg")
	dir := t.TempDir()
	video := filepath.Join(dir, "clip.mov")
	mk(t, "-f", "lavfi", "-i", "testsrc=size=1280x720:rate=30:duration=4", "-f", "lavfi", "-i", "sine=frequency=440:duration=4",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-metadata", "title=secret", video)
	audio := filepath.Join(dir, "tone.wav")
	mk(t, "-f", "lavfi", "-i", "sine=frequency=330:duration=3", "-ac", "2", audio)

	for _, src := range []string{video, audio} {
		items := append(FormatItems(src), ToolItems(src)...)
		for _, it := range items {
			if !it.Enabled {
				continue
			}
			start := time.Now()
			var last float64
			out, err := Run(context.Background(), src, it.ID, func(p float64) { last = p })
			if err != nil {
				t.Errorf("%s %s: %v", filepath.Base(src), it.ID, err)
				continue
			}
			st, err := os.Stat(out)
			if err == nil && st.IsDir() { // tools that make a folder (split)
				parts, _ := os.ReadDir(out)
				t.Logf("%-9s %-11s -> folder with %d files", filepath.Base(src), it.ID, len(parts))
				if len(parts) < 2 {
					t.Errorf("%s %s: folder has %d files", filepath.Base(src), it.ID, len(parts))
				}
				continue
			}
			if err != nil || st.Size() == 0 {
				t.Errorf("%s %s: empty output", filepath.Base(src), it.ID)
				continue
			}
			mi := probe(out)
			desc := ""
			if mi.hasVideo {
				desc += "video:" + mi.vcodec + " "
			}
			if mi.hasAudio {
				desc += "audio:" + mi.acodec + " "
			}
			t.Logf("%-9s %-11s -> %-26s %8d bytes  %.1fs  %s progress=%.2f  (%v)", filepath.Base(src), it.ID, filepath.Base(out), st.Size(), mi.duration, desc, last, time.Since(start).Round(time.Millisecond))
		}
	}
	// metadata really removed
	clean, _ := filepath.Glob(filepath.Join(dir, "clip-clean*"))
	if len(clean) == 1 {
		out, _ := exec.Command("ffmpeg", "-hide_banner", "-i", clean[0]).CombinedOutput()
		if strings.Contains(string(out), "secret") {
			t.Errorf("metadata not removed")
		}
	}
	// channels: stereo -> mono
	mono, _ := filepath.Glob(filepath.Join(dir, "tone-mono*"))
	if len(mono) != 1 || probe(mono[0]).channels != 1 {
		t.Errorf("mono conversion failed: %v", mono)
	}
}

func TestSubtitlesAndArchives(t *testing.T) {
	dir := t.TempDir()
	srt := filepath.Join(dir, "movie.srt")
	os.WriteFile(srt, []byte("\xef\xbb\xbf1\r\n00:00:01,000 --> 00:00:02,500\r\n<i>Hello</i> there\r\n\r\n2\r\n00:00:03,000 --> 00:00:04,000\r\nSecond line\r\nwith two rows\r\n"), 0o644)
	for _, it := range FormatItems(srt) {
		out, err := Run(context.Background(), srt, it.ID, nil)
		if err != nil {
			t.Fatalf("%s: %v", it.ID, err)
		}
		b, _ := os.ReadFile(out)
		t.Logf("%s ->\n%s", it.ID, b)
	}
	vtt := filepath.Join(dir, "movie.vtt")
	back, err := Run(context.Background(), vtt, "to:SRT", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(back)
	if !strings.Contains(string(b), "00:00:01,000 --> 00:00:02,500") {
		t.Errorf("vtt->srt timing wrong:\n%s", b)
	}

	// archive: zip with a nested file
	zp := filepath.Join(dir, "pack.zip")
	f, _ := os.Create(zp)
	zw := zip.NewWriter(f)
	w, _ := zw.Create("folder/a.txt")
	w.Write([]byte("alpha"))
	w, _ = zw.Create("b.txt")
	w.Write([]byte("beta"))
	zw.Close()
	f.Close()
	for _, id := range []string{"to:TAR", "to:GZIP", "x:extract"} {
		out, err := Run(context.Background(), zp, id, nil)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		t.Logf("%s -> %s", id, filepath.Base(out))
	}
	if d, _ := os.ReadFile(filepath.Join(dir, "pack", "folder", "a.txt")); string(d) != "alpha" {
		t.Errorf("extract failed")
	}
	// round trip tar.gz -> zip -> extract
	out, err := Run(context.Background(), filepath.Join(dir, "pack.tar.gz"), "to:ZIP", nil)
	if err != nil {
		t.Fatal(err)
	}
	ex, err := Run(context.Background(), out, "x:extract", nil)
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := os.ReadFile(filepath.Join(ex, "b.txt")); string(d) != "beta" {
		t.Errorf("round trip failed: %s", ex)
	}
	// zip-slip protection
	evil := filepath.Join(dir, "evil.zip")
	f, _ = os.Create(evil)
	zw = zip.NewWriter(f)
	w, _ = zw.Create("../escape.txt")
	w.Write([]byte("x"))
	zw.Close()
	f.Close()
	if _, err := Run(context.Background(), evil, "x:extract", nil); err == nil {
		t.Errorf("unsafe path was not refused")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.txt")); err == nil {
		t.Errorf("file escaped the folder")
	}
}

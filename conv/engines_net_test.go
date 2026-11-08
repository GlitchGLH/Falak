//go:build enginesnet && windows

package conv

// Downloads every pinned engine (about 130 MB) into a temporary folder,
// checks it against its SHA-256 and runs it once. Not part of the normal
// gate; run it whenever a pin changes:
//
//	go test -tags enginesnet -run TestEnginePins -v ./conv

import (
	"context"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnginePins(t *testing.T) {
	old := ToolsDir
	// not t.TempDir(): pdfium.dll stays loaded in this process, so Windows
	// refuses to delete it and t.TempDir's cleanup would fail the test
	dir, err := os.MkdirTemp("", "falak-engines-")
	if err != nil {
		t.Fatal(err)
	}
	ToolsDir = dir
	defer func() { ToolsDir = old; os.RemoveAll(dir) }()
	FFmpegOverride, PDFiumOverride, QPDFOverride, ResvgOverride = "", "", "", ""
	work := t.TempDir()

	t.Run("ffmpeg", func(t *testing.T) {
		if err := EnsureFFmpeg(nil); err != nil {
			t.Fatal(err)
		}
		if !HaveFFmpeg() {
			t.Fatal("not marked installed")
		}
		out, err := exec.Command(FFmpegPath(), "-hide_banner", "-version").CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		for _, lib := range []string{"libx264", "libx265", "libvpx", "libmp3lame", "libopus", "libvorbis"} {
			if !strings.Contains(string(out), "--enable-"+lib) {
				t.Errorf("FFmpeg build lacks %s", lib)
			}
		}
		t.Log(strings.SplitN(string(out), "\n", 2)[0])
	})

	// a picture -> PDF (no engine) -> PNG (PDFium) and compressed (qpdf)
	pic := filepath.Join(work, "page.png")
	f, _ := os.Create(pic)
	png.Encode(f, photo(1200, 1600, false))
	f.Close()
	pdf, err := Run(context.Background(), pic, "to:PDF", nil)
	if err != nil {
		t.Fatalf("png -> pdf: %v", err)
	}

	t.Run("pdfium", func(t *testing.T) {
		if err := EnsurePDFium(nil); err != nil {
			t.Fatal(err)
		}
		if !HavePDFium() {
			t.Fatal("not marked installed")
		}
		out, err := Run(context.Background(), pdf, "to:PNG", nil)
		if err != nil {
			t.Fatalf("pdf -> png: %v", err)
		}
		t.Log("rendered", filepath.Base(out))
	})

	t.Run("qpdf", func(t *testing.T) {
		if err := EnsureQPDF(nil); err != nil {
			t.Fatal(err)
		}
		if !HaveQPDF() {
			t.Fatal("not marked installed")
		}
		out, err := exec.Command(QPDFPath(), "--version").CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		t.Log(strings.SplitN(string(out), "\n", 2)[0])
		if _, err := Run(context.Background(), pdf, "p:compress", nil); err != nil {
			t.Logf("compress: %v", err) // "already compact" is a valid answer
		}
	})

	t.Run("resvg", func(t *testing.T) {
		if err := EnsureResvg(nil); err != nil {
			t.Fatal(err)
		}
		if !HaveResvg() {
			t.Fatal("not marked installed")
		}
		svg := filepath.Join(work, "c.svg")
		os.WriteFile(svg, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><circle cx="32" cy="32" r="30" fill="#36c"/></svg>`), 0o644)
		out, err := Run(context.Background(), svg, "to:PNG", nil)
		if err != nil {
			t.Fatalf("svg -> png: %v", err)
		}
		t.Log("rendered", filepath.Base(out))
	})

	// every engine folder carries its pin file
	for _, p := range []enginePin{ffmpegPin, pdfiumPin, qpdfPin, resvgPin} {
		if !p.installed() {
			t.Errorf("%s: pin file missing", p.Dir)
		}
	}
}

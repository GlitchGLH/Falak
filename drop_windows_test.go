package main

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"raha/conv"
)

// TestDropCompressMixed reproduces the owner's real case: two JPGs, a MOV
// and a PDF from one folder, dropped together on COMPRESS (the wheel shows
// the first file's own Compress, here the PDF's). Through the same path as
// the app's drop (planDrop, runDropFiles), all four must be compressed into
// one "Compressed <date> <time>" folder under their original names.
func TestDropCompressMixed(t *testing.T) {
	ff, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("no ffmpeg")
	}
	conv.FFmpegOverride = ff
	if q := filepath.Join(os.Getenv("LOCALAPPDATA"), "Falak", "tools", "qpdf", "qpdf.exe"); conv.QPDFOverride == "" {
		conv.QPDFOverride = q // Falak's own installed copy (read only)
		defer func() { conv.QPDFOverride = "" }()
	}
	if !conv.HaveQPDF() {
		t.Skip("qpdf is not installed")
	}
	dir := t.TempDir()
	jpg := func(name string) string {
		p := filepath.Join(dir, name)
		img := image.NewRGBA(image.Rect(0, 0, 640, 480))
		for y := 0; y < 480; y++ {
			for x := 0; x < 640; x++ {
				img.Set(x, y, color.RGBA{uint8(x ^ y), uint8(x * 3), uint8(y * 5), 255})
			}
		}
		f, err := os.Create(p)
		if err != nil {
			t.Fatal(err)
		}
		jpeg.Encode(f, img, &jpeg.Options{Quality: 100})
		f.Close()
		return p
	}
	a, b := jpg("snapshots-ar-dark.jpg"), jpg("snapshots-ar-light.jpg")
	mov := filepath.Join(dir, "63434.mov")
	if out, err := exec.Command(ff, "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i",
		"testsrc=size=640x360:rate=30:duration=2", "-c:v", "libx264", "-pix_fmt", "yuv420p", mov).CombinedOutput(); err != nil {
		t.Fatalf("make video: %v %s", err, out)
	}
	made, err := conv.Run(context.Background(), a, "to:PDF", nil)
	if err != nil {
		t.Fatal(err)
	}
	pdf := filepath.Join(dir, "Plant Biotechnology and Genetics, 2008.pdf")
	if err := os.Rename(made, pdf); err != nil {
		t.Fatal(err)
	}

	paths := []string{pdf, a, b, mov} // Explorer gave the PDF first
	id := conv.CompressIDFor(paths[0])
	if id != "p:compress" {
		t.Fatalf("first file's Compress is %q", id)
	}
	todo, unfit := planDrop(paths, id)
	if len(unfit) != 0 {
		t.Fatalf("files left out: %v", unfit)
	}
	outs, errs := runDropFiles(context.Background(), todo, func(p, id string) error { return nil }, func(float64) {})
	if len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	var names []string
	folder := ""
	for _, o := range outs {
		names = append(names, filepath.Base(o))
		if folder == "" {
			folder = filepath.Dir(o)
		} else if filepath.Dir(o) != folder {
			t.Errorf("%s is not in %s", o, folder)
		}
	}
	sort.Strings(names)
	want := []string{"63434.mp4", "Plant Biotechnology and Genetics, 2008.pdf", "snapshots-ar-dark.jpg", "snapshots-ar-light.jpg"}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Fatalf("outputs %v, want %v", names, want)
	}
	if filepath.Dir(folder) != dir || !strings.HasPrefix(filepath.Base(folder), "Compressed ") {
		t.Fatalf("outputs are in %s, want a Compressed folder in %s", folder, dir)
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("original %s: %v", filepath.Base(p), err)
		}
	}

	// a file that can't be compressed is reported, never skipped silently;
	// a single file keeps the old naming beside the original
	txt := filepath.Join(dir, "notes.txt")
	os.WriteFile(txt, []byte("hi"), 0o644)
	if _, unfit := planDrop([]string{a, txt}, "compress"); len(unfit) != 1 || unfit[0] != txt {
		t.Errorf("notes.txt: unfit %v", unfit)
	}
	todo, _ = planDrop([]string{mov}, "v:compress")
	outs, errs = runDropFiles(context.Background(), todo, func(p, id string) error { return nil }, func(float64) {})
	if len(errs) != 0 || len(outs) != 1 || outs[0] != filepath.Join(dir, "63434-compressed.mp4") {
		t.Fatalf("one file: %v %v", outs, errs)
	}
}

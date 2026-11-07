package conv

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func hasItem(items []Item, id string) bool {
	for _, it := range items {
		if it.ID == id {
			return true
		}
	}
	return false
}

func itemOf(items []Item, id string) *Item {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func writeGIF(t *testing.T, dir, name string, frames int) string {
	t.Helper()
	var buf bytes.Buffer
	pal := color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{255, 0, 0, 255}}
	g := gif.GIF{}
	for i := 0; i < frames; i++ {
		p := image.NewPaletted(image.Rect(0, 0, 6, 6), pal)
		p.Set(i%6, i%6, pal[1])
		g.Image = append(g.Image, p)
		g.Delay = append(g.Delay, 10)
	}
	if err := gif.EncodeAll(&buf, &g); err != nil {
		t.Fatal(err)
	}
	return writeFile(t, dir, name, buf.String())
}

func TestTxtToSRTKeepsTimings(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "timed.txt", "1\n00:00:01,000 --> 00:00:02,500\nHello\n\n2\n00:00:03,000 --> 00:00:04,000\nWorld\n")
	out, err := Run(context.Background(), p, "to:SRT", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	s := string(b)
	if !strings.Contains(s, "00:00:01,000 --> 00:00:02,500") || !strings.Contains(s, "00:00:03,000 --> 00:00:04,000") ||
		!strings.Contains(s, "Hello") || !strings.Contains(s, "World") {
		t.Fatalf("timings not kept:\n%s", s)
	}
}

func TestTxtToSRTLinesArabic(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "lines.txt", "مرحبا\n\nكيف حالك\n")
	out, err := Run(context.Background(), p, "to:SRT", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	s := string(b)
	// مرحبا is 5 chars: 1 s + 0.3 s = 1.3 s, clamped to 1.5 s
	if !strings.Contains(s, "1\n00:00:00,000 --> 00:00:01,500\nمرحبا") ||
		!strings.Contains(s, "2\n00:00:01,500 --> 00:00:03,000\nكيف حالك") {
		t.Fatalf("unexpected cues:\n%s", s)
	}
}

func TestTxtToVTTLines(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "lines.txt", "first line\nsecond line\n")
	out, err := Run(context.Background(), p, "to:VTT", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	s := string(b)
	if !strings.HasPrefix(s, "WEBVTT\n\n") || !strings.Contains(s, "00:00:00.000 --> 00:00:01.600") {
		t.Fatalf("unexpected VTT:\n%s", s)
	}
}

// "first line" is 10 chars: 1 s + 0.6 s = 1.6 s.
func TestLineCuesClamp(t *testing.T) {
	long := strings.Repeat("x", 200)
	cues := lineCues([]string{"ab", "", long})
	if len(cues) != 2 {
		t.Fatalf("want 2 cues, got %d", len(cues))
	}
	if cues[0].timing != "00:00:00,000 --> 00:00:01,500" {
		t.Errorf("short line not clamped to 1.5 s: %q", cues[0].timing)
	}
	if cues[1].timing != "00:00:01,500 --> 00:00:08,500" {
		t.Errorf("long line not clamped to 7 s: %q", cues[1].timing)
	}
}

func TestSRTToTXT(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "sub.srt", "1\n00:00:01,000 --> 00:00:02,000\n<i>Hello</i>\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond\n")
	out, err := Run(context.Background(), p, "to:TXT", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	if got, want := strings.TrimSpace(string(b)), "Hello\nSecond"; got != want {
		t.Fatalf("TXT = %q, want %q", got, want)
	}
}

func TestTargetLists(t *testing.T) {
	dir := t.TempDir()
	oldHas := msOfficeHas
	defer func() { msOfficeHas = oldHas }()

	// JPG and PNG gain DOCX; other images do not
	if !hasItem(FormatItems(writeFile(t, dir, "photo.jpg", "\xff\xd8\xff\xe0")), "to:DOCX") {
		t.Error("JPG: DOCX target missing")
	}
	if !hasItem(FormatItems(writeFile(t, dir, "pic.png", "\x89PNG")), "to:DOCX") {
		t.Error("PNG: DOCX target missing")
	}
	if hasItem(FormatItems(writeFile(t, dir, "draw.webp", "RIFF")), "to:DOCX") {
		t.Error("WEBP should not offer DOCX")
	}

	// PDF gains DOCX only when Microsoft Word is installed
	pdf := writeFile(t, dir, "doc.pdf", "%PDF-1.4\n")
	msOfficeHas = func(app string) bool { return app == "word" }
	if it := itemOf(FormatItems(pdf), "to:DOCX"); it == nil || !it.Enabled {
		t.Error("PDF: DOCX should be offered and enabled with Word installed")
	}
	msOfficeHas = func(app string) bool { return false }
	if it := itemOf(FormatItems(pdf), "to:DOCX"); it == nil || it.Enabled {
		t.Error("PDF: DOCX should hide without Word")
	}

	// every audio input gains AIFF (except AIFF itself)
	if !hasItem(FormatItems(writeFile(t, dir, "song.mp3", "x")), "to:AIFF") {
		t.Error("MP3: AIFF target missing")
	}
	if hasItem(FormatItems(writeFile(t, dir, "song.aif", "x")), "to:AIFF") {
		t.Error("AIFF input should not offer AIFF")
	}

	// animated GIF gains AVI and WMV; a still GIF does not
	anim := FormatItems(writeGIF(t, dir, "anim.gif", 2))
	for _, tgt := range []string{"to:AVI", "to:WMV", "to:MP4"} {
		if !hasItem(anim, tgt) {
			t.Errorf("animated GIF: %s missing", tgt)
		}
	}
	still := FormatItems(writeGIF(t, dir, "still.gif", 1))
	if hasItem(still, "to:AVI") || hasItem(still, "to:WMV") {
		t.Error("still GIF should not offer AVI/WMV")
	}

	// TXT always offers PDF, JPG, PNG, SRT, VTT (+ DOCX/RTF/HTML)
	items := FormatItems(writeFile(t, dir, "plain.txt", "hello\nworld\n"))
	for _, tgt := range []string{"to:PDF", "to:JPG", "to:PNG", "to:SRT", "to:VTT", "to:DOCX", "to:RTF", "to:HTML"} {
		if !hasItem(items, tgt) {
			t.Errorf("TXT: %s missing", tgt)
		}
	}
	if hasItem(items, "to:ASS") {
		t.Error("plain TXT should not offer ASS")
	}

	// SRT gains TXT (and keeps ASS)
	srtItems := FormatItems(writeFile(t, dir, "sub.srt", "1\n00:00:01,000 --> 00:00:02,000\nHi\n"))
	for _, tgt := range []string{"to:TXT", "to:VTT", "to:ASS"} {
		if !hasItem(srtItems, tgt) {
			t.Errorf("SRT: %s missing", tgt)
		}
	}

	// ZIP offers TAR and GZIP (and 7Z), RAR only when WinRAR is there
	zipItems := FormatItems(writeFile(t, dir, "bundle.zip", "x"))
	for _, tgt := range []string{"to:TAR", "to:GZIP", "to:7Z"} {
		if !hasItem(zipItems, tgt) {
			t.Errorf("ZIP: %s missing", tgt)
		}
	}
	if it := itemOf(zipItems, "to:RAR"); it == nil || it.Enabled != (RarExe() != "") {
		t.Error("ZIP: RAR availability must follow WinRAR")
	}

	// RAR input converts to ZIP/TAR/GZIP
	rarItems := FormatItems(writeFile(t, dir, "bundle.rar", "x"))
	for _, tgt := range []string{"to:ZIP", "to:TAR", "to:GZIP"} {
		if !hasItem(rarItems, tgt) {
			t.Errorf("RAR: %s missing", tgt)
		}
	}
}

func TestImageDocx(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 150))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	var pb bytes.Buffer
	if err := png.Encode(&pb, img); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := writeFile(t, dir, "pic.png", pb.String())
	var buf bytes.Buffer
	if err := writeImageDocx(&buf, img, src); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	extent := []int{}
	for _, f := range zr.File {
		names = append(names, f.Name)
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			m := regexp.MustCompile(`wp:extent cx="([0-9]+)" cy="([0-9]+)"`).FindSubmatch(b)
			if m == nil {
				t.Fatalf("document.xml has no wp:extent: %.400s", string(b))
			}
			cx, _ := strconv.Atoi(string(m[1]))
			cy, _ := strconv.Atoi(string(m[2]))
			extent = append(extent, cx, cy)
		}
	}
	for _, want := range []string{"[Content_Types].xml", "word/document.xml", "word/_rels/document.xml.rels", "media/image1.png"} {
		if !contains(names, want) {
			t.Errorf("docx is missing %s (has %v)", want, names)
		}
	}
	// 300x150 at natural size: 2:1, inside the A4 page area
	if len(extent) != 2 || extent[0] != 2*extent[1] {
		t.Fatalf("extent %v is not the picture's 2:1 shape", extent)
	}
	if int64(extent[0]) > docxPageWEMU || int64(extent[1]) > docxPageHEMU {
		t.Fatalf("extent %v does not fit the page", extent)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestGzToZip(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	zw.Write([]byte("hello gz"))
	zw.Close()
	p := writeFile(t, dir, "note.gz", buf.String())
	out, err := Run(context.Background(), p, "to:ZIP", nil)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "note" {
		t.Fatalf("want one entry named note, got %v", zr.File)
	}
	rc, _ := zr.File[0].Open()
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "hello gz" {
		t.Fatalf("content = %q", string(b))
	}
}

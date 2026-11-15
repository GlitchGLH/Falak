package conv

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"
)

func TestIsCompanion(t *testing.T) {
	for name, want := range map[string]bool{
		"Thumbs.db":               true,
		"photos/thumbs.db":        true,
		"desktop.ini":             true,
		"a/b/Desktop.ini":         true,
		".DS_Store":               true,
		"photos/.DS_Store":        true,
		"__MACOSX/._1.jpg":        true,
		"__MACOSX/photos/._1.jpg": true,
		`photos\__macosx\x.png`:   true,
		"1.jpg":                   false,
		"photos/._1.jpg":          false, // only inside __MACOSX
		"thumbs.db.jpg":           false,
		"my __MACOSX notes/1.jpg": false,
	} {
		if got := isCompanion(name); got != want {
			t.Errorf("isCompanion(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestSortPages(t *testing.T) {
	in := []string{"10.jpg", "img 2.jpg", "B/3.jpg", "2.jpg", "a/10.png", "img.jpg", "a/2.png", "1.jpg"}
	want := []string{"1.jpg", "2.jpg", "10.jpg", "a/2.png", "a/10.png", "B/3.jpg", "img.jpg", "img 2.jpg"}
	sortPages(in)
	if !reflect.DeepEqual(in, want) {
		t.Fatalf("got %v\nwant %v", in, want)
	}
}

func TestImagesOnly(t *testing.T) {
	cases := []struct {
		names []string
		ok    bool
	}{
		{[]string{"1.jpg", "2.PNG", "scan.tiff", "raw.dng", "logo.svg", "pic.heic"}, true},
		{[]string{"1.jpg", "Thumbs.db", "desktop.ini", ".DS_Store", "__MACOSX/._1.jpg"}, true},
		{[]string{"1.jpg", "readme.txt"}, false},
		{[]string{"1.jpg", "clip.mp4"}, false},
		{[]string{"1.jpg", "noextension"}, false},
		{[]string{"Thumbs.db", "__MACOSX/._1.jpg"}, false}, // no image at all
		{nil, false},
	}
	for _, c := range cases {
		if _, ok := imagesOnly(c.names); ok != c.ok {
			t.Errorf("imagesOnly(%v) = %v, want %v", c.names, ok, c.ok)
		}
	}
}

// testPNG is a w×10 picture, so each page's width tells which image it is.
func testPNG(t *testing.T, w int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, 10))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(0, 0, color.NRGBA{1, 2, 3, 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func writeZip(t *testing.T, p string, files map[string][]byte, order []string) {
	t.Helper()
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, n := range order {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(files[n])
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func pdfItem(items []Item) *Item {
	for i := range items {
		if items[i].ID == "to:PDF" {
			return &items[i]
		}
	}
	return nil
}

var reMediaBox = regexp.MustCompile(`/MediaBox \[0 0 ([0-9.]+) ([0-9.]+)\]`)

// pageWidths returns each page's width in pixels (the writer uses 0.75 pt
// per pixel), in page order.
func pageWidths(t *testing.T, p string) []int {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, m := range reMediaBox.FindAllSubmatch(b, -1) {
		w, _ := strconv.ParseFloat(string(m[1]), 64)
		out = append(out, int(w/0.75+0.5))
	}
	return out
}

func TestArchiveToPDF(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		"pics/10.png":           testPNG(t, 30),
		"pics/2.png":            testPNG(t, 20),
		"1.png":                 testPNG(t, 10),
		"Thumbs.db":             []byte("not an image"),
		"__MACOSX/pics/._2.png": []byte("not an image either"),
	}
	zp := filepath.Join(dir, "album.zip")
	writeZip(t, zp, files, []string{"pics/10.png", "Thumbs.db", "pics/2.png", "__MACOSX/pics/._2.png", "1.png"})

	it := pdfItem(FormatItems(zp))
	if it == nil || !it.Enabled {
		t.Fatalf("ZIP of images: PDF item %+v, want enabled", it)
	}
	out, err := Run(context.Background(), zp, "to:PDF", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "album.pdf"); out != want {
		t.Fatalf("output %s, want %s", out, want)
	}
	if got, want := pageWidths(t, out), []int{10, 20, 30}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pages %v, want %v (1.png, pics/2.png, pics/10.png)", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, tempDirName)); err == nil {
		t.Error("temp folder left behind")
	}
	// never replaces: the second PDF gets the next name
	out2, err := Run(context.Background(), zp, "to:PDF", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "album (1).pdf"); out2 != want {
		t.Fatalf("second output %s, want %s", out2, want)
	}

	// the same archive as 7Z (Falak's own 7Z writer)
	sz, err := to7Z(context.Background(), zp, nil)
	if err != nil {
		t.Fatal(err)
	}
	if it := pdfItem(FormatItems(sz)); it == nil || !it.Enabled {
		t.Fatalf("7Z of images: PDF item %+v, want enabled", it)
	}
	out3, err := Run(context.Background(), sz, "to:PDF", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pageWidths(t, out3), []int{10, 20, 30}; !reflect.DeepEqual(got, want) {
		t.Fatalf("7Z pages %v, want %v", got, want)
	}
}

func TestArchivePDFDisabled(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "mixed.zip")
	writeZip(t, zp, map[string][]byte{"1.png": testPNG(t, 10), "notes.txt": []byte("hi")}, []string{"1.png", "notes.txt"})
	it := pdfItem(FormatItems(zp))
	if it == nil {
		t.Fatal("PDF must be shown (disabled), never hidden")
	}
	if it.Enabled {
		t.Fatal("PDF must be disabled when a file is not an image")
	}
	if _, err := Run(context.Background(), zp, "to:PDF", nil); err == nil {
		t.Fatal("running PDF on a mixed archive must fail")
	}

	empty := filepath.Join(dir, "empty.zip")
	writeZip(t, empty, nil, nil)
	if it := pdfItem(FormatItems(empty)); it == nil || it.Enabled {
		t.Fatalf("empty ZIP: PDF item %+v, want shown and disabled", it)
	}

	// TAR is not one of the formats that offer PDF
	tp := filepath.Join(dir, "pics.tar")
	f, _ := os.Create(tp)
	tw := tar.NewWriter(f)
	data := testPNG(t, 10)
	tw.WriteHeader(&tar.Header{Name: "1.png", Size: int64(len(data)), Mode: 0o644, Typeflag: tar.TypeReg})
	tw.Write(data)
	tw.Close()
	f.Close()
	if it := pdfItem(FormatItems(tp)); it != nil {
		t.Fatal("TAR should not offer PDF")
	}
}

func TestArchiveNeedsResvg(t *testing.T) {
	dir := t.TempDir()
	zp := filepath.Join(dir, "draw.zip")
	writeZip(t, zp, map[string][]byte{"a.svg": []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"/>`), "b.png": testPNG(t, 4)}, []string{"a.svg", "b.png"})
	if !NeedsResvg(zp, "to:PDF") {
		t.Error("an archive with an SVG needs resvg for PDF")
	}
	if NeedsResvg(zp, "to:7Z") || NeedsResvg(zp, "x:extract") {
		t.Error("only the PDF needs resvg")
	}
	if !NeedsResvg(filepath.Join(dir, "x.svg"), "to:PNG") {
		t.Error("an SVG file still needs resvg")
	}
}

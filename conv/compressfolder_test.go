package conv

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompressFolderName(t *testing.T) {
	at := time.Date(2026, 11, 15, 5, 30, 59, 0, time.Local)
	if got, want := CompressFolderName(at), "Compressed 2026-11-15 05-30"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestIsCompress(t *testing.T) {
	for _, id := range []string{"compress", "v:compress", "a:compress", "p:compress"} {
		if !IsCompress(id) {
			t.Errorf("%s should be Compress", id)
		}
	}
	for _, id := range []string{"to:PDF", "metadata", "v:mute", "i:merge", ""} {
		if IsCompress(id) {
			t.Errorf("%s is not Compress", id)
		}
	}
}

// TestCompressFolders: one folder per source folder; an existing folder of
// the same name is never reused, the " (n)" series continues.
func TestCompressFolders(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	os.Mkdir(a, 0o755)
	os.Mkdir(b, 0o755)
	at := time.Date(2026, 11, 15, 5, 30, 0, 0, time.Local)
	name := CompressFolderName(at)
	os.Mkdir(filepath.Join(a, name), 0o755) // already there in a
	os.WriteFile(filepath.Join(a, name, "keep.txt"), []byte("x"), 0o644)
	dirs := CompressFolders([]string{filepath.Join(a, "1.jpg"), filepath.Join(b, "2.jpg"), filepath.Join(a, "3.jpg")}, at)
	if len(dirs) != 2 {
		t.Fatalf("want 2 folders, got %v", dirs)
	}
	if got, want := dirs[a], filepath.Join(a, name+" (1)"); got != want {
		t.Errorf("folder in a: got %q, want %q", got, want)
	}
	if got, want := dirs[b], filepath.Join(b, name); got != want {
		t.Errorf("folder in b: got %q, want %q", got, want)
	}
	for _, d := range dirs {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Errorf("%s was not created", d)
		}
	}
	if _, err := os.Stat(filepath.Join(a, name, "keep.txt")); err != nil {
		t.Error("the existing folder was touched")
	}
}

func writeTestJPEG(t *testing.T, p string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{uint8(x ^ y), uint8(x * 3), uint8(y * 5), 255})
		}
	}
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

// TestCompressIntoFolder: with WithOutDir the copy goes into the folder
// (same name as today); without it, beside the original as before.
func TestCompressIntoFolder(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "photo.jpg")
	writeTestJPEG(t, src)

	out, err := Run(context.Background(), src, "compress", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "photo-compressed.jpg"); out != want {
		t.Fatalf("single file: got %s, want %s", out, want)
	}

	folder := filepath.Join(dir, "Compressed 2026-11-15 05-30")
	os.Mkdir(folder, 0o755)
	out, err = Run(WithOutDir(context.Background(), folder), src, "compress", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(folder, "photo-compressed.jpg"); out != want {
		t.Fatalf("in folder: got %s, want %s", out, want)
	}
	// a second run into the same folder never replaces the first copy
	out2, err := Run(WithOutDir(context.Background(), folder), src, "compress", nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(folder, "photo-compressed (1).jpg"); out2 != want {
		t.Fatalf("second copy: got %s, want %s", out2, want)
	}
	if _, err := os.Stat(filepath.Join(folder, tempDirName)); err == nil {
		t.Error("a temp folder was left in the compress folder")
	}
}

package conv

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func copyFile(t *testing.T, src, dir string) string {
	d, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, filepath.Base(src))
	os.WriteFile(p, d, 0o644)
	return p
}

func photo(w, h int, alpha bool) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			if alpha && x < w/3 {
				a = uint8(x * 255 / (w / 3))
			}
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 255 / w), uint8(y * 255 / h), uint8((x + y) % 256), a})
		}
	}
	return img
}

func TestAll(t *testing.T) {
	FFmpegOverride, _ = exec.LookPath("ffmpeg") // HEIC output uses FFmpeg
	dir := t.TempDir()
	// Synthetic sources.
	jp := filepath.Join(dir, "photo.jpg")
	var jb bytes.Buffer
	jpeg.Encode(&jb, photo(640, 480, false), &jpeg.Options{Quality: 95})
	// add EXIF with orientation 6 and a comment
	exif := minimalExif(6)
	data := append([]byte{0xFF, 0xD8}, exif...)
	data = append(data, 0xFF, 0xFE, 0, 6, 'h', 'i', '!', '!')
	data = append(data, jb.Bytes()[2:]...)
	os.WriteFile(jp, data, 0o644)
	pp := filepath.Join(dir, "logo.png")
	f, _ := os.Create(pp)
	png.Encode(f, photo(300, 200, true))
	f.Close()

	srcs := []string{jp, pp,
		copyFile(t, "testdata/test.heic", dir),
		copyFile(t, "testdata/exif.heic", dir),
		copyFile(t, "testdata/test_rot.avif", dir),
		copyFile(t, "testdata/exif.webp", dir),
	}
	for _, src := range srcs {
		srcImg, err := Decode(src)
		if err != nil {
			t.Fatalf("decode %s: %v", src, err)
		}
		sb := srcImg.Bounds()
		t.Logf("%s: %dx%d", filepath.Base(src), sb.Dx(), sb.Dy())
		items := append(FormatItems(src), ToolItems(src)...)
		for _, it := range items {
			if !it.Enabled {
				continue
			}
			start := time.Now()
			out, err := Run(context.Background(), src, it.ID, nil)
			if err != nil {
				if strings.Contains(err.Error(), "already compressed") {
					t.Logf("  %-10s skipped: %v", it.ID, err)
					continue
				}
				t.Errorf("%s %s: %v", src, it.ID, err)
				continue
			}
			st, _ := os.Stat(out)
			msg := ""
			if !strings.HasSuffix(out, ".pdf") && !strings.HasSuffix(out, ".docx") {
				img, err := Decode(out)
				if err != nil {
					t.Errorf("re-decode %s: %v", out, err)
					continue
				}
				b := img.Bounds()
				msg = " " + itoa(b.Dx()) + "x" + itoa(b.Dy())
				want := sb
				switch it.ID {
				case "rotate":
					want = image.Rect(0, 0, sb.Dy(), sb.Dx())
				case "resize":
					want = image.Rect(0, 0, sb.Dx()/2, sb.Dy()/2)
				}
				if it.ID == "to:ICO" {
					want = image.Rect(0, 0, b.Dx(), b.Dx())
				}
				if b.Dx() != want.Dx() || b.Dy() != want.Dy() {
					t.Errorf("%s %s: size %v want %v", filepath.Base(src), it.ID, b, want)
				}
			}
			t.Logf("  %-10s -> %-28s %7d bytes %s%v", it.ID, filepath.Base(out), st.Size(), msg, time.Since(start).Round(time.Millisecond))
		}
	}
	// Lossless metadata strip keeps image data identical.
	clean, _ := filepath.Glob(filepath.Join(dir, "photo-clean*.jpg"))
	if len(clean) == 1 {
		d, _ := os.ReadFile(clean[0])
		if jpegOrientation(d) != 6 || bytes.Contains(d, []byte("hi!!")) {
			t.Errorf("jpeg strip wrong")
		}
		if !bytes.HasSuffix(d, jb.Bytes()[len(jb.Bytes())-1000:]) {
			t.Errorf("jpeg scan data changed")
		}
	}
	// Orientation applied: photo.jpg is 640x480 with orientation 6 -> 480x640.
	img, _ := Decode(jp)
	if img.Bounds().Dx() != 480 {
		t.Errorf("orientation not applied")
	}
	// Name collisions.
	a, _ := Convert(pp, "JPG")
	b, _ := Convert(pp, "JPG")
	if a == b || !strings.Contains(b, "(") {
		t.Errorf("unique naming failed: %s %s", a, b)
	}
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Replace(string(rune('0'+0)), "0", "", 1)) + fmtInt(i)
}

func fmtInt(i int) string {
	if i == 0 {
		return "0"
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	return s
}

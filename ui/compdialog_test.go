package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func sizeBox(orig float64) *CompressDialog {
	d := &CompressDialog{Target: true, OriginalBytes: orig}
	d.TargetMB = d.DefaultTargetMB()
	return d
}

func typeKeys(d *CompressDialog, keys string) {
	for _, r := range keys {
		d.Char(r)
	}
}

// TestSizeBoxEditing: the field takes typing only while focused; the first
// digit after focusing replaces the offered number (it is shown selected),
// later digits append, Backspace works, at most 4 digits, Arabic-Indic
// digits count as digits, other keys are ignored.
func TestSizeBoxEditing(t *testing.T) {
	d := sizeBox(766e6)
	typeKeys(d, "5")
	if d.TargetMB != "100" {
		t.Fatalf("typing without focus changed the box: %q", d.TargetMB)
	}
	d.Focus()
	typeKeys(d, "25")
	if d.TargetMB != "25" {
		t.Fatalf("first digit should replace the selection: %q", d.TargetMB)
	}
	typeKeys(d, "x.-0")
	if d.TargetMB != "250" {
		t.Fatalf("only digits are taken: %q", d.TargetMB)
	}
	typeKeys(d, "12")
	if d.TargetMB != "2501" {
		t.Fatalf("at most 4 digits: %q", d.TargetMB)
	}
	typeKeys(d, "\b\b")
	if d.TargetMB != "25" {
		t.Fatalf("backspace: %q", d.TargetMB)
	}
	d.Focus() // focusing again selects the whole number
	typeKeys(d, "\b")
	if d.TargetMB != "" || d.StartActive() {
		t.Fatalf("backspace on the selection clears it and disables Compress: %q", d.TargetMB)
	}
	typeKeys(d, "٣٠٠")
	if d.TargetMB != "300" || !d.StartActive() {
		t.Fatalf("Arabic-Indic digits: %q active=%v", d.TargetMB, d.StartActive())
	}
	d.Blur()
	typeKeys(d, "9")
	if d.TargetMB != "300" {
		t.Fatalf("typing after the focus left changed the box: %q", d.TargetMB)
	}
}

// mib is one megabyte as File Explorer counts it.
const mib = 1 << 20

// TestSizeBoxRules: Compress runs only for a whole number > 0 that is
// smaller than the original (unchanged rules).
func TestSizeBoxRules(t *testing.T) {
	d := sizeBox(130 * mib)
	d.Focus()
	for _, c := range []struct {
		typed  string
		active bool
	}{{"0", false}, {"130", false}, {"131", false}, {"129", true}, {"1", true}} {
		d.Focus()
		typeKeys(d, c.typed)
		if got := d.StartActive(); got != c.active {
			t.Errorf("%s MB of a 130 MB file: active=%v, want %v", c.typed, got, c.active)
		}
	}
}

// TestSizeBoxUnit: the size box counts MB like File Explorer (1,048,576
// bytes). A file of 130,000,000 bytes shows as 124.0 MB, so 124 is not
// smaller than the original and 123 is.
func TestSizeBoxUnit(t *testing.T) {
	if BytesPerMB != 1048576 {
		t.Fatalf("BytesPerMB = %d, want 1048576", BytesPerMB)
	}
	d := sizeBox(130e6)
	for _, c := range []struct {
		typed  string
		active bool
	}{{"125", false}, {"124", false}, {"123", true}} {
		d.Focus()
		typeKeys(d, c.typed)
		if got := d.StartActive(); got != c.active {
			t.Errorf("%s MB of a 130,000,000-byte file: active=%v, want %v", c.typed, got, c.active)
		}
	}
}

// TestDefaultTargetMB: 100 MB for bigger files; half the original (whole
// MB, at least 1) for files of 100 MB or less, so the offer can be used.
func TestDefaultTargetMB(t *testing.T) {
	for _, c := range []struct {
		orig float64
		want string
	}{{0, "100"}, {766 * mib, "100"}, {130 * mib, "100"}, {100.5 * mib, "100"}, {100 * mib, "50"},
		{61 * mib, "30"}, {3 * mib, "1"}, {1.5 * mib, "1"}, {100e6, "47"}} {
		d := &CompressDialog{OriginalBytes: c.orig}
		if got := d.DefaultTargetMB(); got != c.want {
			t.Errorf("original %.1f MB: default %s, want %s", c.orig/mib, got, c.want)
		}
		if c.orig >= 3*mib && !sizeBox(c.orig).StartActive() {
			t.Errorf("original %.1f MB: the default %s MB cannot be used", c.orig/mib, c.want)
		}
	}
}

// TestSizeBoxRender draws the focused box (selection, caret) and the
// unfocused one without panicking, in both directions, when a font is
// available on this machine.
func TestSizeBoxRender(t *testing.T) {
	font := ""
	for _, f := range []string{filepath.Join(os.Getenv("WINDIR"), "Fonts", "segoeui.ttf"), "/usr/share/fonts/opentype/inter/Inter-SemiBold.otf"} {
		if _, err := os.Stat(f); err == nil {
			font = f
			break
		}
	}
	if font == "" {
		t.Skip("no font")
	}
	fonts := LoadFonts([]string{font}, []string{font})
	for _, rtl := range []bool{false, true} {
		d := NewCompressDialog(1.5, nil, fonts, true, rtl, CompressLabels{Target: "t", TargetLabel: "Target size (MB)", Start: "Compress"}, "130 MB")
		d.Target, d.OriginalBytes = true, 130e6
		d.TargetMB = d.DefaultTargetMB()
		if d.Render() == nil {
			t.Fatal("no image")
		}
		d.Focus()
		d.Render()
		typeKeys(d, "4")
		d.CaretOn = false
		d.Render()
	}
}

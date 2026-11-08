package conv

// SVG input: rendered by resvg (the most standards-accurate open-source SVG
// renderer), downloaded once on first use. The drawing is rendered so its
// longer side is at least svgMinSide pixels, which keeps it crisp.

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/srwiley/oksvg"
)

const (
	svgMinSide = 1024
	resvgURL   = "https://github.com/linebender/resvg/releases/latest/download/resvg-win-x86_64.zip"
)

// ResvgOverride lets tests use a specific resvg binary.
var ResvgOverride string

func ResvgPath() string {
	if ResvgOverride != "" {
		return ResvgOverride
	}
	return filepath.Join(ToolsDir, "resvg", exeName("resvg"))
}

func HaveResvg() bool {
	_, err := os.Stat(ResvgPath())
	return err == nil
}

// NeedsResvg reports whether a file is an SVG drawing.
func NeedsResvg(path string) bool { return Kind(path) == "SVG" }

// EnsureResvg downloads resvg if it is missing.
func EnsureResvg(progress func(float64)) error {
	if HaveResvg() {
		return nil
	}
	dir := filepath.Join(ToolsDir, "resvg")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(ToolsDir, "resvg-download.zip")
	defer os.Remove(tmp)
	if err := download(resvgURL, tmp, progress); err != nil {
		return fmt.Errorf(tr("Couldn't download the SVG engine: %w", "تعذّر تنزيل محرك SVG: %w"), err)
	}
	zr, err := zip.OpenReader(tmp)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) != "resvg.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out := filepath.Join(dir, "resvg.exe")
		w, err := os.Create(out + ".part")
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		w.Close()
		if err != nil {
			return err
		}
		return os.Rename(out+".part", out)
	}
	return errors.New(tr("resvg.exe wasn't found in the download", "لم يُعثر على resvg.exe داخل الملف"))
}

// svgSize reads the drawing's own size (0, 0 if unknown).
func svgSize(path string) (float64, float64) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	icon, err := oksvg.ReadIconStream(f, oksvg.IgnoreErrorMode)
	if err != nil {
		return 0, 0
	}
	return icon.ViewBox.W, icon.ViewBox.H
}

func decodeSVG(path string) (image.Image, error) {
	if !HaveResvg() {
		return nil, errors.New(tr("The SVG engine isn't available", "محرك SVG غير متاح"))
	}
	w, h := svgSize(path)
	args := []string{}
	switch {
	case w <= 0 || h <= 0:
		args = append(args, "--width", strconv.Itoa(svgMinSide))
	case math.Max(w, h) < svgMinSide && w >= h:
		args = append(args, "--width", strconv.Itoa(svgMinSide))
	case math.Max(w, h) < svgMinSide:
		args = append(args, "--height", strconv.Itoa(svgMinSide))
	case math.Max(w, h) > 8192 && w >= h:
		args = append(args, "--width", "8192")
	case math.Max(w, h) > 8192:
		args = append(args, "--height", "8192")
	}
	out := tempFor(path + ".png")
	defer dropTemp(out)
	cmd := exec.Command(ResvgPath(), append(args, path, out)...)
	PrepareCmd(cmd)
	if msg, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf(tr("This SVG couldn't be drawn: %s", "تعذّر رسم ملف SVG هذا: %s"), lastLine(string(msg)))
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

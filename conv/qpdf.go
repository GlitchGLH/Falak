package conv

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// qpdf (Apache-2.0) rewrites PDFs compactly: streams are recompressed at the
// strongest setting, objects are packed, and large uncompressed images are
// stored as high-quality JPEG only when that is smaller. Text and vector
// drawings are never touched, so pages stay sharp.
const qpdfURL = "https://github.com/qpdf/qpdf/releases/download/v12.3.2/qpdf-12.3.2-mingw64.zip"

var QPDFOverride string

func QPDFPath() string {
	if QPDFOverride != "" {
		return QPDFOverride
	}
	return filepath.Join(ToolsDir, "qpdf", "qpdf.exe")
}

func HaveQPDF() bool {
	_, err := os.Stat(QPDFPath())
	return err == nil
}

// NeedsQPDF reports whether the job uses qpdf.
func NeedsQPDF(path, id string) bool { return Category(path) == "pdf" && id == "p:compress" }

// EnsureQPDF downloads qpdf (its program and DLLs) if it is missing.
func EnsureQPDF(progress func(float64)) error {
	if HaveQPDF() {
		return nil
	}
	dir := filepath.Join(ToolsDir, "qpdf")
	part := dir + ".part"
	os.RemoveAll(part)
	if err := os.MkdirAll(part, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(part)
	tmp := filepath.Join(ToolsDir, "qpdf-download.zip")
	defer os.Remove(tmp)
	if err := download(qpdfURL, tmp, progress); err != nil {
		return fmt.Errorf(tr("Couldn't download the PDF compressor: %w", "تعذّر تنزيل محرك ضغط PDF: %w"), err)
	}
	zr, err := zip.OpenReader(tmp)
	if err != nil {
		return err
	}
	defer zr.Close()
	found := false
	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if !strings.Contains(f.Name, "/bin/") || !(strings.HasSuffix(name, ".dll") || name == "qpdf.exe") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(filepath.Join(part, name))
		if err == nil {
			_, err = io.Copy(w, rc)
			w.Close()
		}
		rc.Close()
		if err != nil {
			return err
		}
		found = found || name == "qpdf.exe"
	}
	if !found {
		return errors.New(tr("qpdf.exe wasn't found in the download", "لم يُعثر على qpdf.exe داخل الملف"))
	}
	os.RemoveAll(dir)
	return os.Rename(part, dir)
}

func compressPDF(ctx context.Context, path string, progress func(float64)) (string, error) {
	if !HaveQPDF() {
		return "", errors.New(tr("The PDF compressor isn't available", "محرك ضغط PDF غير متاح"))
	}
	dst := uniquePath(path, "-compressed", ".pdf")
	tmp := tempFor(dst)
	if progress != nil {
		progress(0.05)
	}
	cmd := exec.CommandContext(ctx, QPDFPath(), "--object-streams=generate", "--compress-streams=y",
		"--recompress-flate", "--compression-level=9", "--optimize-images", "--jpeg-quality=92",
		"--oi-min-area=40000", "--remove-unreferenced-resources=yes", "--", path, tmp)
	PrepareCmd(cmd)
	stop := estimate(progress, 0.05, 0.99, 1+fileMB(path)*0.12)
	out, err := cmd.CombinedOutput()
	stop()
	// exit code 3 means "done, with warnings"
	if err != nil {
		var ee *exec.ExitError
		if !(errors.As(err, &ee) && ee.ExitCode() == 3) {
			dropTemp(tmp)
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			if strings.Contains(string(out), "password") {
				return "", errors.New(tr("This PDF is password-protected", "ملف PDF هذا محمي بكلمة مرور"))
			}
			return "", fmt.Errorf(tr("Compression failed: %s", "فشل الضغط: %s"), lastLine(string(out)))
		}
	}
	a, b := fileSize(path), fileSize(tmp)
	if b <= 0 || b >= a*99/100 {
		dropTemp(tmp)
		return "", errors.New(tr("This PDF is already compact and can't get smaller without losing quality", "ملف PDF هذا مضغوط بالفعل ولا يمكن تصغيره أكثر دون أن تقل جودته"))
	}
	if progress != nil {
		progress(0.99)
	}
	return dst, finishTemp(tmp, dst)
}

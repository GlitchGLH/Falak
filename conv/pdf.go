package conv

// PDF input: pages are rendered by PDFium (Google's PDF engine, the one in
// Chrome), downloaded once on first use like FFmpeg.

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// PDFDPI is the render resolution for PDF pages: print quality.
const PDFDPI = 300

// pdfTargets are the formats a PDF converts to (one image per page).
var pdfTargets = []string{"PNG", "JPG", "WEBP", "HEIC", "TIFF", "TXT"}

// PDFiumOverride lets tests point at a specific library.
var PDFiumOverride string

func PDFiumPath() string {
	if PDFiumOverride != "" {
		return PDFiumOverride
	}
	return filepath.Join(ToolsDir, "pdfium", "pdfium.dll")
}

func HavePDFium() bool { return haveEngine(PDFiumOverride, PDFiumPath(), pdfiumPin) }

// NeedsPDFium reports whether running id on path uses PDFium.
func NeedsPDFium(path, id string) bool {
	if Category(path) == "pdf" {
		return true
	}
	// text to JPG/PNG goes through a PDF, whose pages PDFium renders
	t := strings.TrimPrefix(id, "to:")
	return Category(path) == "text" && (t == "JPG" || t == "PNG")
}

// EnsurePDFium downloads PDFium if it is missing.
func EnsurePDFium(progress func(float64)) error {
	if HavePDFium() {
		return nil
	}
	dir := filepath.Join(ToolsDir, "pdfium")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(ToolsDir, "pdfium-download.tgz")
	defer os.Remove(tmp)
	if err := download(pdfiumPin, tmp, progress); err != nil {
		return fmt.Errorf(tr("Couldn't download the PDF engine: %w", "تعذّر تنزيل محرك PDF: %w"), err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr0 := tar.NewReader(gz)
	for {
		h, err := tr0.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if strings.EqualFold(filepath.Base(h.Name), "pdfium.dll") {
			out := filepath.Join(dir, "pdfium.dll")
			w, err := os.Create(out + ".part")
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, tr0); err != nil {
				w.Close()
				return err
			}
			w.Close()
			if err := os.Rename(out+".part", out); err != nil {
				return err
			}
			return pdfiumPin.mark(dir)
		}
	}
	return errors.New(tr("pdfium.dll wasn't found in the download", "لم يُعثر على pdfium.dll داخل الملف"))
}

func pdfItems(path string) []Item {
	var out []Item
	for _, t := range pdfTargets {
		out = append(out, Item{ID: "to:" + t, Label: t, Enabled: true})
	}
	// DOCX needs Microsoft Word, which re-flows the PDF into an editable
	// document (like the Office targets of Word/Excel/PowerPoint files)
	out = append(out, Item{ID: "to:DOCX", Label: "DOCX", Enabled: msOfficeHas("word")})
	return out
}

// pdfDoc is implemented by the platform PDFium binding.
type pdfDoc interface {
	Pages() int
	Render(i int, dpi float64) (*image.RGBA, error)
	Text(i int) string
	Rotated() ([]byte, error)        // the whole document, every page turned 90° clockwise
	PageBytes(i int) ([]byte, error) // page i alone as a new PDF
	Close()
}

func pdfTools(paths []string) []Item {
	return []Item{
		{"p:compress", "COMPRESS", "compress", true},
		{"p:rotate", "ROTATE", "rotate", true},
		{"p:split", "SPLIT", "split", true},
		{"p:merge", "MERGE", "merge", countCategory(paths, "pdf") >= 2},
	}
}

// runPDFTool rotates every page, or splits the PDF into one file per page
// (in a new folder beside it). Text and quality stay exactly as they were.
func runPDFTool(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	if id == "p:compress" {
		return compressPDF(ctx, path, progress)
	}
	doc, err := openPDF(path)
	if err != nil {
		return "", err
	}
	defer doc.Close()
	n := doc.Pages()
	if n < 1 {
		return "", errors.New(tr("The PDF has no pages", "ملف PDF لا يحتوي على صفحات"))
	}
	switch id {
	case "p:rotate":
		b, err := doc.Rotated()
		if err != nil {
			return "", err
		}
		dst := uniquePath(path, "-rotated", ".pdf")
		return dst, writeAtomic(dst, func(w io.Writer) error { _, err := w.Write(b); return err })
	case "p:split":
		if n == 1 {
			return "", errors.New(tr("This PDF has only one page", "ملف PDF هذا يحتوي على صفحة واحدة فقط"))
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		root := uniqueDir(filepath.Dir(path), base+" - pages")
		tmp := tempFor(root)
		if err := os.MkdirAll(tmp, 0o755); err != nil {
			return "", err
		}
		width := len(fmt.Sprint(n))
		for i := 0; i < n; i++ {
			if err := ctx.Err(); err != nil {
				dropTemp(tmp)
				return "", err
			}
			b, err := doc.PageBytes(i)
			if err == nil {
				err = os.WriteFile(filepath.Join(tmp, fmt.Sprintf("%s - %0*d.pdf", base, width, i+1)), b, 0o644)
			}
			if err != nil {
				dropTemp(tmp)
				return "", err
			}
			if progress != nil {
				progress(float64(i+1) / float64(n))
			}
		}
		return root, finishTemp(tmp, root)
	}
	return "", errors.New(tr("This tool isn't available", "هذه الأداة غير متاحة"))
}

func runPDF(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	t := strings.TrimPrefix(id, "to:")
	if !strings.HasPrefix(id, "to:") {
		return runPDFTool(ctx, path, id, progress)
	}
	if t == "DOCX" {
		return pdfToDOCX(ctx, path, progress)
	}
	doc, err := openPDF(path)
	if err != nil {
		return "", err
	}
	defer doc.Close()
	n := doc.Pages()
	if n < 1 {
		return "", errors.New(tr("The PDF has no pages", "ملف PDF لا يحتوي على صفحات"))
	}
	if t == "TXT" {
		dst := uniquePath(path, "", ".txt")
		return dst, writeAtomic(dst, func(w io.Writer) error {
			w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM so Notepad reads Arabic correctly
			for i := 0; i < n; i++ {
				if i > 0 {
					io.WriteString(w, "\r\n\f\r\n")
				}
				io.WriteString(w, strings.ReplaceAll(doc.Text(i), "\r\n", "\n"))
				if progress != nil {
					progress(float64(i+1) / float64(n))
				}
			}
			return nil
		})
	}
	page := func(i int, dst string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		// each page moves the bar through its own share
		lo, hi := float64(i)/float64(n), float64(i+1)/float64(n)
		stopR := estimate(progress, lo, lo+(hi-lo)*0.3, 0.6)
		img, err := doc.Render(i, PDFDPI)
		stopR()
		if err != nil {
			return err
		}
		return writeAtomic(dst, func(w io.Writer) error {
			if progress == nil {
				return EncodeBest(w, img, t, true)
			}
			stop := estimate(progress, lo+(hi-lo)*0.3, lo+(hi-lo)*0.99, encodeSeconds(img, t, true))
			defer stop()
			return EncodeBest(w, img, t, true)
		})
	}
	if n == 1 {
		dst := uniquePath(path, "", extFor(t))
		return dst, page(0, dst)
	}
	// Several pages: one image per page, in a new folder beside the PDF.
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	root := uniqueDir(filepath.Dir(path), base+" - "+t)
	tmp := tempFor(root)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	width := len(fmt.Sprint(n))
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%s - %0*d%s", base, width, i+1, extFor(t))
		if err := page(i, filepath.Join(tmp, name)); err != nil {
			dropTemp(tmp)
			return "", err
		}
		if progress != nil {
			progress(float64(i+1) / float64(n))
		}
	}
	return root, finishTemp(tmp, root)
}

// pdfToDOCX turns a PDF into a Word document with Microsoft Word's own
// re-flow (the same automation the Office files use).
func pdfToDOCX(ctx context.Context, path string, progress func(float64)) (string, error) {
	if !msOfficeHas("word") {
		return "", errors.New(tr("This needs Microsoft Office or LibreOffice installed", "هذا يحتاج تثبيت Microsoft Office أو LibreOffice"))
	}
	stop := estimate(progress, 0.02, 0.99, 5+fileMB(path)*0.8)
	defer stop()
	dst := uniquePath(path, "", ".docx")
	tmp := tempFor(dst)
	abs, _ := filepath.Abs(path)
	err := officeMS(ctx, "word", abs, tmp, msFormat["word"]["DOCX"])
	if err == nil {
		if _, e := os.Stat(tmp); e != nil {
			err = errors.New(tr("The office program didn't produce a file", "لم يُنتج برنامج Office ملفًا"))
		}
	}
	if err != nil {
		dropTemp(tmp)
		return "", err
	}
	return dst, finishTemp(tmp, dst)
}

func utf16ToString(u []uint16) string {
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	return string(utf16.Decode(u))
}

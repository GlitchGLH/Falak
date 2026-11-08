package conv

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// IsMerge reports whether a tool combines the whole selection into one file.
func IsMerge(id string) bool { return id == "i:merge" || id == "p:merge" }

func countCategory(paths []string, cat string) int {
	n := 0
	for _, p := range paths {
		if Category(p) == cat {
			n++
		}
	}
	return n
}

// Merge combines the selected images (as pages) or PDFs into one PDF next
// to the first file. Files are taken in name order, numbers counted as
// numbers (2 before 10), like File Explorer sorts them.
func Merge(ctx context.Context, paths []string, id string, progress func(float64)) (string, error) {
	cat := "image"
	if id == "p:merge" {
		cat = "pdf"
	}
	var files []string
	for _, p := range paths {
		if Category(p) == cat {
			files = append(files, p)
		}
	}
	if len(files) < 2 {
		return "", errors.New(tr("Select two or more files to merge", "حدّد ملفين أو أكثر للدمج"))
	}
	stem := func(p string) string { b := filepath.Base(p); return strings.TrimSuffix(b, filepath.Ext(b)) }
	sort.SliceStable(files, func(i, j int) bool { return naturalLess(stem(files[i]), stem(files[j])) })
	dst := uniquePath(files[0], "-merged", ".pdf")
	if cat == "pdf" {
		b, err := mergePDFs(ctx, files, progress)
		if err != nil {
			return "", err
		}
		return writeAtomic(dst, func(w io.Writer) error { _, err := w.Write(b); return err })
	}
	return writeAtomic(dst, func(w io.Writer) error { return imagesToPDF(ctx, w, files, progress) })
}

// imagesToPDF writes one page per image, each page the size of its image.
// JPEG files are embedded as they are; other images are stored losslessly.
func imagesToPDF(ctx context.Context, out io.Writer, files []string, progress func(float64)) error {
	bw := bufio.NewWriterSize(out, 1<<20)
	pw := &pdfStream{w: bw}
	pw.raw("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	// 1 = catalog, 2 = page tree, written at the end
	pw.next = 3
	var kids []int
	for i, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		lo, hi := float64(i)/float64(len(files)), float64(i+1)/float64(len(files))
		imgObj, iw, ih, err := pw.addImage(f, part(progress, lo*0.99, hi*0.99))
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
		pwPt, phPt := float64(iw)*0.75, float64(ih)*0.75
		content := []byte(fmt.Sprintf("q %.2f 0 0 %.2f 0 0 cm /Im0 Do Q", pwPt, phPt))
		c := pw.obj(stream("", content))
		kids = append(kids, pw.obj([]byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /XObject << /Im0 %d 0 R >> >> /Contents %d 0 R >>", pwPt, phPt, imgObj, c))))
		if progress != nil {
			progress(float64(i+1) / float64(len(files)) * 0.99)
		}
	}
	var k strings.Builder
	for _, id := range kids {
		fmt.Fprintf(&k, "%d 0 R ", id)
	}
	pw.objAt(1, []byte("<< /Type /Catalog /Pages 2 0 R >>"))
	pw.objAt(2, []byte(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.TrimSpace(k.String()), len(kids))))
	pw.finish(1)
	return bw.Flush()
}

type pdfStream struct {
	w       *bufio.Writer
	n       int64
	offsets map[int]int64
	next    int
	err     error
}

func (p *pdfStream) raw(s string) { p.bytes([]byte(s)) }

func (p *pdfStream) bytes(b []byte) {
	if p.err != nil {
		return
	}
	k, err := p.w.Write(b)
	p.n += int64(k)
	p.err = err
}

func (p *pdfStream) objAt(id int, body []byte) {
	if p.offsets == nil {
		p.offsets = map[int]int64{}
	}
	p.offsets[id] = p.n
	p.raw(fmt.Sprintf("%d 0 obj\n", id))
	p.bytes(body)
	p.raw("\nendobj\n")
}

func (p *pdfStream) obj(body []byte) int {
	id := p.next
	p.next++
	p.objAt(id, body)
	return id
}

func (p *pdfStream) finish(root int) {
	xref := p.n
	size := p.next
	p.raw(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", size))
	for i := 1; i < size; i++ {
		p.raw(fmt.Sprintf("%010d 00000 n \n", p.offsets[i]))
	}
	p.raw(fmt.Sprintf("trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", size, root, xref))
}

func (p *pdfStream) addImage(path string, prog func(float64)) (int, int, int, error) {
	if Kind(path) == "JPG" {
		if data, err := os.ReadFile(path); err == nil && jpegOrientation(data) <= 1 {
			if w, h, comps, ok := jpegInfo(data); ok && (comps == 1 || comps == 3) {
				cs := "/DeviceRGB"
				if comps == 1 {
					cs = "/DeviceGray"
				}
				id := p.obj(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace %s /BitsPerComponent 8 /Filter /DCTDecode", w, h, cs), data))
				return id, w, h, p.err
			}
		}
	}
	img, err := decodeP(path, part(prog, 0, 0.3))
	if err != nil {
		return 0, 0, 0, err
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	n := toNRGBA(img)
	rgb := make([]byte, 0, iw*ih*3)
	for i := 0; i < len(n.Pix); i += 4 {
		rgb = append(rgb, n.Pix[i], n.Pix[i+1], n.Pix[i+2])
	}
	smask := ""
	if hasAlpha(img) {
		alpha := make([]byte, 0, iw*ih)
		for i := 3; i < len(n.Pix); i += 4 {
			alpha = append(alpha, n.Pix[i])
		}
		sm := p.obj(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode", iw, ih), deflateP(alpha, part(prog, 0.3, 0.45))))
		smask = fmt.Sprintf(" /SMask %d 0 R", sm)
	}
	id := p.obj(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode%s", iw, ih, smask), deflateP(rgb, part(prog, 0.45, 1))))
	return id, iw, ih, p.err
}

// naturalLess compares names with digit runs taken as numbers.
func naturalLess(a, b string) bool {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	i, j := 0, 0
	for i < len(ra) && j < len(rb) {
		if unicode.IsDigit(ra[i]) && unicode.IsDigit(rb[j]) {
			si, sj := i, j
			for i < len(ra) && unicode.IsDigit(ra[i]) {
				i++
			}
			for j < len(rb) && unicode.IsDigit(rb[j]) {
				j++
			}
			na := strings.TrimLeft(string(ra[si:i]), "0")
			nb := strings.TrimLeft(string(rb[sj:j]), "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			continue
		}
		if ra[i] != rb[j] {
			return ra[i] < rb[j]
		}
		i++
		j++
	}
	return len(ra)-i < len(rb)-j
}

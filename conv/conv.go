// Package conv implements the local, offline conversions and tools.
package conv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	"github.com/gen2brain/webp"
	"golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/tiff"
)

// Item describes one wheel entry.
type Item struct {
	ID, Label, Icon string
	Enabled         bool
}

var imageExts = map[string]string{
	".jpg": "JPG", ".jpeg": "JPG", ".jfif": "JPG", ".png": "PNG", ".webp": "WEBP",
	".avif": "AVIF", ".heic": "HEIC", ".heif": "HEIC", ".bmp": "BMP", ".tif": "TIFF",
	".tiff": "TIFF", ".gif": "GIF", ".ico": "ICO", ".svg": "SVG",
	// camera RAW photos
	".cr2": "RAW", ".cr3": "RAW", ".nef": "RAW", ".nrw": "RAW", ".arw": "RAW", ".dng": "RAW",
	".orf": "RAW", ".rw2": "RAW", ".raf": "RAW", ".pef": "RAW", ".srw": "RAW",
}

// GIF is not offered for still images (it only makes sense for motion,
// e.g. video to GIF); HEIC takes its place.
var imageTargets = []string{"JPG", "PNG", "WEBP", "HEIC", "AVIF", "TIFF", "BMP", "ICO", "PDF"}

// Animated GIFs convert to video, plus still formats of the first frame.
var gifTargets = []string{"MP4", "MOV", "WEBM", "MKV", "AVI", "WMV", "PNG", "JPG", "WEBP", "HEIC"}

// Kind returns the short format name for a path ("" if unsupported).
func Kind(path string) string { return imageExts[strings.ToLower(filepath.Ext(path))] }

// ExtLabel is the label shown in the wheel center for unsupported files.
func ExtLabel(path string) string {
	e := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if len(e) > 5 {
		e = e[:5]
	}
	return e
}

// FormatItems lists the conversion targets for a file.
func FormatItems(path string) []Item {
	switch Category(path) {
	case "video":
		return videoItems(path)
	case "audio":
		return audioItems(path)
	case "subtitle":
		return subtitleItems(path)
	case "text":
		return textItems(path)
	case "archive":
		return archiveItems(path)
	case "pdf":
		return pdfItems(path)
	case "office":
		return officeItems(path)
	}
	k := Kind(path)
	if k == "" {
		return nil
	}
	targets := imageTargets
	if k == "GIF" && gifIsAnimated(path) {
		targets = gifTargets
	}
	var out []Item
	for _, t := range targets {
		if t == k {
			continue
		}
		out = append(out, Item{ID: "to:" + t, Label: t, Enabled: true})
	}
	if k == "JPG" || k == "PNG" {
		// a Word file holding the picture, page-fitted, written directly
		// (no office suite needed)
		out = append(out, Item{ID: "to:DOCX", Label: "DOCX", Enabled: true})
	}
	return out
}

// ToolItems lists the editing tools for a file.
func ToolItems(path string) []Item { return ToolItemsFor([]string{path}) }

// ToolItemsFor lists the tools for a selection (its first file decides the
// kind; merging needs two or more files of that kind).
func ToolItemsFor(paths []string) []Item {
	if len(paths) == 0 {
		return nil
	}
	path := paths[0]
	switch Category(path) {
	case "video":
		return videoTools(path)
	case "audio":
		return audioTools(path)
	case "archive":
		return archiveTools(path)
	case "pdf":
		return pdfTools(paths)
	case "subtitle", "office", "text":
		return nil
	}
	k := Kind(path)
	if k == "" || k == "SVG" || k == "RAW" {
		return nil // these only convert to other formats
	}
	// Only tools that really work for this file are offered.
	var out []Item
	if k != "ICO" {
		out = append(out, Item{"compress", "COMPRESS", "compress", true})
	}
	if k != "HEIC" && k != "ICO" {
		out = append(out, Item{"metadata", "METADATA", "metadata", true})
	}
	out = append(out,
		Item{"rotate", "ROTATE", "rotate", true},
		Item{"resize", "RESIZE 50%", "resize", true})
	// a background can only be added to formats that can be transparent
	alpha := false
	switch k {
	case "PNG", "WEBP", "TIFF", "AVIF", "GIF", "ICO":
		alpha = true
	}
	out = append(out,
		Item{"addbg", "WHITE BG", "addbg", alpha},
		Item{"blackbg", "BLACK BG", "blackbg", alpha},
		Item{"i:merge", "MERGE PDF", "merge", countCategory(paths, "image") >= 2})
	return out
}

// solidBG puts an opaque background of grey level v behind the image.
func solidBG(v uint32) func(image.Image) image.Image {
	return func(img image.Image) image.Image { return onBG(img, v) }
}

func onBG(img image.Image, v uint32) image.Image {
	src := toNRGBA(img)
	out := image.NewNRGBA(src.Rect)
	for i := 0; i+3 < len(src.Pix); i += 4 {
		a := uint32(src.Pix[i+3])
		for c := 0; c < 3; c++ {
			out.Pix[i+c] = uint8((uint32(src.Pix[i+c])*a + v*(255-a) + 127) / 255)
		}
		out.Pix[i+3] = 255
	}
	return out
}

// Run executes an item ID ("to:PNG", "compress", ...) on path and returns
// the new file. progress (may be nil) receives 0..1 for long operations.
func Run(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	switch Category(path) {
	case "video":
		return runVideo(ctx, path, id, progress)
	case "audio":
		return runAudio(ctx, path, id, progress)
	case "subtitle":
		return runSubtitle(path, id)
	case "text":
		return runText(ctx, path, id, progress)
	case "archive":
		return runArchive(ctx, path, id, progress)
	case "pdf":
		return runPDF(ctx, path, id, progress)
	case "office":
		return runOffice(ctx, path, id, progress)
	}
	if strings.HasPrefix(id, "to:") {
		t := strings.TrimPrefix(id, "to:")
		if Kind(path) == "GIF" && videoTarget[t] {
			return gifToVideo(ctx, path, t, progress)
		}
		return convertP(path, t, progress)
	}
	switch id {
	case "compress":
		return compressP(path, progress)
	case "metadata":
		return StripMetadata(path)
	case "rotate":
		return transform(path, "rotated", rotate90, progress)
	case "resize":
		return transform(path, "50%", half, progress)
	case "addbg":
		return transform(path, "white-bg", solidBG(255), progress)
	case "blackbg":
		return transform(path, "black-bg", solidBG(0), progress)
	}
	return "", errors.New(tr("This tool isn't available yet", "هذه الأداة غير متاحة بعد"))
}

// Decode reads any supported image, applying JPEG EXIF orientation.
func Decode(path string) (image.Image, error) { return decodeP(path, nil) }

// progReader reports how much of the file the decoder has consumed.
type progReader struct {
	r    *bytes.Reader
	n    int
	last float64
	f    func(float64)
}

func (p *progReader) Read(b []byte) (int, error) {
	k, err := p.r.Read(b)
	p.report()
	return k, err
}

func (p *progReader) ReadByte() (byte, error) {
	c, err := p.r.ReadByte()
	p.report()
	return c, err
}

func (p *progReader) report() {
	if p.f == nil || p.n == 0 {
		return
	}
	v := float64(p.n-p.r.Len()) / float64(p.n)
	if v-p.last >= 0.01 || v >= 1 {
		p.last = v
		p.f(v)
	}
}

// image work: reading the file is the first 45%, writing the result the rest
func decodeStage(f func(float64)) func(float64) {
	if f == nil {
		return nil
	}
	return func(v float64) { f(0.45 * v) }
}

// encodeP writes img in format; reading took the first 45% of the bar, so
// writing fills 45% to 99% (the last step happens when the file is in place).
func encodeP(w io.Writer, img image.Image, format string, lossless bool, f func(float64)) error {
	if f == nil {
		return EncodeBest(w, img, format, lossless)
	}
	f(0.45)
	if format == "PDF" { // our own writer reports exactly how far it is
		return encodePDFP(w, img, func(v float64) { f(0.45 + 0.54*v) })
	}
	stop := estimate(f, 0.45, 0.99, encodeSeconds(img, format, lossless))
	defer stop()
	return EncodeBest(w, img, format, lossless)
}

func fileSize(path string) int64 {
	if fi, err := os.Stat(path); err == nil {
		return fi.Size()
	}
	return 0
}

// maxPixels is the largest picture handled (about 250 megapixels, e.g.
// 20000×12500), which needs roughly 1 GB of memory per copy.
const maxPixels = 250_000_000

func decodeP(path string, prog func(float64)) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch Kind(path) {
	case "ICO":
		return decodeICO(data)
	case "SVG":
		return decodeSVG(path)
	case "RAW":
		return decodeRAW(path, data)
	}
	// Refuse pictures so large that decoding them would exhaust memory
	// (a crafted or extreme file must never take the whole app down).
	if cfg, _, cerr := image.DecodeConfig(bytes.NewReader(data)); cerr == nil {
		if int64(cfg.Width)*int64(cfg.Height) > maxPixels {
			return nil, fmt.Errorf(tr("This image is too large (%d×%d)", "هذه الصورة كبيرة جدًا (%d×%d)"), cfg.Width, cfg.Height)
		}
	}
	var img image.Image
	rd := &progReader{r: bytes.NewReader(data), n: len(data), f: prog}
	switch Kind(path) {
	case "WEBP":
		img, err = webp.Decode(rd, webp.Options{AutoRotate: true})
	case "AVIF":
		img, err = avif.Decode(rd, avif.Options{AutoRotate: true})
	default:
		img, _, err = image.Decode(rd)
	}
	if err != nil {
		return nil, fmt.Errorf(tr("This image couldn't be read: %w", "تعذّرت قراءة هذه الصورة: %w"), err)
	}
	if o := jpegOrientation(data); o > 1 {
		img = applyOrientation(img, o)
	}
	return img, nil
}

func extFor(format string) string {
	switch format {
	case "JPG":
		return ".jpg"
	case "TIFF":
		return ".tiff"
	}
	return "." + strings.ToLower(format)
}

// uniquePath returns dir/base+suffix+ext, adding " (n)" if it exists.
func uniquePath(src, suffix, ext string) string {
	dir := filepath.Dir(src)
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + suffix
	p := filepath.Join(dir, base+ext)
	for i := 1; ; i++ {
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
	}
}

// writeAtomic writes via a temp file so a failed conversion leaves nothing behind.
func writeAtomic(dst string, write func(io.Writer) error) error {
	tmp := tempFor(dst)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := write(f); err != nil {
		f.Close()
		dropTemp(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		dropTemp(tmp)
		return err
	}
	return finishTemp(tmp, dst)
}

// Convert converts an image file into the target format beside the original.
func Convert(path, target string) (string, error) { return convertP(path, target, nil) }

func convertP(path, target string, prog func(float64)) (string, error) {
	img, err := decodeP(path, decodeStage(prog))
	if err != nil {
		return "", err
	}
	dst := uniquePath(path, "", extFor(target))
	if target == "PDF" && Kind(path) == "JPG" {
		// Embed the original JPEG data untouched (no re-compression).
		if data, err := os.ReadFile(path); err == nil && jpegOrientation(data) <= 1 {
			if pdf, ok := pdfFromJPEG(data); ok {
				return dst, writeAtomic(dst, func(w io.Writer) error { _, err := w.Write(pdf); return err })
			}
		}
	}
	if target == "DOCX" {
		return dst, writeAtomic(dst, func(w io.Writer) error { return writeImageDocx(w, img, path) })
	}
	return dst, writeAtomic(dst, func(w io.Writer) error {
		return encodeP(w, img, target, losslessKind(Kind(path)), prog)
	})
}

// losslessKind reports whether a source format stores pixels without loss.
func losslessKind(k string) bool {
	switch k {
	case "PNG", "BMP", "TIFF", "GIF", "ICO", "SVG":
		return true
	}
	return false
}

// EncodeBest writes img at the highest useful quality: lossless when the
// source was lossless, otherwise the top quality setting of the format.
func EncodeBest(w io.Writer, img image.Image, format string, srcLossless bool) error {
	switch format {
	case "HEIC":
		return EncodeHEIC(w, img)
	case "WEBP":
		if srcLossless {
			return webp.Encode(w, img, webp.Options{Lossless: true, Exact: true, Method: 4})
		}
		return webp.Encode(w, img, webp.Options{Quality: 95, Method: 6})
	case "AVIF":
		if srcLossless {
			return avif.Encode(w, img, avif.Options{Lossless: true, Speed: 6})
		}
		return avif.Encode(w, img, avif.Options{Quality: 95, QualityAlpha: 100, Speed: 6, ChromaSubsampling: image.YCbCrSubsampleRatio444})
	}
	return Encode(w, img, format, 0)
}

// Encode writes img in format. quality 0 means the default for that format.
func Encode(w io.Writer, img image.Image, format string, quality int) error {
	switch format {
	case "JPG":
		if quality == 0 {
			quality = 100
		}
		return jpeg.Encode(w, flatten(img), &jpeg.Options{Quality: quality})
	case "PNG":
		return png.Encode(w, img)
	case "WEBP":
		if quality == 0 {
			quality = 88
		}
		return webp.Encode(w, img, webp.Options{Quality: quality, Method: 4})
	case "AVIF":
		if quality == 0 {
			quality = 70
		}
		return avif.Encode(w, img, avif.Options{Quality: quality, QualityAlpha: quality, Speed: 8, ChromaSubsampling: image.YCbCrSubsampleRatio420})
	case "TIFF":
		return tiff.Encode(w, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
	case "BMP":
		return bmp.Encode(w, img)
	case "GIF":
		return encodeGIF(w, img)
	case "ICO":
		return encodeICO(w, img)
	case "PDF":
		return encodePDF(w, img)
	}
	return fmt.Errorf("unsupported target %s", format)
}

func hasAlpha(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return !o.Opaque()
	}
	return true
}

// flatten composites img onto white (for formats without transparency).
func flatten(img image.Image) image.Image {
	if !hasAlpha(img) {
		return img
	}
	b := img.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Over)
	return out
}

func toNRGBA(img image.Image) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), img, b.Min, draw.Src)
	return out
}

func encodeGIF(w io.Writer, img image.Image) error {
	pal := medianCut(img, 255)
	if hasAlpha(img) {
		pal = append(pal, color.RGBA{})
	}
	b := img.Bounds()
	p := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), pal)
	draw.FloydSteinberg.Draw(p, p.Bounds(), img, b.Min)
	return gif.Encode(w, p, &gif.Options{NumColors: len(pal)})
}

// Compress writes a smaller copy in the same format (HEIC/BMP/TIFF -> JPG/PNG).
func Compress(path string) (string, error) { return compressP(path, nil) }

func compressP(path string, prog func(float64)) (string, error) {
	k := Kind(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	img, err := decodeP(path, decodeStage(prog))
	if err != nil {
		return "", err
	}
	var bb bytes.Buffer
	buf := &bb
	if prog != nil {
		prog(0.45)
		sec := encodeSeconds(img, k, false)
		if k == "PNG" || k == "GIF" || hasAlpha(img) {
			sec += encodeSeconds(img, "PNG", false) * 3 // palette reduction comes first
		}
		stop := estimate(prog, 0.45, 0.99, sec)
		defer stop()
	}
	format := k
	switch k {
	case "JPG":
		err = Encode(buf, img, "JPG", 72)
	case "WEBP":
		err = Encode(buf, img, "WEBP", 68)
	case "AVIF":
		err = Encode(buf, img, "AVIF", 50)
	case "PNG", "GIF":
		err = encodeQuantizedPNG(buf, img)
		format = "PNG"
	default: // HEIC, BMP, TIFF
		if hasAlpha(img) {
			err = encodeQuantizedPNG(buf, img)
			format = "PNG"
		} else {
			err = Encode(buf, img, "JPG", 80)
			format = "JPG"
		}
	}
	if err != nil {
		return "", err
	}
	if format == k && bb.Len() >= len(data) {
		return "", errors.New(tr("The file is already compressed and can't get smaller", "الملف مضغوط بالفعل ولا يمكن تصغيره أكثر"))
	}
	dst := uniquePath(path, "-compressed", extFor(format))
	return dst, writeAtomic(dst, func(w io.Writer) error { _, err := w.Write(bb.Bytes()); return err })
}

func encodeQuantizedPNG(w io.Writer, img image.Image) error {
	pal := medianCut(img, 256)
	b := img.Bounds()
	p := image.NewPaletted(image.Rect(0, 0, b.Dx(), b.Dy()), pal)
	draw.FloydSteinberg.Draw(p, p.Bounds(), img, b.Min)
	e := png.Encoder{CompressionLevel: png.BestCompression}
	return e.Encode(w, p)
}

func transform(path, suffix string, f func(image.Image) image.Image, prog func(float64)) (string, error) {
	img, err := decodeP(path, decodeStage(prog))
	if err != nil {
		return "", err
	}
	k := Kind(path)
	if k == "HEIC" {
		k = "JPG"
	}
	out := f(img)
	dst := uniquePath(path, "-"+suffix, extFor(k))
	return dst, writeAtomic(dst, func(w io.Writer) error {
		return encodeP(w, out, k, losslessKind(Kind(path)), prog)
	})
}

func rotate90(img image.Image) image.Image {
	src := toNRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			copy(out.Pix[out.PixOffset(h-1-y, x):][:4], src.Pix[src.PixOffset(x, y):][:4])
		}
	}
	return out
}

func half(img image.Image) image.Image {
	b := img.Bounds()
	w, h := max(1, b.Dx()/2), max(1, b.Dy()/2)
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Src, nil)
	return out
}

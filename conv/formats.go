package conv

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"sort"

	"golang.org/x/image/bmp"
	xdraw "golang.org/x/image/draw"
)

// ---------- EXIF orientation (JPEG) ----------

func jpegOrientation(d []byte) int {
	if len(d) < 4 || d[0] != 0xFF || d[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(d) {
		if d[i] != 0xFF {
			return 1
		}
		m := d[i+1]
		if m == 0xDA || m == 0xD9 {
			return 1
		}
		l := int(binary.BigEndian.Uint16(d[i+2:]))
		if m == 0xE1 && i+4+l-2 <= len(d) && l > 8 && string(d[i+4:i+10]) == "Exif\x00\x00" {
			return tiffOrientation(d[i+10 : i+2+l])
		}
		i += 2 + l
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	off := int(bo.Uint32(t[4:]))
	if off+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[off:]))
	for k := 0; k < n; k++ {
		e := off + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			v := int(bo.Uint16(t[e+8:]))
			if v >= 1 && v <= 8 {
				return v
			}
		}
	}
	return 1
}

func applyOrientation(img image.Image, o int) image.Image {
	src := toNRGBA(img)
	w, h := src.Rect.Dx(), src.Rect.Dy()
	ow, oh := w, h
	if o >= 5 {
		ow, oh = h, w
	}
	out := image.NewNRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			default:
				dx, dy = x, y
			}
			copy(out.Pix[out.PixOffset(dx, dy):][:4], src.Pix[src.PixOffset(x, y):][:4])
		}
	}
	return out
}

// ---------- ICO ----------

func encodeICO(w io.Writer, img image.Image) error {
	b := img.Bounds()
	side := max(b.Dx(), b.Dy())
	square := image.NewNRGBA(image.Rect(0, 0, side, side))
	draw.Draw(square, image.Rect((side-b.Dx())/2, (side-b.Dy())/2, (side-b.Dx())/2+b.Dx(), (side-b.Dy())/2+b.Dy()), img, b.Min, draw.Src)
	var entries [][]byte
	var sizes []int
	for _, s := range []int{16, 24, 32, 48, 64, 128, 256} {
		if s > side && s > 16 {
			break
		}
		dst := image.NewNRGBA(image.Rect(0, 0, s, s))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), square, square.Bounds(), draw.Src, nil)
		var buf bytes.Buffer
		if err := png.Encode(&buf, dst); err != nil {
			return err
		}
		entries = append(entries, buf.Bytes())
		sizes = append(sizes, s)
	}
	hdr := []byte{0, 0, 1, 0, byte(len(entries)), 0}
	off := 6 + 16*len(entries)
	var dir bytes.Buffer
	for i, e := range entries {
		s := byte(sizes[i])
		if sizes[i] >= 256 {
			s = 0
		}
		dir.Write([]byte{s, s, 0, 0, 1, 0, 32, 0})
		binary.Write(&dir, binary.LittleEndian, uint32(len(e)))
		binary.Write(&dir, binary.LittleEndian, uint32(off))
		off += len(e)
	}
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	if _, err := w.Write(dir.Bytes()); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := w.Write(e); err != nil {
			return err
		}
	}
	return nil
}

// decodeICO returns the largest image stored in an .ico file.
func decodeICO(d []byte) (image.Image, error) {
	if len(d) < 6 || binary.LittleEndian.Uint16(d[2:]) != 1 {
		return nil, errors.New("not an ICO file")
	}
	n := int(binary.LittleEndian.Uint16(d[4:]))
	best, bestSize := -1, -1
	for i := 0; i < n; i++ {
		e := 6 + i*16
		if e+16 > len(d) {
			break
		}
		s := int(d[e])
		if s == 0 {
			s = 256
		}
		if s > bestSize {
			best, bestSize = i, s
		}
	}
	if best < 0 {
		return nil, errors.New("empty ICO file")
	}
	e := 6 + best*16
	size := int(binary.LittleEndian.Uint32(d[e+8:]))
	off := int(binary.LittleEndian.Uint32(d[e+12:]))
	if off+size > len(d) || size < 8 {
		return nil, errors.New("corrupt ICO file")
	}
	data := d[off : off+size]
	if bytes.HasPrefix(data, []byte("\x89PNG")) {
		return png.Decode(bytes.NewReader(data))
	}
	return decodeICOBitmap(data, bestSize)
}

// decodeICOBitmap handles 32-bit DIB entries (the common case for modern icons).
func decodeICOBitmap(data []byte, side int) (image.Image, error) {
	if len(data) < 40 {
		return nil, errors.New("corrupt ICO bitmap")
	}
	w := int(int32(binary.LittleEndian.Uint32(data[4:])))
	h := int(int32(binary.LittleEndian.Uint32(data[8:]))) / 2
	bpp := int(binary.LittleEndian.Uint16(data[14:]))
	hs := int(binary.LittleEndian.Uint32(data[0:]))
	if bpp != 32 || w <= 0 || h <= 0 {
		// Let the BMP decoder try by synthesizing a file header.
		var buf bytes.Buffer
		buf.WriteString("BM")
		binary.Write(&buf, binary.LittleEndian, uint32(14+len(data)))
		binary.Write(&buf, binary.LittleEndian, uint32(0))
		binary.Write(&buf, binary.LittleEndian, uint32(14+hs))
		fixed := append([]byte(nil), data...)
		binary.LittleEndian.PutUint32(fixed[8:], uint32(h))
		buf.Write(fixed)
		return bmp.Decode(&buf)
	}
	px := data[hs:]
	if len(px) < w*h*4 {
		return nil, errors.New("corrupt ICO bitmap")
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := px[(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			p := row[x*4:]
			out.SetNRGBA(x, y, color.NRGBA{p[2], p[1], p[0], p[3]})
		}
	}
	return out, nil
}

// ---------- PDF ----------

func encodePDF(w io.Writer, img image.Image) error { return encodePDFP(w, img, nil) }

// encodePDFP is encodePDF reporting how much of the picture is compressed.
func encodePDFP(w io.Writer, img image.Image, prog func(float64)) error {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	var objs [][]byte
	add := func(o []byte) int { objs = append(objs, o); return len(objs) }

	var imgObj, smaskObj int
	if hasAlpha(img) {
		n := toNRGBA(img)
		rgb := make([]byte, 0, iw*ih*3)
		alpha := make([]byte, 0, iw*ih)
		for i := 0; i < len(n.Pix); i += 4 {
			rgb = append(rgb, n.Pix[i], n.Pix[i+1], n.Pix[i+2])
			alpha = append(alpha, n.Pix[i+3])
		}
		smaskObj = add(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode", iw, ih), deflateP(alpha, part(prog, 0, 0.25))))
		imgObj = add(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /SMask %d 0 R", iw, ih, smaskObj), deflateP(rgb, part(prog, 0.25, 1))))
	} else {
		// Lossless: raw RGB with Flate compression.
		n := toNRGBA(img)
		rgb := make([]byte, 0, iw*ih*3)
		for i := 0; i < len(n.Pix); i += 4 {
			rgb = append(rgb, n.Pix[i], n.Pix[i+1], n.Pix[i+2])
		}
		imgObj = add(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode", iw, ih), deflateP(rgb, prog)))
	}
	return writePDF(w, &objs, add, imgObj, iw, ih)
}

// pdfFromJPEG wraps original JPEG bytes in a one-page PDF without
// re-encoding. Only plain RGB or grayscale JPEGs are handled.
func pdfFromJPEG(data []byte) ([]byte, bool) {
	w, h, comps, ok := jpegInfo(data)
	if !ok || (comps != 1 && comps != 3) {
		return nil, false
	}
	cs := "/DeviceRGB"
	if comps == 1 {
		cs = "/DeviceGray"
	}
	var objs [][]byte
	add := func(o []byte) int { objs = append(objs, o); return len(objs) }
	imgObj := add(stream(fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace %s /BitsPerComponent 8 /Filter /DCTDecode", w, h, cs), data))
	var b bytes.Buffer
	if err := writePDF(&b, &objs, add, imgObj, w, h); err != nil {
		return nil, false
	}
	return b.Bytes(), true
}

// jpegInfo reads width, height and component count from the SOF marker.
func jpegInfo(d []byte) (w, h, comps int, ok bool) {
	if len(d) < 4 || d[0] != 0xFF || d[1] != 0xD8 {
		return
	}
	i := 2
	for i+9 < len(d) {
		if d[i] != 0xFF {
			return
		}
		m := d[i+1]
		l := int(binary.BigEndian.Uint16(d[i+2:]))
		if m >= 0xC0 && m <= 0xCF && m != 0xC4 && m != 0xC8 && m != 0xCC {
			h = int(binary.BigEndian.Uint16(d[i+5:]))
			w = int(binary.BigEndian.Uint16(d[i+7:]))
			comps = int(d[i+9])
			return w, h, comps, w > 0 && h > 0
		}
		i += 2 + l
	}
	return
}

func writePDF(w io.Writer, op *[][]byte, add func([]byte) int, imgObj, iw, ih int) error {
	pw, ph := float64(iw)*0.75, float64(ih)*0.75
	content := []byte(fmt.Sprintf("q %.2f 0 0 %.2f 0 0 cm /Im0 Do Q", pw, ph))
	contentObj := add(stream("", content))
	pagesObj := len(*op) + 2
	pageObj := add([]byte(fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /XObject << /Im0 %d 0 R >> >> /Contents %d 0 R >>", pagesObj, pw, ph, imgObj, contentObj)))
	add([]byte(fmt.Sprintf("<< /Type /Pages /Kids [%d 0 R] /Count 1 >>", pageObj)))
	catalog := add([]byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesObj)))
	objs := *op

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(o)
		out.WriteString("\nendobj\n")
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, catalog, xref)
	_, err := w.Write(out.Bytes())
	return err
}

func stream(dict string, data []byte) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "<< %s /Length %d >>\nstream\n", dict, len(data))
	b.Write(data)
	b.WriteString("\nendstream")
	return b.Bytes()
}

func deflate(d []byte) []byte { return deflateP(d, nil) }

// deflateP compresses d in slices, reporting the share done.
func deflateP(d []byte, prog func(float64)) []byte {
	var b bytes.Buffer
	z, _ := zlib.NewWriterLevel(&b, zlib.BestCompression)
	const chunk = 1 << 20
	for i := 0; i < len(d); i += chunk {
		z.Write(d[i:min(i+chunk, len(d))])
		if prog != nil {
			prog(float64(min(i+chunk, len(d))) / float64(len(d)))
		}
	}
	z.Close()
	return b.Bytes()
}

// part maps 0..1 onto lo..hi of prog.
func part(prog func(float64), lo, hi float64) func(float64) {
	if prog == nil {
		return nil
	}
	return func(v float64) { prog(lo + (hi-lo)*v) }
}

// ---------- Median-cut palette ----------

type box struct{ px [][4]uint8 }

func (b *box) rangeOf() (int, int) {
	lo, hi := [4]int{255, 255, 255, 255}, [4]int{}
	for _, p := range b.px {
		for c := 0; c < 4; c++ {
			v := int(p[c])
			lo[c] = min(lo[c], v)
			hi[c] = max(hi[c], v)
		}
	}
	best, span := 0, -1
	for c := 0; c < 4; c++ {
		if hi[c]-lo[c] > span {
			best, span = c, hi[c]-lo[c]
		}
	}
	return best, span
}

func medianCut(img image.Image, n int) color.Palette {
	src := toNRGBA(img)
	// Sample at most ~250k pixels for speed.
	step := max(1, len(src.Pix)/4/250000)
	var px [][4]uint8
	for i := 0; i+3 < len(src.Pix); i += 4 * step {
		a := src.Pix[i+3]
		if a < 8 {
			px = append(px, [4]uint8{0, 0, 0, 0})
			continue
		}
		px = append(px, [4]uint8{src.Pix[i], src.Pix[i+1], src.Pix[i+2], a})
	}
	boxes := []*box{{px}}
	for len(boxes) < n {
		bi, bspan, bc := -1, 0, 0
		for i, b := range boxes {
			if len(b.px) < 2 {
				continue
			}
			c, s := b.rangeOf()
			score := s * len(b.px)
			if score > bspan {
				bi, bspan, bc = i, score, c
			}
		}
		if bi < 0 || bspan == 0 {
			break
		}
		b := boxes[bi]
		sort.Slice(b.px, func(i, j int) bool { return b.px[i][bc] < b.px[j][bc] })
		mid := len(b.px) / 2
		boxes[bi] = &box{b.px[:mid]}
		boxes = append(boxes, &box{b.px[mid:]})
	}
	pal := make(color.Palette, 0, len(boxes))
	for _, b := range boxes {
		var s [4]int
		for _, p := range b.px {
			for c := 0; c < 4; c++ {
				s[c] += int(p[c])
			}
		}
		l := max(1, len(b.px))
		a := s[3] / l
		// palette colors are premultiplied RGBA
		pal = append(pal, color.NRGBA{uint8(s[0] / l), uint8(s[1] / l), uint8(s[2] / l), uint8(a)})
	}
	return pal
}

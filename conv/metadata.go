package conv

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// StripMetadata removes EXIF/XMP/IPTC/comments. JPEG, PNG and WebP are
// processed losslessly (pixels untouched); other formats are re-encoded.
func StripMetadata(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	k := Kind(path)
	var out []byte
	switch k {
	case "JPG":
		out, err = stripJPEG(data)
	case "PNG":
		out, err = stripPNG(data)
	case "WEBP":
		out, err = stripWEBP(data)
	case "HEIC", "ICO":
		return "", errors.New(tr("Removing metadata isn't available for this format yet", "حذف البيانات غير متاح لهذه الصيغة بعد"))
	default:
		img, derr := Decode(path)
		if derr != nil {
			return "", derr
		}
		var b bytes.Buffer
		err = Encode(&b, img, k, 0)
		out = b.Bytes()
	}
	if err != nil {
		return "", err
	}
	dst := uniquePath(path, "-clean", extFor(k))
	return writeAtomic(dst, func(w io.Writer) error { _, err := w.Write(out); return err })
}

func stripJPEG(d []byte) ([]byte, error) {
	if len(d) < 4 || d[0] != 0xFF || d[1] != 0xD8 {
		return nil, errors.New(tr("Invalid JPEG file", "ملف JPEG غير صالح"))
	}
	orient := jpegOrientation(d)
	wroteOrient := false
	var out bytes.Buffer
	out.Write(d[:2])
	i := 2
	for i+4 <= len(d) {
		if d[i] != 0xFF {
			return nil, errors.New(tr("Invalid JPEG file", "ملف JPEG غير صالح"))
		}
		m := d[i+1]
		if m == 0xDA { // start of scan: copy the rest verbatim
			out.Write(d[i:])
			break
		}
		l := int(binary.BigEndian.Uint16(d[i+2:]))
		if i+2+l > len(d) {
			return nil, errors.New(tr("Damaged JPEG file", "ملف JPEG تالف"))
		}
		seg := d[i : i+2+l]
		drop := m == 0xFE || m == 0xED || (m >= 0xE3 && m <= 0xEF && m != 0xEE)
		if m == 0xE1 {
			drop = true
			if orient > 1 && !wroteOrient { // keep only the orientation so the photo isn't shown sideways
				out.Write(minimalExif(orient))
				wroteOrient = true
			}
		}
		if !drop {
			out.Write(seg)
		}
		i += 2 + l
	}
	return out.Bytes(), nil
}

// minimalExif builds an APP1 segment holding only the Orientation tag.
func minimalExif(o int) []byte {
	var t bytes.Buffer
	t.WriteString("MM\x00\x2a")
	binary.Write(&t, binary.BigEndian, uint32(8))
	binary.Write(&t, binary.BigEndian, uint16(1))
	binary.Write(&t, binary.BigEndian, uint16(0x0112))
	binary.Write(&t, binary.BigEndian, uint16(3))
	binary.Write(&t, binary.BigEndian, uint32(1))
	binary.Write(&t, binary.BigEndian, uint16(o))
	binary.Write(&t, binary.BigEndian, uint16(0))
	binary.Write(&t, binary.BigEndian, uint32(0))
	payload := append([]byte("Exif\x00\x00"), t.Bytes()...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func stripPNG(d []byte) ([]byte, error) {
	sig := []byte("\x89PNG\r\n\x1a\n")
	if !bytes.HasPrefix(d, sig) {
		return nil, errors.New(tr("Invalid PNG file", "ملف PNG غير صالح"))
	}
	var out bytes.Buffer
	out.Write(sig)
	i := 8
	for i+12 <= len(d) {
		l := int(binary.BigEndian.Uint32(d[i:]))
		if i+12+l > len(d) {
			return nil, errors.New(tr("Damaged PNG file", "ملف PNG تالف"))
		}
		typ := string(d[i+4 : i+8])
		switch typ {
		case "tEXt", "zTXt", "iTXt", "eXIf", "tIME":
		default:
			out.Write(d[i : i+12+l])
		}
		i += 12 + l
		if typ == "IEND" {
			break
		}
	}
	return out.Bytes(), nil
}

func stripWEBP(d []byte) ([]byte, error) {
	if len(d) < 12 || string(d[:4]) != "RIFF" || string(d[8:12]) != "WEBP" {
		return nil, errors.New(tr("Invalid WebP file", "ملف WebP غير صالح"))
	}
	var body bytes.Buffer
	body.WriteString("WEBP")
	i := 12
	vp8x := -1
	keptExif := false
	for i+8 <= len(d) {
		typ := string(d[i : i+4])
		l := int(binary.LittleEndian.Uint32(d[i+4:]))
		end := i + 8 + l + l%2
		if i+8+l > len(d) {
			return nil, errors.New(tr("Damaged WebP file", "ملف WebP تالف"))
		}
		if end > len(d) {
			end = len(d)
		}
		if typ == "EXIF" {
			raw := d[i+8 : i+8+l]
			if bytes.HasPrefix(raw, []byte("Exif\x00\x00")) {
				raw = raw[6:]
			}
			if o := tiffOrientation(raw); o > 1 {
				keptExif = true
				t := minimalExif(o)[10:] // raw TIFF block without APP1 header and "Exif\0\0"
				body.WriteString("EXIF")
				binary.Write(&body, binary.LittleEndian, uint32(len(t)))
				body.Write(t)
				if len(t)%2 == 1 {
					body.WriteByte(0)
				}
			}
		} else if typ != "XMP " {
			if typ == "VP8X" {
				vp8x = body.Len()
			}
			body.Write(d[i:end])
		}
		i = end
	}
	b := body.Bytes()
	if vp8x >= 0 && vp8x+8 < len(b) {
		b[vp8x+8] &^= 0x04 // clear XMP flag
		if !keptExif {
			b[vp8x+8] &^= 0x08 // clear EXIF flag
		}
	}
	out := make([]byte, 8, 8+len(b))
	copy(out, "RIFF")
	binary.LittleEndian.PutUint32(out[4:], uint32(len(b)))
	return append(out, b...), nil
}

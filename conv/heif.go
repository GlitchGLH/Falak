package conv

// HEIC output: FFmpeg's libx265 encodes the picture (and, for transparent
// images, an alpha plane) as HEVC, and the HEIF container (ISO/IEC 23008-12)
// is written here, because FFmpeg has no HEIF muxer.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"io"
	"os"
	"os/exec"
)

// heicCRF is the x265 quality: 14 is visually transparent for photos while
// keeping HEIC's size advantage.
const heicCRF = "14"

// EncodeHEIC writes img as a HEIC file.
func EncodeHEIC(w io.Writer, img image.Image) error {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw < 1 || ih < 1 {
		return errors.New(tr("Empty image", "صورة فارغة"))
	}
	// 4:2:0 needs even sizes: pad by repeating the edge, then crop it away
	// again with a 'clap' box.
	pw, ph := iw+iw%2, ih+ih%2
	src := image.NewNRGBA(image.Rect(0, 0, pw, ph))
	draw.Draw(src, image.Rect(0, 0, iw, ih), img, b.Min, draw.Src)
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			if x < iw && y < ih {
				continue
			}
			sx, sy := min(x, iw-1), min(y, ih-1)
			copy(src.Pix[src.PixOffset(x, y):][:4], src.Pix[src.PixOffset(sx, sy):][:4])
		}
	}
	hasAlpha := false
	for i := 3; i < len(src.Pix); i += 4 {
		if src.Pix[i] != 255 {
			hasAlpha = true
			break
		}
	}

	color, err := x265Still(src.Pix, "rgba", pw, ph)
	if err != nil {
		return err
	}
	var alpha []byte
	if hasAlpha {
		a := make([]byte, pw*ph)
		for i := range a {
			a[i] = src.Pix[i*4+3]
		}
		if alpha, err = x265Still(a, "gray", pw, ph); err != nil {
			return err
		}
	}
	out, err := heifContainer(color, alpha, pw, ph, iw, ih)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// x265Still encodes one frame of raw pixels and returns the Annex B stream.
func x265Still(pix []byte, pixFmt string, w, h int) ([]byte, error) {
	args := []string{"-hide_banner", "-loglevel", "error",
		"-f", "rawvideo", "-pix_fmt", pixFmt, "-s", fmt.Sprintf("%dx%d", w, h), "-i", "pipe:0",
		"-frames:v", "1", "-sws_flags", "accurate_rnd+full_chroma_int+bitexact",
		"-vf", "scale=out_color_matrix=bt601:out_range=full,format=yuv420p",
		"-c:v", "libx265", "-preset", "slow", "-crf", heicCRF,
		"-x265-params", "info=0:colorprim=bt709:transfer=iec61966-2-1:colormatrix=smpte170m:range=full:log-level=error",
		"-f", "hevc", "pipe:1"}
	cmd := exec.Command(FFmpegPath(), args...)
	PrepareCmd(cmd)
	cmd.Stdin = bytes.NewReader(pix)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf(tr("HEIC encoding failed: %s", "فشل ترميز HEIC: %s"), ffmpegError(err, args, stderr.String()))
	}
	return out.Bytes(), nil
}

// ---------- HEVC bitstream helpers ----------

func splitAnnexB(b []byte) [][]byte {
	var nals [][]byte
	start := -1
	for i := 0; i+2 < len(b); {
		if b[i] == 0 && b[i+1] == 0 && b[i+2] == 1 {
			if start >= 0 {
				end := i
				for end > start && b[end-1] == 0 {
					end--
				}
				nals = append(nals, b[start:end])
			}
			i += 3
			start = i
			continue
		}
		i++
	}
	if start >= 0 && start < len(b) {
		nals = append(nals, b[start:])
	}
	return nals
}

func nalType(n []byte) byte { return (n[0] >> 1) & 0x3f }

func unescapeRBSP(n []byte) []byte {
	out := make([]byte, 0, len(n))
	zeros := 0
	for _, c := range n {
		if zeros >= 2 && c == 3 {
			zeros = 0
			continue
		}
		if c == 0 {
			zeros++
		} else {
			zeros = 0
		}
		out = append(out, c)
	}
	return out
}

type bitReader struct {
	b   []byte
	pos int
}

func (r *bitReader) bit() uint32 {
	if r.pos/8 >= len(r.b) {
		return 0
	}
	v := (r.b[r.pos/8] >> (7 - r.pos%8)) & 1
	r.pos++
	return uint32(v)
}

func (r *bitReader) bits(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | r.bit()
	}
	return v
}

func (r *bitReader) ue() uint32 {
	z := 0
	for r.bit() == 0 && z < 32 {
		z++
	}
	return (1<<z - 1) + r.bits(z)
}

// hvcC builds the HEVC decoder configuration record from VPS/SPS/PPS.
func hvcC(vps, sps, pps []byte) ([]byte, error) {
	rb := unescapeRBSP(sps)
	if len(rb) < 15 {
		return nil, errors.New("short SPS")
	}
	ptl := rb[3:15] // general profile/tier/level: 12 bytes after the header byte
	r := &bitReader{b: rb, pos: 16 + 8 + 96}
	maxSub := int((rb[2] >> 1) & 7)
	chroma := uint32(1)
	if maxSub == 0 {
		r.ue() // sps_seq_parameter_set_id
		chroma = r.ue()
	}
	var b bytes.Buffer
	b.WriteByte(1)
	b.Write(ptl[:1])
	b.Write(ptl[1:5])    // compatibility flags
	b.Write(ptl[5:11])   // constraint flags
	b.WriteByte(ptl[11]) // level
	b.Write([]byte{0xF0, 0x00, 0xFC, 0xFC | byte(chroma&3), 0xF8, 0xF8, 0, 0, 0x0F})
	b.WriteByte(3)
	for _, n := range [][]byte{vps, sps, pps} {
		b.WriteByte(0x80 | nalType(n))
		binary.Write(&b, binary.BigEndian, uint16(1))
		binary.Write(&b, binary.BigEndian, uint16(len(n)))
		b.Write(n)
	}
	return b.Bytes(), nil
}

type hevcImage struct {
	config []byte
	data   []byte // length-prefixed VCL NAL units
}

func parseHEVC(stream []byte) (hevcImage, error) {
	var vps, sps, pps []byte
	var data bytes.Buffer
	for _, n := range splitAnnexB(stream) {
		if len(n) < 2 {
			continue
		}
		switch t := nalType(n); {
		case t == 32:
			vps = n
		case t == 33:
			sps = n
		case t == 34:
			pps = n
		case t < 32: // picture data; SEI and others are left out
			binary.Write(&data, binary.BigEndian, uint32(len(n)))
			data.Write(n)
		}
	}
	if vps == nil || sps == nil || pps == nil || data.Len() == 0 {
		return hevcImage{}, errors.New(tr("HEIC encoding produced no picture", "لم ينتج ترميز HEIC صورة"))
	}
	cfg, err := hvcC(vps, sps, pps)
	return hevcImage{cfg, data.Bytes()}, err
}

// ---------- HEIF boxes ----------

func isoBox(typ string, parts ...[]byte) []byte {
	n := 8
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 8, n)
	binary.BigEndian.PutUint32(out, uint32(n))
	copy(out[4:], typ)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func fullBox(typ string, version byte, flags uint32, parts ...[]byte) []byte {
	h := []byte{version, byte(flags >> 16), byte(flags >> 8), byte(flags)}
	return isoBox(typ, append([][]byte{h}, parts...)...)
}

func u16(v int) []byte { return binary.BigEndian.AppendUint16(nil, uint16(v)) }
func u32(v int) []byte { return binary.BigEndian.AppendUint32(nil, uint32(v)) }

func heifContainer(colorStream, alphaStream []byte, pw, ph, iw, ih int) ([]byte, error) {
	main, err := parseHEVC(colorStream)
	if err != nil {
		return nil, err
	}
	items := []hevcImage{main}
	if alphaStream != nil {
		al, err := parseHEVC(alphaStream)
		if err != nil {
			return nil, err
		}
		items = append(items, al)
	}
	crop := pw != iw || ph != ih

	ftyp := isoBox("ftyp", []byte("heic"), u32(0), []byte("mif1heicmiaf"))
	hdlr := fullBox("hdlr", 0, 0, u32(0), []byte("pict"), make([]byte, 12), []byte{0})
	pitm := fullBox("pitm", 0, 0, u16(1))

	var infes [][]byte
	for i := range items {
		infes = append(infes, fullBox("infe", 2, 0, u16(i+1), u16(0), []byte("hvc1"), []byte{0}))
	}
	iinf := fullBox("iinf", 0, 0, append([][]byte{u16(len(items))}, infes...)...)

	// properties: 1 hvcC main, 2 ispe, 3 colr, 4 pixi, [5 clap], then alpha hvcC, auxC
	props := [][]byte{
		isoBox("hvcC", main.config),
		fullBox("ispe", 0, 0, u32(pw), u32(ph)),
		isoBox("colr", []byte("nclx"), u16(1), u16(13), u16(6), []byte{0x80}),
		fullBox("pixi", 0, 0, []byte{3, 8, 8, 8}),
	}
	clapIdx := 0
	if crop {
		props = append(props, isoBox("clap",
			u32(iw), u32(1), u32(ih), u32(1),
			u32(iw-pw), u32(2), u32(ih-ph), u32(2)))
		clapIdx = len(props)
	}
	type assoc struct {
		idx       int
		essential bool
	}
	ipmaFor := [][]assoc{{{1, true}, {2, false}, {3, false}, {4, false}}}
	if crop {
		ipmaFor[0] = append(ipmaFor[0], assoc{clapIdx, true})
	}
	if len(items) > 1 {
		props = append(props, isoBox("hvcC", items[1].config))
		alphaCfg := len(props)
		props = append(props, fullBox("auxC", 0, 0, []byte("urn:mpeg:hevc:2015:auxid:1\x00")))
		auxIdx := len(props)
		props = append(props, fullBox("pixi", 0, 0, []byte{1, 8}))
		pixiA := len(props)
		a := []assoc{{alphaCfg, true}, {2, false}, {auxIdx, true}, {pixiA, false}}
		if crop {
			a = append(a, assoc{clapIdx, true})
		}
		ipmaFor = append(ipmaFor, a)
	}
	ipco := isoBox("ipco", props...)
	var ipmaBody [][]byte
	ipmaBody = append(ipmaBody, u32(len(ipmaFor)))
	for i, list := range ipmaFor {
		e := append(u16(i+1), byte(len(list)))
		for _, a := range list {
			v := byte(a.idx & 0x7f)
			if a.essential {
				v |= 0x80
			}
			e = append(e, v)
		}
		ipmaBody = append(ipmaBody, e)
	}
	ipma := fullBox("ipma", 0, 0, ipmaBody...)
	iprp := isoBox("iprp", ipco, ipma)

	var iref []byte
	if len(items) > 1 {
		iref = fullBox("iref", 0, 0, isoBox("auxl", u16(2), u16(1), u16(1)))
	}

	// iloc needs absolute offsets into mdat, so lay it out twice.
	makeIloc := func(base int) []byte {
		body := [][]byte{{0x44, 0x00}, u16(len(items))}
		off := base
		for i, it := range items {
			body = append(body, u16(i+1), u16(0), u16(1), u32(off), u32(len(it.data)))
			off += len(it.data)
		}
		return fullBox("iloc", 0, 0, body...)
	}
	metaParts := func(iloc []byte) []byte {
		parts := [][]byte{hdlr, pitm, iloc, iinf}
		if iref != nil {
			parts = append(parts, iref)
		}
		parts = append(parts, iprp)
		return fullBox("meta", 0, 0, parts...)
	}
	meta := metaParts(makeIloc(0))
	dataStart := len(ftyp) + len(meta) + 8
	meta = metaParts(makeIloc(dataStart))

	var mdatBody []byte
	for _, it := range items {
		mdatBody = append(mdatBody, it.data...)
	}
	var out bytes.Buffer
	out.Write(ftyp)
	out.Write(meta)
	out.Write(isoBox("mdat", mdatBody))
	return out.Bytes(), nil
}

// gifIsAnimated reports whether a GIF has more than one frame. It walks the
// block structure without decoding pixels.
func gifIsAnimated(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 13 || string(b[:3]) != "GIF" {
		return false
	}
	p := 13
	if b[10]&0x80 != 0 {
		p += 3 << (int(b[10]&7) + 1)
	}
	frames := 0
	skipSub := func() bool {
		for p < len(b) {
			n := int(b[p])
			p++
			if n == 0 {
				return true
			}
			p += n
		}
		return false
	}
	for p < len(b) {
		switch b[p] {
		case 0x21: // extension
			p += 2
			if !skipSub() {
				return frames > 1
			}
		case 0x2C: // image
			frames++
			if frames > 1 {
				return true
			}
			if p+10 > len(b) {
				return false
			}
			flags := b[p+9]
			p += 10
			if flags&0x80 != 0 {
				p += 3 << (int(flags&7) + 1)
			}
			p++ // LZW minimum code size
			if !skipSub() {
				return false
			}
		default: // 0x3B trailer or junk
			return frames > 1
		}
	}
	return frames > 1
}

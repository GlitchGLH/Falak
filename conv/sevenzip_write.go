package conv

// A 7z writer: one solid LZMA2 stream (7-Zip's own default method), with
// names, modification times, attributes and CRCs, following the 7z format
// specification (7zFormat.txt).

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf16"

	"github.com/ulikunitz/xz/lzma"
)

const sevenZipDict = 16 << 20 // 16 MiB dictionary: strong yet light on memory

type szFile struct {
	name  string
	dir   bool
	mtime time.Time
	size  uint64
	crc   uint32
}

type szWriter struct {
	f       *os.File
	lz      *lzma.Writer2
	counter *countWriter
	files   []szFile
	total   uint64
}

type countWriter struct {
	w io.Writer
	n uint64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += uint64(n)
	return n, err
}

func newSZWriter(f *os.File) (*szWriter, error) {
	if _, err := f.Write(make([]byte, 32)); err != nil { // signature header, filled in at the end
		return nil, err
	}
	cw := &countWriter{w: f}
	cfg := lzma.Writer2Config{DictCap: sevenZipDict}
	lz, err := cfg.NewWriter2(cw)
	if err != nil {
		return nil, err
	}
	return &szWriter{f: f, lz: lz, counter: cw}, nil
}

func (z *szWriter) addDir(name string, mt time.Time) {
	z.files = append(z.files, szFile{name: name, dir: true, mtime: mt})
}

func (z *szWriter) addFile(name string, mt time.Time, r io.Reader) error {
	h := crc32.NewIEEE()
	n, err := io.Copy(io.MultiWriter(z.lz, h), r)
	if err != nil {
		return err
	}
	z.files = append(z.files, szFile{name: name, mtime: mt, size: uint64(n), crc: h.Sum32()})
	z.total += uint64(n)
	return nil
}

func szNumber(b *bytes.Buffer, v uint64) {
	first := byte(0)
	mask := byte(0x80)
	i := 0
	for ; i < 8; i++ {
		if v < uint64(1)<<(7*(i+1)) {
			first |= byte(v >> (8 * i))
			break
		}
		first |= mask
		mask >>= 1
	}
	b.WriteByte(first)
	for j := 0; j < i; j++ {
		b.WriteByte(byte(v >> (8 * j)))
	}
}

func szBits(b *bytes.Buffer, bits []bool) {
	var cur byte
	for i, v := range bits {
		if v {
			cur |= 0x80 >> (i % 8)
		}
		if i%8 == 7 {
			b.WriteByte(cur)
			cur = 0
		}
	}
	if len(bits)%8 != 0 {
		b.WriteByte(cur)
	}
}

func filetime(t time.Time) uint64 {
	if t.IsZero() {
		t = time.Now()
	}
	return uint64(t.UnixNano()/100) + 116444736000000000
}

func (z *szWriter) close() error {
	if err := z.lz.Close(); err != nil {
		return err
	}
	packSize := z.counter.n
	var h bytes.Buffer
	h.WriteByte(0x01) // Header
	var streams []szFile
	for _, f := range z.files {
		if !f.dir && f.size > 0 {
			streams = append(streams, f)
		}
	}
	if len(streams) > 0 {
		h.WriteByte(0x04) // MainStreamsInfo
		h.WriteByte(0x06) // PackInfo
		szNumber(&h, 0)
		szNumber(&h, 1)
		h.WriteByte(0x09)
		szNumber(&h, packSize)
		h.WriteByte(0x00)
		h.WriteByte(0x07) // UnPackInfo
		h.WriteByte(0x0B)
		szNumber(&h, 1) // one folder
		h.WriteByte(0)  // not external
		szNumber(&h, 1) // one coder
		h.WriteByte(0x21)
		h.WriteByte(0x21) // LZMA2
		szNumber(&h, 1)
		h.WriteByte(lzma2DictProp(sevenZipDict))
		h.WriteByte(0x0C)
		szNumber(&h, z.total)
		h.WriteByte(0x00)
		h.WriteByte(0x08) // SubStreamsInfo
		h.WriteByte(0x0D)
		szNumber(&h, uint64(len(streams)))
		if len(streams) > 1 {
			h.WriteByte(0x09)
			for _, s := range streams[:len(streams)-1] {
				szNumber(&h, s.size)
			}
		}
		h.WriteByte(0x0A)
		h.WriteByte(1) // all defined
		for _, s := range streams {
			binary.Write(&h, binary.LittleEndian, s.crc)
		}
		h.WriteByte(0x00)
		h.WriteByte(0x00)
	}
	h.WriteByte(0x05) // FilesInfo
	szNumber(&h, uint64(len(z.files)))
	empty := make([]bool, len(z.files))
	var emptyFile []bool
	anyEmpty := false
	for i, f := range z.files {
		if f.dir || f.size == 0 {
			empty[i], anyEmpty = true, true
			emptyFile = append(emptyFile, !f.dir)
		}
	}
	prop := func(id byte, body []byte) {
		h.WriteByte(id)
		szNumber(&h, uint64(len(body)))
		h.Write(body)
	}
	if anyEmpty {
		var b bytes.Buffer
		szBits(&b, empty)
		prop(0x0E, b.Bytes())
		hasEmptyFile := false
		for _, v := range emptyFile {
			hasEmptyFile = hasEmptyFile || v
		}
		if hasEmptyFile {
			b.Reset()
			szBits(&b, emptyFile)
			prop(0x0F, b.Bytes())
		}
	}
	var names bytes.Buffer
	names.WriteByte(0) // not external
	for _, f := range z.files {
		for _, u := range utf16.Encode([]rune(f.name)) {
			binary.Write(&names, binary.LittleEndian, u)
		}
		names.Write([]byte{0, 0})
	}
	prop(0x11, names.Bytes())
	var times bytes.Buffer
	times.Write([]byte{1, 0}) // all defined, not external
	for _, f := range z.files {
		binary.Write(&times, binary.LittleEndian, filetime(f.mtime))
	}
	prop(0x14, times.Bytes())
	var attrs bytes.Buffer
	attrs.Write([]byte{1, 0})
	for _, f := range z.files {
		a := uint32(0x20) // archive
		if f.dir {
			a = 0x10 // directory
		}
		binary.Write(&attrs, binary.LittleEndian, a)
	}
	prop(0x15, attrs.Bytes())
	h.WriteByte(0x00) // end of FilesInfo
	h.WriteByte(0x00) // end of Header

	if _, err := z.f.Write(h.Bytes()); err != nil {
		return err
	}
	var start bytes.Buffer
	binary.Write(&start, binary.LittleEndian, packSize)        // NextHeaderOffset
	binary.Write(&start, binary.LittleEndian, uint64(h.Len())) // NextHeaderSize
	binary.Write(&start, binary.LittleEndian, crc32.ChecksumIEEE(h.Bytes()))
	sig := []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C, 0, 4}
	sig = binary.LittleEndian.AppendUint32(sig, crc32.ChecksumIEEE(start.Bytes()))
	sig = append(sig, start.Bytes()...)
	if _, err := z.f.WriteAt(sig, 0); err != nil {
		return err
	}
	return nil
}

// lzma2DictProp encodes a dictionary size as the LZMA2 property byte.
func lzma2DictProp(d uint32) byte {
	for p := byte(0); p < 40; p++ {
		size := uint32(2|(p&1)) << (p/2 + 11)
		if size >= d {
			return p
		}
	}
	return 40
}

// to7Z repacks any readable archive as 7z.
func to7Z(ctx context.Context, path string, progress func(float64)) (string, error) {
	dir := filepath.Dir(path)
	dst := filepath.Join(dir, uniqueName(dir, archiveBase(path)+".7z"))
	tmp := tempFor(dst)
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	z, err := newSZWriter(f)
	if err != nil {
		f.Close()
		dropTemp(tmp)
		return "", err
	}
	err = readArchive(path, progress, func(e entry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := e.name
		for len(name) > 0 && name[len(name)-1] == '/' {
			name = name[:len(name)-1]
		}
		if name == "" {
			return nil
		}
		if e.dir {
			z.addDir(name, e.modTime)
			return nil
		}
		rc, err := e.open()
		if err != nil {
			return err
		}
		defer rc.Close()
		return z.addFile(name, e.modTime, rc)
	})
	if err == nil {
		err = z.close()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		dropTemp(tmp)
		return "", err
	}
	return dst, finishTemp(tmp, dst)
}

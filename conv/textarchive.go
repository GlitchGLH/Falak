package conv

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ---------- subtitles ----------

func subtitleItems(path string) []Item {
	own := normalizeOwn(path)
	return formatItems([]string{"SRT", "VTT", "ASS", "TXT"}, own)
}

var reTime = regexp.MustCompile(`^\s*(\d{1,2}:)?\d{2}:\d{2}[.,]\d{3}\s*-->\s*(\d{1,2}:)?\d{2}:\d{2}[.,]\d{3}`)

type cue struct {
	timing string
	lines  []string
}

func parseSubs(data []byte) []cue {
	text := strings.ReplaceAll(decodeText(data), "\r\n", "\n")
	if isASS(text) {
		return parseASS(text)
	}
	var cues []cue
	var cur *cue
	for _, line := range strings.Split(text, "\n") {
		switch {
		case reTime.MatchString(line):
			cues = append(cues, cue{timing: strings.TrimSpace(line)})
			cur = &cues[len(cues)-1]
		case strings.TrimSpace(line) == "":
			cur = nil
		case cur != nil:
			cur.lines = append(cur.lines, line)
		}
	}
	return cues
}

func normTiming(t string, comma bool) string {
	parts := strings.SplitN(t, "-->", 2)
	fix := func(s string) string {
		s = strings.TrimSpace(strings.Fields(s + " ")[0])
		if strings.Count(s, ":") == 1 {
			s = "00:" + s
		}
		if comma {
			return strings.Replace(s, ".", ",", 1)
		}
		return strings.Replace(s, ",", ".", 1)
	}
	return fix(parts[0]) + " --> " + fix(parts[1])
}

var reTags = regexp.MustCompile(`<[^>]+>`)

func runSubtitle(path, id string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	cues := parseSubs(data)
	if len(cues) == 0 {
		return "", errors.New(tr("No subtitles found in the file", "لم يُعثر على ترجمات داخل الملف"))
	}
	t := strings.TrimPrefix(id, "to:")
	var b strings.Builder
	switch t {
	case "VTT":
		b.WriteString("WEBVTT\n\n")
		for _, c := range cues {
			fmt.Fprintf(&b, "%s\n%s\n\n", normTiming(c.timing, false), strings.Join(c.lines, "\n"))
		}
	case "SRT":
		for i, c := range cues {
			fmt.Fprintf(&b, "%d\n%s\n%s\n\n", i+1, normTiming(c.timing, true), strings.Join(c.lines, "\n"))
		}
	case "ASS":
		writeASS(&b, cues)
	case "TXT":
		for _, c := range cues {
			for _, l := range c.lines {
				b.WriteString(strings.TrimSpace(reTags.ReplaceAllString(l, "")) + "\n")
			}
		}
	default:
		return "", fmt.Errorf("unsupported %s", t)
	}
	dst := uniquePath(path, "", "."+strings.ToLower(t))
	return writeAtomic(dst, func(w io.Writer) error { _, err := io.WriteString(w, b.String()); return err })
}

// ---------- archives ----------

func archiveKind(path string) string {
	low := strings.ToLower(path)
	switch {
	case strings.HasSuffix(low, ".zip"):
		return "ZIP"
	case strings.HasSuffix(low, ".rar"):
		return "RAR"
	case strings.HasSuffix(low, ".7z"):
		return "7Z"
	case strings.HasSuffix(low, ".tar.gz") || strings.HasSuffix(low, ".tgz"):
		return "TGZ"
	case strings.HasSuffix(low, ".tar"):
		return "TAR"
	case strings.HasSuffix(low, ".gz"):
		return "GZ"
	}
	return ""
}

func archiveItems(path string) []Item {
	kind := archiveKind(path)
	switch kind {
	case "ZIP", "RAR", "7Z", "TAR", "TGZ", "GZ":
		var out []Item
		for _, t := range []string{"ZIP", "7Z", "RAR", "TAR", "GZIP"} {
			// a bare .gz holds one file, not a tar; re-taring it as
			// .tar.gz makes no sense, and a .tar.gz already is one
			if t == kind || (t == "GZIP" && (kind == "TGZ" || kind == "GZ")) {
				continue
			}
			out = append(out, Item{"to:" + t, t, "", t != "RAR" || RarExe() != ""})
		}
		return out
	}
	return nil
}

func archiveTools(path string) []Item {
	return []Item{{"x:extract", "EXTRACT", "extract", archiveKind(path) != ""}}
}

// entry is one file read from an archive.
type entry struct {
	name    string
	mode    os.FileMode
	modTime time.Time
	dir     bool
	open    func() (io.ReadCloser, error)
}

// readArchive calls fn for every entry. prog (may be nil) receives the
// share of the archive's contents read so far.
func readArchive(path string, prog func(float64), fn func(entry) error) error {
	var tarProg *progCount // TAR: progress by position in the file
	if prog != nil {
		pc := &progCount{f: prog, total: archiveTotal(path)}
		if pc.total > 0 {
			inner := fn
			fn = func(e entry) error {
				open := e.open
				e.open = func() (io.ReadCloser, error) {
					rc, err := open()
					if err != nil {
						return nil, err
					}
					return &countRC{rc, pc}, nil
				}
				return inner(e)
			}
		} else if archiveKind(path) == "TAR" || archiveKind(path) == "TGZ" {
			pc.total = fileSize(path)
			tarProg = pc
		}
	}
	switch archiveKind(path) {
	case "7Z":
		zr, err := sevenzip.OpenReader(path)
		if err != nil {
			return sevenZipError(path, err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			f := f
			fi := f.FileInfo()
			if fi.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if err := fn(entry{f.Name, fi.Mode(), f.Modified, fi.IsDir(), f.Open}); err != nil {
				return sevenZipError(path, err)
			}
		}
		return nil
	case "RAR":
		rc, err := rardecode.OpenReader(path)
		if err != nil {
			return rarError(err)
		}
		defer rc.Close()
		for {
			h, err := rc.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return rarError(err)
			}
			if h.LinkType != 0 { // links are skipped, like in tar
				continue
			}
			open := func() (io.ReadCloser, error) { return io.NopCloser(rc), nil }
			if err := fn(entry{h.Name, h.Mode(), h.ModificationTime, h.IsDir, open}); err != nil {
				return rarError(err)
			}
		}
	case "ZIP":
		zr, err := zip.OpenReader(path)
		if err != nil {
			return err
		}
		defer zr.Close()
		for _, f := range zr.File {
			f := f
			if err := fn(entry{f.Name, f.Mode(), f.Modified, f.FileInfo().IsDir(), f.Open}); err != nil {
				return err
			}
		}
		return nil
	case "TAR", "TGZ":
		fh, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fh.Close()
		var src io.Reader = fh
		if pc := tarProg; pc != nil { // progress by position in the file
			src = &countR{fh, pc}
		}
		var r io.Reader = bufio.NewReader(src)
		if archiveKind(path) == "TGZ" {
			gz, err := gzip.NewReader(r)
			if err != nil {
				return err
			}
			defer gz.Close()
			r = gz
		}
		tr := tar.NewReader(r)
		for {
			h, err := tr.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
				continue
			}
			e := entry{h.Name, os.FileMode(h.Mode), h.ModTime, h.Typeflag == tar.TypeDir,
				func() (io.ReadCloser, error) { return io.NopCloser(tr), nil }}
			if err := fn(e); err != nil {
				return err
			}
		}
	case "GZ":
		// a bare .gz yields the one file it holds
		fh, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fh.Close()
		gz, err := gzip.NewReader(fh)
		if err != nil {
			return err
		}
		defer gz.Close()
		return fn(entry{name: archiveBase(path), mode: 0o644, open: func() (io.ReadCloser, error) {
			return io.NopCloser(gz), nil
		}})
	}
	return errors.New(tr("Unsupported archive format", "صيغة أرشيف غير مدعومة"))
}

// safeJoin refuses entries that would escape the destination folder.
func safeJoin(root, name string) (string, error) {
	p := filepath.Join(root, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(name) {
		return "", fmt.Errorf(tr("Unsafe path inside the archive: %s", "مسار غير آمن داخل الأرشيف: %s"), name)
	}
	return p, nil
}

func archiveBase(path string) string {
	b := filepath.Base(path)
	low := strings.ToLower(b)
	for _, suf := range []string{".tar.gz", ".tgz", ".tar", ".zip", ".rar", ".7z", ".gz"} {
		if strings.HasSuffix(low, suf) {
			return b[:len(b)-len(suf)]
		}
	}
	return b
}

func uniqueDir(parent, base string) string {
	return nameSeries{parent, base, ""}.pick()
}

func runArchive(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	kind := archiveKind(path)
	if id == "x:extract" {
		if kind == "GZ" {
			dst := filepath.Join(filepath.Dir(path), uniqueName(filepath.Dir(path), archiveBase(path)))
			fh, err := os.Open(path)
			if err != nil {
				return "", err
			}
			defer fh.Close()
			gz, err := gzip.NewReader(fh)
			if err != nil {
				return "", err
			}
			defer gz.Close()
			return writeAtomic(dst, func(w io.Writer) error { _, err := io.Copy(w, gz); return err })
		}
		root := uniqueDir(filepath.Dir(path), archiveBase(path))
		tmp := tempFor(root)
		err := readArchive(path, progress, func(e entry) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			p, err := safeJoin(tmp, e.name)
			if err != nil {
				return err
			}
			if e.dir {
				return os.MkdirAll(p, 0o755)
			}
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				return err
			}
			rc, err := e.open()
			if err != nil {
				return err
			}
			defer rc.Close()
			f, err := os.Create(p)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, rc); err != nil {
				f.Close()
				return err
			}
			f.Close()
			if !e.modTime.IsZero() {
				os.Chtimes(p, e.modTime, e.modTime)
			}
			return nil
		})
		if err != nil {
			dropTemp(tmp)
			return "", err
		}
		return finishTemp(tmp, root)
	}
	t := strings.TrimPrefix(id, "to:")
	if t == "RAR" {
		return toRAR(ctx, path, progress)
	}
	if t == "7Z" {
		return to7Z(ctx, path, progress)
	}
	ext := map[string]string{"ZIP": ".zip", "TAR": ".tar", "GZIP": ".tar.gz"}[t]
	dst := filepath.Join(filepath.Dir(path), uniqueName(filepath.Dir(path), archiveBase(path)+ext))
	return writeAtomic(dst, func(w io.Writer) error {
		switch t {
		case "ZIP":
			zw := zip.NewWriter(w)
			err := readArchive(path, progress, func(e entry) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				h := &zip.FileHeader{Name: e.name, Method: zip.Deflate, Modified: e.modTime}
				if e.dir {
					if !strings.HasSuffix(h.Name, "/") {
						h.Name += "/"
					}
					_, err := zw.CreateHeader(h)
					return err
				}
				h.SetMode(e.mode)
				fw, err := zw.CreateHeader(h)
				if err != nil {
					return err
				}
				rc, err := e.open()
				if err != nil {
					return err
				}
				defer rc.Close()
				_, err = io.Copy(fw, rc)
				return err
			})
			if err != nil {
				return err
			}
			return zw.Close()
		default: // TAR or GZIP (tar.gz)
			var gz *gzip.Writer
			out := w
			if t == "GZIP" {
				gz = gzip.NewWriter(w)
				out = gz
			}
			tw := tar.NewWriter(out)
			err := readArchive(path, progress, func(e entry) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if e.dir {
					return tw.WriteHeader(&tar.Header{Name: strings.TrimSuffix(e.name, "/") + "/", Typeflag: tar.TypeDir, Mode: 0o755, ModTime: e.modTime})
				}
				rc, err := e.open()
				if err != nil {
					return err
				}
				defer rc.Close()
				data, err := io.ReadAll(rc)
				if err != nil {
					return err
				}
				mode := int64(e.mode.Perm())
				if mode == 0 {
					mode = 0o644
				}
				if err := tw.WriteHeader(&tar.Header{Name: e.name, Size: int64(len(data)), Mode: mode, ModTime: e.modTime, Typeflag: tar.TypeReg}); err != nil {
					return err
				}
				_, err = tw.Write(data)
				return err
			})
			if err != nil {
				return err
			}
			if err := tw.Close(); err != nil {
				return err
			}
			if gz != nil {
				return gz.Close()
			}
			return nil
		}
	})
}

func uniqueName(dir, name string) string {
	ext := ""
	base := name
	for _, suf := range []string{".tar.gz", ".zip", ".tar"} {
		if strings.HasSuffix(strings.ToLower(name), suf) {
			ext = name[len(name)-len(suf):]
			base = name[:len(name)-len(suf)]
			break
		}
	}
	if ext == "" {
		ext = filepath.Ext(name)
		base = strings.TrimSuffix(name, ext)
	}
	return filepath.Base(nameSeries{dir, base, ext}.pick())
}

func rarError(err error) error {
	switch {
	case errors.Is(err, rardecode.ErrArchiveEncrypted), errors.Is(err, rardecode.ErrBadPassword):
		return errors.New(tr("This archive is password-protected", "هذا الأرشيف محمي بكلمة مرور"))
	}
	return err
}

// RarHint may return Rar.exe from where WinRAR says it is installed.
var RarHint func() string

// RarExe finds WinRAR's command-line tool. Only WinRAR may create RAR
// archives (the format's license), so ZIP to RAR needs it installed.
func RarExe() string {
	if RarHint != nil {
		if p := RarHint(); p != "" {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	var dirs []string
	for _, v := range []string{"ProgramW6432", "ProgramFiles", "ProgramFiles(x86)"} {
		if d := os.Getenv(v); d != "" {
			dirs = append(dirs, d)
		}
	}
	for _, d := range dirs {
		p := filepath.Join(d, "WinRAR", "Rar.exe")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// toRAR repacks an archive as RAR with WinRAR at its best compression.
func toRAR(ctx context.Context, path string, progress func(float64)) (string, error) {
	unpack := part(progress, 0, 0.4)
	rar := RarExe()
	if rar == "" {
		return "", errors.New(tr("Creating RAR files needs WinRAR installed", "إنشاء ملفات RAR يحتاج تثبيت WinRAR"))
	}
	dir := filepath.Dir(path)
	dst := filepath.Join(dir, uniqueName(dir, archiveBase(path)+".rar"))
	work := tempFor(dst + ".d")
	defer dropTemp(work)
	if err := os.MkdirAll(work, 0o755); err != nil {
		return "", err
	}
	err := readArchive(path, unpack, func(e entry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		p, err := safeJoin(work, e.name)
		if err != nil {
			return err
		}
		if e.dir {
			return os.MkdirAll(p, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		rc, err := e.open()
		if err != nil {
			return err
		}
		defer rc.Close()
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, rc)
		f.Close()
		if err == nil && !e.modTime.IsZero() {
			os.Chtimes(p, e.modTime, e.modTime)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	out := tempFor(dst)
	// a: add, -r: recurse, -ep1: no base folder, -m5: best compression,
	// -ma5: RAR5 format, -idq: quiet, -y: yes to all
	cmd := exec.CommandContext(ctx, rar, "a", "-r", "-ep1", "-m5", "-ma5", "-idq", "-y", out, filepath.Join(work, "*"))
	stop := estimate(progress, 0.4, 0.99, 1+float64(archiveTotal(path))/(12<<20))
	defer stop()
	PrepareCmd(cmd)
	if msg, err := cmd.CombinedOutput(); err != nil {
		dropTemp(out)
		return "", fmt.Errorf(tr("WinRAR couldn't create the archive: %s", "تعذّر على WinRAR إنشاء الأرشيف: %s"), lastLine(string(msg)))
	}
	return finishTemp(out, dst)
}

func sevenZipError(path string, err error) error {
	if err == nil {
		return nil
	}
	var re sevenzip.ReadError
	var rp *sevenzip.ReadError
	encrypted := (errors.As(err, &re) && re.Encrypted) || (errors.As(err, &rp) && rp.Encrypted)
	if encrypted || sevenZipEncrypted(path) || strings.Contains(strings.ToLower(err.Error()), "password") {
		return errors.New(tr("This archive is password-protected", "هذا الأرشيف محمي بكلمة مرور"))
	}
	if strings.Contains(err.Error(), "unsupported compression") {
		return errors.New(tr("This 7z uses a compression method Falak can't read (e.g. PPMd)", "ملف 7z هذا يستخدم طريقة ضغط لا يقرؤها فلك (مثل PPMd)"))
	}
	return err
}

// sevenZipEncrypted looks for the AES coder ID (06 F1 07 01) in the archive's
// header area, which 7-Zip writes when a password is used.
func sevenZipEncrypted(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return false
	}
	n := min(st.Size(), 64<<10)
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, st.Size()-n); err != nil {
		return false
	}
	return bytes.Contains(buf, []byte{0x06, 0xF1, 0x07, 0x01})
}

// ---------- progress while reading archives ----------

type progCount struct {
	f           func(float64)
	done, total int64
	last        float64
}

func (p *progCount) add(n int) {
	p.done += int64(n)
	v := math.Min(float64(p.done)/float64(p.total), 1)
	if v-p.last >= 0.005 {
		p.last = v
		p.f(v * 0.99)
	}
}

type countRC struct {
	io.ReadCloser
	p *progCount
}

func (c *countRC) Read(b []byte) (int, error) {
	n, err := c.ReadCloser.Read(b)
	c.p.add(n)
	return n, err
}

type countR struct {
	r io.Reader
	p *progCount
}

func (c *countR) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	c.p.add(n)
	return n, err
}

// archiveTotal is the unpacked size of everything in a ZIP, 7Z or RAR
// (0 when unknown, e.g. for TAR, which is measured by file position).
func archiveTotal(path string) int64 {
	var n int64
	switch archiveKind(path) {
	case "ZIP":
		zr, err := zip.OpenReader(path)
		if err != nil {
			return 0
		}
		defer zr.Close()
		for _, f := range zr.File {
			n += int64(f.UncompressedSize64)
		}
	case "7Z":
		zr, err := sevenzip.OpenReader(path)
		if err != nil {
			return 0
		}
		defer zr.Close()
		for _, f := range zr.File {
			n += int64(f.UncompressedSize)
		}
	case "RAR":
		files, err := rardecode.List(path)
		if err != nil {
			return 0
		}
		for _, f := range files {
			if !f.UnKnownSize && f.UnPackedSize > 0 {
				n += f.UnPackedSize
			}
		}
	}
	return n
}

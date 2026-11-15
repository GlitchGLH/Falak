package conv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Image-only archives convert to PDF: when every file in a ZIP, RAR or 7Z
// is an image Falak can read, the wheel offers PDF (one image per page, the
// same pages as MERGE PDF). With any other file the option is shown
// disabled. Companion files that systems add on their own (Thumbs.db,
// desktop.ini, .DS_Store, anything under __MACOSX) are ignored (owner's
// decision 2026-11-15).

// archivePDFKind reports whether the archive format can offer PDF.
func archivePDFKind(p string) bool {
	switch archiveKind(p) {
	case "ZIP", "RAR", "7Z":
		return true
	}
	return false
}

// isCompanion reports whether an archive entry is a file a system adds by
// itself, which the PDF check and the PDF ignore.
func isCompanion(name string) bool {
	n := strings.ReplaceAll(name, `\`, "/")
	for _, part := range strings.Split(n, "/") {
		if strings.EqualFold(part, "__MACOSX") {
			return true
		}
	}
	switch strings.ToLower(path.Base(n)) {
	case "thumbs.db", "desktop.ini", ".ds_store":
		return true
	}
	return false
}

// imagesOnly returns the image entries in page order, and whether the list
// qualifies for a PDF: at least one image and nothing but images (after
// leaving out folders and companion files).
func imagesOnly(names []string) ([]string, bool) {
	var imgs []string
	for _, n := range names {
		if isCompanion(n) {
			continue
		}
		if Kind(n) == "" {
			return nil, false
		}
		imgs = append(imgs, n)
	}
	sortPages(imgs)
	return imgs, len(imgs) > 0
}

// sortPages orders entry names like File Explorer orders names, folder by
// folder along the path: numbers counted as numbers (2 before 10), case
// ignored, and a name's extension only breaking ties ("img" before "img 2").
func sortPages(names []string) {
	key := func(n string) []string { return strings.Split(strings.ReplaceAll(n, `\`, "/"), "/") }
	stem := func(s string) string { return strings.TrimSuffix(s, path.Ext(s)) }
	sort.SliceStable(names, func(i, j int) bool {
		a, b := key(names[i]), key(names[j])
		for k := 0; k < len(a) && k < len(b); k++ {
			x, y := a[k], b[k]
			if k == len(a)-1 {
				x = stem(x)
			}
			if k == len(b)-1 {
				y = stem(y)
			}
			if naturalLess(x, y) {
				return true
			}
			if naturalLess(y, x) {
				return false
			}
			if a[k] != b[k] { // same stem: the full name decides
				return naturalLess(a[k], b[k])
			}
		}
		return len(a) < len(b)
	})
}

// archiveNames lists the files (not folders) in an archive without reading
// their contents: ZIP and 7Z read only the index, RAR skips over the data.
func archiveNames(p string) ([]string, error) {
	var names []string
	err := readArchive(p, nil, func(e entry) error {
		if !e.dir && !strings.HasSuffix(e.name, "/") {
			names = append(names, e.name)
		}
		return nil
	})
	return names, err
}

type archiveKey struct {
	path string
	size int64
	mod  time.Time
}

type archiveScan struct {
	images []string
	ok     bool
}

var (
	archiveMu    sync.Mutex
	archiveCache = map[archiveKey]archiveScan{}
)

// archiveImages is imagesOnly for an archive file. The wheel asks again and
// again while it is open, so each archive is read once per version (path,
// size, modification time).
func archiveImages(p string) archiveScan {
	st, err := os.Stat(p)
	if err != nil {
		return archiveScan{}
	}
	k := archiveKey{p, st.Size(), st.ModTime()}
	archiveMu.Lock()
	s, hit := archiveCache[k]
	archiveMu.Unlock()
	if hit {
		return s
	}
	t := time.Now()
	names, err := archiveNames(p)
	if err == nil {
		s.images, s.ok = imagesOnly(names)
	}
	Logf("archive %s: %d entries, images only=%v, read in %v (err %v)", filepath.Base(p), len(names), s.ok, time.Since(t).Round(time.Millisecond), err)
	archiveMu.Lock()
	if len(archiveCache) > 64 {
		archiveCache = map[archiveKey]archiveScan{}
	}
	archiveCache[k] = s
	archiveMu.Unlock()
	return s
}

// archiveHasSVG reports whether an image-only archive holds an SVG drawing
// (its PDF then needs the SVG engine).
func archiveHasSVG(p string) bool {
	if !archivePDFKind(p) {
		return false
	}
	for _, n := range archiveImages(p).images {
		if Kind(n) == "SVG" {
			return true
		}
	}
	return false
}

// archiveToPDF extracts the images into a hidden temp folder and writes
// them as one PDF beside the archive, one image per page.
func archiveToPDF(ctx context.Context, p string, progress func(float64)) (string, error) {
	s := archiveImages(p)
	if !s.ok {
		// not reachable from the wheel (the option is disabled then); the
		// message has no approved Arabic text yet, so it stays English
		return "", errors.New("This archive doesn't hold only images")
	}
	page := map[string]int{}
	for i, n := range s.images {
		page[n] = i
	}
	dir := filepath.Dir(p)
	dst := filepath.Join(dir, uniqueName(dir, archiveBase(p)+".pdf"))
	work := tempFor(dst + ".images")
	if err := os.MkdirAll(work, 0o755); err != nil {
		return "", err
	}
	defer dropTemp(work)
	// each image is saved as <page number><its extension>: no name from the
	// archive is used as a path, and two entries with the same name stay two
	files := make([]string, len(s.images))
	err := readArchive(p, part(progress, 0, 0.3), func(e entry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		i, want := page[e.name]
		if e.dir || !want || files[i] != "" {
			return nil
		}
		f := filepath.Join(work, fmt.Sprintf("%06d%s", i, strings.ToLower(path.Ext(e.name))))
		rc, err := e.open()
		if err != nil {
			return err
		}
		defer rc.Close()
		out, err := os.Create(f)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		files[i] = f
		return nil
	})
	if err != nil {
		return "", err
	}
	for i, f := range files {
		if f == "" {
			// the archive changed while it was read; English only, like above
			return "", fmt.Errorf("Couldn't read %s from the archive", s.images[i])
		}
	}
	return writeAtomic(dst, func(w io.Writer) error {
		err := imagesToPDF(ctx, w, files, part(progress, 0.3, 1))
		if err != nil {
			// name the image as it is in the archive, not its temp name
			for i, f := range files {
				if strings.HasPrefix(err.Error(), filepath.Base(f)+":") {
					return fmt.Errorf("%s%s", s.images[i], strings.TrimPrefix(err.Error(), filepath.Base(f)))
				}
			}
		}
		return err
	})
}

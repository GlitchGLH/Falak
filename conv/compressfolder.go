package conv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Compressing two or more files together puts the compressed copies in a
// new folder, "Compressed 2026-11-15 05-30", beside the originals; files
// from different folders get one such folder in each of their folders.
// Compressing one file is unchanged (the copy goes beside it).

// IsCompress reports whether id is the Compress tool of any file kind.
func IsCompress(id string) bool {
	switch id {
	case "compress", "v:compress", "a:compress", "p:compress":
		return true
	}
	return false
}

// CompressFolderName is the folder's name for a job started at t (local time).
func CompressFolderName(t time.Time) string {
	return "Compressed " + t.Format("2006-01-02 15-04")
}

// CompressFolders creates one dated folder in each folder that holds one of
// paths and returns them by source folder. An existing folder of that name
// is never reused: the " (n)" series continues. A folder that could not be
// created is left out (its files are then compressed beside the originals).
func CompressFolders(paths []string, t time.Time) map[string]string {
	out := map[string]string{}
	for _, p := range paths {
		dir := filepath.Dir(p)
		if _, done := out[dir]; done {
			continue
		}
		out[dir] = ""
		if f, err := makeFolder(dir, CompressFolderName(t)); err == nil {
			out[dir] = f
		} else {
			Logf("compress folder in %s: %v", dir, err)
		}
	}
	return out
}

// makeFolder creates dir/name, or the next free name of its " (n)" series.
func makeFolder(dir, name string) (string, error) {
	s := nameSeries{dir, name, ""}
	for tries := 0; tries < 100; tries++ {
		p := s.pick()
		err := os.Mkdir(p, 0o755)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
		// taken between pick and Mkdir: try the next name
	}
	return "", errors.New("no free folder name")
}

type outDirKey struct{}

// WithOutDir makes the Compress tool write its result into dir instead of
// beside the original.
func WithOutDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, outDirKey{}, dir)
}

// outPath is uniquePath, inside the folder set by WithOutDir when there is one.
func outPath(ctx context.Context, src, suffix, ext string) string {
	if d, _ := ctx.Value(outDirKey{}).(string); d != "" {
		base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src)) + suffix
		return nameSeries{d, base, ext}.pick()
	}
	return uniquePath(src, suffix, ext)
}

package conv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// HidePath marks a path hidden (set by the Windows app; no-op elsewhere).
var HidePath = func(string) {}

const tempDirName = ".falak-tmp"

var tempSeq atomic.Int64

// tempName is the shape of every name tempFor makes: "<UnixNano>-<seq>"
// plus whatever follows (an extension, "-pass…", ".docx", ".d" …). The
// startup cleanup deletes only names of this shape inside .falak-tmp.
var tempName = regexp.MustCompile(`^[0-9]{15,20}-[0-9]+`)

// tempFor returns a path for work-in-progress output that belongs at dst.
// It lives in a hidden folder next to dst (same drive, so the final move is
// instant), so half-written files never show up in Explorer. The folder is
// recorded so a later start can remove what an interrupted job left.
func tempFor(dst string) string {
	parent := filepath.Dir(dst)
	dir := filepath.Join(parent, tempDirName)
	if err := os.MkdirAll(dir, 0o755); err == nil {
		HidePath(dir)
		recordTempDir(parent)
	}
	n := tempSeq.Add(1)
	return filepath.Join(dir, fmt.Sprintf("%d-%d%s", time.Now().UnixNano(), n, filepath.Ext(dst)))
}

// retryWait is how long a rename or delete keeps trying while another
// program (a folder watcher, an antivirus scan of the new file) holds it.
var retryWait = 5 * time.Second

func retry(op func() error) error {
	deadline := time.Now().Add(retryWait)
	for {
		err := op()
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// removeTemp deletes tmp (a file or folder), retrying while it is held,
// then the .falak-tmp folder if that is now empty.
func removeTemp(tmp string) {
	if err := retry(func() error {
		err := os.RemoveAll(tmp)
		if err == nil {
			if _, statErr := os.Lstat(tmp); statErr == nil {
				return fmt.Errorf("%s is still there", tmp)
			}
		}
		return err
	}); err != nil {
		Logf("temp: could not remove %s: %v", tmp, err)
	}
	os.Remove(filepath.Dir(tmp)) // only succeeds once the folder is empty
}

// finishTemp moves finished work into place and tidies the hidden folder.
// It never replaces anything: if dst was created while the job ran (by
// another job or program), the work gets the next free name, made the same
// way as at job start (uniquePath, uniqueDir, uniqueName). It returns the
// path used.
func finishTemp(tmp, dst string) (string, error) {
	final := dst
	err := retry(func() error {
		final = dst
		if _, err := os.Lstat(final); err == nil {
			final = seriesFor(dst).pick()
		}
		return renameNoReplace(tmp, final)
	})
	if err != nil {
		Logf("temp: could not move %s to %s: %v", tmp, final, err)
		removeTemp(tmp)
		return "", err
	}
	if final != dst {
		Logf("temp: %s was created during the job; the result is %s", dst, final)
	}
	os.Remove(filepath.Dir(tmp)) // only succeeds once the folder is empty
	return final, nil
}

// renameNoReplace moves from to to and fails (os.ErrExist) if to exists.
// The Windows version asks Windows for exactly that in one step.
var renameNoReplace = func(from, to string) error {
	if _, err := os.Lstat(to); err == nil {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: os.ErrExist}
	}
	return os.Rename(from, to)
}

// ---------- output names ----------

// nameSeries is how an output is named: dir/base+ext, else
// "base (1)ext", "base (2)ext" …
type nameSeries struct{ dir, base, ext string }

func (s nameSeries) name(i int) string {
	if i == 0 {
		return filepath.Join(s.dir, s.base+s.ext)
	}
	return filepath.Join(s.dir, fmt.Sprintf("%s (%d)%s", s.base, i, s.ext))
}

var (
	seriesMu sync.Mutex
	seriesOf = map[string]nameSeries{} // each picked name -> its series
)

// pick returns the first free name of the series and remembers the series,
// so finishTemp can continue it if that name is taken by the end.
func (s nameSeries) pick() string {
	for i := 0; ; i++ {
		p := s.name(i)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			seriesMu.Lock()
			seriesOf[p] = s
			seriesMu.Unlock()
			return p
		}
	}
}

// seriesFor returns the series dst was picked from (or, for a name made
// elsewhere, dst's own name followed by " (n)").
func seriesFor(dst string) nameSeries {
	seriesMu.Lock()
	s, ok := seriesOf[dst]
	seriesMu.Unlock()
	if ok {
		return s
	}
	ext := filepath.Ext(dst)
	if st, err := os.Stat(dst); err == nil && st.IsDir() {
		ext = ""
	}
	return nameSeries{filepath.Dir(dst), strings.TrimSuffix(filepath.Base(dst), ext), ext}
}

// dropTemp discards failed or cancelled work.
func dropTemp(tmp string) { removeTemp(tmp) }

// ---------- leftovers from interrupted jobs ----------

// TempRecord is the file listing the folders that got a .falak-tmp folder
// (set by the app; empty: nothing is recorded).
var TempRecord string

var (
	recMu    sync.Mutex
	recSeen  map[string]bool
	recFresh bool
)

func readRecord() []string {
	b, err := os.ReadFile(TempRecord)
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func writeRecord(dirs []string) {
	if len(dirs) == 0 {
		os.Remove(TempRecord)
		return
	}
	os.MkdirAll(filepath.Dir(TempRecord), 0o755)
	os.WriteFile(TempRecord, []byte(strings.Join(dirs, "\n")+"\n"), 0o644)
}

func recordTempDir(parent string) {
	if TempRecord == "" {
		return
	}
	recMu.Lock()
	defer recMu.Unlock()
	if !recFresh {
		recSeen = map[string]bool{}
		for _, d := range readRecord() {
			recSeen[strings.ToLower(d)] = true
		}
		recFresh = true
	}
	key := strings.ToLower(filepath.Clean(parent))
	if recSeen[key] {
		return
	}
	recSeen[key] = true
	writeRecord(append(readRecord(), filepath.Clean(parent)))
}

// CleanStaleTemp removes what interrupted jobs left behind (the app ended
// mid-job, a crash, a shutdown): in every recorded folder's .falak-tmp, only
// entries whose name has tempFor's shape and that are older than maxAge;
// then the .falak-tmp folder when it is empty, and the folder's record when
// no .falak-tmp is left. In the system temp folder: old falak-pass-* and
// falak-palette-*.png. Nothing else is ever touched. Run at start-up, when
// no job of this Falak is running.
func CleanStaleTemp(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	recMu.Lock()
	defer recMu.Unlock()
	var keep []string
	for _, parent := range readRecord() {
		dir := filepath.Join(parent, tempDirName)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				keep = append(keep, parent) // e.g. a drive that is not connected now
			}
			continue
		}
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || !tempName.MatchString(e.Name()) || info.ModTime().After(cutoff) {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if err := os.RemoveAll(p); err == nil {
				Logf("temp: removed leftover %s", p)
			}
		}
		os.Remove(dir) // only when empty
		if _, err := os.Stat(dir); err == nil {
			keep = append(keep, parent)
		}
	}
	writeRecord(keep)
	recFresh = false

	tmp := os.TempDir()
	entries, _ := os.ReadDir(tmp)
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, passDirPrefix) && !(strings.HasPrefix(n, "falak-palette-") && strings.HasSuffix(n, ".png")) {
			continue
		}
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			os.RemoveAll(filepath.Join(tmp, n))
		}
	}
}

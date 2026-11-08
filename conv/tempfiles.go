package conv

import (
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
func finishTemp(tmp, dst string) error {
	err := retry(func() error { return os.Rename(tmp, dst) })
	if err != nil {
		Logf("temp: could not move %s to %s: %v", tmp, dst, err)
		removeTemp(tmp)
		return err
	}
	os.Remove(filepath.Dir(tmp)) // only succeeds once the folder is empty
	return nil
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

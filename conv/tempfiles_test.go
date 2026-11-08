package conv

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestDropTempWaitsForLock: a failed or cancelled job's file that another
// program still holds (an antivirus scan of a big file just closed) is
// removed once it is released, and the empty .falak-tmp folder goes too.
func TestDropTempWaitsForLock(t *testing.T) {
	dir := t.TempDir()
	tmp := tempFor(filepath.Join(dir, "out.wmv"))
	if err := os.WriteFile(tmp, []byte("half-written"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	go func() { time.Sleep(300 * time.Millisecond); f.Close() }()
	start := time.Now()
	dropTemp(tmp)
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("the temp file is still there")
	}
	if _, err := os.Stat(filepath.Join(dir, tempDirName)); !os.IsNotExist(err) {
		t.Fatalf(".falak-tmp is still there")
	}
	if runtime.GOOS == "windows" && time.Since(start) < 250*time.Millisecond {
		t.Errorf("removed while still held? took %v", time.Since(start))
	}
}

// TestCleanStaleTemp: at start-up only old entries named like tempFor's
// output are removed from recorded .falak-tmp folders; other files and new
// entries stay; an emptied .falak-tmp goes and its folder leaves the record.
func TestCleanStaleTemp(t *testing.T) {
	oldRec := TempRecord
	TempRecord = filepath.Join(t.TempDir(), "temp-folders.txt")
	defer func() { TempRecord = oldRec; recFresh = false }()
	recFresh = false

	a, b := t.TempDir(), t.TempDir()
	tempFor(filepath.Join(a, "x.mp4")) // records a and b, creates their .falak-tmp
	tempFor(filepath.Join(b, "x.mp4"))
	old := time.Now().Add(-2 * time.Hour)
	mk := func(dir, name string, mod time.Time, isDir bool) string {
		p := filepath.Join(dir, tempDirName, name)
		if isDir {
			os.MkdirAll(filepath.Join(p, "sub"), 0o755)
		} else {
			os.WriteFile(p, []byte("x"), 0o644)
		}
		os.Chtimes(p, mod, mod)
		return p
	}
	staleWMV := mk(a, "1793971885472228000-5.wmv", old, false)
	staleLog := mk(a, "1794084335458884300-3.mp4-pass-0.log.mbtree.temp", old, false)
	staleDir := mk(a, "1794084335458884300-7", old, true)
	fresh := mk(a, "1794200000000000000-1.mp4", time.Now(), false)
	foreign := mk(a, "notes.txt", old, false)
	onlyStale := mk(b, "1793971885472228000-9.mkv", old, false)

	CleanStaleTemp(10 * time.Minute)

	for _, p := range []string{staleWMV, staleLog, staleDir, onlyStale} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("stale %s was kept", filepath.Base(p))
		}
	}
	for _, p := range []string{fresh, foreign} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s was removed", filepath.Base(p))
		}
	}
	if _, err := os.Stat(filepath.Join(b, tempDirName)); !os.IsNotExist(err) {
		t.Error("the emptied .falak-tmp in b was kept")
	}
	rec, _ := os.ReadFile(TempRecord)
	if !strings.Contains(strings.ToLower(string(rec)), strings.ToLower(a)) || strings.Contains(strings.ToLower(string(rec)), strings.ToLower(b)) {
		t.Errorf("record should keep only a: %q", rec)
	}
}

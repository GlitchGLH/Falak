package conv

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestCompletePassLogs reproduces the "Compress to a set size" failure seen
// in NVIDIA's capture folder: x264 could not rename its pass-1 statistics
// from "*.temp" because another program had the new file open, so pass 2
// failed with "can't open stats file". completePassLogs must finish the
// rename, waiting while the file is held, and leave other files alone.
func TestCompletePassLogs(t *testing.T) {
	dir := t.TempDir()
	passlog := filepath.Join(dir, "pass")
	write := func(name string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("pass-0.log.temp")
	write("pass-0.log.mbtree.temp")
	write("other.temp")

	// Another program holds the stats file open for a moment (on Windows an
	// os.Open handle does not allow renaming, like the watcher's handle).
	f, err := os.Open(filepath.Join(dir, "pass-0.log.temp"))
	if err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		time.Sleep(400 * time.Millisecond)
		f.Close()
		close(released)
	}()

	if err := completePassLogs(passlog, 5*time.Second); err != nil {
		t.Fatalf("completePassLogs: %v", err)
	}
	<-released
	for _, name := range []string{"pass-0.log", "pass-0.log.mbtree", "other.temp"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s missing: %v", name, err)
		}
	}
	for _, name := range []string{"pass-0.log.temp", "pass-0.log.mbtree.temp"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s still there", name)
		}
	}
}

// TestCompletePassLogsGivesUp: a file held longer than the wait is reported
// as an error instead of letting pass 2 fail with a vague message.
func TestCompletePassLogsGivesUp(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("an open handle blocks renaming only on Windows")
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "pass-0.log.temp")
	os.WriteFile(p, []byte("x"), 0o644)
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := completePassLogs(filepath.Join(dir, "pass"), 300*time.Millisecond); err == nil {
		t.Fatal("a held stats file was reported as renamed")
	}
}

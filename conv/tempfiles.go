package conv

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// HidePath marks a path hidden (set by the Windows app; no-op elsewhere).
var HidePath = func(string) {}

const tempDirName = ".falak-tmp"

var tempSeq atomic.Int64

// tempFor returns a path for work-in-progress output that belongs at dst.
// It lives in a hidden folder next to dst (same drive, so the final move is
// instant), so half-written files never show up in Explorer.
func tempFor(dst string) string {
	dir := filepath.Join(filepath.Dir(dst), tempDirName)
	if err := os.MkdirAll(dir, 0o755); err == nil {
		HidePath(dir)
	}
	n := tempSeq.Add(1)
	return filepath.Join(dir, fmt.Sprintf("%d-%d%s", time.Now().UnixNano(), n, filepath.Ext(dst)))
}

// finishTemp moves finished work into place and tidies the hidden folder.
func finishTemp(tmp, dst string) error {
	err := os.Rename(tmp, dst)
	if err != nil {
		os.RemoveAll(tmp)
	}
	os.Remove(filepath.Dir(tmp)) // only succeeds once the folder is empty
	return err
}

// dropTemp discards failed work.
func dropTemp(tmp string) {
	os.RemoveAll(tmp)
	os.Remove(filepath.Dir(tmp))
}

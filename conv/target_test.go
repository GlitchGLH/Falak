package conv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestCompressTargetSize: compressing to a set size must land at or just
// under the target (here 2 MB for an 8-second clip).
func TestCompressTargetSize(t *testing.T) {
	clip := makeClip(t, 8)
	out, err := RunWithPlan(context.Background(), clip, "v:compress", CompressPlan{TargetMB: 2}, nil)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	defer os.Remove(out)
	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// the two-pass logs must not stay behind in the hidden work folder
	if _, err := os.Stat(filepath.Join(filepath.Dir(clip), tempDirName)); !os.IsNotExist(err) {
		left, _ := os.ReadDir(filepath.Join(filepath.Dir(clip), tempDirName))
		t.Fatalf("%s still exists after compressing (%d files left)", tempDirName, len(left))
	}
	mb := float64(st.Size()) / 1048576
	t.Logf("target 2.00 MB, actual %.2f MB", mb)
	if mb > 2.2 || mb < 1.0 {
		t.Fatalf("actual %.2f MB too far from the 2 MB target", mb)
	}
}

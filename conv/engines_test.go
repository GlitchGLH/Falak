package conv

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestDownloadChecksHash: a download is kept only when its SHA-256 is the
// pinned one; otherwise the file is deleted and the error is the existing
// "The download didn't finish" message.
func TestDownloadChecksHash(t *testing.T) {
	body := []byte("falak engine archive")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer srv.Close()
	good := sha256.Sum256(body)
	dst := filepath.Join(t.TempDir(), "dl.zip")

	if err := download(enginePin{"x", srv.URL, hex.EncodeToString(good[:])}, dst, nil); err != nil {
		t.Fatalf("matching hash: %v", err)
	}
	if b, _ := os.ReadFile(dst); string(b) != string(body) {
		t.Fatalf("downloaded %q", b)
	}

	bad := enginePin{"x", srv.URL, "00" + hex.EncodeToString(good[1:])}
	err := download(bad, dst, nil)
	if err == nil || err.Error() != errDownloadIncomplete().Error() {
		t.Fatalf("wrong hash: got %v, want %q", err, errDownloadIncomplete())
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("a download with the wrong hash was left on disk")
	}
}

// TestEnginePinFile: an engine folder counts as installed only with the
// current pin file; copies from earlier builds (no pin file) or another
// pin are downloaded again.
func TestEnginePinFile(t *testing.T) {
	old := ToolsDir
	ToolsDir = t.TempDir()
	defer func() { ToolsDir = old }()
	p := enginePin{"eng", "https://example.invalid/e.zip", "abc123"}
	exe := filepath.Join(ToolsDir, "eng", "eng.exe")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("x"), 0o644)

	if haveEngine("", exe, p) {
		t.Fatal("no pin file (earlier build) counted as installed")
	}
	enginePin{"eng", p.URL, "def456"}.mark(filepath.Dir(exe))
	if haveEngine("", exe, p) {
		t.Fatal("another pin counted as installed")
	}
	p.mark(filepath.Dir(exe))
	if !haveEngine("", exe, p) {
		t.Fatal("current pin not recognised")
	}
	os.Remove(exe)
	if haveEngine("", exe, p) {
		t.Fatal("pin file without the program counted as installed")
	}
	if haveEngine(exe+".missing", "", p) {
		t.Fatal("a missing override counted as installed")
	}
}

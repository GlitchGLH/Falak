package conv

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func getText(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func putText(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFinishNeverReplaces: a file that appears under the output name while
// the job runs is kept; the result takes the next free name, the same
// series as at job start ("name (1)", "name (2)" …).
func TestFinishNeverReplaces(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "clip.mp4")
	dst := uniquePath(src, "-compressed", ".mp4")
	if filepath.Base(dst) != "clip-compressed.mp4" {
		t.Fatalf("start name %s", dst)
	}
	tmp := tempFor(dst)
	putText(t, tmp, "new")
	putText(t, dst, "made during the job")
	putText(t, filepath.Join(dir, "clip-compressed (1).mp4"), "also there")
	got, err := finishTemp(tmp, dst)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "clip-compressed (2).mp4" {
		t.Fatalf("result %s, want clip-compressed (2).mp4", filepath.Base(got))
	}
	if getText(t, dst) != "made during the job" || getText(t, filepath.Join(dir, "clip-compressed (1).mp4")) != "also there" {
		t.Fatal("an existing file was replaced")
	}
	if getText(t, got) != "new" {
		t.Fatal("the result is not the finished work")
	}
	if _, err := os.Stat(filepath.Join(dir, tempDirName)); !os.IsNotExist(err) {
		t.Fatalf("%s left behind", tempDirName)
	}
}

// TestFinishFreeName: with the name still free, the result is exactly dst.
func TestFinishFreeName(t *testing.T) {
	dir := t.TempDir()
	dst := uniquePath(filepath.Join(dir, "a.png"), "", ".jpg")
	tmp := tempFor(dst)
	putText(t, tmp, "x")
	got, err := finishTemp(tmp, dst)
	if err != nil || got != dst {
		t.Fatalf("got %s, %v; want %s", got, err, dst)
	}
}

// TestWriteAtomicNeverReplaces: the same through writeAtomic, with the
// other file created while the output is being written.
func TestWriteAtomicNeverReplaces(t *testing.T) {
	dir := t.TempDir()
	dst := uniquePath(filepath.Join(dir, "notes.txt"), "", ".pdf")
	got, err := writeAtomic(dst, func(w io.Writer) error {
		putText(t, dst, "user's file")
		_, err := io.WriteString(w, "converted")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "notes (1).pdf" {
		t.Fatalf("result %s, want notes (1).pdf", filepath.Base(got))
	}
	if getText(t, dst) != "user's file" || getText(t, got) != "converted" {
		t.Fatal("wrong contents")
	}
}

// TestFinishFolderNeverReplaces: a results folder ("doc - PNG") that
// appears during the job is kept with its contents.
func TestFinishFolderNeverReplaces(t *testing.T) {
	dir := t.TempDir()
	root := uniqueDir(dir, "doc - PNG")
	tmp := tempFor(root)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	putText(t, filepath.Join(tmp, "doc - 1.png"), "page")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	putText(t, filepath.Join(root, "keep.txt"), "keep")
	got, err := finishTemp(tmp, root)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "doc - PNG (1)" {
		t.Fatalf("result %s, want doc - PNG (1)", filepath.Base(got))
	}
	if getText(t, filepath.Join(root, "keep.txt")) != "keep" || getText(t, filepath.Join(got, "doc - 1.png")) != "page" {
		t.Fatal("wrong contents")
	}
}

// TestFinishNameMadeElsewhere: a name not picked by uniquePath continues
// with its own " (n)".
func TestFinishNameMadeElsewhere(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "x.txt")
	tmp := tempFor(dst)
	putText(t, tmp, "new")
	putText(t, dst, "old")
	got, err := finishTemp(tmp, dst)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "x (1).txt" || getText(t, dst) != "old" {
		t.Fatalf("result %s", got)
	}
}

// TestRenameNoReplace: the move itself refuses an existing name and leaves
// both files as they were.
func TestRenameNoReplace(t *testing.T) {
	dir := t.TempDir()
	from, to := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	putText(t, from, "a")
	putText(t, to, "b")
	if err := renameNoReplace(from, to); !errors.Is(err, os.ErrExist) {
		t.Fatalf("err = %v, want os.ErrExist", err)
	}
	if getText(t, from) != "a" || getText(t, to) != "b" {
		t.Fatal("files changed")
	}
	os.Remove(to)
	if err := renameNoReplace(from, to); err != nil || getText(t, to) != "a" {
		t.Fatalf("move to a free name: %v", err)
	}
}

package conv

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestSnapshotAt: the snapshot of frame k is the frame the window shows for
// k (FrameAt at the same seek position), and snapshots never replace each
// other.
func TestSnapshotAt(t *testing.T) {
	clip := makeClip(t, 4)
	info := ProbeVideo(clip)
	ctx := context.Background()
	k := 50
	f, err := FrameAt(ctx, clip, info.FrameSeek(k), 4000, 4000, true)
	if err != nil {
		t.Fatal(err)
	}
	next, err := FrameAt(ctx, clip, info.FrameSeek(k+1), 4000, 4000, true)
	if err != nil {
		t.Fatal(err)
	}
	out, err := SnapshotAt(ctx, clip, info.FrameSeek(k))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip-snapshot.png"); out != want {
		t.Fatalf("snapshot %s, want %s", out, want)
	}
	fh, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	pimg, err := png.Decode(fh)
	fh.Close()
	if err != nil {
		t.Fatal(err)
	}
	snap := image.NewRGBA(pimg.Bounds())
	for y := 0; y < snap.Rect.Dy(); y++ {
		for x := 0; x < snap.Rect.Dx(); x++ {
			snap.Set(x, y, pimg.At(x, y))
		}
	}
	same, other := diffImages(snap, f), diffImages(next, f)
	t.Logf("snapshot vs preview %.2f, next frame vs preview %.2f", same, other)
	if same > 2 || other < same*3+0.5 {
		t.Fatalf("the snapshot is not the previewed frame (diff %.2f; next frame %.2f)", same, other)
	}
	out2, err := SnapshotAt(ctx, clip, info.FrameSeek(k+1))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip-snapshot (1).png"); out2 != want {
		t.Fatalf("second snapshot %s, want %s", out2, want)
	}

}

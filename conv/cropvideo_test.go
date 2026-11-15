package conv

import (
	"context"
	"image"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCropVideo: "<name>-cropped" beside the original holds exactly the
// chosen area (the same pixels as that part of the original's frame), the
// sound copied as it is; never replacing a file; odd numbers become even.
func TestCropVideo(t *testing.T) {
	clip := makeClip(t, 2)
	ctx := context.Background()
	if w, h := VideoSize(clip); w != 320 || h != 180 {
		t.Fatalf("VideoSize %dx%d", w, h)
	}
	out, err := CropVideo(ctx, clip, 40, 20, 160, 90, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(clip), "clip-cropped.mp4"); out != want {
		t.Errorf("output %s, want %s", out, want)
	}
	if w, h := VideoSize(out); w != 160 || h != 90 {
		t.Errorf("cropped to %dx%d, want 160x90", w, h)
	}
	mi, src := probe(out), probe(clip)
	if !mi.hasAudio || mi.acodec != src.acodec || mi.abr != src.abr {
		t.Errorf("sound %v %s %d kb/s, want the original %s %d kb/s copied", mi.hasAudio, mi.acodec, mi.abr, src.acodec, src.abr)
	}
	a, _ := FrameAt(ctx, out, 1, 4000, 4000, true)
	full, _ := FrameAt(ctx, clip, 1, 4000, 4000, true)
	if a == nil || full == nil {
		t.Fatal("frames missing")
	}
	part := image.NewRGBA(image.Rect(0, 0, 160, 90))
	wrong := image.NewRGBA(image.Rect(0, 0, 160, 90))
	for y := 0; y < 90; y++ {
		copy(part.Pix[y*part.Stride:(y+1)*part.Stride], full.Pix[full.PixOffset(40, 20+y):full.PixOffset(200, 20+y)])
		copy(wrong.Pix[y*wrong.Stride:(y+1)*wrong.Stride], full.Pix[full.PixOffset(80, 60+y):full.PixOffset(240, 60+y)])
	}
	if d1, d2 := diffImages(a, part), diffImages(a, wrong); d1 >= d2/3 {
		t.Errorf("not the chosen area: %.2f against it, %.2f against another", d1, d2)
	}
	out2, err := CropVideo(ctx, clip, 41, 21, 101, 51, nil)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(out2) != "clip-cropped (1).mp4" {
		t.Errorf("second output %s", filepath.Base(out2))
	}
	if w, h := VideoSize(out2); w != 100 || h != 50 {
		t.Errorf("odd size gave %dx%d, want 100x50", w, h)
	}
}

// TestVideoSizeRotated: a video stored sideways with a rotation flag (as
// phones make them) has its sides swapped, as FFmpeg's filters see it,
// and the crop works on that picture.
func TestVideoSizeRotated(t *testing.T) {
	clip := makeClip(t, 1)
	rot := filepath.Join(filepath.Dir(clip), "rot.mp4")
	cmd := exec.Command("ffmpeg", "-y", "-loglevel", "error", "-display_rotation:v:0", "90", "-i", clip, "-c", "copy", rot)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	if w, h := VideoSize(rot); w != 180 || h != 320 {
		t.Fatalf("rotated: %dx%d, want 180x320", w, h)
	}
	out, err := CropVideo(context.Background(), rot, 0, 100, 180, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := VideoSize(out); w != 180 || h != 100 {
		t.Errorf("rotated crop %dx%d, want 180x100", w, h)
	}
}

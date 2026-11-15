package conv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"math"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/image/bmp"
)

// The shared base of the video windows (Trim, Snapshots, later Split, Speed
// and Crop): the video's length and frame rate, single frames for the
// preview, a strip of thumbnails, and the cuts (the time display is
// ui.FormatClock). Frames
// come from the installed FFmpeg, one short run per picture: on a 1080p
// H.264 video an exact frame takes about 0.15-0.2 s (measured 2026-11-15).

// VideoInfo is what the video windows need to know about a video.
type VideoInfo struct {
	Duration float64 // seconds (0 = unknown)
	FPS      float64 // frames per second (0 = unknown)
}

// ProbeVideo reads the video's length and frame rate.
func ProbeVideo(path string) VideoInfo {
	mi := probe(path)
	return VideoInfo{Duration: mi.duration, FPS: mi.fps}
}

// FrameRate is the frame rate used for frame steps: the video's own, else 30.
func (v VideoInfo) FrameRate() float64 {
	if v.FPS > 0 && v.FPS < 1000 {
		return v.FPS
	}
	return 30
}

// Frames is how many frames the video has (at least 1).
func (v VideoInfo) Frames() int {
	return max(1, int(math.Floor(v.Duration*v.FrameRate()+1e-6)))
}

// FrameTime is the time of frame k, which is also what the time display shows.
func (v VideoInfo) FrameTime(k int) float64 { return float64(k) / v.FrameRate() }

// FrameSeek is the seek position that lands exactly on frame k: FFmpeg's
// accurate seek returns the first frame at or after the position, so the
// position sits a quarter frame before the frame's own time.
func (v VideoInfo) FrameSeek(k int) float64 {
	return math.Max(0, (float64(k)-0.25)/v.FrameRate())
}

// FrameAt returns the picture at t seconds, scaled to fit maxW×maxH (never
// enlarged). exact=false takes the key frame at or before t instead (much
// faster; good enough for thumbnails).
func FrameAt(ctx context.Context, path string, t float64, maxW, maxH int, exact bool) (*image.RGBA, error) {
	if !HaveFFmpeg() {
		return nil, errors.New("FFmpeg is not available")
	}
	var args []string
	if !exact {
		args = append(args, "-noaccurate_seek")
	}
	args = append(args, "-ss", fmt.Sprintf("%.3f", math.Max(0, t)), "-i", path, "-frames:v", "1", "-an", "-sn",
		"-vf", fmt.Sprintf("scale=w='min(iw,%d)':h='min(ih,%d)':force_original_aspect_ratio=decrease:flags=bilinear", maxW, maxH),
		"-c:v", "bmp", "-f", "image2pipe", "pipe:1")
	cmd := exec.CommandContext(ctx, FFmpegPath(), append([]string{"-hide_banner", "-nostdin", "-loglevel", "error"}, args...)...)
	PrepareCmd(cmd)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		Logf("frame at %.3f of %s failed: %v %s", t, filepath.Base(path), err, strings.TrimSpace(stderr.String()))
		return nil, err
	}
	if out.Len() == 0 {
		return nil, errors.New("no frame at this time")
	}
	img, err := bmp.Decode(&out)
	if err != nil {
		return nil, err
	}
	rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(rgba, rgba.Rect, img, img.Bounds().Min, draw.Src)
	return rgba, nil
}

// ThumbTimes spreads n thumbnails evenly: each one shows the middle of its
// share of the video.
func ThumbTimes(duration float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = duration * (float64(i) + 0.5) / float64(n)
	}
	return out
}

// Thumbs reads n thumbnails (key frames, fast) of at most maxW×maxH, three
// at a time, and hands each to got as it arrives (any order, from other
// goroutines). It returns when all are done or ctx ends.
func Thumbs(ctx context.Context, path string, duration float64, n, maxW, maxH int, got func(i int, img *image.RGBA)) {
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for i, t := range ThumbTimes(duration, n) {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, t float64) {
			defer wg.Done()
			defer func() { <-sem }()
			if img, err := FrameAt(ctx, path, t, maxW, maxH, false); err == nil {
				got(i, img)
			}
		}(i, t)
	}
	wg.Wait()
}

// OpensWindow reports whether a wheel item opens a video window instead of
// running at once (the window then starts the work itself).
func OpensWindow(id string) bool {
	switch id {
	case "v:trim", "v:snapshot":
		return true
	}
	return false
}

// trimExt keeps the container when it holds H.264 + AAC well, else MP4.
func trimExt(path string) string {
	switch e := strings.ToLower(filepath.Ext(path)); e {
	case ".mp4", ".mov", ".mkv":
		return e
	}
	return ".mp4"
}

// MinTrim is the shortest piece Trim makes (seconds).
const MinTrim = 0.1

// TrimVideo keeps start..end (seconds) as "<name>-trimmed" beside the
// original. The cut is exact: the piece is re-encoded at a visually
// lossless quality (like the split's exact cut), never cut at the nearest
// key frame.
func TrimVideo(ctx context.Context, path string, start, end float64, progress func(float64)) (string, error) {
	if end-start < MinTrim {
		return "", errors.New(tr("The selection is too short", "المقطع المحدد قصير جدًا"))
	}
	dst := uniquePath(path, "-trimmed", trimExt(path))
	length := end - start
	args := []string{"-t", fmt.Sprintf("%.3f", length), "-map", "0:v:0", "-map", "0:a:0?",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "16", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "192k"}
	if trimExt(path) != ".mkv" {
		args = append(args, "-movflags", "+faststart")
	}
	return ffmpegPre(ctx, []string{"-ss", fmt.Sprintf("%.3f", start)}, path, dst, args, length, progress)
}

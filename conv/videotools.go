package conv

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The Split Video and Change Video Speed windows' jobs, and the Speed
// window's playing preview.

// SplitGap is the shortest part the Split window allows (seconds): a split
// point stays at least this far from the start, the end and the other
// points. It is a whole number of frames, at least MinTrim.
func SplitGap(info VideoInfo) float64 {
	fr := info.FrameRate()
	return math.Max(1, math.Ceil(MinTrim*fr-1e-9)) / fr
}

// SplitVideo cuts the video at the points (seconds) into len(points)+1
// parts, "<name> - part N" in a new "<name> - parts" folder beside it
// (never replacing one: " (1)" is added). The cuts are exact: the video is
// re-encoded at a visually lossless quality with a key frame forced at
// every point (as Trim does), never cut at the nearest key frame.
func SplitVideo(ctx context.Context, path string, points []float64, progress func(float64)) (string, error) {
	mi := probe(path)
	pts := append([]float64(nil), points...)
	sort.Float64s(pts)
	if len(pts) == 0 {
		return "", errors.New(tr("Add at least one split point", "أضف نقطة تقسيم واحدة على الأقل"))
	}
	prev := 0.0
	for _, p := range append(pts, mi.duration) {
		if p-prev < MinTrim/2 { // the window keeps parts longer (SplitGap)
			return "", errors.New(tr("The selection is too short", "المقطع المحدد قصير جدًا"))
		}
		prev = p
	}
	times := make([]string, len(pts))
	for i, p := range pts {
		times[i] = fmt.Sprintf("%.3f", p)
	}
	list := strings.Join(times, ",")
	ext := trimExt(path)
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	root := uniqueDir(filepath.Dir(path), base+" - parts")
	tmp := tempFor(root)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(tmp, strings.ReplaceAll(base, "%", "%%")+" - part %d"+ext)
	args := []string{"-map", "0:v:0", "-map", "0:a:0?", "-force_key_frames", list,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "16", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "192k",
		"-f", "segment", "-segment_times", list, "-segment_start_number", "1", "-reset_timestamps", "1"}
	if ext != ".mkv" {
		args = append(args, "-segment_format_options", "movflags=+faststart")
	}
	if err := ffmpegRaw(ctx, nil, path, out, args, mi.duration, progress); err != nil {
		dropTemp(tmp)
		return "", err
	}
	return finishTemp(tmp, root)
}

// Speeds are the Change Video Speed window's choices, in order.
var Speeds = []float64{0.25, 0.5, 0.75, 1.25, 1.5, 2, 3, 4}

// SpeedText shows a speed the way the window and the file name do: "0.5",
// "2", "1.25".
func SpeedText(speed float64) string { return strconv.FormatFloat(speed, 'f', -1, 64) }

// atempoChain keeps the sound's pitch while changing its speed: FFmpeg's
// atempo is chained so every step stays within 0.5..2 (0.25 = 0.5 × 0.5,
// 3 = 2 × 1.5, 4 = 2 × 2).
func atempoChain(speed float64) string {
	var steps []string
	for speed < 0.5-1e-9 {
		steps = append(steps, "atempo=0.5")
		speed /= 0.5
	}
	for speed > 2+1e-9 {
		steps = append(steps, "atempo=2")
		speed /= 2
	}
	return strings.Join(append(steps, "atempo="+strconv.FormatFloat(speed, 'f', -1, 64)), ",")
}

// SpeedVideo plays the video at speed (0.25 = four times slower, 4 = four
// times faster) into "<name>-<speed>x.mp4" beside it, never replacing a
// file. The sound keeps its pitch; a video without sound stays silent. The
// frame rate stays the video's own: faster drops frames, slower repeats them.
func SpeedVideo(ctx context.Context, path string, speed float64, progress func(float64)) (string, error) {
	if speed <= 0 {
		return "", errors.New(tr("This tool isn't available yet", "هذه الأداة غير متاحة بعد"))
	}
	mi := probe(path)
	fr := VideoInfo{FPS: mi.fps}.FrameRate()
	dst := uniquePath(path, "-"+SpeedText(speed)+"x", ".mp4")
	args := []string{"-map", "0:v:0", "-map", "0:a:0?",
		"-filter:v", fmt.Sprintf("setpts=PTS/%s,fps=%s", SpeedText(speed), strconv.FormatFloat(fr, 'f', -1, 64)),
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "18", "-pix_fmt", "yuv420p", "-movflags", "+faststart"}
	if mi.hasAudio {
		args = append(args, "-filter:a", atempoChain(speed), "-c:a", "aac", "-b:a", "192k")
	}
	return ffmpeg(ctx, path, dst, args, mi.duration/speed, progress)
}

// PreviewFPS is how many pictures a second the playing preview shows.
const PreviewFPS = 24

// PlayPreview plays the video silently from from seconds at speed, as w×h
// pictures (fitted, on black), handing each to show at its moment with the
// time in the video it comes from. It returns at the end of the video, or
// when ctx ends.
func PlayPreview(ctx context.Context, path string, from, speed float64, w, h int, show func(img *image.RGBA, at float64)) error {
	if !HaveFFmpeg() {
		return errors.New("FFmpeg is not available")
	}
	vf := fmt.Sprintf("setpts=(PTS-STARTPTS)/%s,fps=%d,scale=%d:%d:force_original_aspect_ratio=decrease:flags=bilinear,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black",
		SpeedText(speed), PreviewFPS, w, h, w, h)
	cmd := exec.CommandContext(ctx, FFmpegPath(), "-hide_banner", "-nostdin", "-loglevel", "error",
		"-ss", fmt.Sprintf("%.3f", math.Max(0, from)), "-i", path, "-an", "-sn", "-vf", vf,
		"-pix_fmt", "rgba", "-f", "rawvideo", "pipe:1")
	PrepareCmd(cmd)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Wait()
	r := bufio.NewReaderSize(pipe, w*h*4)
	start := time.Now()
	for i := 0; ; i++ {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		if _, err := io.ReadFull(r, img.Pix); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil // the end of the video
		}
		due := start.Add(time.Duration(float64(i) / PreviewFPS * float64(time.Second)))
		if d := time.Until(due); d > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
			}
		}
		show(img, from+float64(i)/PreviewFPS*speed)
	}
}

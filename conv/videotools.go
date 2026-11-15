package conv

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The Split Video window's job.

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

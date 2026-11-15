package conv

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// The Crop Video window's job.

var (
	reVideoSize = regexp.MustCompile(`Stream #\S+.*?: Video: [^\n]*?, (\d{2,5})x(\d{2,5})[ ,\[]`)
	reRotation  = regexp.MustCompile(`rotation of (-?[\d.]+) degrees`)
)

// MinCrop is the smallest crop side in pixels (when the video is at least
// that big): the Crop window never goes smaller, so no message is needed.
const MinCrop = 32

// VideoSize is the picture's size as it is shown (and as FFmpeg's filters
// see it): a video stored sideways with a rotation flag (phones) has its
// sides swapped. 0, 0 when unknown.
func VideoSize(path string) (int, int) {
	cmd := exec.Command(FFmpegPath(), "-hide_banner", "-i", path)
	PrepareCmd(cmd)
	out, _ := cmd.CombinedOutput()
	s := string(out)
	m := reVideoSize.FindStringSubmatchIndex(s)
	if m == nil {
		return 0, 0
	}
	w, _ := strconv.Atoi(s[m[2]:m[3]])
	h, _ := strconv.Atoi(s[m[4]:m[5]])
	rest := s[m[1]:] // this stream's side data, up to the next stream
	if i := strings.Index(rest, "Stream #"); i >= 0 {
		rest = rest[:i]
	}
	if r := reRotation.FindStringSubmatch(rest); r != nil {
		deg, _ := strconv.ParseFloat(r[1], 64)
		if q := int(math.Round(math.Abs(deg)/90)) % 2; q == 1 {
			w, h = h, w
		}
	}
	return w, h
}

// CropVideo keeps the w×h area at x, y (pixels of the picture as shown;
// even numbers) into "<name>-cropped" beside the video (.mp4/.mov/.mkv
// kept, else .mp4), never replacing a file. The picture is re-encoded at
// Trim's quality; the sound is copied as it is (re-encoded to AAC only if
// the output format can't hold it).
func CropVideo(ctx context.Context, path string, x, y, w, h int, progress func(float64)) (string, error) {
	even := func(v int) int { return v &^ 1 }
	x, y, w, h = even(x), even(y), even(w), even(h)
	if w <= 0 || h <= 0 || x < 0 || y < 0 {
		return "", errors.New(tr("This tool isn't available yet", "هذه الأداة غير متاحة بعد"))
	}
	mi := probe(path)
	ext := trimExt(path)
	dst := uniquePath(path, "-cropped", ext)
	args := func(audio ...string) []string {
		a := []string{"-map", "0:v:0", "-map", "0:a:0?",
			"-filter:v", fmt.Sprintf("crop=%d:%d:%d:%d", w, h, x, y),
			"-c:v", "libx264", "-preset", "veryfast", "-crf", "16", "-pix_fmt", "yuv420p"}
		a = append(a, audio...)
		if ext != ".mkv" {
			a = append(a, "-movflags", "+faststart")
		}
		return a
	}
	out, err := ffmpeg(ctx, path, dst, args("-c:a", "copy"), mi.duration, progress)
	if err != nil && ctx.Err() == nil && mi.hasAudio {
		// the sound's format doesn't fit this container (e.g. WMA in MP4)
		out, err = ffmpeg(ctx, path, dst, args("-c:a", "aac", "-b:a", "192k"), mi.duration, progress)
	}
	return out, err
}

package conv

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ---------- categories ----------

var videoExts = map[string]bool{".mp4": true, ".mov": true, ".mkv": true, ".webm": true, ".avi": true, ".wmv": true,
	".m4v": true, ".flv": true, ".mpg": true, ".mpeg": true, ".ts": true, ".3gp": true, ".mts": true}
var audioExts = map[string]bool{".mp3": true, ".m4a": true, ".wav": true, ".flac": true, ".ogg": true, ".opus": true,
	".aiff": true, ".aif": true, ".wma": true, ".aac": true}

// Category returns "image", "video", "audio", "subtitle", "archive" or "".
func Category(path string) string {
	e := strings.ToLower(filepath.Ext(path))
	low := strings.ToLower(path)
	switch {
	case imageExts[e] != "":
		return "image"
	case videoExts[e]:
		return "video"
	case audioExts[e]:
		return "audio"
	case e == ".srt" || e == ".vtt" || e == ".ass" || e == ".ssa":
		return "subtitle"
	case e == ".txt":
		return "text"
	case e == ".pdf":
		return "pdf"
	case officeExts[e] != "":
		return "office"
	case e == ".zip" || e == ".rar" || e == ".7z" || e == ".tar" || e == ".tgz" || strings.HasSuffix(low, ".tar.gz") || e == ".gz":
		return "archive"
	}
	return ""
}

// NeedsFFmpeg reports whether running id on path uses FFmpeg.
func NeedsFFmpeg(path, id string) bool {
	c := Category(path)
	if c == "video" || c == "audio" {
		return true
	}
	t := strings.TrimPrefix(id, "to:")
	if c == "image" && Kind(path) == "HEIC" && (id == "compress" || id == "rotate" || id == "resize") {
		return true // re-saving as HEIC uses FFmpeg's HEVC encoder
	}
	return (c == "image" || c == "pdf") && (t == "HEIC" || (Kind(path) == "GIF" && videoTarget[t]))
}

var videoTarget = map[string]bool{"MP4": true, "MOV": true, "WEBM": true, "MKV": true, "AVI": true, "WMV": true}

// gifToVideo turns an animated GIF into a video at high quality, keeping
// every frame and its own timing.
func gifToVideo(ctx context.Context, path, t string, progress func(float64)) (string, error) {
	mi := probe(path)
	dst := uniquePath(path, "", "."+strings.ToLower(t))
	even := "scale=trunc(iw/2)*2:trunc(ih/2)*2:flags=lanczos"
	var args []string
	switch t {
	case "WEBM": // VP9 keeps the GIF's transparency
		args = []string{"-vf", even + ",format=yuva420p", "-c:v", "libvpx-vp9", "-crf", "15", "-b:v", "0",
			"-deadline", "good", "-cpu-used", "2", "-row-mt", "1", "-auto-alt-ref", "0"}
	case "AVI":
		args = []string{"-vf", even + ",format=yuv420p", "-c:v", "mpeg4", "-q:v", "3"}
	case "WMV":
		args = []string{"-vf", even + ",format=yuv420p", "-c:v", "wmv2", "-q:v", "3"}
	default:
		args = []string{"-vf", even + ",format=yuv420p", "-c:v", "libx264", "-preset", "slow", "-crf", "14",
			"-tune", "animation"}
	}
	args = append(args, "-fps_mode", "vfr", "-an")
	return dst, ffmpeg(ctx, path, dst, args, mi.duration, progress)
}

func extUpper(path string) string {
	return strings.ToUpper(strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."))
}

func formatItems(targets []string, own string) []Item {
	var out []Item
	for _, t := range targets {
		if t == own {
			continue
		}
		out = append(out, Item{ID: "to:" + t, Label: t, Enabled: true})
	}
	return out
}

var videoTargets = []string{"MP4", "MOV", "MKV", "WEBM", "AVI", "WMV", "GIF", "MP3"}

var audioTargets = []string{"MP3", "M4A", "WAV", "FLAC", "OGG", "OPUS", "AIFF", "WMA"}

func normalizeOwn(path string) string {
	switch u := extUpper(path); u {
	case "AIF":
		return "AIFF"
	case "M4V":
		return "MP4"
	default:
		return u
	}
}

// ---------- FFmpeg location and download ----------

// ToolsDir is where downloaded helper programs live (set by the app).
var ToolsDir = filepath.Join(os.TempDir(), "falak-tools")

// FFmpegOverride lets tests use a system ffmpeg.
var FFmpegOverride string

// PrepareCmd is called on every external command before it starts (the
// Windows app uses it to hide console windows).
var PrepareCmd = func(*exec.Cmd) {}

func exeName(n string) string {
	if runtime.GOOS == "windows" {
		return n + ".exe"
	}
	return n
}

func FFmpegPath() string {
	if FFmpegOverride != "" {
		return FFmpegOverride
	}
	return filepath.Join(ToolsDir, "ffmpeg", exeName("ffmpeg"))
}

func HaveFFmpeg() bool { return haveEngine(FFmpegOverride, FFmpegPath(), ffmpegPin) }

// EnsureFFmpeg downloads FFmpeg once (about 100 MB, see ffmpegPin) and
// extracts the programs and libraries it needs. progress receives 0..1.
func EnsureFFmpeg(progress func(float64)) error {
	if HaveFFmpeg() {
		return nil
	}
	return EnsureFFmpegFrom(progress)
}

// EnsureFFmpegFrom always downloads and installs FFmpeg.
func EnsureFFmpegFrom(progress func(float64)) error {
	dir := filepath.Join(ToolsDir, "ffmpeg")
	if err := os.MkdirAll(ToolsDir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(ToolsDir, "ffmpeg-download.zip")
	if err := download(ffmpegPin, tmp, progress); err != nil {
		os.Remove(tmp)
		return fmt.Errorf(tr("Couldn't download the video engine: %w", "تعذّر تنزيل محرك الفيديو: %w"), err)
	}
	defer os.Remove(tmp)
	staging := dir + ".new"
	os.RemoveAll(staging)
	if err := extractBin(tmp, staging); err != nil {
		os.RemoveAll(staging)
		return fmt.Errorf(tr("Couldn't unpack the video engine: %w", "تعذّر فك محرك الفيديو: %w"), err)
	}
	if err := ffmpegPin.mark(staging); err != nil {
		os.RemoveAll(staging)
		return err
	}
	os.RemoveAll(dir)
	return os.Rename(staging, dir)
}

// errDownloadIncomplete is reported for a cut-off download and for one
// whose SHA-256 is not the pinned one (damaged or changed on the server).
func errDownloadIncomplete() error {
	return errors.New(tr("The download didn't finish", "التنزيل لم يكتمل"))
}

// download fetches p.URL into dst and checks it against p.SHA256; on a
// mismatch dst is deleted.
func download(p enginePin, dst string, progress func(float64)) error {
	url := p.URL
	client := &http.Client{Timeout: 30 * time.Minute}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Falak")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	total := resp.ContentLength
	var done int64
	buf := make([]byte, 256<<10)
	last := time.Time{}
	sum := sha256.New()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			sum.Write(buf[:n])
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return werr
			}
			done += int64(n)
			if progress != nil && total > 0 && time.Since(last) > 100*time.Millisecond {
				progress(float64(done) / float64(total))
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	if total > 0 && done != total {
		f.Close()
		return errDownloadIncomplete()
	}
	if err := f.Close(); err != nil {
		return err
	}
	if hex.EncodeToString(sum.Sum(nil)) != p.SHA256 {
		os.Remove(dst)
		return errDownloadIncomplete()
	}
	return nil
}

// extractBin copies every file from the archive's bin/ folder into dir.
func extractBin(zipPath, dir string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zr.Close()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	found := false
	for _, f := range zr.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) < 2 || parts[len(parts)-2] != "bin" || f.FileInfo().IsDir() {
			continue
		}
		name := parts[len(parts)-1]
		if strings.Contains(name, "ffplay") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
		if strings.HasPrefix(name, "ffmpeg") {
			found = true
		}
	}
	if !found {
		return errors.New(tr("ffmpeg wasn't found in the download", "لم يُعثر على ffmpeg داخل الملف"))
	}
	return nil
}

// ---------- running FFmpeg ----------

type mediaInfo struct {
	duration float64
	channels int
	hasVideo bool
	hasAudio bool
	rate     string
	sfmt     string // sample format, e.g. s16, s32, fltp
	abr      int    // audio bitrate in kb/s (0 = unknown)
	vcodec   string
	acodec   string
	fps      float64
}

var (
	reDur     = regexp.MustCompile(`Duration: (\d+):(\d+):(\d+(?:\.\d+)?)`)
	reAudio   = regexp.MustCompile(`Stream #\S+.*?: Audio: (\w+)[^,]*, (\d+) Hz, ([^,]+), (\w+)(?:.*?(\d+) kb/s)?`)
	reVideo   = regexp.MustCompile(`Stream #\S+.*?: Video: (\w+)`)
	reFPS     = regexp.MustCompile(`Stream #\S+.*?: Video: .*?([\d.]+) fps`)
	reOutTime = regexp.MustCompile(`out_time_us=(\d+)`)
)

func probe(path string) mediaInfo {
	cmd := exec.Command(FFmpegPath(), "-hide_banner", "-i", path)
	PrepareCmd(cmd)
	out, _ := cmd.CombinedOutput() // exits with an error because there is no output file
	s := string(out)
	var mi mediaInfo
	if m := reDur.FindStringSubmatch(s); m != nil {
		h, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		sec, _ := strconv.ParseFloat(m[3], 64)
		mi.duration = float64(h*3600+mm*60) + sec
	}
	if m := reFPS.FindStringSubmatch(s); m != nil {
		mi.fps, _ = strconv.ParseFloat(m[1], 64)
	}
	if m := reVideo.FindStringSubmatch(s); m != nil && m[1] != "mjpeg" && m[1] != "png" {
		mi.hasVideo, mi.vcodec = true, m[1]
	}
	if m := reAudio.FindStringSubmatch(s); m != nil {
		mi.hasAudio, mi.acodec, mi.rate, mi.sfmt = true, m[1], m[2], m[4]
		if m[5] != "" {
			mi.abr, _ = strconv.Atoi(m[5])
		}
		layout := strings.TrimSpace(m[3])
		switch {
		case layout == "mono" || layout == "1 channels":
			mi.channels = 1
		case layout == "stereo" || layout == "2 channels":
			mi.channels = 2
		default:
			mi.channels = 6
		}
	}
	return mi
}

// ffmpeg runs FFmpeg writing to dst (via a temporary file), reporting 0..1.
func ffmpeg(ctx context.Context, src, dst string, args []string, duration float64, progress func(float64)) error {
	return ffmpegPre(ctx, nil, src, dst, args, duration, progress)
}

// ffmpegPre is ffmpeg with options placed before -i (e.g. fast seeking).
func ffmpegPre(ctx context.Context, pre []string, src, dst string, args []string, duration float64, progress func(float64)) error {
	tmp := tempFor(dst)
	if err := ffmpegRaw(ctx, pre, src, tmp, args, duration, progress); err != nil {
		dropTemp(tmp)
		return err
	}
	return finishTemp(tmp, dst)
}

// ffmpegRaw runs FFmpeg writing straight to out (a file or a pattern).
func ffmpegRaw(ctx context.Context, pre []string, src, tmp string, args []string, duration float64, progress func(float64)) error {
	full := append([]string{"-hide_banner", "-nostdin", "-y"}, pre...)
	full = append(full, "-i", src)
	full = append(full, args...)
	// progress ten times a second, so the bar follows the real work closely
	full = append(full, "-progress", "pipe:1", "-stats_period", "0.1", "-nostats", tmp)
	cmd := exec.CommandContext(ctx, FFmpegPath(), full...)
	PrepareCmd(cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf(tr("Couldn't start the video engine: %w", "تعذّر تشغيل محرك الفيديو: %w"), err)
	}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if m := reOutTime.FindStringSubmatch(sc.Text()); m != nil && duration > 0 && progress != nil {
			us, _ := strconv.ParseFloat(m[1], 64)
			progress(clamp01(us / 1e6 / duration))
		}
	}
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf(tr("Conversion failed: %s", "فشل التحويل: %s"), lastLine(stderr.String()))
	}
	return nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); l != "" {
			if len(l) > 160 {
				l = l[:160]
			}
			return l
		}
	}
	return tr("Unknown error", "خطأ غير معروف")
}

// ---------- video ----------

var videoEncode = map[string][]string{
	"MP4":  {"-map", "0:v:0?", "-map", "0:a:0?", "-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "192k"},
	"MKV":  {"-map", "0", "-c", "copy"},
	"WEBM": {"-map", "0:v:0?", "-map", "0:a:0?", "-c:v", "libvpx-vp9", "-crf", "32", "-b:v", "0", "-row-mt", "1", "-deadline", "realtime", "-cpu-used", "8", "-c:a", "libopus", "-b:a", "128k"},
	"AVI":  {"-map", "0:v:0?", "-map", "0:a:0?", "-c:v", "mpeg4", "-q:v", "3", "-c:a", "libmp3lame", "-q:a", "2"},
	"WMV":  {"-map", "0:v:0?", "-map", "0:a:0?", "-c:v", "wmv2", "-q:v", "3", "-c:a", "wmav2", "-b:a", "192k"},
	"MP3":  {"-vn", "-c:a", "libmp3lame", "-q:a", "2"},
}

func videoItems(path string) []Item { return formatItems(videoTargets, normalizeOwn(path)) }

func videoTools(path string) []Item {
	return []Item{
		{"v:compress", "COMPRESS", "compress", true},
		{"v:metadata", "METADATA", "metadata", true},
		{"v:mute", "MUTE", "mute", true},
		{"v:speed", "SPEED 2×", "speed", true},
		{"v:snapshot", "SNAPSHOT", "snapshot", true},
		{"v:split", "SPLIT ½", "split", true},
	}
}

func runVideo(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	mi := probe(path)
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case strings.HasPrefix(id, "to:"):
		t := strings.TrimPrefix(id, "to:")
		dst := uniquePath(path, "", "."+strings.ToLower(t))
		if t == "GIF" {
			// palettegen holds every frame in memory, so a long video
			// would crawl and sit at 0%: refuse anything over 10 s
			if mi.duration > 10 {
				return "", errors.New(tr("GIF supports videos up to 10 seconds only", "صيغة GIF تدعم الفيديو حتى 10 ثوانٍ فقط"))
			}
			return dst, videoToGIF(ctx, path, dst, mi, progress)
		}
		if t == "MP3" && !mi.hasAudio {
			return "", errors.New(tr("This video has no audio", "هذا الفيديو لا يحتوي على صوت"))
		}
		return dst, ffmpeg(ctx, path, dst, videoArgs(t, mi), mi.duration, progress)
	case id == "v:compress":
		return runVideoPlan(ctx, path, CompressPlan{}, progress)
	case id == "v:metadata":
		dst := uniquePath(path, "-clean", ext)
		return dst, ffmpeg(ctx, path, dst, []string{"-map", "0", "-map_metadata", "-1", "-map_chapters", "-1", "-c", "copy"}, mi.duration, progress)
	case id == "v:mute":
		dst := uniquePath(path, "-muted", ext)
		return dst, ffmpeg(ctx, path, dst, []string{"-map", "0:v", "-c", "copy", "-an"}, mi.duration, progress)
	case id == "v:snapshot":
		dst := uniquePath(path, "-snapshot", ".png")
		at := 0.0
		if mi.duration > 0 {
			at = mi.duration / 2
		}
		return dst, ffmpegPre(ctx, []string{"-ss", fmt.Sprintf("%.3f", at)}, path, dst, []string{"-frames:v", "1"}, 0, progress)
	case id == "v:speed":
		// twice as fast, sound included (pitch kept)
		dst := uniquePath(path, "-2x", ".mp4")
		args := []string{"-map", "0:v:0", "-map", "0:a:0?", "-filter:v", "setpts=0.5*PTS",
			"-c:v", "libx264", "-preset", "veryfast", "-crf", "18", "-pix_fmt", "yuv420p"}
		if mi.hasAudio {
			args = append(args, "-filter:a", "atempo=2.0", "-c:a", "aac", "-b:a", "192k")
		}
		return dst, ffmpeg(ctx, path, dst, args, mi.duration/2, progress)
	case id == "v:split":
		// two halves, cut without re-encoding (no quality loss) at the
		// key frame nearest the middle, into a folder beside the video
		if mi.duration <= 1 {
			return "", errors.New(tr("This video is too short to split", "هذا الفيديو أقصر من أن يُقسَم"))
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		root := uniqueDir(filepath.Dir(path), base+" - parts")
		tmp := tempFor(root)
		if err := os.MkdirAll(tmp, 0o755); err != nil {
			return "", err
		}
		half := mi.duration / 2
		out := filepath.Join(tmp, strings.ReplaceAll(base, "%", "%%")+" - part %d"+ext)
		seg := []string{"-f", "segment", "-segment_times", fmt.Sprintf("%.3f", math.Max(0.5, half-0.05)),
			"-segment_start_number", "1", "-reset_timestamps", "1"}
		// First try without re-encoding (no quality loss): the cut lands on
		// the key frame nearest the middle.
		err := ffmpegRaw(ctx, nil, path, out, append([]string{"-map", "0", "-c", "copy"}, seg...), mi.duration, func(f float64) {
			if progress != nil {
				progress(f * 0.3)
			}
		})
		if err == nil {
			parts, _ := os.ReadDir(tmp)
			first := filepath.Join(tmp, fmt.Sprintf("%s - part 1%s", base, ext))
			if len(parts) != 2 || math.Abs(probe(first).duration-half) > math.Max(1, mi.duration*0.05) {
				// key frames too far apart for an even cut: re-encode, cutting
				// exactly in the middle (at a visually lossless quality)
				for _, e := range parts {
					os.Remove(filepath.Join(tmp, e.Name()))
				}
				enc := []string{"-map", "0:v:0", "-map", "0:a?", "-force_key_frames", fmt.Sprintf("%.3f", half),
					"-c:v", "libx264", "-preset", "veryfast", "-crf", "16", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "192k"}
				if ext != ".mp4" && ext != ".mov" && ext != ".mkv" {
					out = filepath.Join(tmp, strings.ReplaceAll(base, "%", "%%")+" - part %d.mp4")
				}
				err = ffmpegRaw(ctx, nil, path, out, append(enc, seg...), mi.duration, func(f float64) {
					if progress != nil {
						progress(0.3 + f*0.7)
					}
				})
			}
		}
		if err != nil {
			dropTemp(tmp)
			return "", err
		}
		return root, finishTemp(tmp, root)
	}
	return "", errors.New(tr("This tool isn't available yet", "هذه الأداة غير متاحة بعد"))
}

// ---------- video compress plans (the compress dialog) ----------

// CompressPlan holds the options chosen in the compress dialog: Balanced
// or Strong, or a target size in megabytes (0 = quality mode).
type CompressPlan struct {
	Strong   bool
	TargetMB float64
}

// RunWithPlan runs id on path with the tool's options.
func RunWithPlan(ctx context.Context, path, id string, plan CompressPlan, progress func(float64)) (string, error) {
	if id == "v:compress" {
		return runVideoPlan(ctx, path, plan, progress)
	}
	return Run(ctx, path, id, progress)
}

// VideoDuration returns the video's length in seconds (0 if unknown).
func VideoDuration(path string) float64 {
	return probe(path).duration
}

func runVideoPlan(ctx context.Context, path string, plan CompressPlan, progress func(float64)) (string, error) {
	if plan.TargetMB > 0 {
		return compressTwoPass(ctx, path, plan.TargetMB, progress)
	}
	mi := probe(path)
	dst := uniquePath(path, "-compressed", ".mp4")
	if plan.Strong {
		// Strong: the long side capped at 1280 px too — the biggest win
		// at low bitrates.
		return dst, ffmpeg(ctx, path, dst, []string{"-map", "0:v:0?", "-map", "0:a:0?",
			"-vf", "scale='if(gt(iw,ih),min(1280,iw),-2)':'if(gt(iw,ih),-2,min(1280,ih))'",
			"-c:v", "libx264", "-preset", "veryfast", "-crf", "32",
			"-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "96k"}, mi.duration, progress)
	}
	return dst, ffmpeg(ctx, path, dst, []string{"-map", "0:v:0?", "-map", "0:a:0?",
		"-vf", "scale='if(gt(ih,1080),-2,iw)':'min(ih,1080)'", "-c:v", "libx264", "-preset", "veryfast", "-crf", "28",
		"-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "128k"}, mi.duration, progress)
}

// videoToGIF makes a high-quality GIF in two passes, both with a moving
// progress bar: pass one builds the optimised palette while a null output
// mirrors the input, so the progress keeps getting real times (palettegen
// on its own sits at 0% until the very end), pass two maps the frames
// onto it. Same filters, fps, scale and dithering as the old single
// filter_complex. Cancellation works in both passes.
func videoToGIF(ctx context.Context, path, dst string, mi mediaInfo, progress func(float64)) error {
	fps := "30"
	if mi.fps > 0 && mi.fps < 30 {
		fps = strconv.FormatFloat(mi.fps, 'f', 3, 64)
	}
	vf := "fps=" + fps + ",scale='min(1280,iw)':-2:flags=lanczos"
	pal, err := os.CreateTemp("", "falak-palette-*.png")
	if err != nil {
		return err
	}
	palName := pal.Name()
	pal.Close()
	defer os.Remove(palName)
	nullDev := "/dev/null"
	if runtime.GOOS == "windows" {
		nullDev = "NUL"
	}
	if err := gifPalettePass(ctx, path, palName, vf, nullDev, mi.duration, progress); err != nil {
		return err
	}
	return ffmpegPre(ctx, nil, path, dst, []string{"-i", palName,
		"-filter_complex", "[0:v]" + vf + "[x];[x][1:v]paletteuse=dither=sierra2_4a:diff_mode=rectangle",
		"-an", "-loop", "0"}, mi.duration, func(f float64) {
		if progress != nil {
			progress(0.4 + f*0.6)
		}
	})
}

// gifPalettePass runs the palettegen pass: the palette goes to pal, and a
// second null output consumes the input in step, keeping out_time (and the
// progress bar) moving through the whole pass.
func gifPalettePass(ctx context.Context, src, pal, vf, nullDev string, duration float64, progress func(float64)) error {
	full := []string{"-hide_banner", "-nostdin", "-y",
		"-progress", "pipe:1", "-stats_period", "0.1", "-nostats",
		"-i", src,
		"-filter_complex", "[0:v]" + vf + ",palettegen=max_colors=256:stats_mode=diff[p]",
		"-map", "[p]", "-frames:v", "1", "-update", "1", pal,
		"-map", "0:v", "-f", "null", nullDev}
	cmd := exec.CommandContext(ctx, FFmpegPath(), full...)
	PrepareCmd(cmd)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf(tr("Couldn't start the video engine: %w", "تعذّر تشغيل محرك الفيديو: %w"), err)
	}
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		if m := reOutTime.FindStringSubmatch(sc.Text()); m != nil && duration > 0 && progress != nil {
			us, _ := strconv.ParseFloat(m[1], 64)
			progress(clamp01(us/1e6/duration) * 0.4)
		}
	}
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf(tr("Conversion failed: %s", "فشل التحويل: %s"), lastLine(stderr.String()))
	}
	return nil
}

// compressTwoPass encodes to about the requested size: pass one analyses
// the video, pass two writes it at the exact average bitrate the size
// needs (about 128 kb/s of it goes to the sound).
func compressTwoPass(ctx context.Context, path string, targetMB float64, progress func(float64)) (string, error) {
	mi := probe(path)
	if mi.duration <= 0 {
		return "", errors.New(tr("Couldn't read the video's length", "تعذّر قراءة طول الفيديو"))
	}
	kbits := targetMB*8388.608/mi.duration - 128
	if kbits < 64 {
		return "", errors.New(tr("The target size is too small for this video", "الحجم المطلوب صغير جدًا لهذا الفيديو."))
	}
	dst := uniquePath(path, "-compressed", ".mp4")
	tmp := tempFor(dst)
	defer dropTemp(tmp)
	passlog := tmp + "-pass"
	// FFmpeg names its logs <passlog>-<stream>.log(.mbtree)(.temp); remove
	// every one of them on any exit (done, failed or cancelled), before
	// dropTemp runs, so the hidden .falak-tmp folder can go away.
	defer removePassLogs(passlog)
	nullDev := "/dev/null"
	if runtime.GOOS == "windows" {
		nullDev = "NUL"
	}
	br := fmt.Sprintf("%.0fk", kbits)
	vf := "scale='if(gt(ih,1080),-2,iw)':'min(ih,1080)'"
	// pass 1: analyse only (no sound, no real output file)
	p1 := []string{"-map", "0:v:0?", "-vf", vf, "-c:v", "libx264", "-preset", "veryfast",
		"-b:v", br, "-pass:v", "1", "-passlogfile:v", passlog, "-an", "-f", "null"}
	if err := ffmpegRaw(ctx, nil, path, nullDev, p1, mi.duration, func(f float64) {
		if progress != nil {
			progress(f * 0.45)
		}
	}); err != nil {
		return "", err
	}
	// pass 2: the real encode, into the temporary file
	// -pass:v / -passlogfile:v: only the video is two-pass. A plain "-pass 2"
	// also applies to the AAC sound in current FFmpeg builds, which then looks
	// for a pass-1 log the sound never wrote ("Error opening output files").
	p2 := []string{"-map", "0:v:0?", "-map", "0:a:0?", "-vf", vf, "-c:v", "libx264", "-preset", "veryfast",
		"-b:v", br, "-pass:v", "2", "-passlogfile:v", passlog, "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "128k"}
	if err := ffmpegRaw(ctx, nil, path, tmp, p2, mi.duration, func(f float64) {
		if progress != nil {
			progress(0.45 + f*0.54)
		}
	}); err != nil {
		return "", err
	}
	removePassLogs(passlog)
	return dst, finishTemp(tmp, dst)
}

// removePassLogs deletes the two-pass log files that start with passlog.
// (A directory listing, not filepath.Glob: folder names may contain '['.)
func removePassLogs(passlog string) {
	dir, prefix := filepath.Dir(passlog), filepath.Base(passlog)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// ---------- audio ----------

var audioEncode = map[string][]string{
	"MP3":  {"-c:a", "libmp3lame", "-q:a", "2"},
	"M4A":  {"-c:a", "aac", "-b:a", "192k"},
	"WAV":  {"-c:a", "pcm_s16le"},
	"FLAC": {"-c:a", "flac"},
	"OGG":  {"-c:a", "libvorbis", "-q:a", "5"},
	"OPUS": {"-c:a", "libopus", "-b:a", "128k"},
	"AIFF": {"-c:a", "pcm_s16be"},
	"WMA":  {"-c:a", "wmav2", "-b:a", "192k"},
}

func audioItems(path string) []Item { return formatItems(audioTargets, normalizeOwn(path)) }

func audioTools(path string) []Item {
	return []Item{
		{"a:compress", "COMPRESS", "compress", true},
		{"a:metadata", "METADATA", "metadata", true},
		{"a:normalize", "NORMALIZE", "normalize", true},
		{"a:visualize", "VISUALIZE", "visualize", true},
		{"a:channels", "CHANNELS", "channels", true},
	}
}

// sameFormatArgs re-encodes to the file's own format with quality q
// ("high" keeps quality, "small" lowers the bitrate).
func sameFormatArgs(own string, small bool) (string, []string) {
	if small {
		switch own {
		case "MP3":
			return ".mp3", []string{"-c:a", "libmp3lame", "-b:a", "96k"}
		case "OGG":
			return ".ogg", []string{"-c:a", "libvorbis", "-q:a", "2"}
		case "OPUS":
			return ".opus", []string{"-c:a", "libopus", "-b:a", "64k"}
		case "WMA":
			return ".wma", []string{"-c:a", "wmav2", "-b:a", "96k"}
		default: // WAV, FLAC, AIFF, M4A, AAC -> compact M4A
			return ".m4a", []string{"-c:a", "aac", "-b:a", "128k"}
		}
	}
	if own == "AAC" {
		own = "M4A"
	}
	if a, ok := audioEncode[own]; ok {
		return "." + strings.ToLower(own), a
	}
	return ".m4a", audioEncode["M4A"]
}

func runAudio(ctx context.Context, path, id string, progress func(float64)) (string, error) {
	mi := probe(path)
	own := normalizeOwn(path)
	ext := strings.ToLower(filepath.Ext(path))
	vn := []string{"-vn"}
	switch {
	case strings.HasPrefix(id, "to:"):
		t := strings.TrimPrefix(id, "to:")
		dst := uniquePath(path, "", "."+strings.ToLower(t))
		return dst, ffmpeg(ctx, path, dst, append(vn, audioArgs(t, mi)...), mi.duration, progress)
	case id == "a:compress":
		e, a := sameFormatArgs(own, true)
		dst := uniquePath(path, "-compressed", e)
		return dst, ffmpeg(ctx, path, dst, append(vn, a...), mi.duration, progress)
	case id == "a:metadata":
		dst := uniquePath(path, "-clean", ext)
		return dst, ffmpeg(ctx, path, dst, []string{"-map", "0:a", "-map_metadata", "-1", "-c", "copy"}, mi.duration, progress)
	case id == "a:normalize":
		e, a := sameFormatArgs(own, false)
		dst := uniquePath(path, "-normalized", e)
		rate := mi.rate
		if rate == "" {
			rate = "48000"
		}
		// loudnorm works at 192 kHz internally; keep the file's own rate.
		return dst, ffmpeg(ctx, path, dst, append(append(vn, "-af", "loudnorm=I=-16:TP=-1.5:LRA=11", "-ar", rate), a...), mi.duration, progress)
	case id == "a:channels":
		e, a := sameFormatArgs(own, false)
		suffix, ch := "-mono", "1"
		if mi.channels == 1 {
			suffix, ch = "-stereo", "2"
		}
		dst := uniquePath(path, suffix, e)
		return dst, ffmpeg(ctx, path, dst, append(append(vn, "-ac", ch), a...), mi.duration, progress)
	case id == "a:visualize":
		dst := uniquePath(path, "-waveform", ".png")
		// the picture appears only at the end: follow the expected time
		stop := estimate(progress, 0, 0.99, 0.5+mi.duration/60)
		defer stop()
		return dst, ffmpeg(ctx, path, dst, []string{"-filter_complex",
			"aformat=channel_layouts=mono,showwavespic=s=2400x600:colors=0x007AFF:scale=sqrt", "-frames:v", "1"}, 0, progress)
	}
	return "", errors.New(tr("This tool isn't available yet", "هذه الأداة غير متاحة بعد"))
}

// ---------- quality: never below the source ----------

var losslessAudio = map[string]bool{"flac": true, "alac": true, "wavpack": true, "ape": true, "truehd": true, "tta": true}

func (mi mediaInfo) audioLossless() bool {
	return strings.HasPrefix(mi.acodec, "pcm_") || losslessAudio[mi.acodec]
}

// deepAudio: the source has more than 16 bits of precision.
func (mi mediaInfo) deepAudio() bool {
	switch strings.TrimSuffix(mi.sfmt, "p") {
	case "s32", "s24", "flt", "dbl", "s64":
		return true
	}
	return false
}

// lossyBitrate returns the bitrate to use for a lossy target: the source's
// own bitrate (never lower), or the format's top quality for lossless or
// unknown sources.
func lossyBitrate(mi mediaInfo, lo, hi int) string {
	br := mi.abr
	if br == 0 || mi.audioLossless() || br > hi {
		br = hi
	}
	if br < lo {
		br = lo
	}
	return fmt.Sprintf("%dk", br)
}

// audioArgs gives encoder settings for an audio target at full quality.
func audioArgs(target string, mi mediaInfo) []string {
	switch target {
	case "MP3":
		if mi.acodec == "mp3" {
			return []string{"-c:a", "copy"}
		}
		return []string{"-c:a", "libmp3lame", "-b:a", lossyBitrate(mi, 32, 320)}
	case "M4A":
		if mi.acodec == "aac" {
			return []string{"-c:a", "copy"}
		}
		if mi.audioLossless() {
			return []string{"-c:a", "alac"} // lossless inside M4A
		}
		return []string{"-c:a", "aac", "-b:a", lossyBitrate(mi, 32, 320)}
	case "OGG":
		if mi.acodec == "vorbis" {
			return []string{"-c:a", "copy"}
		}
		return []string{"-c:a", "libvorbis", "-b:a", lossyBitrate(mi, 45, 320)}
	case "OPUS":
		if mi.acodec == "opus" {
			return []string{"-c:a", "copy"}
		}
		return []string{"-c:a", "libopus", "-b:a", lossyBitrate(mi, 16, 320)}
	case "WMA":
		return []string{"-c:a", "wmav2", "-b:a", lossyBitrate(mi, 32, 320)}
	case "WAV":
		if mi.deepAudio() {
			return []string{"-c:a", "pcm_s24le"}
		}
		return []string{"-c:a", "pcm_s16le"}
	case "AIFF":
		if mi.deepAudio() {
			return []string{"-c:a", "pcm_s24be"}
		}
		return []string{"-c:a", "pcm_s16be"}
	case "FLAC":
		if mi.deepAudio() {
			return []string{"-c:a", "flac", "-sample_fmt", "s32"}
		}
		return []string{"-c:a", "flac"}
	}
	return nil
}

var mp4Video = map[string]bool{"h264": true, "hevc": true, "av1": true, "mpeg4": true}
var mp4Audio = map[string]bool{"aac": true, "mp3": true, "ac3": true, "eac3": true, "alac": true}
var webmVideo = map[string]bool{"vp8": true, "vp9": true, "av1": true}
var webmAudio = map[string]bool{"opus": true, "vorbis": true}

// videoArgs gives settings for a video target: an exact stream copy when the
// new container can hold the streams, otherwise visually lossless encoding.
func videoArgs(target string, mi mediaInfo) []string {
	audioOK := func(set map[string]bool) bool { return !mi.hasAudio || set[mi.acodec] }
	maps := []string{"-map", "0:v:0?", "-map", "0:a?"}
	switch target {
	case "MP4", "MOV":
		if mp4Video[mi.vcodec] && audioOK(mp4Audio) {
			return append(maps, "-c", "copy")
		}
		a := []string{"-c:a", "aac", "-b:a", lossyBitrate(mi, 64, 320)}
		if mp4Audio[mi.acodec] {
			a = []string{"-c:a", "copy"}
		}
		v := []string{"-c:v", "libx264", "-preset", "slow", "-crf", "16", "-pix_fmt", "yuv420p"}
		if mp4Video[mi.vcodec] {
			v = []string{"-c:v", "copy"}
		}
		return append(append(maps, v...), a...)
	case "MKV":
		return append(maps, "-c", "copy") // MKV holds any codec: exact copy
	case "WEBM":
		if webmVideo[mi.vcodec] && audioOK(webmAudio) {
			return append(maps, "-c", "copy")
		}
		v := []string{"-c:v", "libvpx-vp9", "-crf", "18", "-b:v", "0", "-row-mt", "1", "-deadline", "good", "-cpu-used", "2"}
		if webmVideo[mi.vcodec] {
			v = []string{"-c:v", "copy"}
		}
		a := []string{"-c:a", "libopus", "-b:a", lossyBitrate(mi, 64, 320)}
		if webmAudio[mi.acodec] {
			a = []string{"-c:a", "copy"}
		}
		return append(append(maps, v...), a...)
	case "AVI":
		a := []string{"-c:a", "libmp3lame", "-b:a", lossyBitrate(mi, 64, 320)}
		if mi.acodec == "mp3" {
			a = []string{"-c:a", "copy"}
		}
		return append(append(maps, "-c:v", "mpeg4", "-q:v", "1"), a...)
	case "WMV":
		return append(maps, "-c:v", "wmv2", "-q:v", "1", "-c:a", "wmav2", "-b:a", lossyBitrate(mi, 64, 320))
	case "MP3":
		return append([]string{"-vn"}, audioArgs("MP3", mi)...)
	}
	return nil
}

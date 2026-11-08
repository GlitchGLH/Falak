package conv

import (
	"os"
	"path/filepath"
	"strings"
)

// The helper programs are downloaded from fixed, numbered releases and each
// download is checked against its SHA-256, so an upstream update can never
// change (or break) what Falak runs. To move to a newer release, change the
// URL and the hash together, then run the download test:
//
//	go test -tags enginesnet -run TestEnginePins -v ./conv
type enginePin struct {
	Dir    string // the engine's folder under ToolsDir
	URL    string
	SHA256 string // of the downloaded archive, lowercase hex
}

var (
	// FFmpeg 9.0.2 "full_build-shared" by gyan.dev (GPL; has libx264,
	// libx265, libvpx, libmp3lame, libopus, libvorbis). Numbered releases
	// in this repository are kept permanently.
	ffmpegPin = enginePin{"ffmpeg",
		"https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-full_build-shared.zip",
		"8d31e162f1616e37aab3fa2db991b97e1b8dbeb1c8465fd81a81be16ff91b328"}
	pdfiumPin = enginePin{"pdfium",
		"https://github.com/bblanchon/pdfium-binaries/releases/download/chromium/8138/pdfium-win-x64.tgz",
		"8d48973e3eb472dda0f5f2435e595ac043f9f2219dd71883392f549445afa8a5"}
	qpdfPin = enginePin{"qpdf",
		"https://github.com/qpdf/qpdf/releases/download/v12.3.2/qpdf-12.3.2-mingw64.zip",
		"ebeb4692434a8cb8a27f876584c0c2b402b6251dc6c8ed4b39cf2304cada10e0"}
	resvgPin = enginePin{"resvg",
		"https://github.com/linebender/resvg/releases/download/v0.49.0/resvg-win-x86_64.zip",
		"38af9631b37e61118a32a5ef31e6be6af083eda63b4c17ffc4fdc63e449277df"}
)

// pinFile sits in each engine's folder and names the download it came from.
const pinFile = "falak-engine.txt"

// installed reports whether the engine's folder was filled from exactly
// this pin. Copies downloaded by earlier builds (from "latest" links, with
// no pin file) or from another pin count as missing and are replaced.
func (p enginePin) installed() bool {
	b, err := os.ReadFile(filepath.Join(ToolsDir, p.Dir, pinFile))
	if err != nil {
		return false
	}
	f := strings.Fields(string(b))
	return len(f) > 0 && f[0] == p.SHA256
}

// mark records the pin in dir: the engine's folder, or the staging folder
// that is renamed to it. Written last, so a half-finished install never
// looks complete.
func (p enginePin) mark(dir string) error {
	return os.WriteFile(filepath.Join(dir, pinFile), []byte(p.SHA256+"\n"+p.URL+"\n"), 0o644)
}

// haveEngine: an override (tests) only has to exist; a downloaded engine
// must also carry the current pin.
func haveEngine(override, path string, p enginePin) bool {
	if override != "" {
		_, err := os.Stat(override)
		return err == nil
	}
	_, err := os.Stat(path)
	return err == nil && p.installed()
}

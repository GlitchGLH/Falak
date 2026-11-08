# HANDOVER

## Paths
- Project (this repo): `/home/user/workspace/falak-build/falak-src` (Go module `raha`)
- Dependencies: `/home/user/workspace/falak-build/deps` — must sit NEXT TO `falak-src` (go.mod `replace` directives point to `../deps/...`). They cannot be swapped for public modules: several pins (`gen2brain/avif|heic|webp v0.0.0`, `srwiley/oksvg|rasterx v0.0.0`) do not exist upstream (wasm2go builds generated for this project).

## Build Falak.exe
From the repo root, with Go >= 1.24 on PATH:

    GOFLAGS=-mod=mod GOPROXY=off GOSUMDB=off GOMEMLIMIT=2500MiB \
    GOOS=windows GOARCH=amd64 go build -p 1 -tags "wasm2go nodynamic" \
      -ldflags "-s -w -H windowsgui" -trimpath -o FalakSetup-staging/Falak.exe .

The checked-in `rsrc_windows_amd64.syso` (icon + VERSIONINFO 1.0.0.0 + manifest) is picked up by the linker automatically; `res/raha.rc` mirrors it for reference.

## Build FalakSetup-1.0.0.exe (NSIS 3.x)
1. Put a built `Falak.exe` into `installer/`.
2. From the repo root run (absolute script path recommended):

       makensis -INPUTCHARSET UTF8 "$(pwd)/installer/falak.nsi"

   The script resolves its assets (icon `installer/raha-nsis.ico`, license
   `installer/license.txt`, `installer/Falak.exe`) relative to its own
   directory (`${__FILEDIR__}`) and writes `FalakSetup-1.0.0.exe` two
   directory levels above `installer/`.

## Preview tools (cmd/, run from the repo root: `go run -p 1 ./cmd/<name>`)
- `preview` — wheel over a sample desktop → PNG files (see its source for output paths)
- `themeprev` — wheel light/dark → `/tmp/theme.png`
- `menuprev` — tray menu (EN/AR, dark/light, with logo) → `/tmp/menu.png` (arg = Inter fonts variant)
- `headerprev` — menu header sheet (EN/AR × dark/light) → `../previews/task3-header.png`
- `compressprev` — Compress dialog grid; needs fonts:
  `go run -p 1 ./cmd/compressprev -o out.png -bold /usr/share/fonts/opentype/fonts-hosny-amiri/Amiri-Bold.ttf -semi /usr/share/fonts/opentype/fonts-hosny-amiri/Amiri-Regular.ttf -debug=false`
- `helpprev`, `panelprev`, `iconprev`, `icons` — help card / panel / icon sheets
- Fonts used by the app's UI renders: Inter (Latin), Amiri for Arabic previews; DejaVu as fallback.

## What Falak writes OUTSIDE its own folder (uninstaller must remove all)
| Location | What |
|---|---|
| `HKCU\Software\Falak` | values: `Language`, `Theme`, `SoundOff`, `UseHelper`, `Welcomed`, `RelaunchedAt` |
| `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` | value `Falak` = `"<install>\Falak.exe" --startup` (legacy value `Raha` is deleted by the app) |
| `HKCU\...\Explorer\StartupApproved\Run` | value `Falak` (startup enable marker) |
| `HKCU\...\AppCompatFlags\Layers` | only ever *removed* (Run-as-admin marks on `\falak.exe`) |
| `%LOCALAPPDATA%\Programs\Falak` | self-installed copy `Falak.exe` (`.new`/`.old` transient) |
| `%LOCALAPPDATA%\Falak` | `falak.log` (+`.old`), crash reports |
| Scheduled task `Falak\KeysHelper` | created via `schtasks` only by the opt-in elevated key helper |
| `%TEMP%\falak-tools` | downloaded engines: FFmpeg, PDFium, qpdf, resvg |
| session-only | mutex `Local\FalakFileWheel`; reads `Themes\Personalize\AppsUseLightTheme`; no shell extensions / file associations |

## Standing rules
- One commit per task.
- Commit trailer: `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` only — never a Claude-Session line.
- All files UTF-8.
- Never touch any Arabic string unless the exact replacement text is given; show every changed/new Arabic line in full.
- **Never modify the sound code.** `sound_windows.go`, `res/*.wav` and every call site of `playHover` / `playAppear` / `playDrop` / `soundKeepAlive` / `soundOn` are frozen verbatim at commit `4bd539b` (the tested-good build). No redesign, no "improvements", no volume changes; if a sound task ever comes up again, restore from `4bd539b` instead of editing.

## Final state (2026-11-07)
- Final head: `1.0.0: restore sounds from 4bd539b` — the sound code (engine, `res/`, and every call site, incl. the menu sound toggle playing `playHover()` at its old level) is byte-identical to `4bd539b`; `confirmPCM` / `playConfirm` are gone.
- Kept from this session, untouched: compress target size fix, tray menu animation, GIF 10 s cap + progress, solid popups, licenses file, shortcut/capture fixes.
- Version stays 1.0.0; `FalakSetup-1.0.0.exe`, the portable zip, `SHA256SUMS.txt` and the two GitHub-upload batches are rebuilt from this head.
- User-side verification only: `installer/verify-install.ps1` on real Windows; VirusTotal stays manual (no API key).
